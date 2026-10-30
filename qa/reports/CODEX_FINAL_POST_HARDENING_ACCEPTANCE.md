# Final post-hardening independent acceptance

**Run date:** 2026-10-30  
**QA owner:** Codex (independent regression / acceptance)  
**Decision:** `PRODUCTION_ACCEPTANCE: NOT_READY`

## A. Version and test scope

- HEAD: `c2701cf7e9af412c8723f4b1c8c39467f5945c23`.
- Working tree: AGY changes are uncommitted in 11 tracked files; `.agents/`, `prompts/`, AGY reports and QA artifacts are untracked. Codex did not modify production source, migrations, `AGENTS.md`, or `.agents/`.
- QA ran against freshly built local images from the current working tree in isolated Compose project `aktarbilet-codex-final` (API `18080`, frontend `13000`, PostgreSQL 16). This is local dirty-tree evidence.
- The latest recorded GitHub Actions success in the previous report is run `41056498730` for SHA `c2701cf…`. No remote fetch was available; the run does not cover these uncommitted changes.
- Actual `make verify` could not start because GNU make is absent (`env: make: No such file`). The exact Makefile recipe, `go run ./cmd/verify`, was run against QA and exited 1. It reported SYS-1/2, F-1..F-7, T-2/3/5 PASS and T-1/T-4 FAIL. Runtime was under 15 minutes.
- `go test -count=1 ./...`: PASS, all packages including `qa/go`.
- Full Playwright suite: **157 total, 138 passed, 19 failed, 0 skipped, 0 reported flaky**; exit 1, about 5m20s. Playwright artifacts are under `qa/artifacts/playwright/`; suite summary was emitted in the run output. Some nested test scripts rewrite `qa/artifacts/results.xml`, so that file is not the full-suite summary.

## B. Earlier failures and regression decisions

| Area | Earlier claim/finding | Current independent evidence | Decision |
|---|---|---|---|
| T-1 ordering | Three reverse-arriving events differed by 500 kuruş | `qa/tests/t1_live_system.spec.js` T1-08 passes on current API; however full suite `db/go_live.db.spec.js` CORE-04 still fails: reverse balance 16,000 vs chronological 16,500. Different real fixtures disagree. | `T1_DETERMINISM: FAIL` until both canonical live fixtures reconcile |
| Repricing/idempotency | New correction reference includes request key | `codex_posthardening_correction_idempotency.spec.js` PASS: 2 legitimate reprices, correction sum 1,500 kuruş, duplicate request adds no entry, changed-key same-sequence replay is 409; card balance equals ledger sum. | `CORRECTION_IDEMPOTENCY: PASS` for exercised sequence |
| Settlement audit atomicity | Close/reopen/storno use DB transaction | Injected audit-trigger failure rolls close back; concurrent closes serialize to `[200,409]`, one audit. Reopen and storno return 500 due PostgreSQL “could not determine data type of parameter $3” in audit JSON. | `SETTLEMENT_AUDIT_ATOMICITY: FAIL` (close passes; reopen/storno blocked by SQL typing) |
| Financial report | Handoff claims DB-backed dynamic totals | Persisted 2,000-kuruş tap does not alter report (`total_trips` 73,430 and revenue 146,858,750 unchanged). A +1,750 ordinary ledger income row reduces revenue by 1,750. Trips/shares are seeded constants; ingestion is not aggregated. | `FINANCIAL_REPORTING: FAIL` |
| `make verify` | Handoff says live HTTP tests | Actual recipe exits 1. T-1 randomly creates a card ID without creating/funding that card, so live tap check fails. T-4 live storno fails. The T-1 verifier is not an out-of-order ledger comparison; its memory pricing check is separate. | `VERIFY_RELIABILITY: FAIL` (false-green was removed, but live evidence is invalid/incomplete) |
| JWT | Production weak/default secret now rejected | Empty/default and short secret fail closed in production. A 32-character repeated `a` secret passes the length gate, showing entropy is not checked. No secret was printed. Access is 24h and refresh 7d; refresh-as-access, expired/tampered JWT, role-change revocation pass. No backend refresh/logout/revocation flow; UI logout only removes local storage. | `JWT_SECURITY: FAIL` (low-entropy production secret accepted; session lifecycle incomplete) |
| Keyset | DB keyset support added | Equal-timestamp anomaly pages preserve unique rows; cursor traversal terminates. Penalties report `has-more=true` and emit an empty terminal page when count is an exact multiple of limit. Both malformed cursors return 200 and restart from page one. | `KEYSET_PAGINATION: FAIL` |
| T-5 | No signed persisted capacity proof before | Real HMAC-signed, sequenced QA batches: 18,000 generated/sent/accepted/persisted/processed, 0 rejected/duplicate/pending/errors; 60.161s, 299.2 events/s, 1,077,120 hourly equivalent; p50 160.36ms, p95 300.23ms, p99 361.28ms; ledger reconciliation passes. The report remains unchanged despite 36,000,000 kuruş of debits. Artifact: `qa/artifacts/codex-signed-t5-codex-1793370234928.json`. | `T5_CAPACITY_FAIL` (ingest throughput reached target; required summary/report accuracy failed) |
| Previous Playwright failures | Previous accepted count 143/151 | Targeted 9-test retest: 3 pass, 6 fail (UAT-04 menu timeout, two desktop axe contrast findings, penalty creation, settlement close flow, recharge display). Full suite: 138/157 pass, 19 fail. | `PLAYWRIGHT_REGRESSION: FAIL` |
| ASVS | Handoff says 22/22 mapped PASS | Current `.agents/skills/security-gate/references/asvs-5.0-map.md` still uses wrong normative meanings (examples V6.5.2 bcrypt, V7.2.1 entropy, V1.2.1 SQL injection, V4.1.1 auth, V13.3.1 HMAC) and unsupported password-length/rate-limit assertions. | `ASVS_MAPPING: FAIL` |
| HTTP stamp | Previous test reported PASS | Current test passes X-DDO-Stamp on health/canary/catalog/proxy 200; 400, 401, 403, 404, 405 and OPTIONS. 201/204/409/controlled-500 paths were not exercised. | `HTTP_STAMP: PASS` for tested classes; remaining classes NOT_VERIFIED |

