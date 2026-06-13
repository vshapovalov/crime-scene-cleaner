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
	return editor.CheckTooling(context.Background(), editor.ExecRunner{}, editor.DefaultPythonExecutable())
}

func (a *App) InstallEditorTooling() (editor.ToolingStatus, error) {
	python := editor.DefaultPythonExecutable()
	a.emitEditorProgress("tooling", 10, "Checking Python")
	status := editor.CheckTooling(context.Background(), editor.ExecRunner{}, python)
	if !status.PythonAvailable {
		a.emitEditorProgress("tooling", 100, status.Message)
		return status, nil
	}
	if status.UnityPyAvailable {
		a.emitEditorProgress("tooling", 100, "Editor tooling is ready")
		return status, nil
	}

	a.emitEditorProgress("tooling", 35, "Installing UnityPy")
	if err := editor.InstallUnityPy(context.Background(), editor.ExecRunner{}, python); err != nil {
		a.emitEditorProgress("tooling", 100, "UnityPy installation failed")
		return editor.CheckTooling(context.Background(), editor.ExecRunner{}, python), err
	}
	a.emitEditorProgress("tooling", 85, "Verifying UnityPy")
	status = editor.CheckTooling(context.Background(), editor.ExecRunner{}, python)
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
	python := editor.DefaultPythonExecutable()

	a.emitEditorProgress("load", 10, "Checking editor tooling")
	status := editor.CheckTooling(context.Background(), editor.ExecRunner{}, python)
	if !status.Ready {
		a.emitEditorProgress("load", 100, status.Message)
		return editor.EditorData{}, os.ErrNotExist
	}
	a.emitEditorProgress("load", 25, "Preparing editable translation bundle")
	info, err := steam.DetectGame()
	if err != nil {
		a.emitEditorProgress("load", 100, "Failed to detect game")
		return editor.EditorData{}, err
	}
	if !info.Installed {
		a.emitEditorProgress("load", 100, "Game was not found")
		return editor.EditorData{}, os.ErrNotExist
	}
	if err := editor.EnsureEditorBundles(bundlePath, dictionaryPath, info.Path); err != nil {
		a.emitEditorProgress("load", 100, "Failed to prepare editable bundle")
		return editor.EditorData{}, err
	}

	a.emitEditorProgress("load", 45, "Unpacking ukrainian-localization.bundle")
	data, err := editor.ExportBundle(context.Background(), editor.ExecRunner{}, python, bundlePath, dictionaryPath)
	if err != nil {
		a.emitEditorProgress("load", 100, "Failed to read translation bundle")
		return editor.EditorData{}, err
	}
	a.emitEditorProgress("load", 100, "Translation table loaded")
	return data, nil
}

func (a *App) SaveTranslationEditor(rows []editor.TranslationRow) error {
	executable, err := os.Executable()
	if err != nil {
		return err
	}
	bundlePath := patcher.RuntimeBundlePath(executable)
	python := editor.DefaultPythonExecutable()

	a.emitEditorProgress("save", 10, "Checking editor tooling")
	status := editor.CheckTooling(context.Background(), editor.ExecRunner{}, python)
	if !status.Ready {
		a.emitEditorProgress("save", 100, status.Message)
		return os.ErrNotExist
	}
	info, err := steam.DetectGame()
	if err != nil {
		a.emitEditorProgress("save", 100, "Failed to detect game")
		return err
	}
	if !info.Installed {
		a.emitEditorProgress("save", 100, "Game was not found")
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
	a.emitEditorProgress("save", 35, "Packing translations into bundle")
	if err := editor.ImportBundle(context.Background(), editor.ExecRunner{}, python, bundlePath, rows, englishTemplate, polishTemplate); err != nil {
		a.emitEditorProgress("save", 100, "Failed to save translation bundle")
		return err
	}
	a.emitEditorProgress("save", 100, "Translation bundle saved")
	return nil
}

func (a *App) ExportTranslationJSON(rows []editor.TranslationRow) (string, error) {
	path, err := runtime.SaveFileDialog(a.ctx, runtime.SaveDialogOptions{
		Title:           "Export translation JSON",
		DefaultFilename: "ukrainian-localization.json",
		Filters: []runtime.FileFilter{{
			DisplayName: "JSON files (*.json)",
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
		Title: "Import translation JSON",
		Filters: []runtime.FileFilter{{
			DisplayName: "JSON files (*.json)",
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
