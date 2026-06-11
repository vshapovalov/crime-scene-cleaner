# Crime Scene Cleaner Localizer Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build and publish the initial Wails desktop app for applying a Ukrainian localization bundle over English or Polish text resources.

**Architecture:** Keep Steam/game detection and patch application in focused Go services with tests. Keep the Wails `App` layer thin and expose simple DTOs to the Vue frontend.

**Tech Stack:** Go, Wails v2, Vue 3, Vite, Tailwind CSS, shadcn-style components, GitHub CLI.

---

### Task 1: Repository Setup

**Files:**
- Create: `G:/projects/crime-scene-cleaner`
- Create: GitHub repo `vshapovalov/crime-scene-cleaner`

- [x] Scaffold Wails Vue app with Git initialized.
- [x] Create public GitHub repository and configure `origin`.

### Task 2: Backend Services

**Files:**
- Create: `internal/steam/steam.go`
- Create: `internal/steam/steam_test.go`
- Create: `internal/patcher/patcher.go`
- Create: `internal/patcher/patcher_test.go`
- Modify: `app.go`

- [ ] Write failing tests for Steam manifest parsing, library detection, version reporting, target path generation, backup creation, and replacement copy.
- [ ] Implement minimal Go code to satisfy tests.
- [ ] Expose `GetGameStatus` and `ApplyTranslation` through Wails.

### Task 3: Frontend UI

**Files:**
- Modify: `frontend/package.json`
- Create: `frontend/tailwind.config.js`
- Create: `frontend/postcss.config.js`
- Modify: `frontend/src/App.vue`
- Modify: `frontend/src/style.css`

- [ ] Install Tailwind and class helper dependencies.
- [ ] Implement shadcn-style controls for select, button, and status display.
- [ ] Match the supplied window layout.
- [ ] Hide or disable edit-translation behavior for the first version.

### Task 4: Build Assets and Verification

**Files:**
- Create: `translations/.gitkeep`
- Modify: `README.md`

- [ ] Document that the runtime bundle must be placed next to the app binary.
- [ ] Run Go tests.
- [ ] Run frontend build.
- [ ] Run Wails build.
- [ ] Commit and push to `origin/master`.
