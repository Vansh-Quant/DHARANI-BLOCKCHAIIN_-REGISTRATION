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
```

### What DHARANI adds

- **Cross-source verification** — compare property information instead of trusting a single record.
- **Conflict detection** — surface mismatches in fields such as owner, survey details, area, encumbrance and litigation status.
- **Authority workflow** — authorized reviewers remain the final decision-makers.
- **Property passport** — provide a structured, user-facing view of a property's verification state and history.
- **Transfer workflow** — block demo transfers until the property has reached a verified state and require buyer acceptance.
- **Auditability** — preserve important verification and transfer events for traceability.
- **Blockchain proof layer** — use cryptographic anchoring to make a verification artifact tamper-evident; blockchain is not treated as the legal source of title.

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
├── frontend/             # Web application
├── ignition/             # Hardhat Ignition deployment modules
├── scripts/              # Blockchain/development scripts
├── test/                 # Smart-contract tests
├── hardhat.config.ts     # Hardhat configuration
└── README.md
```

## Backend quick start

The backend expects MongoDB to be reachable through `MONGODB_URI` and listens on port `8080` by default.

```powershell
cd backend
$env:MONGODB_URI="mongodb://127.0.0.1:27017"
go run main.go
```

Health endpoint:

```text
GET http://localhost:8080/api/v1/health
```

For local demonstrations, the authentication flow currently uses a fixed demo OTP (`1234`). This is intentionally a development/demo mechanism and must be replaced by a real OTP provider before production use.

## Smart-contract development

Install dependencies from the repository root and run:

```shell
npm install
npx hardhat compile
npx hardhat test
```

The repository uses Hardhat 3 configuration with Solidity `0.8.28`. The current `PropertyRegistry.sol` is an intermediate contract implementation and is being hardened toward authority-controlled verification and cryptographic verification anchoring.

## Security and governance boundary

DHARANI does **not** claim that putting information on a blockchain makes that information legally true. The platform is intended to improve reconciliation, traceability and reviewability around land information. Legal title and official decisions remain with the competent authority and underlying government records.

Sensitive personal information and raw land documents should remain off-chain. Blockchain should contain only the minimum cryptographic references required for tamper-evident verification.

## Demo principle

The strongest demonstration is not "we put property data on blockchain." It is:

> **We compare evidence, explain conflicts, obtain an authorized decision, and anchor the resulting verification artifact so later changes can be detected.**

---

DHARANI — SIH 2026 project.