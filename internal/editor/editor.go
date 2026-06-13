package editor

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
	Ready               bool   `json:"ready"`
	BundleToolAvailable bool   `json:"bundleToolAvailable"`
	BundleToolPath      string `json:"bundleToolPath"`
	Message             string `json:"message"`
}

type TranslationRow struct {
	Table    string `json:"table"`
	ID       string `json:"id"`
	Text     string `json:"text"`
	Original string `json:"original"`
}

type EditorData struct {
	Rows    []TranslationRow `json:"rows"`
	TempDir string           `json:"tempDir"`
}

const (
	RuntimeEditableBundle   = "ukrainian-localization.bundle"
	RuntimeDictionaryBundle = "ukrainian-localization-dictionary.bundle"
	RuntimeEnglishBundle    = "ukrainian-localization_en.bundle"
	RuntimePolishBundle     = "ukrainian-localization_pl.bundle"
)

func RuntimeBundleToolPath(executablePath string) string {
	return filepath.Join(filepath.Dir(executablePath), "BundleTool.exe")
}

func CheckTooling(bundleToolPath string) ToolingStatus {
	status := ToolingStatus{BundleToolPath: bundleToolPath}
	if err := requireFile(bundleToolPath); err != nil {
		status.Message = "BundleTool.exe не знайдено поруч із файлом програми"
		return status
	}
	status.BundleToolAvailable = true
	status.Ready = true
	status.Message = "Інструменти редактора готові"
	return status
}

func MarshalRows(rows []TranslationRow) ([]byte, error) {
	return json.MarshalIndent(rows, "", "  ")
}

func UnmarshalRows(data []byte) ([]TranslationRow, error) {
	var rows []TranslationRow
	if err := json.Unmarshal(data, &rows); err != nil {
		return nil, err
	}
	if rows == nil {
		return []TranslationRow{}, nil
	}
	return rows, nil
}

func ExportBundle(ctx context.Context, runner CommandRunner, bundleToolPath string, bundlePath string, dictionaryPath string) (EditorData, error) {
	if err := requireFile(bundlePath); err != nil {
		return EditorData{}, err
	}
	if err := requireFile(bundleToolPath); err != nil {
		return EditorData{}, fmt.Errorf("інструмент для бандлів відсутній: %w", err)
	}

	tempDir, err := os.MkdirTemp("", "crime-scene-cleaner-localization-*")
	if err != nil {
		return EditorData{}, err
	}
	outputPath := filepath.Join(tempDir, "editable.json")

	if _, err := runner.Run(ctx, bundleToolPath, "export", bundlePath, outputPath); err != nil {
		return EditorData{}, err
	}
	data, err := os.ReadFile(outputPath)
	if err != nil {
		return EditorData{}, err
	}
	rows, err := UnmarshalRows(data)
	if err != nil {
		return EditorData{}, err
	}
	if dictionaryPath != "" {
		if err := requireFile(dictionaryPath); err == nil {
			dictionaryOutputPath := filepath.Join(tempDir, "dictionary.json")
			if _, err := runner.Run(ctx, bundleToolPath, "export", dictionaryPath, dictionaryOutputPath); err != nil {
				return EditorData{}, err
			}
			dictionaryData, err := os.ReadFile(dictionaryOutputPath)
			if err != nil {
				return EditorData{}, err
			}
			dictionaryRows, err := UnmarshalRows(dictionaryData)
			if err != nil {
				return EditorData{}, err
			}
			originals := map[string]string{}
			for _, row := range dictionaryRows {
				originals[rowKey(row.Table, row.ID)] = row.Text
			}
			for i := range rows {
				if original, ok := originals[rowKey(rows[i].Table, rows[i].ID)]; ok {
					rows[i].Original = original
				} else {
					rows[i].Original = rows[i].Text
				}
			}
		}
	}
	for i := range rows {
		if rows[i].Original == "" {
			rows[i].Original = rows[i].Text
		}
	}
	return EditorData{Rows: rows, TempDir: tempDir}, nil
}

func ImportBundle(ctx context.Context, runner CommandRunner, bundleToolPath string, bundlePath string, rows []TranslationRow, workingTemplatePath string, englishTemplatePath string, polishTemplatePath string) error {
	if err := requireFile(bundleToolPath); err != nil {
		return fmt.Errorf("інструмент для бандлів відсутній: %w", err)
	}
	if err := requireFile(bundlePath); err != nil {
		return err
	}
	if err := requireFile(workingTemplatePath); err != nil {
		return fmt.Errorf("відсутній шаблон робочого бандла: %w", err)
	}
	if err := requireFile(englishTemplatePath); err != nil {
		return fmt.Errorf("відсутній шаблон експорту англійської: %w", err)
	}
	if err := requireFile(polishTemplatePath); err != nil {
		return fmt.Errorf("відсутній шаблон експорту польської: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "crime-scene-cleaner-localization-save-*")
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
			return fmt.Errorf("створення резервної копії редактора: %w", err)
		}
	}
	baseDir := filepath.Dir(bundlePath)
	englishPath := filepath.Join(baseDir, RuntimeEnglishBundle)
	polishPath := filepath.Join(baseDir, RuntimePolishBundle)

	if err := importBundleTo(ctx, runner, bundleToolPath, workingTemplatePath, rowsPath, bundlePath, "", ""); err != nil {
		return err
	}
	if err := importBundleTo(ctx, runner, bundleToolPath, englishTemplatePath, rowsPath, englishPath, "en", "_en"); err != nil {
		return err
	}
	return importBundleTo(ctx, runner, bundleToolPath, polishTemplatePath, rowsPath, polishPath, "pl", "_pl")
}

