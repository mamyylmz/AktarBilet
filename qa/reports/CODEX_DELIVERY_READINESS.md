# AktarBilet — Delivery Readiness

**Final state:** **NOT_READY**  
**Git action:** **NO COMMIT / NO PUSH**

## Git and remote

- Branch: `main`
- Candidate/local/remote SHA observed at closure: `268fd8e9ef33161252a0d98f3241fdd413082fac`
- Repository: `https://github.com/mamyylmz/AktarBilet`, visibility **PUBLIC**.
- History contains multiple meaningful commits.
- The working tree contains independent QA evidence/report changes, so it is not clean.
- No force operation, history rewrite, commit, or push was performed.

## GitHub Actions

- Workflow: AktarBilet CI / CD Pipeline
- Run: `40353938296`
- SHA: `268fd8e9ef33161252a0d98f3241fdd413082fac`
- Conclusion: **failure**.
- Exact failure: aggregate internal statement coverage **67.1%**, required **70%**. Independently, `internal/sync` measured **63.5%**.

## Mandatory blockers

1. Valid multi-event device batch returns 409 because one batch nonce is inserted into a globally unique per-sequence column.
2. Simulator persists 300 unique events in WAL but sends/persists zero; live Section 8 behavior is not complete.
3. Database outage silently enables in-memory service, violating DB-backed health/readiness and financial durability.
4. Current GitHub Actions is red and coverage is below the required threshold.
5. OpenAPI documents only nine paths and is materially behind the router.
6. Frontend “lint” is a syntax/secret regex script, not a real lint gate.
7. Map is not MapLibre/real PMTiles/official GTFS: the PMTiles asset is 1,162 bytes and routes are hard-coded canvas points.
8. Browser source contains fake local card/ledger/transit state and always emits a turnstile success toast after an invalid placeholder signature.
9. Deployable default JWT secret and all demo passwords remain in tracked/browser sources.
10. 600-second and 1M performance gates are correctly NOT_TESTED because mandatory prerequisites failed.

## Delivery timing and owner action

The observed remote commits are before the stated 26 October 2026 23:59 Europe/Istanbul deadline, but a red CI run and mandatory FAILs prevent release approval. GitHub push does not submit the assignment. After a verified green release, the repository URL must still be submitted by the user through `https://banukampus.bandirma.edu.tr/a/s/ZB76YDBA`; this audit did not and cannot claim that survey submission.
