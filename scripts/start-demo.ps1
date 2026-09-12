$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$ApiPort = 8080
$FrontendPort = 5173
$MongoPort = 27017

Write-Host "Starting DHARANI local demo..." -ForegroundColor Cyan

$mongoService = Get-Service MongoDB -ErrorAction SilentlyContinue
if ($mongoService) {
    if ($mongoService.Status -ne 'Running') { Start-Service MongoDB; Start-Sleep -Seconds 2 }
    Write-Host "MongoDB Windows service: Running" -ForegroundColor Green
} else {
    $mongoBin = Get-ChildItem "C:\Program Files\MongoDB\Server" -Recurse -Filter mongod.exe -ErrorAction SilentlyContinue | Select-Object -First 1 -ExpandProperty FullName
    if (-not $mongoBin) { throw "MongoDB executable/service not found" }
    $mongoData = Join-Path $env:USERPROFILE "dharani-mongodb-data"
    if (-not (Test-Path $mongoData)) { New-Item -ItemType Directory -Path $mongoData | Out-Null }
    if (-not (Get-NetTCPConnection -LocalPort $MongoPort -State Listen -ErrorAction SilentlyContinue)) {
        Start-Process powershell -ArgumentList "-NoExit", "-Command", "& '$mongoBin' --dbpath '$mongoData' --port $MongoPort --bind_ip 127.0.0.1"
        Start-Sleep -Seconds 3
    }
}

if (-not (Get-NetTCPConnection -LocalPort $MongoPort -State Listen -ErrorAction SilentlyContinue)) {
    throw "MongoDB is not listening on port $MongoPort"
}

if (-not (Get-NetTCPConnection -LocalPort 8545 -State Listen -ErrorAction SilentlyContinue)) {
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$Root'; npm run node:local"
    Start-Sleep -Seconds 4
}

Push-Location $Root
try {
    npm run deploy:localhost
    $addressFile = Join-Path $Root "ignition\deployments\chain-31337\deployed_addresses.json"
    if (-not (Test-Path $addressFile)) { throw "Deployment address file not found: $addressFile" }
    $addresses = Get-Content $addressFile -Raw | ConvertFrom-Json
    $contractAddress = $addresses."DharaniModule#PropertyRegistry"
    if (-not $contractAddress) { throw "PropertyRegistry address was not found in deployment output" }

    $env:MONGODB_URI = "mongodb://127.0.0.1:$MongoPort/dharani"
    $env:API_PORT = "$ApiPort"
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$Root\backend'; `$env:MONGODB_URI='mongodb://127.0.0.1:$MongoPort/dharani'; `$env:API_PORT='$ApiPort'; go run ."
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$Root'; npx --yes http-server frontend -p $FrontendPort -c-1"

    Write-Host "PropertyRegistry: $contractAddress" -ForegroundColor Green
    Write-Host "DHARANI API: http://localhost:$ApiPort/api/v1/health" -ForegroundColor Green
    Write-Host "Frontend: http://localhost:$FrontendPort" -ForegroundColor Green
    Write-Host "MongoDB: mongodb://127.0.0.1:$MongoPort/dharani" -ForegroundColor Green
    Write-Host "Blockchain RPC: http://127.0.0.1:8545" -ForegroundColor Green
} finally {
    Pop-Location
}
