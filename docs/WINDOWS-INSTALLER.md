# Astra-System Windows Installer & Update System

## Overview

The Astra-System Windows Installer provides a production-grade installation and
automatic update mechanism for deploying the Astra-System self-checkout platform
on Windows kiosk machines. The system is built from three layers:

| Layer | Technology | Purpose |
|---|---|---|
| **astra-installer** | Go CLI | Prerequisite checking, directory setup, service registration |
| **astra-updater** | Go Windows Service | Background update polling, download, and apply |
| **Inno Setup** | setup.iss | Windows installer shell (EXE, shortcuts, uninstall) |

All source code lives under `installer/`. The installer is built by CI on tag
push and published as a GitHub Release asset.

---

## Directory Layout

```
installer/
├── astra-installer/               # Go installer CLI
│   ├── cmd/astra-installer/main.go
│   └── internal/
│       ├── prereq/prereq.go       # Docker Desktop detection
│       └── setup/setup.go         # Directory creation, config, service install
├── astra-updater/                 # Go update agent (Windows service)
│   ├── cmd/astra-updater/main.go  # Service entry point + foreground mode
│   └── internal/
│       ├── check/check.go         # GitHub Releases API client + semver compare
│       ├── download/download.go   # Asset download with SHA-256 verification
│       └── apply/apply.go         # Stack stop/backup/install/restart lifecycle
├── resources/
│   ├── astra.conf.template        # Runtime configuration template
│   └── .env.template              # Environment variable template
├── scripts/
│   ├── bootstrap.ps1              # One-liner: fresh Windows install
│   └── release.ps1                # Tag & push helper for maintainers
├── setup.iss                      # Inno Setup compiler script
├── CHANNELS.md                    # Channel system reference
└── version.json                   # Manifest schema template
```

---

## Architecture

### Installation Flow

```
Astra-System-Setup.exe
        │
        ▼
  ┌─────────────┐
  │  Inno Setup │  Copies files → Program Files\Astra-System
  │  (setup.iss)│  Creates shortcuts, runs astra-installer --silent
  └──────┬──────┘
         │
         ▼
  ┌────────────────┐
  │ astra-installer │  Checks Docker Desktop
  │  (Go CLI)      │  Creates %PROGRAMDATA%\Astra-System\
  └──────┬─────────┘  Writes runtime config
         │            Registers AstraUpdateAgent service
         ▼
  ┌────────────────┐
  │   System Ready │   Kiosk available at http://localhost
  └────────────────┘
```

### Update Flow

```
  ┌──────────────────────┐
  │  AstraUpdateAgent    │  Windows service, runs as LOCAL SYSTEM
  │  (Go Windows Service)│  Polls GitHub Releases every 6 hours
  └──────────┬───────────┘
             │
             ▼
  ┌────────────────────┐
  │  check.LatestRelease│  GET api.github.com/repos/.../releases
  │                    │  Filter by channel (stable/beta/canary)
  │                    │  Compare semver against current version
  └──────────┬─────────┘
             │ (new version found)
             ▼
  ┌────────────────────┐
  │  download.Asset    │  Download installer EXE → %DATA%\staging\
  │                    │  Verify SHA-256 checksum
  └──────────┬─────────┘
             │
             ▼
  ┌────────────────────┐
  │  apply.Update      │  1. docker compose down
  │                    │  2. Backup %DATA%\config\
  │                    │  3. Run new installer --silent
  │                    │  4. docker compose up -d --pull always
  │                    │  5. Record applied update
  └────────────────────┘
```

---

## Channel System

Three channels control update delivery:

| Channel | Tag Suffix | Stability | CI Publish |
|---|---|---|---|
| **stable** | (none) or `-stable` | Production-ready | Full release |
| **beta** | `-beta` | Pre-release | Pre-release on GitHub |
| **canary** | `-canary` | Bleeding-edge | Pre-release on GitHub |

Tag examples:

```
v0.2.0         → stable
v0.2.1-beta    → beta
v0.2.2-canary  → canary
```

The update agent filters releases by channel. A kiosk configured to the `beta`
channel will only see releases tagged with `-beta`. Channel is set at install
time and stored in `%PROGRAMDATA%\Astra-System\config\astra.conf`.

---

## Creating a Release

### Prerequisites

- Git push access to the repository
- A GitHub personal access token with `repo` scope (for `gh` CLI)

### With the helper script (recommended)

```powershell
.\installer\scripts\release.ps1 -Version 0.3.0 -Channel beta -Message "Bug fixes and improvements"
```

This commits pending changes, creates an annotated tag, and pushes it. The CI
workflow `build-installer.yml` automatically builds the installer and creates a
GitHub Release.

### Manually

```bash
git tag v0.3.0-stable
git push origin v0.3.0-stable
```

### What CI does

The `build-installer.yml` workflow (`.github/workflows/build-installer.yml`):

1. Checks out the repository
2. Builds `astra-installer.exe` and `astra-updater.exe` with Go 1.25
3. Downloads and installs Inno Setup
4. Patches the version number into `setup.iss`
5. Compiles `Astra-System-Setup.exe`
6. Generates `SHA-256` checksum and `update-manifest.json`
7. Determines channel from the tag name
8. Creates a GitHub Release with all three assets attached
9. Marks as pre-release if channel is not `stable`

