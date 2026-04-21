param(
    [string]$ConfigPath = ".\config.json",
    [string]$CookiesPath = ".\cookies.json",
    [string]$Output = ".\dist\cd-tiktok-streak-standalone.exe"
)

$ErrorActionPreference = "Stop"

if (-not (Test-Path -LiteralPath $ConfigPath)) {
    throw "Config file not found: $ConfigPath"
}

if (-not (Test-Path -LiteralPath $CookiesPath)) {
    throw "Cookies file not found: $CookiesPath"
}

$configJson = Get-Content -LiteralPath $ConfigPath -Raw
$cookiesJson = Get-Content -LiteralPath $CookiesPath -Raw

$configB64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($configJson))
$cookiesB64 = [Convert]::ToBase64String([Text.Encoding]::UTF8.GetBytes($cookiesJson))

$outputDir = Split-Path -Parent $Output
if ($outputDir) {
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

$ldflags = @(
    "-X", "main.cdrusuEmbeddedConfigJSONB64=$configB64",
    "-X", "main.cdrusuEmbeddedCookiesJSONB64=$cookiesB64"
)

go build -ldflags ($ldflags -join " ") -o $Output .
