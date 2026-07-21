# Andino Kiosk — Local Package Installer
# Installs the Andino kiosk as a Docker container connected to Astra-System.
#
# Usage:
#   .\packages\andino-kiosk\install.ps1
#   .\packages\andino-kiosk\install.ps1 -BuildLocal -SourcePath "D:\selfservice-cafeteria"

param(
    [switch]$BuildLocal,
    [string]$SourcePath = "D:\selfservice-cafeteria",
    [string]$ComposeFile = "$PSScriptRoot\docker-compose.andino.yml",
    [string]$EnvFile = "$PSScriptRoot\.env.andino"
)

$ErrorActionPreference = "Stop"

Write-Host "Andino Kiosk — Package Installer" -ForegroundColor Cyan
Write-Host "=================================" -ForegroundColor Cyan
Write-Host ""

if (-not (Test-Path $EnvFile)) {
    Write-Host "  Creating .env.andino from template..." -ForegroundColor Yellow
    Copy-Item "$PSScriptRoot\.env.andino.example" $EnvFile
    Write-Host "  ! Edit $EnvFile with your Andino API credentials" -ForegroundColor Yellow
    Write-Host "  ! Then re-run this script." -ForegroundColor Yellow
    exit 0
}

if ($BuildLocal) {
    Write-Host "  Building Andino kiosk image from source..." -ForegroundColor White
    if (-not (Test-Path $SourcePath)) {
        Write-Host "  ✗ Source path not found: $SourcePath" -ForegroundColor Red
        exit 1
    }

    Push-Location $SourcePath
    try {
        docker build -t ghcr.io/xdfkenny/astra-system/andino-kiosk:latest -f packages/andino-kiosk/Dockerfile .
        Write-Host "  ✓ Image built" -ForegroundColor Green
    }
    finally {
        Pop-Location
    }
}

Write-Host "  Starting Andino kiosk container..." -ForegroundColor White
docker compose -f $ComposeFile --env-file $EnvFile up -d

Write-Host ""
Write-Host "  ✓ Andino Kiosk started!" -ForegroundColor Green
Write-Host "    Kiosk UI: http://localhost:3000" -ForegroundColor Cyan
Write-Host "    Health:   http://localhost:3000/api/health" -ForegroundColor Cyan
Write-Host "    Products: http://localhost:3000/api/products" -ForegroundColor Cyan
Write-Host ""
Write-Host "  To stop:   docker compose -f $ComposeFile down" -ForegroundColor Gray
