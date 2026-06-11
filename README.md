# Crime Scene Cleaner Ukrainian Localizer

Windows desktop patcher for installing a Ukrainian text localization over an existing Crime Scene Cleaner text language.

The app does not modify Wwise soundbanks or voice/audio files. It replaces only one Unity Localization string-table bundle.

## User Flow

1. Start the app.
2. The app detects Crime Scene Cleaner through local Steam libraries.
3. Select `замість англійської` or `замість польської`.
4. Click `Застосувати переклад`.

The app creates a `.bak` backup of the selected original language bundle before copying the Ukrainian bundle.

## Translation Editor

The editor button is hidden by default. Click `v1.0.0` in the bottom-right corner to show `Редагувати переклад`.

When the editor opens, the app checks for Python and UnityPy. If UnityPy is missing, the app asks for confirmation before installing it with pip. After tooling is ready, the app loads `ukrainian-localization.bundle` from next to the executable.

If `ukrainian-localization.bundle` does not exist yet, the app copies the Russian game string-table bundle and uses that as the first editable translation bundle.

The editor view contains:

- `Назад` to return to the main screen.
- `Зберегти` to pack edited rows back into `ukrainian-localization.bundle`.
- A translation table with Unity table name, string ID, and editable text.

## Runtime Translation File

Place the Ukrainian replacement bundle next to the built executable:

```text
build/bin/crime-scene-cleaner.exe
build/bin/ukrainian-localization.bundle
```

The replacement bundle is copied to one of these game files:

```text
CrimeCleaner_Data/StreamingAssets/aa/StandaloneWindows64/localization-string-tables-english(en)_assets_all.bundle
CrimeCleaner_Data/StreamingAssets/aa/StandaloneWindows64/localization-string-tables-polish(pl)_assets_all.bundle
```

## Development

```powershell
go test ./...
cd frontend
npm.cmd install
npm.cmd run build
cd ..
wails build
```

Use `npm.cmd` on Windows PowerShell if script execution policy blocks `npm.ps1`.
