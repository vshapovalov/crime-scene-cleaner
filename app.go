package main

import (
	"context"
	"os"

	"crime-scene-cleaner/internal/patcher"
	"crime-scene-cleaner/internal/steam"
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
	return patcher.Apply(info.Path, patcher.RuntimeBundlePath(executable), request.Target)
}
