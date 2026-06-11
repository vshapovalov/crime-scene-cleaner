package patcher

import (
	"os"
	"path/filepath"
	"testing"
)

func TestTargetBundlePathUsesSelectedLanguage(t *testing.T) {
	gameDir := filepath.Join("G:", "SteamLibrary", "steamapps", "common", "Crime Scene Cleaner")

	english, err := TargetBundlePath(gameDir, TargetEnglish)
	if err != nil {
		t.Fatalf("TargetBundlePath English returned error: %v", err)
	}
	polish, err := TargetBundlePath(gameDir, TargetPolish)
	if err != nil {
		t.Fatalf("TargetBundlePath Polish returned error: %v", err)
	}

	if filepath.Base(english) != "localization-string-tables-english(en)_assets_all.bundle" {
		t.Fatalf("English target = %q", english)
	}
	if filepath.Base(polish) != "localization-string-tables-polish(pl)_assets_all.bundle" {
		t.Fatalf("Polish target = %q", polish)
	}
}

func TestApplyCreatesBackupAndCopiesTranslation(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	targetDir := filepath.Join(gameDir, "CrimeCleaner_Data", "StreamingAssets", "aa", "StandaloneWindows64")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, "localization-string-tables-english(en)_assets_all.bundle")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "ukrainian-localization.bundle")
	if err := os.WriteFile(source, []byte("translated"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := Apply(gameDir, source, TargetEnglish)

	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "translated" {
		t.Fatalf("target content = %q", string(got))
	}
	backup, err := os.ReadFile(target + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "original" {
		t.Fatalf("backup content = %q", string(backup))
	}
	if result.BackupPath != target+".bak" {
		t.Fatalf("BackupPath = %q", result.BackupPath)
	}
}

func TestApplyDoesNotOverwriteExistingBackup(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	targetDir := filepath.Join(gameDir, "CrimeCleaner_Data", "StreamingAssets", "aa", "StandaloneWindows64")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, "localization-string-tables-polish(pl)_assets_all.bundle")
	if err := os.WriteFile(target, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target+".bak", []byte("first-original"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "ukrainian-localization.bundle")
	if err := os.WriteFile(source, []byte("translated"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := Apply(gameDir, source, TargetPolish)

	if err != nil {
		t.Fatalf("Apply returned error: %v", err)
	}
	backup, err := os.ReadFile(target + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(backup) != "first-original" {
		t.Fatalf("backup content = %q", string(backup))
	}
}
