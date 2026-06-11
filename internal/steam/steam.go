package steam

import (
	"bufio"
	"errors"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strings"

	"golang.org/x/sys/windows/registry"
)

const CrimeSceneCleanerAppID = "1040200"

type GameInfo struct {
	Installed bool   `json:"installed"`
	Path      string `json:"path"`
	Version   string `json:"version"`
	Message   string `json:"message"`
}

func ParseACF(content string) map[string]string {
	values := map[string]string{}
	re := regexp.MustCompile(`^\s*"([^"]+)"\s+"([^"]*)"\s*$`)
	scanner := bufio.NewScanner(strings.NewReader(content))
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 3 {
			values[matches[1]] = matches[2]
		}
	}
	return values
}

func ReadLibraryFolders(steamRoot string) ([]string, error) {
	data, err := os.ReadFile(filepath.Join(steamRoot, "steamapps", "libraryfolders.vdf"))
	if err != nil {
		return nil, err
	}

	re := regexp.MustCompile(`^\s*"path"\s+"([^"]+)"\s*$`)
	var libraries []string
	scanner := bufio.NewScanner(strings.NewReader(string(data)))
	for scanner.Scan() {
		matches := re.FindStringSubmatch(scanner.Text())
		if len(matches) == 2 {
			libraries = append(libraries, strings.ReplaceAll(matches[1], `\\`, `\`))
		}
	}
	if len(libraries) == 0 {
		libraries = append(libraries, steamRoot)
	}
	return uniqueStrings(libraries), scanner.Err()
}

func FindGameInLibraries(libraries []string) (GameInfo, error) {
	for _, library := range uniqueStrings(libraries) {
		manifestPath := filepath.Join(library, "steamapps", "appmanifest_"+CrimeSceneCleanerAppID+".acf")
		data, err := os.ReadFile(manifestPath)
		if err != nil {
			continue
		}
		values := ParseACF(string(data))
		installDir := values["installdir"]
		if installDir == "" {
			installDir = "Crime Scene Cleaner"
		}
		gamePath := filepath.Join(library, "steamapps", "common", installDir)
		if stat, err := os.Stat(gamePath); err == nil && stat.IsDir() {
			return GameInfo{
				Installed: true,
				Path:      gamePath,
				Version:   values["buildid"],
				Message:   "Game detected",
			}, nil
		}
	}
	return GameInfo{Installed: false, Message: "Crime Scene Cleaner was not found in local Steam libraries"}, nil
}

func DetectGame() (GameInfo, error) {
	steamRoot, err := SteamRoot()
	if err != nil {
		return GameInfo{Installed: false, Message: err.Error()}, nil
	}
	libraries, err := ReadLibraryFolders(steamRoot)
	if err != nil {
		return GameInfo{Installed: false, Message: "Steam library configuration was not found"}, nil
	}
	if !containsString(libraries, steamRoot) {
		libraries = append([]string{steamRoot}, libraries...)
	}
	return FindGameInLibraries(libraries)
}

func SteamRoot() (string, error) {
	if runtime.GOOS != "windows" {
		return "", errors.New("Steam auto-detection is only implemented for Windows")
	}

	for _, keyName := range []string{
		`SOFTWARE\WOW6432Node\Valve\Steam`,
		`SOFTWARE\Valve\Steam`,
	} {
		key, err := registry.OpenKey(registry.LOCAL_MACHINE, keyName, registry.QUERY_VALUE)
		if err != nil {
			continue
		}
		path, _, err := key.GetStringValue("InstallPath")
		_ = key.Close()
		if err == nil && path != "" {
			return path, nil
		}
	}

	key, err := registry.OpenKey(registry.CURRENT_USER, `Software\Valve\Steam`, registry.QUERY_VALUE)
	if err == nil {
		path, _, valueErr := key.GetStringValue("SteamPath")
		_ = key.Close()
		if valueErr == nil && path != "" {
			return filepath.Clean(path), nil
		}
	}

	return "", errors.New("Steam installation was not found")
}

func uniqueStrings(values []string) []string {
	seen := map[string]bool{}
	var result []string
	for _, value := range values {
		clean := filepath.Clean(value)
		key := strings.ToLower(clean)
		if !seen[key] {
			seen[key] = true
			result = append(result, clean)
		}
	}
	return result
}

func containsString(values []string, needle string) bool {
	needle = strings.ToLower(filepath.Clean(needle))
	for _, value := range values {
		if strings.ToLower(filepath.Clean(value)) == needle {
			return true
		}
	}
	return false
}
