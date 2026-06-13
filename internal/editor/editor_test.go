package editor

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"crime-scene-cleaner/internal/patcher"
)

type fakeRunner struct {
	outputs map[string]runResult
	calls   []string
}

type runResult struct {
	out string
	err error
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	result := f.outputs[call]
	return result.out, result.err
}

func TestCheckToolingReportsPythonAndUnityPyAvailable(t *testing.T) {
	runner := &fakeRunner{outputs: map[string]runResult{
		"python --version":         {out: "Python 3.12.10"},
		"python -c import UnityPy": {out: ""},
	}}

	status := CheckTooling(context.Background(), runner, "python")

	if !status.PythonAvailable {
		t.Fatal("expected PythonAvailable")
	}
	if !status.UnityPyAvailable {
		t.Fatal("expected UnityPyAvailable")
	}
	if !status.Ready {
		t.Fatal("expected Ready")
	}
}

func TestCheckToolingReportsMissingUnityPy(t *testing.T) {
	runner := &fakeRunner{outputs: map[string]runResult{
		"python --version":         {out: "Python 3.12.10"},
		"python -c import UnityPy": {err: errTest("missing module")},
	}}

	status := CheckTooling(context.Background(), runner, "python")

	if !status.PythonAvailable {
		t.Fatal("expected PythonAvailable")
	}
	if status.UnityPyAvailable {
		t.Fatal("expected UnityPyAvailable to be false")
	}
	if status.Ready {
		t.Fatal("expected Ready to be false")
	}
}

func TestMarshalRowsPreservesLargeUnityIdsAsStrings(t *testing.T) {
	rows := []TranslationRow{{
		Table:    "Languages_en",
		ID:       "272843213265616896",
		Text:     "Magyar",
		Original: "Magyar",
	}}

	data, err := MarshalRows(rows)
	if err != nil {
		t.Fatalf("MarshalRows returned error: %v", err)
	}

	var decoded []TranslationRow
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal returned error: %v", err)
	}
	if decoded[0].ID != rows[0].ID {
		t.Fatalf("ID = %q, want %q", decoded[0].ID, rows[0].ID)
	}
}

