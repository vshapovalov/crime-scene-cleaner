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

func TestTargetAssetTableBundlePathUsesSelectedLanguage(t *testing.T) {
	gameDir := filepath.Join("G:", "SteamLibrary", "steamapps", "common", "Crime Scene Cleaner")

	english, err := TargetAssetTableBundlePath(gameDir, TargetEnglish)
	if err != nil {
		t.Fatalf("TargetAssetTableBundlePath English returned error: %v", err)
	}
	polish, err := TargetAssetTableBundlePath(gameDir, TargetPolish)
	if err != nil {
		t.Fatalf("TargetAssetTableBundlePath Polish returned error: %v", err)
	}

	if filepath.Base(english) != "localization-asset-tables-english(en)_assets_all.bundle" {
		t.Fatalf("English asset table target = %q", english)
	}
	if filepath.Base(polish) != "localization-asset-tables-polish(pl)_assets_all.bundle" {
		t.Fatalf("Polish asset table target = %q", polish)
	}
}

func TestRuntimeBundlePathForTargetUsesLanguageSpecificExports(t *testing.T) {
	executable := filepath.Join("C:", "Tools", "crime-scene-cleaner.exe")

	english, err := RuntimeBundlePathForTarget(executable, TargetEnglish)
	if err != nil {
		t.Fatalf("RuntimeBundlePathForTarget English returned error: %v", err)
	}
	polish, err := RuntimeBundlePathForTarget(executable, TargetPolish)
	if err != nil {
		t.Fatalf("RuntimeBundlePathForTarget Polish returned error: %v", err)
	}

	if filepath.Base(english) != "ukrainian-localization_en.bundle" {
		t.Fatalf("English runtime bundle = %q", english)
	}
	if filepath.Base(polish) != "ukrainian-localization_pl.bundle" {
		t.Fatalf("Polish runtime bundle = %q", polish)
	}
}

func TestTargetBundleTemplatePathPrefersBackupWhenPresent(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	targetDir := filepath.Join(gameDir, "CrimeCleaner_Data", "StreamingAssets", "aa", "StandaloneWindows64")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := TargetBundlePath(gameDir, TargetEnglish)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target+".bak", []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}

	template, err := TargetBundleTemplatePath(gameDir, TargetEnglish)
	if err != nil {
		t.Fatalf("TargetBundleTemplatePath returned error: %v", err)
	}
	if template != target+".bak" {
		t.Fatalf("template = %q, want backup", template)
	}
}

func TestTargetBundleTemplatePathFallsBackToCurrentBundle(t *testing.T) {
	root := t.TempDir()
	gameDir := filepath.Join(root, "Crime Scene Cleaner")
	targetDir := filepath.Join(gameDir, "CrimeCleaner_Data", "StreamingAssets", "aa", "StandaloneWindows64")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := TargetBundlePath(gameDir, TargetPolish)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(target, []byte("current"), 0o644); err != nil {
		t.Fatal(err)
	}

	template, err := TargetBundleTemplatePath(gameDir, TargetPolish)
	if err != nil {
		t.Fatalf("TargetBundleTemplatePath returned error: %v", err)
	}
	if template != target {
		t.Fatalf("template = %q, want current target", template)
	}
}

func TestApplyCreatesBackupAndCopiesTranslation(t *testing.T) {
	root := t.TempDir()
	gameDir, catalogPath := writeTestCatalog(t, root)
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
	if result.CatalogPath != catalogPath {
		t.Fatalf("CatalogPath = %q", result.CatalogPath)
	}
	if result.CatalogBackupPath != catalogPath+".bak" {
		t.Fatalf("CatalogBackupPath = %q", result.CatalogBackupPath)
	}
	english := readCatalogOptions(t, catalogPath, testEnglishInternalID)
	if english.CRC != 0 || english.UseCRCForCachedBundles {
		t.Fatalf("English catalog entry was not patched: %+v", english)
	}
}

func TestApplyDoesNotOverwriteExistingBackup(t *testing.T) {
	root := t.TempDir()
	gameDir, catalogPath := writeTestCatalog(t, root)
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
	if err := os.WriteFile(catalogPath+".bak", []byte("first-catalog"), 0o644); err != nil {
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
	catalogBackup, err := os.ReadFile(catalogPath + ".bak")
	if err != nil {
		t.Fatal(err)
	}
	if string(catalogBackup) != "first-catalog" {
		t.Fatalf("catalog backup content = %q", string(catalogBackup))
	}
}

func TestApplyAssetTableCreatesBackupCopiesBundleAndPatchesCatalog(t *testing.T) {
	root := t.TempDir()
	gameDir, catalogPath := writeTestCatalog(t, root)
	targetDir := filepath.Join(gameDir, "CrimeCleaner_Data", "StreamingAssets", "aa", "StandaloneWindows64")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, "localization-asset-tables-english(en)_assets_all.bundle")
	if err := os.WriteFile(target, []byte("original asset table"), 0o644); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(root, "fonts_en.bundle")
	if err := os.WriteFile(source, []byte("patched asset table"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ApplyAssetTable(gameDir, source, TargetEnglish)

	if err != nil {
		t.Fatalf("ApplyAssetTable returned error: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "patched asset table" {
		t.Fatalf("target content = %q", string(got))
	}
	if result.TargetPath != target {
		t.Fatalf("TargetPath = %q", result.TargetPath)
	}
	englishAsset := readCatalogOptions(t, catalogPath, testEnglishAssetID)
	if englishAsset.CRC != 0 || englishAsset.UseCRCForCachedBundles || englishAsset.BundleSize != 19 {
		t.Fatalf("English asset catalog entry was not patched: %+v", englishAsset)
	}
	englishStrings := readCatalogOptions(t, catalogPath, testEnglishInternalID)
	if englishStrings.CRC != 1111111111 || !englishStrings.UseCRCForCachedBundles {
		t.Fatalf("English string catalog entry was changed: %+v", englishStrings)
	}
}
