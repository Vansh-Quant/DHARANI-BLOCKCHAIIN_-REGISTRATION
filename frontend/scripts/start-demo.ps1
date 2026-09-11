$ErrorActionPreference = "Stop"
$Root = Split-Path -Parent $PSScriptRoot
$MongoBin = "C:\Program Files\MongoDB\Server\8.2\bin\mongod.exe"
$MongoData = Join-Path $env:USERPROFILE "dharani-mongodb-data"
$MongoPort = 27018
$ApiPort = 8080

Write-Host "Starting DHARANI local demo..." -ForegroundColor Cyan

if (-not (Test-Path $MongoBin)) {
    throw "MongoDB executable not found at $MongoBin"
}
if (-not (Test-Path $MongoData)) { New-Item -ItemType Directory -Path $MongoData | Out-Null }

$mongoListening = Get-NetTCPConnection -LocalPort $MongoPort -State Listen -ErrorAction SilentlyContinue
if (-not $mongoListening) {
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "& '$MongoBin' --dbpath '$MongoData' --port $MongoPort"
    Start-Sleep -Seconds 2
}

$chainListening = Get-NetTCPConnection -LocalPort 8545 -State Listen -ErrorAction SilentlyContinue
if (-not $chainListening) {
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$Root'; npm run node:local"
    Start-Sleep -Seconds 3
}

Push-Location $Root
try {
    npx hardhat ignition deploy ignition/modules/dharanimodule.ts --network localhost
    $addressFile = Join-Path $Root "ignition\deployments\chain-31337\deployed_addresses.json"
    if (-not (Test-Path $addressFile)) { throw "Deployment address file not found: $addressFile" }
    $addresses = Get-Content $addressFile | ConvertFrom-Json
    $contractAddress = $addresses."DharaniModule#PropertyRegistry"
    if (-not $contractAddress) { throw "PropertyRegistry address was not found in deployment output" }
    Write-Host "PropertyRegistry: $contractAddress" -ForegroundColor Green

    $env:MONGODB_URI = "mongodb://127.0.0.1:$MongoPort/dharani"
    $env:API_PORT = "$ApiPort"
    Start-Process powershell -ArgumentList "-NoExit", "-Command", "Set-Location '$Root\backend'; `$env:MONGODB_URI='mongodb://127.0.0.1:$MongoPort/dharani'; `$env:API_PORT='$ApiPort'; go run ."
    Write-Host "DHARANI API: http://localhost:$ApiPort" -ForegroundColor Green
    Write-Host "MongoDB: mongodb://127.0.0.1:$MongoPort/dharani" -ForegroundColor Green
    Write-Host "Blockchain RPC: http://127.0.0.1:8545" -ForegroundColor Green
    Write-Host "Contract: $contractAddress" -ForegroundColor Green
} finally {
    Pop-Location
}