func BuildLocalizedAssetTable(ctx context.Context, runner CommandRunner, bundleToolPath string, sourceRowsBundlePath string, templatePath string, outputPath string, locale string, suffix string) error {
	if err := requireFile(bundleToolPath); err != nil {
		return fmt.Errorf("інструмент для бандлів відсутній: %w", err)
	}
	if err := requireFile(sourceRowsBundlePath); err != nil {
		return fmt.Errorf("відсутня вихідна таблиця ассетів: %w", err)
	}
	if err := requireFile(templatePath); err != nil {
		return fmt.Errorf("відсутній шаблон таблиці ассетів: %w", err)
	}
	tempDir, err := os.MkdirTemp("", "crime-scene-cleaner-fonts-*")
	if err != nil {
		return err
	}
	rowsPath := filepath.Join(tempDir, "font-assets.json")
	if _, err := runner.Run(ctx, bundleToolPath, "export", sourceRowsBundlePath, rowsPath); err != nil {
		return err
	}
	if err := requireFile(rowsPath); err != nil {
		return fmt.Errorf("інструмент для бандлів не експортував таблицю ассетів: %w", err)
	}
	outputTempPath := filepath.Join(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+".tmp")
	if _, err := runner.Run(ctx, bundleToolPath, "import", templatePath, rowsPath, outputTempPath, locale, suffix); err != nil {
		return err
	}
	if err := requireFile(outputTempPath); err != nil {
		return fmt.Errorf("інструмент для бандлів не створив таблицю ассетів: %w", err)
	}
	if err := os.Remove(outputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(outputTempPath, outputPath)
}

func EnsureEditorBundles(bundlePath string, dictionaryPath string, gameDir string) error {
	needsEditable := false
	if _, err := os.Stat(bundlePath); errors.Is(err, os.ErrNotExist) {
		needsEditable = true
	} else if err != nil {
		return err
	}

	needsDictionary := false
	if _, err := os.Stat(dictionaryPath); errors.Is(err, os.ErrNotExist) {
		needsDictionary = true
	} else if err != nil {
		return err
	}

	if !needsEditable && !needsDictionary {
		return nil
	}

	sourcePath, err := patcher.TargetBundlePath(gameDir, patcher.TargetRussian)
	if err != nil {
		return err
	}
	if err := requireFile(sourcePath); err != nil {
		return fmt.Errorf("відсутній російський вихідний бандл: %w", err)
	}

	if needsEditable {
		if err := copyFile(sourcePath, bundlePath); err != nil {
			return err
		}
	}

	if needsDictionary {
		if err := copyFile(sourcePath, dictionaryPath); err != nil {
			return err
		}
	}
	return nil
}

func EnsureEditableBundle(bundlePath string, gameDir string) error {
	return EnsureEditorBundles(bundlePath, filepath.Join(filepath.Dir(bundlePath), RuntimeDictionaryBundle), gameDir)
}

func DictionaryBundlePath(bundlePath string) string {
	return filepath.Join(filepath.Dir(bundlePath), RuntimeDictionaryBundle)
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

func importBundleTo(ctx context.Context, runner CommandRunner, bundleToolPath string, templatePath string, rowsPath string, outputPath string, locale string, suffix string) error {
	outputTempPath := filepath.Join(filepath.Dir(outputPath), "."+filepath.Base(outputPath)+".tmp")
	args := []string{"import", templatePath, rowsPath, outputTempPath}
	if locale != "" {
		args = append(args, locale, suffix)
	}
	if _, err := runner.Run(ctx, bundleToolPath, args...); err != nil {
		return err
	}
	if err := requireFile(outputTempPath); err != nil {
		return fmt.Errorf("інструмент для бандлів не створив вихідний бандл: %w", err)
	}
	if err := os.Remove(outputPath); err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return os.Rename(outputTempPath, outputPath)
}

func rowKey(table string, id string) string {
	return normalizeTableName(table) + "\x00" + id
}

func normalizeTableName(table string) string {
	for _, suffix := range []string{"_ru", "_en", "_pl"} {
		if strings.HasSuffix(table, suffix) {
			return strings.TrimSuffix(table, suffix)
		}
	}
	return table
}
