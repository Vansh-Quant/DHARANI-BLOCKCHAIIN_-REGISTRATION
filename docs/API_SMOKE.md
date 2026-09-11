# API Smoke Test

The fastest runtime validation after pulling the repository is:

```powershell
Set-Location .\scripts
.\smoke-demo.ps1
```

The script checks the API, creates a synthetic citizen/property, authenticates an authority, creates three evidence layers, deliberately produces an `AREA_MISMATCH`, corrects the GIS extent, reruns reconciliation to `CLEAR`, approves the property, and reads the Property Passport.

The script does not perform a blockchain transaction. After it prints the report hash, use the anchor script against the local Hardhat deployment.

## Manual health check

```powershell
Invoke-RestMethod http://localhost:8080/api/v1/health
```

Expected status:

```text
ok
```
