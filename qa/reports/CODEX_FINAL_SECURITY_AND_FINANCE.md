# AktarBilet — Final Security and Finance Audit

**Decision:** **PASS for unauthorized mutation controls; FAIL for legitimate signed batch availability; overall NOT_READY.**

## Security evidence

- All current negative validator/IDOR suites passed against real PostgreSQL: anonymous registered validator, anonymous batch, ordinary JWT-as-device, known/random idempotency-key IDOR, retired repository credential, cross-device signing, full-payload tamper, assignment mismatch, replay, revocation, and real JWT signature-byte tamper.
- Unauthorized financial mutations observed: **0**.
- Single signed tap exact retry is durable and idempotent.
- Twenty concurrent exact signed copies produced one validator event and one ledger debit.
- Per-device credentials are runtime-generated into a Docker volume; the retired credential cannot authenticate.

## P0/P1 findings

### P1 — A legitimate multi-event signed batch cannot be accepted

- Specification: T-2 / F-2 / §8 simulator.
- Reproduction: strengthened `qa/db/final_financial.audit.spec.js` signs a five-event raw JSON batch with the active `VAL_500T_01` credential; the first request returns HTTP 409.
- Root cause: `migrations/006_security_and_approval_integrity.sql:10` makes `device_nonce` globally unique in `jw_device_requests`, while `VerifyBatchSignature` inserts one reservation row per sequence using the same batch nonce. The second row conflicts inside the transaction.
- API/DB impact: no event or ledger mutation; all legitimate simulator batches fail closed.
- Minimum correction: make the request-level nonce unique once per signed request (separate request table), or remove the per-row global nonce uniqueness and enforce the intended request relationship transactionally. Preserve `(validator_id, sequence_no)` and exact-payload replay checks.

### P1 — Database outage is hidden by in-memory production behavior

- Code: `cmd/server/main.go:55-60` and `:88-95`.
- Reproduction: the independent outage test stops PostgreSQL and recreates backend; backend logs “in-memory modda devam ediliyor” and serves HTTP instead of failing readiness with 503.
- Impact: `/health` does not prove the required PostgreSQL dependency, and financial state can diverge or disappear.
- Minimum correction: production/compose mode must fail startup or remain unready when PostgreSQL is unavailable. Restrict memory implementations to explicit unit-test construction.

### P1 — Deployable credentials and UI demo passwords remain in source/bundle

- `.env.example:8` and `docker-compose.yml:50` contain a deployable default JWT secret.
- `web/src/features/auth/LoginModal.jsx:109-144` embeds all six demo passwords; they appear in the production bundle.
- No active validator private key or cloud/private key was found in tracked runtime source.
- Minimum correction: require an externally supplied JWT secret outside explicit QA; do not ship passwords in the browser bundle.

## Financial integrity

| Control | Result | Evidence |
|---|---|---|
| Append-only ledger | PASS | PostgreSQL rejects UPDATE and DELETE. |
| Balance = ledger sum | PASS | Clean DB integrity and signed concurrency checks. |
| Concurrent signed taps | PASS | 20/20 HTTP 200, 20 events, 20 unique debits. |
| Concurrent tap + recharge | PASS | 15 + 15 operations; 31 ledger rows including seed; reconciled. |
| Duplicate signed tap | PASS | 20 requests, one event and one debit. |
| Negative tolerance | PASS | Contention cannot cross -5,000 kuruş. |
| Recharge restart/idempotency | PASS | One 777-kuruş effect after backend restart and retry. |
| Out-of-order correction | PASS | CORE-04 chronological and reverse delivery converge. |
| Settlement close/storno | PASS | close is one-way/idempotent; retry creates one correction. |
| Optimization approval | PASS | authorized persistence, unauthorized denial, idempotent retry, invalid DTO rejection. |
| Multi-event device batch | FAIL | valid signed batch returns 409 before mutation. |
| Financial float invariant | FAIL | `internal/domain/models.go:12-17` exposes float64 TL conversion helpers. |

The security control itself was not weakened to make old admin-JWT validator tests green. The QA concurrency suite was strengthened to use the same raw-body HMAC contract as real devices.
