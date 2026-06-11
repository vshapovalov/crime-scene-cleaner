# Crime Scene Cleaner Ukrainian Localizer

Windows desktop patcher for installing a Ukrainian text localization over an existing Crime Scene Cleaner text language.

The app does not modify Wwise soundbanks or voice/audio files. It replaces only one Unity Localization string-table bundle.

## User Flow

1. Start the app.
2. The app detects Crime Scene Cleaner through local Steam libraries.
3. Select `замість англійської` or `замість польської`.
4. Click `Застосувати переклад`.

The app creates a `.bak` backup of the selected original language bundle before copying the Ukrainian bundle.

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
