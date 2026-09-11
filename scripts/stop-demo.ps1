$ErrorActionPreference = "SilentlyContinue"
Write-Host "Stopping DHARANI demo processes..." -ForegroundColor Cyan
Get-Process node,go,mongod -ErrorAction SilentlyContinue | Stop-Process -Force
Get-NetTCPConnection -LocalPort 8080,8545,27018 -ErrorAction SilentlyContinue | ForEach-Object { try { Stop-Process -Id $_.OwningProcess -Force } catch {} }
Write-Host "DHARANI demo processes stopped." -ForegroundColor Green
