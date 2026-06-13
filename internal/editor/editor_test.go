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
	onRun   func(name string, args ...string) (string, error)
}

type runResult struct {
	out string
	err error
}

func (f *fakeRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	call := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, call)
	if f.onRun != nil {
		return f.onRun(name, args...)
	}
	result := f.outputs[call]
	return result.out, result.err
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

func TestCheckToolingReportsBundleToolAvailable(t *testing.T) {
	root := t.TempDir()
	toolPath := filepath.Join(root, "BundleTool.exe")
	if err := os.WriteFile(toolPath, []byte("tool"), 0o755); err != nil {
		t.Fatal(err)
	}

	status := CheckTooling(toolPath)

	if !status.BundleToolAvailable {
		t.Fatal("expected BundleToolAvailable")
	}
	if !status.Ready {
		t.Fatal("expected Ready")
	}
	if status.BundleToolPath != toolPath {
		t.Fatalf("BundleToolPath = %q, want %q", status.BundleToolPath, toolPath)
	}
}

func TestCheckToolingReportsMissingBundleTool(t *testing.T) {
	status := CheckTooling(filepath.Join(t.TempDir(), "BundleTool.exe"))

	if status.BundleToolAvailable {
		t.Fatal("expected BundleToolAvailable to be false")
	}
	if status.Ready {
		t.Fatal("expected Ready to be false")
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
	root := t.TempDir()
	bundlePath := filepath.Join(root, RuntimeEditableBundle)
	dictionaryPath := filepath.Join(root, RuntimeDictionaryBundle)
	toolPath := filepath.Join(root, "BundleTool.exe")
	if err := os.WriteFile(toolPath, []byte("tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{onRun: func(name string, args ...string) (string, error) {
		if len(args) != 3 || args[0] != "export" {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		rows := []TranslationRow{{Table: "UIText_ru", ID: "1", Text: "Так"}}
		if strings.Contains(args[1], RuntimeDictionaryBundle) {
			rows[0].Text = "Да"
		}
		data, err := MarshalRows(rows)
		if err != nil {
			return "", err
		}
		return "", os.WriteFile(args[2], data, 0o644)
	}}
	if err := os.WriteFile(bundlePath, []byte("bundle"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(dictionaryPath, []byte("dictionary"), 0o644); err != nil {
		t.Fatal(err)
	}

	data, err := ExportBundle(context.Background(), runner, toolPath, bundlePath, dictionaryPath)
	if err != nil {
		t.Fatalf("ExportBundle returned error: %v", err)
	}
	if len(data.Rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(data.Rows))
	}
	if data.Rows[0].Original != "Да" {
		t.Fatalf("Original = %q, want dictionary text", data.Rows[0].Original)
	}
	if data.TempDir == "" {
		t.Fatal("expected temp dir to be returned")
	}
}

func TestImportBundleUsesAssetsToolForEditableEnglishAndPolishBundles(t *testing.T) {
	root := t.TempDir()
	bundlePath := filepath.Join(root, RuntimeEditableBundle)
	workingTemplate := filepath.Join(root, RuntimeDictionaryBundle)
	englishTemplate := filepath.Join(root, "english.bundle")
	polishTemplate := filepath.Join(root, "polish.bundle")
	toolPath := filepath.Join(root, "BundleTool.exe")
	for _, path := range []string{bundlePath, workingTemplate, englishTemplate, polishTemplate} {
		if err := os.WriteFile(path, []byte("bundle"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(toolPath, []byte("tool"), 0o755); err != nil {
		t.Fatal(err)
	}
	runner := &fakeRunner{onRun: func(name string, args ...string) (string, error) {
		if len(args) < 4 || args[0] != "import" {
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		return "", os.WriteFile(args[3], []byte("packed"), 0o644)
	}}
	rows := []TranslationRow{{Table: "UIText_ru", ID: "1", Text: "Так", Original: "Да"}}

	if err := ImportBundle(context.Background(), runner, toolPath, bundlePath, rows, workingTemplate, englishTemplate, polishTemplate); err != nil {
		t.Fatalf("ImportBundle returned error: %v", err)
	}

	allCalls := strings.Join(runner.calls, "\n")
	for _, expected := range []string{
		" import " + workingTemplate,
		" import " + englishTemplate,
		" import " + polishTemplate,
		" en _en",
		" pl _pl",
	} {
		if !strings.Contains(allCalls, expected) {
			t.Fatalf("expected command calls to contain %q, got:\n%s", expected, allCalls)
		}
	}
}

func TestBuildLocalizedAssetTableExportsSourceRowsAndImportsIntoTemplate(t *testing.T) {
	root := t.TempDir()
	toolPath := filepath.Join(root, "BundleTool.exe")
	sourceRowsPath := filepath.Join(root, "localization-asset-tables-russian(ru)_assets_all.bundle")
	templatePath := filepath.Join(root, "localization-asset-tables-english(en)_assets_all.bundle")
	outputPath := filepath.Join(root, "fonts_en.bundle")
	for _, path := range []string{toolPath, sourceRowsPath, templatePath} {
		if err := os.WriteFile(path, []byte("file"), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	runner := &fakeRunner{onRun: func(name string, args ...string) (string, error) {
		switch {
		case len(args) == 3 && args[0] == "export" && args[1] == sourceRowsPath:
			return "", os.WriteFile(args[2], []byte(`[{"table":"Fonts_ru","id":"1","text":"guid"}]`), 0o644)
		case len(args) == 6 && args[0] == "import" && args[1] == templatePath && args[3] != "" && args[4] == "en" && args[5] == "_en":
			return "", os.WriteFile(args[3], []byte("packed-font-table"), 0o644)
		default:
			t.Fatalf("unexpected command: %s %v", name, args)
		}
		return "", nil
	}}

	if err := BuildLocalizedAssetTable(context.Background(), runner, toolPath, sourceRowsPath, templatePath, outputPath, "en", "_en"); err != nil {
		t.Fatalf("BuildLocalizedAssetTable returned error: %v", err)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "packed-font-table" {
		t.Fatalf("output = %q", string(data))
	}
	allCalls := strings.Join(runner.calls, "\n")
	for _, expected := range []string{
		" export " + sourceRowsPath,
		" import " + templatePath,
		" en _en",
	} {
		if !strings.Contains(allCalls, expected) {
			t.Fatalf("expected command calls to contain %q, got:\n%s", expected, allCalls)
		}
	}
}

func TestBundleToolExportsEnglishAndPolishBundlesWithCorrectSuffixes(t *testing.T) {
	toolPath := filepath.Join("..", "..", "tools", "BundleTool", "bin", "Release", "net8.0", "BundleTool.exe")
	if _, err := os.Stat(toolPath); err != nil {
		t.Skip("BundleTool.exe has not been built")
	}

	russianTemplate, err := targetBundlePathForTest(patcher.TargetRussian)
	if err != nil {
		t.Fatal(err)
	}
	englishTemplate, err := targetBundlePathForTest(patcher.TargetEnglish)
	if err != nil {
		t.Fatal(err)
	}
	polishTemplate, err := targetBundlePathForTest(patcher.TargetPolish)
	if err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{russianTemplate, englishTemplate, polishTemplate} {
		if _, err := os.Stat(path); err != nil {
			t.Skip("local Crime Scene Cleaner localization bundles are not available")
		}
	}

	root := t.TempDir()
	rowsPath := filepath.Join(root, "rows.json")
	if out, err := exec.Command(toolPath, "export", russianTemplate, rowsPath).CombinedOutput(); err != nil {
		t.Fatalf("export failed: %v\n%s", err, out)
	}

	englishPath := filepath.Join(root, RuntimeEnglishBundle)
	polishPath := filepath.Join(root, RuntimePolishBundle)
	if out, err := exec.Command(toolPath, "import", englishTemplate, rowsPath, englishPath, "en", "_en").CombinedOutput(); err != nil {
		t.Fatalf("english import failed: %v\n%s", err, out)
	}
	if out, err := exec.Command(toolPath, "import", polishTemplate, rowsPath, polishPath, "pl", "_pl").CombinedOutput(); err != nil {
		t.Fatalf("polish import failed: %v\n%s", err, out)
	}

	assertBundleMetadata(t, toolPath, englishTemplate, englishPath, "_en", "en")
	assertBundleMetadata(t, toolPath, polishTemplate, polishPath, "_pl", "pl")
}

func TestBundleToolBuildsEnglishAssetTableFromRussianRowsAndEnglishTemplate(t *testing.T) {
	toolPath := filepath.Join("..", "..", "build", "bin", "BundleTool.exe")
	if _, err := os.Stat(toolPath); err != nil {
		t.Skip("BundleTool.exe has not been built")
	}
	russianAssetTable, err := patcher.TargetAssetTableBundlePath(`G:\SteamLibrary\steamapps\common\Crime Scene Cleaner`, patcher.TargetRussian)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(russianAssetTable); err != nil {
		t.Skip("local Crime Scene Cleaner localization asset table is not available")
	}
	englishAssetTableTemplate, err := patcher.TargetAssetTableBundlePath(`G:\SteamLibrary\steamapps\common\Crime Scene Cleaner`, patcher.TargetEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(englishAssetTableTemplate); err != nil {
		t.Skip("local Crime Scene Cleaner English asset table is not available")
	}

	root := t.TempDir()
	rowsPath := filepath.Join(root, "fonts-ru.json")
	englishAssetTable := filepath.Join(root, "fonts_en.bundle")
	if out, err := exec.Command(toolPath, "export", russianAssetTable, rowsPath).CombinedOutput(); err != nil {
		t.Fatalf("export failed: %v\n%s", err, out)
	}
	if out, err := exec.Command(toolPath, "import", englishAssetTableTemplate, rowsPath, englishAssetTable, "en", "_en").CombinedOutput(); err != nil {
		t.Fatalf("asset-table import failed: %v\n%s", err, out)
	}

	assertBundleMetadata(t, toolPath, englishAssetTableTemplate, englishAssetTable, "_en", "en")
}

func targetBundlePathForTest(target patcher.TargetLanguage) (string, error) {
	return patcher.TargetBundlePath(`G:\SteamLibrary\steamapps\common\Crime Scene Cleaner`, target)
}

type errTest string

func (e errTest) Error() string {
	return string(e)
}

func assertBundleMetadata(t *testing.T, toolPath string, templatePath string, bundlePath string, suffix string, locale string) {
	t.Helper()
	outputPath := filepath.Join(t.TempDir(), "inspect.json")
	if out, err := exec.Command(toolPath, "inspect", bundlePath, outputPath).CombinedOutput(); err != nil {
		t.Fatalf("inspect failed: %v\n%s", err, out)
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatal(err)
	}
	templateOutputPath := filepath.Join(t.TempDir(), "template-inspect.json")
	if out, err := exec.Command(toolPath, "inspect", templatePath, templateOutputPath).CombinedOutput(); err != nil {
		t.Fatalf("inspect template failed: %v\n%s", err, out)
	}
	templateData, err := os.ReadFile(templateOutputPath)
	if err != nil {
		t.Fatal(err)
	}
	var result struct {
		Tables []struct {
			Name   string `json:"name"`
			Locale string `json:"locale"`
		} `json:"tables"`
		Directories []string `json:"directories"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("decode inspection output: %v\n%s", err, data)
	}
	var templateResult struct {
		Directories []string `json:"directories"`
	}
	if err := json.Unmarshal(templateData, &templateResult); err != nil {
		t.Fatalf("decode template inspection output: %v\n%s", err, templateData)
	}
	if len(result.Tables) == 0 {
		t.Fatalf("no string tables inspected in %s", bundlePath)
	}
	if strings.Join(result.Directories, "\n") != strings.Join(templateResult.Directories, "\n") {
		t.Fatalf("bundle directories changed from template: got %v, want %v", result.Directories, templateResult.Directories)
	}
	var bad []string
	for _, table := range result.Tables {
		if !strings.HasSuffix(table.Name, suffix) || table.Locale != locale {
			bad = append(bad, table.Name+":"+table.Locale)
		}
	}
	if len(bad) > 0 {
		t.Fatalf("tables with wrong metadata in %s: %v", bundlePath, bad[:min(3, len(bad))])
	}
}
