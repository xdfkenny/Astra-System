#!/usr/bin/env pwsh
# Astra-System Release Script
# Creates a git tag and pushes it to trigger the CI build-installer workflow.
#
# Usage:
#   .\installer\scripts\release.ps1 -Version 0.3.0 -Channel beta -Message "Bug fixes and stability improvements"

param(
    [Parameter(Mandatory = $true)]
    [string]$Version,

    [Parameter(Mandatory = $false)]
    [ValidateSet("stable", "beta", "canary")]
    [string]$Channel = "stable",

    [Parameter(Mandatory = $false)]
    [string]$Message = "Release v${Version}-${Channel}",

    [Parameter(Mandatory = $false)]
    [switch]$DryRun
)

$ErrorActionPreference = "Stop"

$tag = if ($Channel -eq "stable") { "v$Version" } else { "v$Version-$Channel" }

Write-Host "Astra-System Release" -ForegroundColor Cyan
Write-Host "====================" -ForegroundColor Cyan
Write-Host ""
Write-Host "  Version: $Version" -ForegroundColor White
Write-Host "  Channel: $Channel" -ForegroundColor White
Write-Host "  Tag:     $tag" -ForegroundColor White
Write-Host "  Message: $Message" -ForegroundColor White
Write-Host ""

# Check working tree
$status = git status --porcelain
if ($status) {
    Write-Host "! Uncommitted changes detected:" -ForegroundColor Yellow
    $status | ForEach-Object { Write-Host "    $_" -ForegroundColor Yellow }
    Write-Host ""
    $answer = Read-Host "Commit all changes before releasing? (Y/n)"
    if ($answer -ne "n") {
        git add -A
        git commit -m "chore: prepare release $tag"
        Write-Host "  ✓ Committed" -ForegroundColor Green
    }
}

if ($DryRun) {
    Write-Host "[DRY RUN] Would run:" -ForegroundColor Yellow
    Write-Host "  git tag -a $tag -m `"$Message`"" -ForegroundColor Gray
    Write-Host "  git push origin $tag" -ForegroundColor Gray
    exit 0
}

# Create and push tag
Write-Host "Creating tag $tag..." -ForegroundColor White
git tag -a $tag -m $Message
if ($LASTEXITCODE -ne 0) {
    Write-Host "  ✗ Failed to create tag. Does it already exist?" -ForegroundColor Red
    exit 1
}
Write-Host "  ✓ Tag created" -ForegroundColor Green

Write-Host "Pushing tag to origin..." -ForegroundColor White
git push origin $tag
if ($LASTEXITCODE -ne 0) {
    Write-Host "  ✗ Failed to push tag" -ForegroundColor Red
    exit 1
}

Write-Host ""
Write-Host "✓ Release triggered!" -ForegroundColor Cyan
Write-Host "  Tag: $tag"
Write-Host "  Channel: $Channel"
Write-Host "  Monitor: https://github.com/astra-service/Astra-System/actions/workflows/build-installer.yml" -ForegroundColor Gray