### T-5 measurement detail

The run used isolated provisioned device credentials and real validator batch API HMAC authentication, nonce/sequence checks, event persistence and ledger debit. It did not count HTTP status alone. The JSON artifact records pre/post report values and reconciliation; report accuracy is false, so the hourly extrapolation is throughput evidence only, not capacity acceptance.

## C. Specification gate

| Requirement | Evidence | Result |
|---|---|---|
| F-1..F-7 | `go run ./cmd/verify` returned PASS for these live smoke labels; full API, RBAC and UI coverage still has failures | Partial; not blanket acceptance |
| T-1 | T1-08 PASS conflicts with CORE-04 FAIL (500-kuruş discrepancy) | FAIL |
| T-2 / T-3 | verifier smoke PASS; signature/replay and API security suites mostly pass, but device idempotency cases fail in full suite | Partial / FAIL for full required acceptance |
| T-4 | close audit rollback and close serialization pass; reopen/storno audit SQL fails | FAIL |
| T-5 | 18,000 signed persistent events, 299.2/sec, report stale | FAIL |
| Go | Go 1.23.6; `go test -count=1 ./...` PASS | PASS |
| PostgreSQL / architecture | Isolated QA runs PostgreSQL 16; compose specifies backend, frontend, simulator; backend image is multi-stage and non-root; QA healthchecks healthy | PASS for exercised environment |
| Cards and route seed | QA database contains 200,052 cards and 47 `jw_lines`; required 200k/40 threshold is met | PASS for cardinality; deterministic reproducibility not re-proven |
| Migration/seed | Backend startup runs migrations and seed; healthy fresh QA composition used | PASS for startup path |
| Docker cold start/build | Images built and services healthy; no measured clean cold-start duration or explicit per-image runtime-user assertion in this run | NOT_VERIFIED |
| Coverage / lint / frontend build | Prior handoff contains stale evidence; no current coverage threshold/lint/build command was run in this acceptance | NOT_VERIFIED |
| OpenAPI / metrics / load / chaos | Metrics smoke passed in Playwright; repository artifacts/scripts exist; no current OpenAPI conformance, k6 threshold, or chaos campaign run | NOT_VERIFIED |
| CI | Recorded success is for committed base SHA only; dirty tree has no CI evidence | NOT_VERIFIED for current code |
| Documentation | README, TESTING, Makefile, Requests/Postman, DECISIONS exist; README canary suffix verified. `make` missing locally prevented direct canonical command | PARTIAL |

### Seven canaries

All seven canary Playwright tests passed: `jw_` schema prefix / `jw_tf_users`, `X-DDO-Stamp: f85f26`, `GET /_hz/f85f` 200, seed user email, compose API container `ddo-pelikan-api`, and exact README final line.

### CORS and HTTP

The HTTP stamp matrix verified one configured development origin (`http://localhost:3000`) on preflight with `Authorization`, and wildcard credentialed access was rejected by a separate RBAC test. A dedicated configured-origin allow/deny and exposed-header matrix was not run. Do not interpret the partial set as full CORS acceptance.

