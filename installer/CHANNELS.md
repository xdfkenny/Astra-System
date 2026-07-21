# Astra-System Release Channels

The installer and update agent support three release channels to control how updates are delivered.

## Channel Overview

| Channel | Tag Suffix | Stability | Update Frequency | Use Case |
|---|---|---|---|---|
| **stable** | (none) or `-stable` | Production-ready | Manual, tested releases | Customer kiosks |
| **beta** | `-beta` | Pre-release | Weekly | Staging, QA testing |
| **canary** | `-canary` | Bleeding-edge | Per-commit | Development, CI |

## Tag Convention

Git tags determine the channel:

```
v0.2.0            → stable channel
v0.2.0-stable     → stable channel (explicit)
v0.2.1-beta       → beta channel
v0.2.2-canary     → canary channel
```

## How It Works

1. A maintainer pushes a tag: `git tag v0.2.1-beta && git push origin v0.2.1-beta`
2. The `build-installer.yml` GitHub Actions workflow:
   - Detects the tag and determines the channel from the suffix
   - Builds `astra-installer.exe` and `astra-updater.exe`
   - Compiles the Inno Setup installer
   - Creates a GitHub Release with the installer attached
   - Marks as pre-release if not stable
3. Each kiosk's `astra-updater` Windows service:
   - Periodically queries `api.github.com/repos/astra-service/Astra-System/releases`
   - Filters releases matching its configured channel
   - Compares semver to find a newer version
   - Downloads and applies the update

## Creating a Release

### Via CI (recommended)

```bash
# Create and push a tag
git tag v0.3.0-beta
git push origin v0.3.0-beta
```

The CI will automatically build the installer and create the release.

### Via CLI tool

```bash
# Using the release helper
installer\scripts\release.ps1 -Version 0.3.0 -Channel beta
```

## Channel Configuration

The channel is set at installation time and stored in the runtime config:

| Method | How |
|---|---|
| **Installer GUI** | Select channel during setup |
| **Silent install** | `astra-installer.exe --channel beta` |
| **Post-install** | Edit `%PROGRAMDATA%\Astra-System\config\astra.conf` |

## Rollout Strategy

- **stable**: Full rollout to all matching kiosks immediately on release
- **beta**: Available to opt-in kiosks only
- **canary**: Available to development kiosks; may be unstable

The update agent checks every 6 hours by default. This interval can be configured via the `--interval` flag on the `astra-updater run` command.
