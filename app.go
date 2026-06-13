package main

import (
	"context"
	"os"

	"crime-scene-cleaner/internal/editor"
	"crime-scene-cleaner/internal/patcher"
	"crime-scene-cleaner/internal/steam"

	"github.com/wailsapp/wails/v2/pkg/runtime"
)

// App struct
type App struct {
	ctx context.Context
}

// NewApp creates a new App application struct
func NewApp() *App {
	return &App{}
}

// startup is called when the app starts. The context is saved
// so we can call the runtime methods
func (a *App) startup(ctx context.Context) {
	a.ctx = ctx
}

type AppStatus struct {
	Game steam.GameInfo `json:"game"`
}

type ApplyRequest struct {
	Target patcher.TargetLanguage `json:"target"`
}

type EditorProgress struct {
	Stage   string `json:"stage"`
	Percent int    `json:"percent"`
	Message string `json:"message"`
}

func (a *App) GetGameStatus() AppStatus {
	info, _ := steam.DetectGame()
	return AppStatus{Game: info}
}

func (a *App) ApplyTranslation(request ApplyRequest) (patcher.ApplyResult, error) {
	info, err := steam.DetectGame()
	if err != nil {
		return patcher.ApplyResult{}, err
	}
	if !info.Installed {
		return patcher.ApplyResult{}, os.ErrNotExist
	}

	executable, err := os.Executable()
	if err != nil {
		return patcher.ApplyResult{}, err
	}
	sourceBundle, err := patcher.RuntimeBundlePathForTarget(executable, request.Target)
	if err != nil {
		return patcher.ApplyResult{}, err
	}
	return patcher.Apply(info.Path, sourceBundle, request.Target)
}

func (a *App) GetEditorToolingStatus() editor.ToolingStatus {
	executable, err := os.Executable()
	if err != nil {
		return editor.ToolingStatus{Message: err.Error()}
	}
	return editor.CheckTooling(editor.RuntimeBundleToolPath(executable))
}

func (a *App) InstallEditorTooling() (editor.ToolingStatus, error) {
	a.emitEditorProgress("tooling", 20, "Перевіряємо BundleTool.exe")
	status := a.GetEditorToolingStatus()
	a.emitEditorProgress("tooling", 100, status.Message)
	return status, nil
}

func (a *App) LoadTranslationEditor() (editor.EditorData, error) {
	executable, err := os.Executable()
	if err != nil {
		return editor.EditorData{}, err
	}
	bundlePath := patcher.RuntimeBundlePath(executable)
	dictionaryPath := editor.DictionaryBundlePath(bundlePath)
	bundleToolPath := editor.RuntimeBundleToolPath(executable)

	a.emitEditorProgress("load", 10, "Перевіряємо інструменти редактора")
	status := editor.CheckTooling(bundleToolPath)
	if !status.Ready {
		a.emitEditorProgress("load", 100, status.Message)
		return editor.EditorData{}, os.ErrNotExist
	}
	a.emitEditorProgress("load", 25, "Готуємо редагований бандл перекладу")
	info, err := steam.DetectGame()
	if err != nil {
		a.emitEditorProgress("load", 100, "Не вдалося перевірити гру")
		return editor.EditorData{}, err
	}
	if !info.Installed {
		a.emitEditorProgress("load", 100, "Гру не знайдено")
		return editor.EditorData{}, os.ErrNotExist
	}
	if err := editor.EnsureEditorBundles(bundlePath, dictionaryPath, info.Path); err != nil {
		a.emitEditorProgress("load", 100, "Не вдалося підготувати редагований бандл")
		return editor.EditorData{}, err
	}

	a.emitEditorProgress("load", 45, "Розпаковуємо ukrainian-localization.bundle")
	data, err := editor.ExportBundle(context.Background(), editor.ExecRunner{}, bundleToolPath, bundlePath, dictionaryPath)
	if err != nil {
		a.emitEditorProgress("load", 100, "Не вдалося прочитати бандл перекладу")
		return editor.EditorData{}, err
	}
	a.emitEditorProgress("load", 100, "Таблицю перекладу завантажено")
	return data, nil
}

func (a *App) SaveTranslationEditor(rows []editor.TranslationRow) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	bundlePath := patcher.RuntimeBundlePath(executable)
	bundleToolPath := editor.RuntimeBundleToolPath(executable)

	a.emitEditorProgress("save", 10, "Перевіряємо інструменти редактора")
	status := editor.CheckTooling(bundleToolPath)
	if !status.Ready {
		a.emitEditorProgress("save", 100, status.Message)
		return os.ErrNotExist
	}
	info, err := steam.DetectGame()
	if err != nil {
		a.emitEditorProgress("save", 100, "Не вдалося перевірити гру")
		return err
	}
	if !info.Installed {
		a.emitEditorProgress("save", 100, "Гру не знайдено")
		return os.ErrNotExist
	}
	englishTemplate, err := patcher.TargetBundlePath(info.Path, patcher.TargetEnglish)
	if err != nil {
		return err
	}
	polishTemplate, err := patcher.TargetBundlePath(info.Path, patcher.TargetPolish)
	if err != nil {
		return err
	}
	a.emitEditorProgress("save", 35, "Запаковуємо переклад у бандл")
	if err := editor.ImportBundle(context.Background(), editor.ExecRunner{}, bundleToolPath, bundlePath, rows, englishTemplate, polishTemplate); err != nil {
		a.emitEditorProgress("save", 100, "Не вдалося зберегти бандл перекладу")
		return err
	}
	a.emitEditorProgress("save", 100, "Бандл перекладу збережено")
	return nil
}

func (a *App) ExportTranslationJSON(rows []editor.TranslationRow) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Експорт перекладу в JSON",
		DefaultFilename: "ukrainian-localization.json",
		Filters: []runtime.FileFilter{{
			DisplayName: "Файли JSON (*.json)",
			Pattern:     "*.json",
		}},
		CanCreateDirectories: true,
	})
	if err != nil || path == "" {
		return path, err
	}
	data, err := editor.MarshalRows(rows)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return "", err
	}
	return path, nil
}

func (a *App) ImportTranslationJSON() ([]editor.TranslationRow, error) {
	path, err := runtime.OpenFileDialog(a.ctx, runtime.OpenDialogOptions{
		Title: "Імпорт перекладу з JSON",
		Filters: []runtime.FileFilter{{
			DisplayName: "Файли JSON (*.json)",
			Pattern:     "*.json",
		}},
	})
	if err != nil || path == "" {
		return []editor.TranslationRow{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return editor.UnmarshalRows(data)
}

func (a *App) emitEditorProgress(stage string, percent int, message string) {
	if a.ctx == nil {
		return
	}
	runtime.EventsEmit(a.ctx, "translation-editor-progress", EditorProgress{
		Stage:   stage,
		Percent: percent,
		Message: message,
	})
}
