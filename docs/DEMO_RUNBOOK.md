# DHARANI Demo Runbook

## Purpose

Run the DHARANI demonstration as one connected story: property registration → source evidence → reconciliation → authority decision → verification artifact → blockchain anchor → property passport → transfer readiness.

## Local services

1. MongoDB: `127.0.0.1:27018`
2. Go API: `http://localhost:8080`
3. Hardhat node: `http://127.0.0.1:8545`
4. Static frontend: `http://localhost:5173`

## Demo accounts

- Citizen: any phone number, email `demo@dharani.local`
- Authority: any phone number, email `authority@dharani.local`
- Demo OTP: `1234`

The authority role is assigned by the backend during demo registration. The browser must not grant itself authority privileges.

## Judge-facing story

1. Register a property as a citizen.
2. Sign in as authority.
3. Add independent source records for the same property.
4. Run reconciliation and show a conflict such as `AREA_MISMATCH`.
5. Correct the source evidence and rerun reconciliation.
6. Show the `CLEAR` verification artifact and its SHA-256 fingerprint.
7. Authority marks the property verified.
8. Run the blockchain anchor adapter and record the returned proof.
9. Open the Property Passport and show the report hash, chain property ID, transaction hash, network, and block number.
10. Initiate the DHARANI transfer/readiness workflow and accept it from the buyer account.

## Positioning

DHARANI is a cross-source consistency, verification, audit, and proof layer. Blockchain anchors the verification artifact; it does not manufacture the truth of an underlying land record or replace statutory registration.
