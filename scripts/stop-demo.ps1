$ErrorActionPreference = "SilentlyContinue"
Write-Host "Stopping DHARANI demo listeners..." -ForegroundColor Cyan
$ports = @(8080, 8545, 5173)
$connections = Get-NetTCPConnection -LocalPort $ports -ErrorAction SilentlyContinue
$connections | Select-Object -ExpandProperty OwningProcess -Unique | ForEach-Object {
    try { Stop-Process -Id $_ -Force } catch {}
}
Write-Host "DHARANI demo listeners stopped. MongoDB Windows service was left running." -ForegroundColor Green
