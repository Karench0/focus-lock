param([string]$Version = '0.3.2', [switch]$Installer)
$ErrorActionPreference = 'Stop'
if ($Version -notmatch '^\d+\.\d+\.\d+(?:-[a-zA-Z0-9.-]+)?$') { throw 'Invalid version' }
Push-Location (Join-Path $PSScriptRoot '..')
try {
  $releaseDir = Join-Path (Get-Location) 'build/release'
  New-Item -ItemType Directory -Path $releaseDir -Force | Out-Null
  $zip = Join-Path $releaseDir "FocusLock-$Version-windows-amd64.zip"
  $files = @('build/bin/FocusLock.exe', 'build/bin/FocusLock-host.exe', 'build/bin/browser-extension', 'LICENSE', 'README.md', 'THIRD_PARTY_NOTICES.md', 'third-party-licenses', 'docs')
  Compress-Archive -LiteralPath $files -DestinationPath $zip -Force
  if ($Installer) {
    $iscc = Get-Command ISCC.exe -ErrorAction SilentlyContinue
    $compiler = if ($iscc) { $iscc.Source } else { 'C:\Program Files (x86)\Inno Setup 6\ISCC.exe' }
    if (-not (Test-Path -LiteralPath $compiler)) { throw 'Inno Setup 6 is required for -Installer' }
    & $compiler "/DAppVersion=$Version" scripts/installer.iss
    if ($LASTEXITCODE -ne 0) { throw 'Installer build failed' }
  }
  Get-ChildItem -LiteralPath $releaseDir -File | Where-Object { $_.Extension -in @('.zip','.exe') } | ForEach-Object { $hash = (Get-FileHash -LiteralPath $_.FullName -Algorithm SHA256).Hash.ToLowerInvariant(); "$hash  $($_.Name)" } | Set-Content -LiteralPath (Join-Path $releaseDir 'SHA256SUMS.txt') -Encoding ascii
} finally { Pop-Location }
