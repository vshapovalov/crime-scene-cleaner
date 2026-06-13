package patcher

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type TargetLanguage string

const (
	TargetEnglish TargetLanguage = "english"
	TargetPolish  TargetLanguage = "polish"
	TargetRussian TargetLanguage = "russian"
)

const RuntimeTranslationBundle = "ukrainian-localization.bundle"

type ApplyResult struct {
	TargetPath string `json:"targetPath"`
	BackupPath string `json:"backupPath"`
	Message    string `json:"message"`
}

func TargetBundlePath(gameDir string, target TargetLanguage) (string, error) {
	fileName := ""
	switch target {
	case TargetEnglish:
		fileName = "localization-string-tables-english(en)_assets_all.bundle"
	case TargetPolish:
		fileName = "localization-string-tables-polish(pl)_assets_all.bundle"
	case TargetRussian:
		fileName = "localization-string-tables-russian(ru)_assets_all.bundle"
	default:
		return "", fmt.Errorf("непідтримувана мова для заміни: %s", target)
	}

	return filepath.Join(
		gameDir,
		"CrimeCleaner_Data",
		"StreamingAssets",
		"aa",
		"StandaloneWindows64",
		fileName,
	), nil
}

func Apply(gameDir string, sourceBundle string, target TargetLanguage) (ApplyResult, error) {
	targetPath, err := TargetBundlePath(gameDir, target)
	if err != nil {
		return ApplyResult{}, err
	}
	if err := requireFile(sourceBundle); err != nil {
		return ApplyResult{}, fmt.Errorf("бандл перекладу відсутній: %w", err)
	}
	if err := requireFile(targetPath); err != nil {
		return ApplyResult{}, fmt.Errorf("бандл локалізації гри відсутній: %w", err)
	}

	backupPath := targetPath + ".bak"
	if _, err := os.Stat(backupPath); os.IsNotExist(err) {
		if err := copyFile(targetPath, backupPath); err != nil {
			return ApplyResult{}, fmt.Errorf("створення резервної копії: %w", err)
		}
	}
	if err := copyFile(sourceBundle, targetPath); err != nil {
		return ApplyResult{}, fmt.Errorf("встановлення перекладу: %w", err)
	}

	return ApplyResult{
		TargetPath: targetPath,
		BackupPath: backupPath,
		Message:    "Переклад застосовано",
	}, nil
}

func RuntimeBundlePath(executablePath string) string {
	return filepath.Join(filepath.Dir(executablePath), RuntimeTranslationBundle)
}

func RuntimeBundlePathForTarget(executablePath string, target TargetLanguage) (string, error) {
	fileName := ""
	switch target {
	case TargetEnglish:
		fileName = "ukrainian-localization_en.bundle"
	case TargetPolish:
		fileName = "ukrainian-localization_pl.bundle"
	default:
		return "", fmt.Errorf("непідтримувана мова перекладу під час виконання: %s", target)
	}
	return filepath.Join(filepath.Dir(executablePath), fileName), nil
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
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(target)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return out.Sync()
}
