package steam

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseACFReadsNestedValues(t *testing.T) {
	content := `"AppState"
{
    "appid"     "1040200"
    "name"      "Crime Scene Cleaner"
    "buildid"   "18765432"
    "installdir" "Crime Scene Cleaner"
}`

	values := ParseACF(content)

	if values["appid"] != "1040200" {
		t.Fatalf("appid = %q", values["appid"])
	}
	if values["installdir"] != "Crime Scene Cleaner" {
		t.Fatalf("installdir = %q", values["installdir"])
	}
	if values["buildid"] != "18765432" {
		t.Fatalf("buildid = %q", values["buildid"])
	}
}

func TestFindGameInLibrariesReturnsInstalledGame(t *testing.T) {
	root := t.TempDir()
	library := filepath.Join(root, "SteamLibrary")
	steamapps := filepath.Join(library, "steamapps")
	game := filepath.Join(library, "steamapps", "common", "Crime Scene Cleaner")
	if err := os.MkdirAll(game, 0o755); err != nil {
		t.Fatal(err)
	}
	manifest := `"AppState"
{
    "appid" "1040200"
    "name" "Crime Scene Cleaner"
    "buildid" "18765432"
    "installdir" "Crime Scene Cleaner"
}`
	if err := os.WriteFile(filepath.Join(steamapps, "appmanifest_1040200.acf"), []byte(manifest), 0o644); err != nil {
		t.Fatal(err)
	}

	info, err := FindGameInLibraries([]string{library})

	if err != nil {
		t.Fatalf("FindGameInLibraries returned error: %v", err)
	}
	if !info.Installed {
		t.Fatal("expected game to be installed")
	}
	if info.Path != game {
		t.Fatalf("Path = %q, want %q", info.Path, game)
	}
	if info.Version != "18765432" {
		t.Fatalf("Version = %q", info.Version)
	}
}

func TestReadLibraryFoldersSupportsVDFLibraries(t *testing.T) {
	root := t.TempDir()
	steamRoot := filepath.Join(root, "Steam")
	config := filepath.Join(steamRoot, "steamapps")
	if err := os.MkdirAll(config, 0o755); err != nil {
		t.Fatal(err)
	}
	vdf := `"libraryfolders"
{
    "0"
    {
        "path" "C:\\Program Files (x86)\\Steam"
    }
    "1"
    {
        "path" "G:\\SteamLibrary"
    }
}`
	if err := os.WriteFile(filepath.Join(config, "libraryfolders.vdf"), []byte(vdf), 0o644); err != nil {
		t.Fatal(err)
	}

	libraries, err := ReadLibraryFolders(steamRoot)

	if err != nil {
		t.Fatalf("ReadLibraryFolders returned error: %v", err)
	}
	if len(libraries) != 2 {
		t.Fatalf("len(libraries) = %d", len(libraries))
	}
	if libraries[1] != `G:\SteamLibrary` {
		t.Fatalf("libraries[1] = %q", libraries[1])
	}
}
