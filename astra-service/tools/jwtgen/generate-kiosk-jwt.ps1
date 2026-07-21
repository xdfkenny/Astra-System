# Generate EdDSA keys and a kiosk JWT for local development
# Usage: .\generate-kiosk-jwt.ps1 [-KioskId "kiosk-dev-001"]
param(
    [string]$KioskId = "kiosk-dev-001",
    [string]$OutDir = ".\keys"
)

$ErrorActionPreference = "Stop"
$ToolDir = "$PSScriptRoot"

Write-Host "Building jwtgen tool..." -ForegroundColor Cyan
$env:GOWORK = "off"
Push-Location "$ToolDir"
try {
    go build -o jwtgen.exe .
} finally {
    Pop-Location
}

Write-Host "Generating Ed25519 key pair..." -ForegroundColor Cyan
& "$ToolDir\jwtgen.exe" -generate -out $OutDir

Write-Host "Signing JWT for kiosk '$KioskId'..." -ForegroundColor Cyan
$jwt = & "$ToolDir\jwtgen.exe" -sign -key "$OutDir\kiosk-eddsa-private.pem" -sub $KioskId

$publicKeyPath = Resolve-Path "$OutDir\kiosk-eddsa-public.pem"
$privateKeyPath = Resolve-Path "$OutDir\kiosk-eddsa-private.pem"

Write-Host ""
Write-Host "============================================================" -ForegroundColor Green
Write-Host "  Keys generated in: $OutDir" -ForegroundColor Green
Write-Host "============================================================" -ForegroundColor Green
Write-Host ""

Write-Host "1. Set these env vars for the GATEWAY (add to .env or docker-compose env):" -ForegroundColor Yellow
Write-Host ""
Write-Host "   GATEWAY_JWT_EDDSA_PUBLIC_KEY_PATH=$publicKeyPath" -ForegroundColor White
Write-Host ""
Write-Host "   Or alternatively (base64-encoded PEM):" -ForegroundColor DarkGray
Write-Host "   GATEWAY_JWT_EDDSA_PUBLIC_KEY=<base64 from public key file>" -ForegroundColor DarkGray
Write-Host ""

Write-Host "2. Set this env var when BUILDING the kiosk app:" -ForegroundColor Yellow
Write-Host ""
Write-Host "   `$env:VITE_ASTRA_JWT = '$jwt'" -ForegroundColor White
Write-Host ""

Write-Host "3. For local production testing, use VITE_ASTRA_DEV_MODE to use localhost remotes:" -ForegroundColor Yellow
Write-Host ""
Write-Host "   `$env:VITE_ASTRA_DEV_MODE = 'true'" -ForegroundColor White
Write-Host ""

Write-Host "============================================================" -ForegroundColor Green
Write-Host ""
Write-Host "Generated JWT (keep private key secret!):" -ForegroundColor DarkGray
Write-Host $jwt -ForegroundColor DarkGray