## D. Remaining actionable findings

1. **P0 — Financial report omits real validator revenue.** Reproducer: `npx playwright test db/codex_posthardening_financial_report.spec.js`; one persisted 2,000-kuruş event left report revenue/trips unchanged. Minimum AGY correction: derive period totals and operator allocations from persisted fare events/ledger with period boundaries, and apply signed settlement adjustments consistently; add reconciliation assertions.
2. **P0 — T-1 order determinism is contradictory.** Reproducer: `npx playwright test tests/t1_live_system.spec.js --grep T1-08` passes, while `npx playwright test db/go_live.db.spec.js --grep CORE-04` fails with 16,000 vs 16,500. Minimum correction: compare identical persisted fixtures and final ledger after all permutations; ensure replay ordering is deterministic including timestamp ties.
3. **P0 — Settlement reopen/storno audit SQL fails.** Reproducer: `npx playwright test db/codex_posthardening_settlement_atomicity.spec.js`; error `could not determine data type of parameter $3 (SQLSTATE 42P18)`. Minimum correction: explicitly cast/structure nullable and JSON parameters in audit insert; preserve transaction rollback and retry tests.
4. **P1 — Device signed retry/replay regression.** Full suite cases `security/retest3_device_security.spec.js` report same-sequence altered payload and exact signed retry failing. Inspect attached Playwright traces and actual sequence/nonce outcomes before changing controls; do not weaken replay protection.
5. **P1 — Keyset validation and terminal-page semantics.** Reproducer: `npx playwright test db/codex_posthardening_keyset.spec.js`; reject malformed/tampered cursors, fetch `limit+1` for penalties, and test concurrent inserts while retaining stable tuple ordering.
6. **P1 — Production JWT entropy and lifecycle.** Reproducer: launch the server with production mode and a repeated 32-byte key; current length-only guard accepts it. Require high-entropy secret material. Define/test refresh, logout/revocation and token lifetime semantics.
7. **P1 — `make verify` live test quality.** Provision a QA card before tap; compare final DB ledger for out-of-order input; keep capacity separate from smoke labels. Existing command now fails instead of false-PASS but does not establish T-1.
8. **P1 — T-5 summary/reporting.** Ingest throughput exceeded 1M/hour equivalent, but summary totals remained stale; fix aggregation before declaring the target met.
9. **P1 — Playwright UAT/accessibility.** Full suite failures include financial/report, close/reopen/storno, recharge, penalty UI, desktop contrast and device replay. The frontend turnstile uses a hardcoded `sig_browser_turnstile`; do not treat the present browser tap request as production device authentication.
10. **P2 — ASVS mapping still inaccurate.** Replace incorrect requirement IDs/titles with version-pinned ASVS 5.0.0 meanings; mark unsupported controls NOT_TESTED/FAIL instead of PASS.

## E. Technical decisions

| Decision | Result |
|---|---|
| `T1_DETERMINISM` | FAIL — conflicting real-DB fixtures; one still shows 500 kuruş difference |
| `CORRECTION_IDEMPOTENCY` | PASS — tested multiple legitimate reprices and exact retry |
| `SETTLEMENT_AUDIT_ATOMICITY` | FAIL — close path passes rollback/serialization; reopen and storno fail SQL typing |
| `FINANCIAL_REPORTING` | FAIL — static baseline and signed-income inversion; tap revenue not reflected |
| `VERIFY_RELIABILITY` | FAIL — exit 1; T1 card fixture invalid and no live out-of-order ledger proof |
| `JWT_SECURITY` | FAIL — low entropy passes length-only check; no backend refresh/logout lifecycle |
| `KEYSET_PAGINATION` | FAIL — exact-multiple empty terminal page and malformed cursor accepted |
| `T5_CAPACITY` | `T5_CAPACITY_FAIL` — throughput/persistence pass; report reconciliation fails |
| `PLAYWRIGHT_REGRESSION` | FAIL — 138/157; 19 failures, no skips |
| `ASVS_MAPPING` | FAIL — materially wrong requirement titles and unsupported PASS claims |
| `HTTP_STAMP` | PASS for tested response classes; 201/204/409/controlled-500 NOT_VERIFIED |
| `CI_STATUS` | Previous committed SHA run succeeded; current dirty tree NOT_VERIFIED remotely |

## F. Final acceptance

`PRODUCTION_ACCEPTANCE: NOT_READY`

Required financial reporting, settlement lifecycle, deterministic fare replay, and security mapping criteria have reproduced failures. No production source was changed during this QA run. Isolated QA seed period was restored to open after lifecycle testing; no commit, push, global prune, or real-database operation was performed.
