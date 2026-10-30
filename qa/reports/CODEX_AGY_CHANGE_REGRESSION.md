# AGY change regression

**Reviewed commit:** `c2701cf7e9af412c8723f4b1c8c39467f5945c23`. QA mutations ran only in Compose project `aktarbilet-codex-final`. Production code, migrations, AGENTS/.agents, and Git history were not changed by this audit.

## AGY-1

- **Cleanup:** commit `2929ce2` archives 22 old QA reports as content-preserving renames and removes three screenshot scripts. No migration changes in that commit. No deleted-script references were found in the inspected Makefile/README/QA paths. Existing project volumes were not removed. Current documentation report references remain present.
- **Settlement:** close persisted `is_closed=true`, but a live isolated DB query found no `SETTLEMENT_CLOSE` audit row. `internal/api/handlers.go` ignores the audit `Exec` error after updating the period. The independent test `qa/db/codex_settlement_lifecycle.audit.spec.js` fails at line 54. The test left the seeded October period closed; it was explicitly reopened through the QA API afterward and verified as a fixture-only restoration. Reopen reason defaults when empty (observed API behavior); this needs an explicit policy decision/test. Storno success/ledger/audit and restart persistence were not reached because the close-audit assertion failed. Do not claim them verified by this audit.
- **JWT:** `internal/auth/auth.go` issues 24-hour access and 7-day refresh tokens. Refresh-as-access is rejected (HTTP matrix). No refresh/logout/revocation route found; frontend logout clears local storage only. `RequireRole` silently uses the claim role if current DB role lookup errors. Do not accept the 24-hour token as secure without tested revocation/idle/absolute lifetime controls.
- **T-1:** broad Playwright: 143 passed, 8 failed. Existing real-DB T1-08 fails: chronological balance 16,500 kuruş, reverse-delivery balance 16,000 kuruş (500 kuruş mismatch). Source test: `qa/tests/t1_live_system.spec.js:142-201`; full-suite artifact `qa/artifacts/results.json` (the subsequent isolated test overwrote the aggregate result JSON, so use the deterministic reproduction command below). Other T-1 failure T1-04 also failed in the broad run. `make verify` reports T-1 PASS despite the live mismatch: its verifier is too shallow for the requirement.
- **Keyset:** penalties use a PostgreSQL `(issued_at,id)` keyset query. Anomalies are fetched from in-memory detector state (`GetAnomalies`) then paginated; this is not durable PostgreSQL keyset pagination. Existing keyset unit test is memory-only. Real DB cursor continuation, stable ordering under equal timestamps/concurrent inserts and malformed cursor behavior remain unproven.
- **Git/CI:** `HEAD` and `origin/main` tracking ref both showed `c2701cf…`; GitHub Actions run `41056498730` for that SHA showed success. Network fetch (`git ls-remote`/GitHub API) was unavailable, so remote ref freshness is based on local tracking ref plus recorded CI, not a fresh fetch.

## Regression and repository gates

- `go test -count=1 ./...`: PASS (isolated GOCACHE; loopback permission escalation required).
- `go vet ./...`: PASS.
- `gofmt -l cmd internal scripts`: PASS, no paths listed.
- `web` `npm run lint`: PASS, 23 files, no warnings/errors.
- Production core package coverage: anomaly 88.4%, auth 84.6%, config 100%, domain 100%, ledger 77.6%, pricing 95.8%, seed 84.2%, settlement 73.5%, sync 74.6%, transit 78.8%. API 63.2%; it is below 70 but not a core domain package. QA oracle 47.3% is not production code.
- Full Playwright: **143 passed / 8 failed / 151 total**. Failures included T1-04, T1-08, UAT storno control, two desktop accessibility scenarios, browser penalty/settlement/recharge server actions, and wildcard CORS behavior. The aggregate artifacts were replaced by the later isolated test, so use the command for exact rerun and current failure details.
- `make verify` equivalent (`BACKEND_URL=http://127.0.0.1:18080`, dedicated QA validator credentials): exit 0; SYS-1/2, F-1..7 and T-1..5 all PASS. This is contradicted by T1-08 and does not measure T-5 persisted throughput/report refresh.
- Seven canaries: all passed in broad suite (`qa/tests/agy_canary_checks.spec.js`, seven assertions).
- Docker QA backend, PostgreSQL, frontend were healthy. Compose `up --wait` returned nonzero because the successful one-shot storage-init container exits 0; long-running services were healthy. No QA stack teardown/volume deletion was run.
- T-5: NOT TESTED. Existing load clients do not generate current HMAC-signed batches, and no valid persisted-event plus reporting capacity result exists. Do not run the current unsigned harness as capacity proof.

