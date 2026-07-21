# Astra-System Bootstrap Script
# Run this on a fresh Windows machine to set up prerequisites
# and install Astra-System from the latest GitHub Release.
#
# Usage:
#   [Net.ServicePointManager]::SecurityProtocol = [Net.SecurityProtocolType]::Tls12
#   iex ((New-Object System.Net.WebClient).DownloadString('https://raw.githubusercontent.com/astra-service/Astra-System/main/installer/scripts/bootstrap.ps1'))

param(
    [string]$Channel = "stable",
    [string]$InstallDir = "${env:ProgramFiles}\Astra-System",
    [string]$DataDir = "${env:ProgramData}\Astra-System"
)

$ErrorActionPreference = "Stop"
$repoOwner = "astra-service"
$repoName = "Astra-System"

Write-Host "Astra-System Bootstrap" -ForegroundColor Cyan
Write-Host "======================" -ForegroundColor Cyan
Write-Host ""

function Get-LatestRelease {
    param([string]$Channel)
    $url = "https://api.github.com/repos/$repoOwner/$repoName/releases"
    $releases = Invoke-RestMethod -Uri $url -Headers @{ "Accept" = "application/vnd.github.v3+json" }
    foreach ($r in $releases) {
        $tag = $r.tag_name
        $releaseChannel = "stable"
        if ($tag -match "-(\w+)$") { $releaseChannel = $matches[1] }
        if ($releaseChannel -ne $Channel) { continue }
        foreach ($asset in $r.assets) {
            if ($asset.name -match "Astra-System-Setup.*\.exe$") {
                return @{
                    Tag = $tag
                    URL = $asset.browser_download_url
                    Name = $asset.name
                }
            }
        }
    }
    throw "No release found for channel '$Channel'"
}

function Test-DockerDesktop {
    try {
        $version = docker version --format "{{.Server.Version}}" 2>&1
        if ($LASTEXITCODE -eq 0 -and $version) {
            Write-Host "  ✓ Docker Desktop $version is running" -ForegroundColor Green
            return $true
        }
    } catch {}
    try {
        $null = Get-Command docker -ErrorAction Stop
        Write-Host "  ! Docker Desktop is installed but not running" -ForegroundColor Yellow
        return $false
    } catch {
        Write-Host "  ✗ Docker Desktop is not installed" -ForegroundColor Red
        return $false
    }
}

function Install-DockerDesktop {
    Write-Host "  → Downloading Docker Desktop for Windows..." -ForegroundColor Yellow
    $url = "https://desktop.docker.com/win/stable/Docker%20Desktop%20Installer.exe"
    $out = "$env:TEMP\DockerDesktopInstaller.exe"
    Invoke-WebRequest -Uri $url -OutFile $out -UseBasicParsing
    Write-Host "  → Installing Docker Desktop (automatic WSL2 setup)..." -ForegroundColor Yellow
    Start-Process -FilePath $out -ArgumentList "install", "--quiet", "--accept-license" -Wait
    Write-Host "  ✓ Docker Desktop installed. Please restart your computer to complete setup." -ForegroundColor Green
}

## Main
Write-Host "Step 1: Checking prerequisites..." -ForegroundColor White
if (-not (Test-DockerDesktop)) {
    $choice = Read-Host "  Docker Desktop is required. Install now? (Y/n)"
    if ($choice -ne "n") {
        Install-DockerDesktop
        Write-Host "  Re-run this script after restarting." -ForegroundColor Yellow
        exit 0
    } else {
        Write-Host "  Docker Desktop is required for Astra-System. Exiting." -ForegroundColor Red
        exit 1
    }
}

Write-Host ""
Write-Host "Step 2: Fetching latest release ($Channel channel)..." -ForegroundColor White
$release = Get-LatestRelease -Channel $Channel
Write-Host "  Found: $($release.Tag) - $($release.Name)" -ForegroundColor Green

Write-Host ""
Write-Host "Step 3: Downloading installer..." -ForegroundColor White
$installerPath = "$env:TEMP\$($release.Name)"
Invoke-WebRequest -Uri $release.URL -OutFile $installerPath -UseBasicParsing
Write-Host "  Downloaded to: $installerPath" -ForegroundColor Green

Write-Host ""
Write-Host "Step 4: Running installer..." -ForegroundColor White
$params = @("/SILENT", "/DIR=""$InstallDir""", "/DATADIR=""$DataDir""")
Write-Host "  $installerPath $params" -ForegroundColor Gray
Start-Process -FilePath $installerPath -ArgumentList $params -Wait

Write-Host ""
Write-Host "✓ Astra-System installation complete!" -ForegroundColor Cyan
Write-Host "  Open http://localhost to access the kiosk." -ForegroundColor Cyan
Write-Host ""
Write-Host "  The update agent will automatically keep your system up to date." -ForegroundColor Gray
