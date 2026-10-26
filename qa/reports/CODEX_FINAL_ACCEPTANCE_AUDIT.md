# AktarBilet — Final Independent Acceptance Audit

**Verdict: NOT_READY**

## Executive result

The current candidate materially improves the previous security state: independent device-auth, IDOR, replay, revocation, role and financial atomicity tests pass, and unauthorized mutation count is zero. It is still not releasable because a real signed multi-event batch cannot be inserted, the simulator drains zero events, PostgreSQL outage is masked by an in-memory fallback, CI is red at 67.1% coverage, and the claimed map/lint/OpenAPI delivery is not genuine or complete.

| Result | Exact criterion count |
|---|---:|
| PASS | 124 |
| FAIL | 28 |
| BLOCKED | 3 |
| NOT_TESTED | 16 |
| **Total** | **171** |

The 171-row total is the exact count of criterion rows in the specification matrix, including the seven mandatory canaries and the explicitly requested UI/UX and multi-role extensions. `BLOCKED` and `NOT_TESTED` together account for 19 criteria. Any one mandatory `FAIL`, `BLOCKED`, or `NOT_TESTED` prevents release.

## Independently executed results

- Go: all packages, vet and formatting PASS; 67 named Go tests. Current-SHA CI completed `go test -race` before failing the subsequent coverage threshold. The local race attempt was environment-blocked because this host has no C compiler. Coverage: pricing 96.8%, ledger 77.6%, settlement 79.8%, anomaly 88.4%, transit 78.8%, sync 63.5%.
- Clean isolated Playwright selection: **134 PASS / 10 FAIL**. Five financial failures were obsolete tests that attempted admin-JWT validator access. After strengthening those tests to real HMAC, four concurrency cases and restart recharge passed; signed batch remained FAIL. Browser/map failures and upstream timeout remain visible.
- Security: current device/IDOR/role suites PASS; **0 unauthorized mutation**.
- Accessibility: **7/7 PASS**, axe-core zero automated violations in tested views, focus transfer/Escape and 200% reflow PASS.
- Canary: **7/7 PASS**.
- Frontend production build: PASS, 53 modules. Real lint: FAIL/not present.
- Docker verify: prints 14/14 PASS but is insufficient because independent signed batch and simulator tests contradict T-2.
- GitHub Actions: FAIL on current SHA, coverage 67.1%.

## Key defect reproductions

### AB-FINAL-01 — Signed batch and simulator are nonfunctional

- Priority: P1 mandatory core function.
- Specification: F-2, T-2, §8.
- Files: `migrations/006_security_and_approval_integrity.sql:3-10`, `internal/sync/device_auth.go:156-194`.
- Expected: one signed request reserves all unique sequences and exact retry is idempotent.
- Actual: HTTP 409 before the first batch is persisted. Simulator: 300 unique events pending, 0 persisted, 0 sent.
- Minimum fix: correct nonce/request relational model without relaxing signature, sequence, replay or payload binding.

### AB-FINAL-02 — Health/readiness does not require PostgreSQL

- Priority: P1 data integrity.
- Specification: §3 `/health` must include DB; §7 resilience.
- File: `cmd/server/main.go:55-95`.
- Expected: startup/readiness fails without PostgreSQL.
- Actual: backend selects in-memory ledger and continues serving.
- Minimum fix: fail closed in deployed environments; memory services only by explicit test dependency injection.

### AB-FINAL-03 — Map/GTFS claim is synthetic

- Priority: P1 accepted product scope.
- Files: `web/src/map/MapLibreView.jsx:21-139`, `web/public/tiles/izmir.pmtiles`.
- Expected: MapLibre renderer, valid regional PMTiles vector tiles, official GTFS geometry/stops and 206 tile decode.
- Actual: no MapLibre/PMTiles package dependency; hand-drawn canvas, hard-coded stop coordinates, 1,162-byte header-like PMTiles file. The UI labels it “Range 206 OK”.
- Minimum fix: integrate real MapLibre + PMTiles protocol and licensed official GTFS, then test actual tile decode/render and attribution.

### AB-FINAL-04 — Browser fabricates success and operational data

- Priority: P1 product trust.
- Files: `web/src/App.jsx:93-114`, `:150-163`, `:332-356`.
- Expected: server-authoritative card, ledger, transit and turnstile outcomes.
- Actual: local 500 TL ledger, fabricated buses, placeholder signature, swallowed error, unconditional success toast.
- Minimum fix: load authenticated API state; never show success unless the signed server operation succeeds; keep device credentials out of browser.

### AB-FINAL-05 — Quality/delivery contracts are red

- Priority: P1 release.
- Files: `web/lint.js`, `api/openapi.yaml`, `.github/workflows/ci.yml`.
- Expected: real lint, route-complete OpenAPI, green CI and ≥70% core coverage.
- Actual: custom syntax check, nine OpenAPI paths, current CI failure, sync 63.5% / CI aggregate 67.1%.
- Minimum fix: use an established JS lint rule set, synchronize all router contracts, add meaningful sync/device coverage, and obtain green CI on the candidate SHA.

## Performance and Git gate

Because mandatory P1/CI/product gates fail, the audit did not run the 600-second or 1M campaigns. No current p50/p95/p99, persisted events/s, CPU/RAM, error rate or 1M reconciliation is claimed. Per instruction, no commit or push was made.

Detailed criterion-by-criterion status is in `qa/reports/CODEX_SPEC_COMPLIANCE_MATRIX.md`; machine-readable evidence is under `qa/artifacts/codex/final-acceptance/`.
