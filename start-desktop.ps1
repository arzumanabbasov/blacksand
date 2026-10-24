param([switch]$NoBrowser)

$ErrorActionPreference = 'Stop'
$projectRoot = $PSScriptRoot
$binary = Join-Path $projectRoot 'blacksand.exe'
$url = 'http://127.0.0.1:8787'
Set-Location -LiteralPath $projectRoot

if (-not (Test-Path -LiteralPath $binary -PathType Leaf)) {
    throw 'blacksand.exe is missing. Build the Windows executable first.'
}

# Keep existing SQLite workspaces together with their configured database.
$dataDirectory = Join-Path $projectRoot 'data'
$configPath = if ($env:BLACKSAND_CONFIG) { $env:BLACKSAND_CONFIG } else { Join-Path $projectRoot 'config.json' }
if (Test-Path -LiteralPath $configPath) {
    $settings = Get-Content -LiteralPath $configPath -Raw | ConvertFrom-Json
    $databasePath = $settings.database.path
    if ($settings.database.driver -eq 'sqlite' -and $databasePath -and [IO.Path]::IsPathRooted($databasePath)) {
        $dataDirectory = Split-Path -Parent $databasePath
    }
}

$listener = Get-NetTCPConnection -State Listen -LocalPort 8787 -ErrorAction SilentlyContinue | Select-Object -First 1
if ($listener) {
    $owner = Get-Process -Id $listener.OwningProcess -ErrorAction Stop
    if ($owner.Path -ne $binary) {
        throw 'Port 8787 is already occupied by another application. Close it before starting Blacksand.'
    }
} else {
    $logDirectory = Join-Path $projectRoot 'log'
    New-Item -ItemType Directory -Path $logDirectory -Force | Out-Null
    $process = Start-Process -FilePath $binary -WorkingDirectory $projectRoot `
        -ArgumentList @('-addr', '127.0.0.1:8787', '-data', ('"' + $dataDirectory + '"')) `
        -WindowStyle Hidden -PassThru `
        -RedirectStandardOutput (Join-Path $logDirectory 'desktop-stdout.log') `
        -RedirectStandardError (Join-Path $logDirectory 'desktop-stderr.log')
}

for ($attempt = 0; $attempt -lt 60; $attempt++) {
    if ($process -and $process.HasExited) {
        throw 'Blacksand stopped during startup. See log/desktop-stderr.log in the project folder.'
    }
    $ready = $false
    try {
        $response = Invoke-WebRequest -Uri "$url/api/health" -UseBasicParsing -TimeoutSec 2
        $ready = $response.StatusCode -eq 200
    } catch { }
    if ($ready) {
        Write-Host "Blacksand is running at $url"
        if (-not $NoBrowser) { Start-Process $url }
        exit 0
    }
    Start-Sleep -Milliseconds 500
}
throw 'Blacksand did not become ready. See log/desktop-stderr.log in the project folder.'
