$ErrorActionPreference = 'Stop'
$Api = 'http://localhost:8080/api/v1'

function Post-Json($Path, $Body, $Token) {
  $headers = @{}
  if ($Token) { $headers.Authorization = "Bearer $Token" }
  return Invoke-RestMethod -Method Post -Uri "$Api$Path" -Headers $headers -ContentType 'application/json' -Body ($Body | ConvertTo-Json -Depth 10)
}
function Get-Json($Path, $Token) {
  $headers = @{}
  if ($Token) { $headers.Authorization = "Bearer $Token" }
  return Invoke-RestMethod -Method Get -Uri "$Api$Path" -Headers $headers
}

Write-Host 'Checking API...' -ForegroundColor Cyan
Get-Json '/health' | Format-List

$phone = "99999$((Get-Random -Minimum 10000 -Maximum 99999))"
Write-Host "Demo phone: $phone" -ForegroundColor DarkGray
Post-Json '/auth/register' @{ phone_number=$phone; first_name='Demo'; last_name='Citizen'; email='demo@dharani.local' } $null | Out-Null
$citizen = Post-Json '/auth/verify-otp' @{ phone_number=$phone; otp='1234' } $null
$citizenToken = $citizen.token

$property = Post-Json '/property/register' @{ property_name='DHARANI Demo Plot'; property_type='RESIDENTIAL'; location='Jaipur, Rajasthan'; address='Demo Road, Jaipur, Rajasthan'; area=1200; latitude=26.85; longitude=75.80 } $citizenToken
$propertyId = $property.property_id
Write-Host "Created property: $propertyId" -ForegroundColor Green

Post-Json '/auth/register' @{ phone_number=$phone; first_name='Demo'; last_name='Authority'; email='authority@dharani.local' } $null | Out-Null
$authority = Post-Json '/auth/verify-otp' @{ phone_number=$phone; otp='1234' } $null
$authorityToken = $authority.token

$base = @{ property_id=$propertyId; owner_name='Demo Citizen'; ulpin='ULPIN-DEMO-001'; survey_number='123/4'; polygon_hash='poly-demo'; parent_survey='123'; lineage_status='VALID'; encumbrance_status='CLEAR'; litigation_status='CLEAR'; overlap_flag=$false; document_hash='doc-demo' }
Post-Json '/source-records' ($base + @{ source_type='REVENUE'; source_record_id='REV-001'; area=1200 }) $authorityToken | Out-Null
Post-Json '/source-records' ($base + @{ source_type='REGISTRATION'; source_record_id='REG-001'; area=1200 }) $authorityToken | Out-Null
Post-Json '/source-records' ($base + @{ source_type='GIS'; source_record_id='GIS-001'; area=1350 }) $authorityToken | Out-Null

$conflict = Post-Json "/reconciliation/$propertyId/run" @{} $authorityToken
Write-Host "First reconciliation: $($conflict.property_status)" -ForegroundColor Yellow
$conflict.report.conflicts | Format-Table rule_code,severity,title

Post-Json '/source-records' ($base + @{ source_type='GIS'; source_record_id='GIS-001'; area=1200 }) $authorityToken | Out-Null
$clear = Post-Json "/reconciliation/$propertyId/run" @{} $authorityToken
Write-Host "Second reconciliation: $($clear.property_status) / score $($clear.report.sha256)" -ForegroundColor Green

$review = Post-Json "/verification/$propertyId/review" @{ decision='VERIFIED'; notes='Demo authority approval after CLEAR reconciliation' } $authorityToken
Write-Host "Authority review: $($review.status)" -ForegroundColor Green

$passport = Get-Json "/property/$propertyId/passport" $authorityToken
$reportHash = [string]$passport.passport.verification_report.sha256
if ($reportHash -notmatch '^[0-9a-fA-F]{64}$') { throw "Passport did not return a valid 64-character SHA-256 report hash. Received length $($reportHash.Length)." }
$anchorPayload = @{ property_ref=$propertyId; report_hash=$reportHash } | ConvertTo-Json
Set-Content -Path (Join-Path (Get-Location) '.dharani-last-anchor.json') -Value $anchorPayload -Encoding utf8
Write-Host "Passport ready for $propertyId" -ForegroundColor Cyan
Write-Host "Report SHA-256: $reportHash"
Write-Host "Anchor handoff saved to .dharani-last-anchor.json" -ForegroundColor Green
Write-Host "Next: run npm run anchor:verification after setting BLOCKCHAIN_RPC_URL and PROPERTY_REGISTRY_ADDRESS."