func TestUnmarshalRowsPreservesOriginalAndLargeUnityIds(t *testing.T) {
	data := []byte(`[
  {
    "table": "UIText_ru",
    "id": "272843213265616896",
    "original": "Играть",
    "text": "Грати"
  }
]`)

	rows, err := UnmarshalRows(data)
	if err != nil {
		t.Fatalf("UnmarshalRows returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].ID != "272843213265616896" {
		t.Fatalf("ID = %q, want large string ID", rows[0].ID)
	}
	if rows[0].Original != "Играть" {
		t.Fatalf("Original = %q, want %q", rows[0].Original, "Играть")
	}
	if rows[0].Text != "Грати" {
		t.Fatalf("Text = %q, want %q", rows[0].Text, "Грати")
	}
}

func TestEnsureEditableBundleCopiesRussianBundleWhenMissing(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	sourceDir := filepath.Join(gameDir, "CrimeCleaner_Data", "StreamingAssets", "aa", "StandaloneWindows64")
	if err := os.MkdirAll(sourceDir, 0o755); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(sourceDir, "localization-string-tables-russian(ru)_assets_all.bundle")
	if err := os.WriteFile(source, []byte("russian bundle"), 0o644); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(root, "ukrainian-localization.bundle")

	if err := EnsureEditorBundles(target, filepath.Join(root, RuntimeDictionaryBundle), gameDir); err != nil {
		t.Fatalf("EnsureEditableBundle returned error: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "russian bundle" {
		t.Fatalf("target content = %q", string(data))
	}
	dictionary, err := os.ReadFile(filepath.Join(root, RuntimeDictionaryBundle))
	if err != nil {
		t.Fatal(err)
	}
	if string(dictionary) != "russian bundle" {
		t.Fatalf("dictionary content = %q", string(dictionary))
	}
}

func TestEnsureEditableBundleKeepsExistingBundle(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	target := filepath.Join(root, "ukrainian-localization.bundle")
	dictionary := filepath.Join(root, RuntimeDictionaryBundle)
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dictionary, []byte("dictionary"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureEditorBundles(target, dictionary, gameDir); err != nil {
		t.Fatalf("EnsureEditableBundle returned error: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "existing" {
		t.Fatalf("target content = %q", string(data))
	}
}

func TestExportBundleMergesOriginalTextFromDictionary(t *testing.T) {
	source := `G:\SteamLibrary\steamapps\common\Crime Scene Cleaner\CrimeCleaner_Data\StreamingAssets\aa\StandaloneWindows64\localization-string-tables-russian(ru)_assets_all.bundle`
	if _, err := os.Stat(source); err != nil {
		t.Skip("local Crime Scene Cleaner russian bundle is not available")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python is not available")
	}
	status := CheckTooling(context.Background(), ExecRunner{}, python)
	if !status.Ready {
		t.Skip("UnityPy is not available")
	}

	root := t.TempDir()
	bundlePath := filepath.Join(root, RuntimeEditableBundle)
	dictionaryPath := filepath.Join(root, RuntimeDictionaryBundle)
	if err := copyFile(source, bundlePath); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(source, dictionaryPath); err != nil {
		t.Fatal(err)
	}

	data, err := ExportBundle(context.Background(), ExecRunner{}, python, bundlePath, dictionaryPath)
	if err != nil {
		t.Fatalf("ExportBundle returned error: %v", err)
	}
	if len(data.Rows) == 0 {
		t.Fatal("expected exported rows")
	}
	if data.Rows[0].Original == "" {
		t.Fatal("expected Original to be populated from dictionary bundle")
	}
}

func TestImportBundleExportsEnglishAndPolishBundlesWithCorrectSuffixes(t *testing.T) {
	source := `G:\SteamLibrary\steamapps\common\Crime Scene Cleaner\CrimeCleaner_Data\StreamingAssets\aa\StandaloneWindows64\localization-string-tables-russian(ru)_assets_all.bundle`
	if _, err := os.Stat(source); err != nil {
		t.Skip("local Crime Scene Cleaner russian bundle is not available")
	}
	python, err := exec.LookPath("python")
	if err != nil {
		t.Skip("python is not available")
	}
	status := CheckTooling(context.Background(), ExecRunner{}, python)
	if !status.Ready {
		t.Skip("UnityPy is not available")
	}

	root := t.TempDir()
	bundlePath := filepath.Join(root, RuntimeEditableBundle)
	if err := copyFile(source, bundlePath); err != nil {
		t.Fatal(err)
	}
	data, err := ExportBundle(context.Background(), ExecRunner{}, python, bundlePath, "")
	if err != nil {
		t.Fatalf("ExportBundle returned error: %v", err)
	}
	if len(data.Rows) == 0 {
		t.Fatal("expected exported rows")
	}
	data.Rows[0].Text = data.Rows[0].Text + " test"

	englishTemplate, err := targetBundlePathForTest(patcher.TargetEnglish)
	if err != nil {
		t.Fatal(err)
	}
	polishTemplate, err := targetBundlePathForTest(patcher.TargetPolish)
	if err != nil {
		t.Fatal(err)
	}

	if err := ImportBundle(context.Background(), ExecRunner{}, python, bundlePath, data.Rows, englishTemplate, polishTemplate); err != nil {
		t.Fatalf("ImportBundle returned error: %v", err)
	}

	assertBundleSuffixes(t, python, filepath.Join(root, RuntimeEnglishBundle), "_en", "en")
	assertBundleSuffixes(t, python, filepath.Join(root, RuntimePolishBundle), "_pl", "pl")
}

func targetBundlePathForTest(target patcher.TargetLanguage) (string, error) {
	return patcher.TargetBundlePath(`G:\SteamLibrary\steamapps\common\Crime Scene Cleaner`, target)
}

type errTest string

func (e errTest) Error() string {
	return string(e)
}

func assertBundleSuffixes(t *testing.T, python string, bundlePath string, suffix string, locale string) {
	t.Helper()
	script := `
import json
import sys
import UnityPy

env = UnityPy.load(sys.argv[1])
bad_names = []
bad_locales = []
bad_containers = []
checked = 0
for file in env.files.values():
    for container in getattr(file, "container", {}) or {}:
        if not container.endswith(sys.argv[2] + ".asset"):
            bad_containers.append(container)
for obj in env.objects:
    if obj.type.name != "MonoBehaviour":
        continue
    try:
        tree = obj.read_typetree()
    except Exception:
        continue
    name = tree.get("m_Name")
    if not name or "m_TableData" not in tree:
        continue
    checked += 1
    if not name.endswith(sys.argv[2]):
        bad_names.append(name)
    code = tree.get("m_LocaleId", {}).get("m_Code")
    if code != sys.argv[3]:
        bad_locales.append([name, code])
print(json.dumps({"checked": checked, "bad_names": bad_names, "bad_locales": bad_locales, "bad_containers": bad_containers}, ensure_ascii=False))
`
	out, err := exec.Command(python, "-c", script, bundlePath, suffix, locale).CombinedOutput()
	if err != nil {
		t.Fatalf("inspect bundle failed: %v\n%s", err, out)
	}
	var result struct {
		Checked    int        `json:"checked"`
		BadNames   []string   `json:"bad_names"`
		BadLocales [][]string `json:"bad_locales"`
		BadPaths   []string   `json:"bad_containers"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		t.Fatalf("decode inspection output: %v\n%s", err, out)
	}
	if result.Checked == 0 {
		t.Fatalf("no string tables inspected in %s", bundlePath)
	}
	if len(result.BadNames) > 0 {
		t.Fatalf("tables with wrong suffix in %s: %v", bundlePath, result.BadNames[:min(3, len(result.BadNames))])
	}
	if len(result.BadLocales) > 0 {
		t.Fatalf("tables with wrong locale in %s: %v", bundlePath, result.BadLocales[:min(3, len(result.BadLocales))])
	}
	if len(result.BadPaths) > 0 {
		t.Fatalf("containers with wrong suffix in %s: %v", bundlePath, result.BadPaths[:min(3, len(result.BadPaths))])
	}
}