---

## Installing on a Fresh Machine

### Method 1: Download from GitHub Releases

1. Go to https://github.com/astra-service/Astra-System/releases
2. Download the latest `Astra-System-Setup.exe` for your channel
3. Run the installer as Administrator
4. The installer checks for Docker Desktop and sets up the system
5. Open http://localhost to access the kiosk

### Method 2: Bootstrap script (one-liner)

From an elevated PowerShell:

```powershell
[Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/astra-service/Astra-System/main/installer/scripts/bootstrap.ps1'))
```

This automatically:
- Checks if Docker Desktop is installed
- Offers to install Docker Desktop if missing
- Fetches the latest stable release from GitHub
- Runs the installer silently

### Silent Installation

```cmd
Astra-System-Setup.exe /SILENT /DIR="C:\Program Files\Astra-System"
```

---

## Runtime Configuration

### Configuration File

Path: `%PROGRAMDATA%\Astra-System\config\astra.conf`

```
ASTRA_INSTALL_DIR=C:\Program Files\Astra-System
ASTRA_DATA_DIR=C:\ProgramData\Astra-System
ASTRA_UPDATE_CHANNEL=stable
ASTRA_COMPOSE_DIR=C:\ProgramData\Astra-System\compose
```

### Environment File

Path: `%PROGRAMDATA%\Astra-System\.env`

Controls Docker Compose runtime variables (database passwords, ports, etc.).

### Changing the Update Channel

Edit `%PROGRAMDATA%\Astra-System\config\astra.conf` and change:

```
ASTRA_UPDATE_CHANNEL=beta
```

Then restart the update agent:

```cmd
sc stop AstraUpdateAgent
sc start AstraUpdateAgent
```

---

## Component Reference

### `astra-installer.exe`

| Flag | Default | Description |
|---|---|---|
| `--install-dir` | `%ProgramFiles%\Astra-System` | Application installation directory |
| `--data-dir` | `%ProgramData%\Astra-System` | Application data directory |
| `--channel` | `stable` | Update channel |
| `--silent` | `false` | No prompts, minimal output |
| `--no-docker` | `false` | Skip Docker Desktop check |
| `--version` | — | Print version and exit |

### `astra-updater.exe`

| Command | Description |
|---|---|
| `install` | Register as Windows service |
| `remove` | Unregister Windows service |
| `run` | Run in foreground (for debugging) |
| `version` | Print version and exit |

Flags for `install` and `run`:

| Flag | Default | Description |
|---|---|---|
| `--install-dir` | `%ProgramFiles%\Astra-System` | Install directory |
| `--data-dir` | `%ProgramData%\Astra-System` | Data directory |
| `--channel` | `stable` | Update channel |
| `--interval` | `6h` | Poll interval (for `run` command) |

---

## Update Manifest Schema

The CI generates `update-manifest.json` and attaches it to every release:

```json
{
  "version": "v0.3.0-beta",
  "channel": "beta",
  "releasedAt": "2026-07-20T20:45:00Z",
  "artifacts": {
    "astra-installer": {
      "url": "https://github.com/astra-service/Astra-System/releases/download/v0.3.0-beta/Astra-System-Setup.exe",
      "checksum": "sha256:abc123...",
      "platforms": ["windows/amd64"]
    }
  },
  "rollout": {
    "strategy": "idle-only",
    "maxConcurrent": 10,
    "healthCheckSeconds": 300
  }
}
```

This schema mirrors the existing `update-server` manifest format used for kiosk
OTA updates.

---

## Uninstallation

### Via Control Panel

1. Open **Settings > Apps > Installed apps**
2. Search for **Astra-System**
3. Click **Uninstall**

The uninstaller automatically stops and removes the `AstraUpdateAgent` Windows
service and cleans up configuration files.

### Silent Uninstall

```cmd
"C:\Program Files\Astra-System\unins000.exe" /SILENT
```

---

## Troubleshooting

### Docker Desktop not found

The installer checks for Docker Desktop at installation time. If missing:

- Install Docker Desktop from https://docs.docker.com/desktop/setup/install/windows-install/
- Or bypass with `astra-installer --no-docker` and install it later

### Update agent not starting

Check the service status:

```cmd
sc query AstraUpdateAgent
```

View logs in `%ProgramData%\Astra-System\logs\`. The update agent logs all
check/download/apply operations.

### Rollback after a bad update

The update agent creates backups before applying:

```
%ProgramData%\Astra-System\backups\pre-update-20260720T204500Z\
```

To rollback: stop the stack, restore the config backup, and restart.

---

## Development

### Building Locally

```powershell
# Build the Go binaries
cd installer\astra-installer
go build -o ..\bin\astra-installer.exe .\cmd\astra-installer\

cd ..\astra-updater
go build -o ..\bin\astra-updater.exe .\cmd\astra-updater\

# Compile the installer (requires Inno Setup)
ISCC.exe installer\setup.iss
```

### Testing the Update Agent

```powershell
# Run in foreground (not as a service)
astra-updater.exe run --channel beta --interval 5m
```

### Updating Go Dependencies

```powershell
cd installer\astra-installer
go get -u
go mod tidy

cd ..\astra-updater
go get -u
go mod tidy
```
