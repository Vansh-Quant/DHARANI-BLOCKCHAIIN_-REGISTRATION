# Judge Demo Checklist

## Before the demo

- [ ] MongoDB is listening on `27018`.
- [ ] Go API health returns `status: ok` on `8080`.
- [ ] Hardhat node is listening on `8545`.
- [ ] PropertyRegistry is deployed locally.
- [ ] Frontend is served on `5173`.
- [ ] Browser has no stale DHARANI session.

## Live sequence

### 1. Citizen

Register a property and show that it starts as `SUBMITTED`.

### 2. Authority

Sign in with `authority@dharani.local` and open the verification queue.

### 3. Conflict

Load three synthetic evidence layers. Use 1200 area in Revenue and Registration and 1350 in GIS. Run reconciliation and show `AREA_MISMATCH`.

### 4. Resolution

Correct GIS to 1200 and rerun reconciliation. Show `CLEAR` and the generated SHA-256 fingerprint.

### 5. Decision

Authority marks the property `VERIFIED`.

### 6. Proof

Run the blockchain anchor adapter. Show the transaction hash, contract address, chain property ID, block number, and matching report hash.

### 7. Passport

Open Property Passport and explain that it packages the off-chain evidence, verification artifact, blockchain proof, and audit history.

### 8. Transfer

Initiate the DHARANI transfer/readiness workflow and accept it as the buyer. Emphasize that this is a controlled platform workflow, not a replacement for statutory registration.

## Key sentence

> DHARANI verifies consistency across evidence first; blockchain only anchors the resulting verification artifact.