## AGY handoff — prioritized actionable findings

### P1 — reverse-order fare reconciliation is not deterministic

Reproduction from `qa/`:

```sh
QA_PROJECT_NAME=aktarbilet-codex-final QA_API_URL=http://127.0.0.1:18080 QA_FRONTEND_URL=http://127.0.0.1:13000 npx playwright test tests/t1_live_system.spec.js --grep 'T1-08'
```

Expected: both three-event sequences debit 3,500 kuruş and end at 16,500. Actual: reverse sequence ended at 16,000; one event fare/correction differs. Test uses actual PostgreSQL and checks ledger equality. Root cause is within future-event replay/correction path in `internal/sync/service.go` (the exact failing event-level trace should be captured by AGY while fixing). Smallest fix: make suffix replay deterministic and transactional, then preserve append-only correction accounting and idempotency; add the three-event DB regression.

### P1 — close operation can omit its audit record

Reproduction: `QA_PROJECT_NAME=aktarbilet-codex-final QA_API_URL=http://127.0.0.1:18080 QA_FRONTEND_URL=http://127.0.0.1:13000 npx playwright test db/codex_settlement_lifecycle.audit.spec.js` from `qa/`.

Expected: every successful close has one matching append-only audit row. Actual: period closes, but audit query count remains 0; handler ignores insert error. Smallest fix: execute state change and mandatory audit insert transactionally and return failure/rollback if audit insert fails. Investigate the underlying insert failure rather than suppressing it.

### P1 — `make verify` returns false green for T-1/T-5

Expected: T-1 test proves real out-of-order 3-event reconciliation; T-5 proves persisted processing/reporting throughput. Actual: prints all PASS without either proof. Smallest fix: make verifier invoke canonical, bounded real QA checks or label these checks as smoke-only and reserve acceptance PASS for empirical gates.

### P1 — current keyset acceptance is incomplete

Expected: stable DB-backed cursor pagination for each claimed endpoint. Actual: anomaly pagination is over volatile in-memory list; penalty endpoint has DB keyset but no live multi-page acceptance in this audit. Smallest fix: persist/order anomalies using stable `(timestamp,id)` cursor or stop claiming DB keyset; add real-DB page/no-duplicate/concurrent insertion tests.

### P1 — session lifecycle/revocation is not implemented or proven

Expected: logout/revocation and bounded token lifecycle; role removal invalidates access. Actual: 24h bearer access token; no server logout/refresh route observed; DB role-query error falls back to stale claim. Smallest fix: fail closed on role lookup errors and implement/document the approved session lifecycle with tests; do not treat longer expiry as a repair.

### P2 — broad browser regressions

Full-suite failures also include desktop axe violations, wildcard CORS credential policy, and frontend penalty/settlement/recharge controls that did not make successful server requests. Inspect individual existing test traces and fix each root cause; do not weaken assertions.

### P2 — ASVS map and QA skill claims are unreliable

See `CODEX_ASVS_5_0_SKILL_AUDIT.md`: multiple incorrect normative IDs, 18-vs-22 count, and unmeasured claims. Update mappings and procedures before relying on them as an acceptance gate.

### P1 — T-5 capacity evidence absent

No supported current-HMAC persisted-load evidence. The old `qa/load/event-load.mjs` sends unsigned requests and is not suitable. Smallest next step: update the existing QA harness to sign with isolated test credentials, measure committed rows and report refresh/error/latency under a bounded run, then separately support the hourly capacity claim with reproducible evidence.
