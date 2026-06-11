package editor

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
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
		Table: "Languages_en",
		ID:    "272843213265616896",
		Text:  "Magyar",
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

	if err := EnsureEditableBundle(target, gameDir); err != nil {
		t.Fatalf("EnsureEditableBundle returned error: %v", err)
	}
	data, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "russian bundle" {
		t.Fatalf("target content = %q", string(data))
	}
}

func TestEnsureEditableBundleKeepsExistingBundle(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	target := filepath.Join(root, "ukrainian-localization.bundle")
	if err := os.WriteFile(target, []byte("existing"), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := EnsureEditableBundle(target, gameDir); err != nil {
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

type errTest string

func (e errTest) Error() string {
	return string(e)
}
