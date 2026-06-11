package editor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"crime-scene-cleaner/internal/patcher"
)

type CommandRunner interface {
	Run(ctx context.Context, name string, args ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	output, err := cmd.CombinedOutput()
	return string(output), err
}

type ToolingStatus struct {
	Ready            bool   `json:"ready"`
	PythonAvailable  bool   `json:"pythonAvailable"`
	UnityPyAvailable bool   `json:"unityPyAvailable"`
	PythonExecutable string `json:"pythonExecutable"`
	Message          string `json:"message"`
}

type TranslationRow struct {
	Table string `json:"table"`
	ID    string `json:"id"`
	Text  string `json:"text"`
}

type EditorData struct {
	Rows    []TranslationRow `json:"rows"`
	TempDir string           `json:"tempDir"`
}

const (
	RuntimeEditableBundle = "ukrainian-localization.bundle"
	RuntimeEnglishBundle  = "ukrainian-localization_en.bundle"
	RuntimePolishBundle   = "ukrainian-localization_pl.bundle"
)

func DefaultPythonExecutable() string {
	if path, err := exec.LookPath("python"); err == nil {
		return path
	}
	if path, err := exec.LookPath("py"); err == nil {
		return path
	}
	return "python"
}

func CheckTooling(ctx context.Context, runner CommandRunner, python string) ToolingStatus {
	status := ToolingStatus{PythonExecutable: python}
	if _, err := runner.Run(ctx, python, "--version"); err != nil {
		status.Message = "Python is not available"
		return status
	}
	status.PythonAvailable = true

	if _, err := runner.Run(ctx, python, "-c", "import UnityPy"); err != nil {
		status.Message = "UnityPy is not installed"
		return status
	}
	status.UnityPyAvailable = true
	status.Ready = true
	status.Message = "Editor tooling is ready"
	return status
}

func InstallUnityPy(ctx context.Context, runner CommandRunner, python string) error {
	_, err := runner.Run(ctx, python, "-m", "pip", "install", "UnityPy")
	return err
}

func MarshalRows(rows []TranslationRow) ([]byte, error) {
	return json.MarshalIndent(rows, "", "  ")
}

func ExportBundle(ctx context.Context, runner CommandRunner, python string, bundlePath string) (EditorData, error) {
	if err := requireFile(bundlePath); err != nil {
		return EditorData{}, err
	}

	tempDir, err := os.MkdirTemp("", "crime-scene-cleaner-localization-*")
	if err != nil {
		return EditorData{}, err
	}
	scriptPath, err := writeHelperScript(tempDir, "export_bundle.py", exportScript)
	if err != nil {
		return EditorData{}, err
	}
	outputPath := filepath.Join(tempDir, "translations.json")

	if _, err := runner.Run(ctx, python, scriptPath, bundlePath, outputPath); err != nil {
		return EditorData{}, err
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return EditorData{}, err
	}
	var rows []TranslationRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return EditorData{}, err
	}
	return EditorData{Rows: rows, TempDir: tempDir}, nil
}

func ImportBundle(ctx context.Context, runner CommandRunner, python string, bundlePath string, rows []TranslationRow) error {
	if err := requireFile(bundlePath); err != nil {
		return err
	}
	tempDir, err := os.MkdirTemp("", "crime-scene-cleaner-localization-save-*")
	if err != nil {
		return err
	}
	scriptPath, err := writeHelperScript(tempDir, "import_bundle.py", importScript)
	if err != nil {
		return err
	}
	rowsPath := filepath.Join(tempDir, "translations.json")
	data, err := MarshalRows(rows)
	if err != nil {
		return err
	}
	if err := os.WriteFile(rowsPath, data, 0o644); err != nil {
		return err
	}
	backupPath := bundlePath + ".editor.bak"
	if _, err := os.Stat(backupPath); errors.Is(err, os.ErrNotExist) {
		if err := copyFile(bundlePath, backupPath); err != nil {
			return fmt.Errorf("create editor backup: %w", err)
		}
	}
	baseDir := filepath.Dir(bundlePath)
	englishPath := filepath.Join(baseDir, RuntimeEnglishBundle)
	polishPath := filepath.Join(baseDir, RuntimePolishBundle)
	_, err = runner.Run(ctx, python, scriptPath, bundlePath, rowsPath, englishPath, polishPath)
	return err
}

func EnsureEditableBundle(bundlePath string, gameDir string) error {
	if _, err := os.Stat(bundlePath); err == nil {
		return nil
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}

	sourcePath, err := patcher.TargetBundlePath(gameDir, patcher.TargetRussian)
	if err != nil {
		return err
	}
	if err := requireFile(sourcePath); err != nil {
		return fmt.Errorf("russian source bundle is missing: %w", err)
	}
	return copyFile(sourcePath, bundlePath)
}

func writeHelperScript(dir string, name string, content string) (string, error) {
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func requireFile(path string) error {
	stat, err := os.Stat(path)
	if err != nil {
		return err
	}
	if stat.IsDir() {
		return fmt.Errorf("%s is a directory", path)
	}
	return nil
}

func copyFile(source string, target string) error {
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return os.WriteFile(target, data, 0o644)
}
