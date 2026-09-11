# DHARANI

**Don't just store land records — create trust around them.**

DHARANI is a land-record verification and property-trust platform designed to reconcile information from multiple sources, surface discrepancies, route cases to authorized review, and maintain an auditable history of verification and ownership changes.

## Core flow

```text
Source Records
     ↓
Normalize & Compare
     ↓
Detect Conflicts
     ↓
Authority Review
     ↓
Verified / Conflict / Rejected
     ↓
Audit Trail
     ↓
Blockchain Proof Anchor
     ↓
Property Passport
     ↓
Transfer / Readiness Workflow
```

### What DHARANI adds

- **Cross-source verification** — compare property information instead of trusting a single record.
- **Conflict detection** — surface mismatches in owner, ULPIN, survey number, area, overlap, lineage, encumbrance and litigation data.
- **Authority workflow** — only an authorized reviewer can mark a property `VERIFIED`, and verification requires the latest reconciliation report to be `CLEAR`.
- **Property passport** — expose verification evidence, the latest report, blockchain proof and audit history in one response for UI consumption.
- **Transfer workflow** — block demo transfers until the property is verified and require buyer acceptance.
- **Auditability** — preserve important verification, review and transfer events for traceability.
- **Blockchain proof layer** — anchor the verification artifact hash and verify the on-chain hash before the backend accepts an anchor result. Blockchain is not treated as the legal source of title.

## Current stack

| Layer | Technology |
|---|---|
| Backend API | Go + Gin |
| Database | MongoDB |
| Blockchain | Solidity + Hardhat 3 + ethers |
| Web UI | React (frontend integration) |
| Mobile | React Native / Expo (mobile integration) |

## Repository structure

```text
.
├── backend/              # Go API and MongoDB integration
├── contracts/            # Solidity property-registry contracts
├── frontend/             # Frontend integration pointer
├── ignition/             # Hardhat Ignition deployment modules
├── scripts/              # Blockchain/development scripts
├── test/                 # Smart-contract tests
├── hardhat.config.ts     # Hardhat configuration
└── README.md
```

## Backend quick start

The backend expects MongoDB to be reachable through `MONGODB_URI` and listens on port `8080` by default.

For the current local demo environment:

```powershell
cd backend
$env:MONGODB_URI="mongodb://127.0.0.1:27018/dharani"
go run .
```

Health endpoint:

```text
GET http://localhost:8080/api/v1/health
```

For local demonstrations, authentication uses a fixed demo OTP (`1234`). This is intentionally a development/demo mechanism and must be replaced by a real OTP provider before production use.

## Verification rules

The current MVP reconciliation engine evaluates:

1. `AREA_MISMATCH` — source extents differ by more than 1%.
2. `OWNERSHIP_MISMATCH` — normalized owner names differ.
3. `ULPIN_MISMATCH` — supplied ULPIN identifiers differ.
4. `SURVEY_NUMBER_MISMATCH` — supplied survey numbers differ.
5. `DUPLICATE_OVERLAP` — a source flags spatial/duplicate overlap.
6. `BROKEN_SUBDIVISION_LINEAGE` — parent/subdivision lineage is broken or missing.
7. `MISSING_RISK_DATA` — encumbrance or litigation status is missing/unknown.

A property cannot become `VERIFIED` unless its latest stored reconciliation artifact is `CLEAR`. Conflict and inconclusive runs do not leave a usable latest verification hash behind for accidental approval.

## Smart-contract development

Install dependencies from the repository root and run:

```shell
npm install
npx hardhat compile
npx hardhat test
```

The repository uses Hardhat 3 with Solidity `0.8.28`. `PropertyRegistry.sol` is the core proof contract. It stores minimal property references and verification hashes, while sensitive records remain off-chain.

For the local blockchain demo:

```shell
npx hardhat node
```

In another terminal, deploy the contract and run the verification anchor script using the deployed address and a verification report hash. On localhost, the script uses Hardhat account #0 automatically; a private key is required only for remote networks.

The anchor script now reads the property back from the contract after the transaction and checks that the on-chain verification flag and latest verification hash match the expected report hash before reporting success.

## Security and governance boundary

DHARANI does **not** claim that putting information on a blockchain makes that information legally true. The platform is intended to improve reconciliation, traceability and reviewability around land information. Legal title and official decisions remain with the competent authority and underlying government records.

Sensitive personal information and raw land documents should remain off-chain. Blockchain should contain only the minimum cryptographic references required for tamper-evident verification.

## Demo principle

The strongest demonstration is not "we put property data on blockchain." It is:

> **We compare evidence, explain conflicts, obtain an authorized decision, and anchor the resulting verification artifact so later changes can be detected.**

---

DHARANI — SIH 2026 project.
