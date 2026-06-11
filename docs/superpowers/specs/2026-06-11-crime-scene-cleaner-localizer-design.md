# Crime Scene Cleaner Localizer Design

## Goal

Create a Windows desktop patcher for Crime Scene Cleaner that installs a Ukrainian text bundle over an existing supported text language while preserving the game's original audio.

## User Flow

The app detects the local Steam installation and checks whether app `1040200` is installed. The main window shows the game branding area, the user agreement area, a replacement-language dropdown, and an apply button. The user selects either `instead of English` or `instead of Polish`, then clicks `Apply translation`.

The `Edit translation` button is out of scope for the first version.

## Patch Behavior

A replacement translation bundle sits next to the application binary. The backend copies that file over the selected game localization bundle:

- English target: `CrimeCleaner_Data/StreamingAssets/aa/StandaloneWindows64/localization-string-tables-english(en)_assets_all.bundle`
- Polish target: `CrimeCleaner_Data/StreamingAssets/aa/StandaloneWindows64/localization-string-tables-polish(pl)_assets_all.bundle`

Before replacement, the app creates a `.bak` copy of the original bundle if one does not already exist.

## Game Detection

The backend reads local Steam configuration from the Windows registry and Steam library folders. It finds the game by `steamapps/appmanifest_1040200.acf`, verifies the install directory, and exposes detected status, path, and version to the frontend. The first version reports the Steam manifest `buildid` as the game version.

## Technology

The app uses Wails v2, Go, Vue, Vite, Tailwind CSS, and shadcn-style Vue components. The UI follows the supplied mockup while using production controls and status messages.
