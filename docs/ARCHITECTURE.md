# DHARANI Architecture

```text
Citizen / Authority UI
        |
        v
     Go + Gin API
        |
   +----+-----+----------------+
   |          |                |
 MongoDB   Reconciliation   Audit history
   |          |
   |     Verification report
   |          |
   +----------+-----> SHA-256 artifact
                         |
                         v
                 Blockchain adapter
                         |
                         v
                 PropertyRegistry
                         |
                         v
                  Property Passport
```

## Trust boundaries

- MongoDB stores application records, source evidence, reports, and audit events.
- The reconciliation engine compares normalized source evidence and emits explainable conflict codes.
- Authority review is the decision boundary for `VERIFIED` status.
- SHA-256 fingerprints the canonical verification artifact.
- The blockchain stores the minimal proof needed to make the artifact tamper-evident and independently checkable.
- Raw documents and sensitive personal information should remain off-chain.

## Important invariant

A blockchain anchor is accepted by the backend only when the supplied report hash matches a stored `CLEAR` verification report, the property is already `VERIFIED`, and the hash is the property's latest verified artifact.

This separation keeps blockchain in its proper role: proof of the verification artifact, not a substitute for source-of-truth land governance.
