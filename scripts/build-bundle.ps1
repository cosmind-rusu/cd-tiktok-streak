param(
    [string]$ConfigPath = ".\config.json",
    [string]$CookiesPath = ".\cookies.json",
    [string]$OutputDir = ".\dist\bundle",
    [string]$DriverSource = "$env:USERPROFILE\AppData\Local\ms-playwright-go\1.57.0",
    [string]$BrowsersSource = "$env:USERPROFILE\AppData\Local\ms-playwright",
    [switch]$IncludeAllBrowsers,
    [switch]$ZipBundle
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $DriverSource)) {
    throw "Playwright driver source not found: $DriverSource"
}

if (-not (Test-Path -LiteralPath $BrowsersSource)) {
    throw "Playwright browsers source not found: $BrowsersSource"
}

New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null

$exePath = Join-Path $OutputDir "cd-tiktok-streak.exe"
$driverOutput = Join-Path $OutputDir "playwright-driver"
$browsersOutput = Join-Path $OutputDir "ms-playwright"
$launcherPath = Join-Path $OutputDir "run-cd-tiktok-streak.cmd"

& .\scripts\build-standalone.ps1 -ConfigPath $ConfigPath -CookiesPath $CookiesPath -Output $exePath

if (Test-Path -LiteralPath $driverOutput) {
    Remove-Item -LiteralPath $driverOutput -Recurse -Force
}

if (Test-Path -LiteralPath $browsersOutput) {
    Remove-Item -LiteralPath $browsersOutput -Recurse -Force
}

Copy-Item -LiteralPath $DriverSource -Destination $driverOutput -Recurse

New-Item -ItemType Directory -Force -Path $browsersOutput | Out-Null

if ($IncludeAllBrowsers) {
    Copy-Item -LiteralPath (Join-Path $BrowsersSource "*") -Destination $browsersOutput -Recurse
} else {
    $browserFamilies = @(
        @{ Name = "chromium"; Pattern = "chromium-*"; Exclude = @("chromium_headless_shell-*") },
        @{ Name = "chromium_headless_shell"; Pattern = "chromium_headless_shell-*"; Exclude = @() },
        @{ Name = "ffmpeg"; Pattern = "ffmpeg-*"; Exclude = @() },
        @{ Name = "winldd"; Pattern = "winldd-*"; Exclude = @() }
    )

    foreach ($family in $browserFamilies) {
        $matches = Get-ChildItem -LiteralPath $BrowsersSource -Filter $family.Pattern -Directory

        foreach ($excludePattern in $family.Exclude) {
            $matches = $matches | Where-Object { $_.Name -notlike $excludePattern }
        }

        $selected = $matches | Sort-Object LastWriteTimeUtc -Descending | Select-Object -First 1
        if ($selected) {
            Copy-Item -LiteralPath $selected.FullName -Destination $browsersOutput -Recurse
        }
    }
}

$launcher = @"
@echo off
setlocal
set "PLAYWRIGHT_DRIVER_PATH=%~dp0playwright-driver"
set "PLAYWRIGHT_BROWSERS_PATH=%~dp0ms-playwright"
"%~dp0cd-tiktok-streak.exe" %*
"@

Set-Content -LiteralPath $launcherPath -Value $launcher -Encoding ASCII

if ($ZipBundle) {
    $zipPath = "$OutputDir.zip"
    if (Test-Path -LiteralPath $zipPath) {
        Remove-Item -LiteralPath $zipPath -Force
    }
    Compress-Archive -Path (Join-Path $OutputDir "*") -DestinationPath $zipPath
}
