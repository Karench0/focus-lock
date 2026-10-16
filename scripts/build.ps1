param([switch]$SkipFrontend)
$ErrorActionPreference = 'Stop'
function Check-Exit { if ($LASTEXITCODE -ne 0) { throw "Build failed with exit code $LASTEXITCODE" } }
Push-Location (Join-Path $PSScriptRoot '..')
try {
  if (-not $SkipFrontend) {
    Push-Location ui
    try { & npm.cmd ci; Check-Exit; & npm.cmd run build; Check-Exit } finally { Pop-Location }
  }
  & go run github.com/wailsapp/wails/v2/cmd/wails@v2.16.0 build -platform windows/amd64 -webview2 embed -s -skipbindings -trimpath
  Check-Exit
  & go build -trimpath -ldflags '-s -w' -o build/bin/FocusLock-host.exe ./cmd/focus-lock-host
  Check-Exit
  Copy-Item -LiteralPath browser-extension -Destination build/bin -Recurse -Force
  Copy-Item -LiteralPath LICENSE,README.md -Destination build/bin -Force
  Write-Host 'Built build/bin/FocusLock.exe and its companion extension bridge.'
} finally { Pop-Location }
