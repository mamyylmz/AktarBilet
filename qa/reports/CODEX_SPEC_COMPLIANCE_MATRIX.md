# AktarBilet — Final Specification Compliance Matrix

**Candidate:** `268fd8e9ef33161252a0d98f3241fdd413082fac`  
**Audit:** 2026-10-26, isolated Docker/PostgreSQL 16 environment  
**Decision:** **NOT_READY**

`PASS` means independently executed or directly inspected evidence exists. AGY report claims were not treated as evidence.

## §1–5 — architecture, roles and functional scope

| Source | Criterion | Code/Endpoint | Test | Evidence | Status |
|---|---|---|---|---|---|
| §1 | Dockerized frontend + backend + database | Compose, Go, React, PostgreSQL | isolated build/up | All three built and became healthy | PASS |
| §2 | Go 1.22+ | `go.mod`, Go images | build/test | Go 1.23 image and tests | PASS |
| §2 | PostgreSQL 16 | Compose | DB query/integrity suite | PostgreSQL 16; 26 schema objects | PASS |
| §2 | Real frontend | `web/` | Vite build/browser | 53 modules, production build PASS | PASS |
| §2 | Docker + Compose | Dockerfiles/Compose | two isolated projects | image build and orchestration PASS | PASS |
| §2 | Frontend choice justified | README §8 | document inspection | React/Vite reasoning present | PASS |
| §3 | Documented clone/env/up flow | README §2 | static + isolated equivalent | Commands documented; local clone not repeated | PASS |
| §3 | Frontend 3000 / backend 8080 | Compose | config/runtime | container ports correct | PASS |
| §3 | Database not exposed by default | Compose | config inspection | no production host port | PASS |
| §3 | Simulator is a separate service | Compose `simulator` | config/runtime | separate profile service | PASS |
| §3 | Healthchecks and healthy dependencies | Compose | runtime | DB/backend/frontend healthy | PASS |
| §3 | Cold start under 5 minutes | Compose | not measured with empty cache | warm build only | NOT_TESTED |
| §3 | Automatic migration and seed | `cmd/server`, migrations, seed | clean schema start | 200,025 cards, 40 lines, users seeded | PASS |
| §3 | Demo accounts documented | README/TESTING | inspection/login | six accounts log in | PASS |
| §3 | `/health` proves DB | `cmd/server/main.go:55-95` | outage probe | backend falls back to memory and reports HTTP health | FAIL |
| §3 | Non-root containers | three Dockerfiles | static | all runtime stages use app user | PASS |
| §3 | Multi-stage images | three Dockerfiles | static/build | builder + runtime stages | PASS |
| §3 | No repository secrets | `.env.example`, Compose, UI | secret scan | deployable default JWT and browser demo passwords remain | FAIL |
| §3 | Verify container runs | Compose verify | isolated `run --rm verify` | ran without OCI error | PASS |
| §4 | Passenger API isolation | card handlers | RBAC/IDOR tests | ownership and recharge protections PASS | PASS |
| §4 | Inspector API isolation | penalty handlers | role matrix | allowed only inspector/admin | PASS |
| §4 | Line-manager API isolation | line/optimization handlers | role matrix/cost suite | financial and approval controls PASS | PASS |
| §4 | Accounting API isolation | settlement handlers | role matrix | close/storno restrictions PASS | PASS |
| §4 | Operations API isolation | anomaly handlers | role matrix | anomaly access restrictions PASS | PASS |
| user | Admin role | admin JWT/role rows | role suite | authorized operations PASS | PASS |
| user | Anonymous context | middleware/UI | UAT/security | no silent login; protected APIs denied | PASS |
| user | Multi-role assignment is server-backed | `jw_user_roles`, switch-role | RBAC suite | assigned roles checked in DB | PASS |
| user | Unauthorized role switch denied | `/auth/switch-role` | RBAC suite | passenger escalation denied | PASS |
| user | Role change invalidates old authorization | token version/DB role | RBAC test | old token rejected | PASS |
| F-1 | Card, balance and recharge persistence | cards/recharge/ledger | API + DB | idempotent recharge and ledger equality PASS | PASS |
| F-1/UI | Passenger recharge and refresh | passenger portal | current independent browser flow | secure logged-in end-to-end UI flow not executed | NOT_TESTED |
| F-2 | Secure single validator events | `/validators/tap` | signed real-DB tests | persistence, signature, replay and assignment PASS | PASS |
| F-2 | Offline multi-event batch | `/validators/batch-sync` | strengthened audit | first valid signed 5-event batch returns 409 | FAIL |
| F-3 | List/manage lines, stops, tariffs | catalog/CreateLine | API/role tests | reads and authorized line creation PASS | PASS |
| F-4 | Full/student/elderly/transfer fares | pricing engine | unit tests | full/student and transfer covered; elderly live chain absent | NOT_TESTED |
| F-5 | Operator revenue share | settlement service/API | unit/API/DB | largest remainder and close/storno PASS | PASS |
| F-6 | Inspection and penalties | penalties API | API/RBAC | durable create/list and authorization PASS | PASS |
| F-6/UI | Inspector complete UI journey | Inspector portal | current browser | no explicit-login independent journey completed | NOT_TESTED |
| F-7 | Heatmap/revenue reports use DB | reports endpoints | DB/API tests | materialized heatmap and financial contract PASS | PASS |

## §6 — core tasks

| Source | Criterion | Code/Endpoint | Test | Evidence | Status |
|---|---|---|---|---|---|
| T-1.1 | First ride normal fare | pricing | unit | 2,000 kuruş | PASS |
| T-1.2 | First transfer within 90 min | pricing | unit | 1,000 kuruş | PASS |
| T-1.3 | Second transfer within 90 min | pricing | unit | 500 kuruş | PASS |
| T-1.4 | Third transfer not discounted | pricing | unit | full fare | PASS |
| T-1.5 | Exact 90-minute boundary | pricing | search/test inventory | only 95-minute boundary exists | NOT_TESTED |
| T-1.6 | Ride after 90 minutes | pricing | unit | 95-minute case full fare | PASS |
| T-1.7 | Reverse direction on same line | pricing | unit | full fare | PASS |
| T-1.8 | Night fare | pricing | unit | 3,000 kuruş | PASS |
| T-1.9 | Student daily limit | pricing | unit | 10th discounted, next full | PASS |
| T-1.10 | Monthly cap | pricing | unit | cap and partial final charge | PASS |
| T-1.11 | Fare by all card types | pricing/real pipeline | inventory | elderly real ledger path not proven | NOT_TESTED |
| T-1.12 | Out-of-order deterministic result | sync service | real DB CORE-04 | chronological/reverse delivery equal | PASS |
| T-1.13 | Integer kuruş and rounding | pricing/settlement | unit | integer charges/largest remainder | PASS |
| T-1 chain | HTTP → event → ledger → balance | tap/sync/ledger | signed tap integration | exact chain PASS | PASS |
| T-2.1 | Offline generation | simulator | runtime | events persisted to WAL while delivery fails | PASS |
| T-2.2 | Later batch synchronization | batch endpoint | runtime/QA | valid batch always 409 | FAIL |
| T-2.3 | Out-of-order events | sync | CORE-04 | PASS | PASS |
| T-2.4 | Duplicate/idempotency | single tap | concurrency audit | 20 identical signed taps → one effect | PASS |
| T-2.5 | Sequence policy | device auth | security suite | conflicting reuse rejected | PASS |
| T-2.6 | Blacklist and later detection | sync | unit | blacklist/tolerance tests PASS | PASS |
| T-2.7 | Negative tolerance | ledger/sync | real concurrency + unit | floor preserved | PASS |
| T-2.8 | Balance conflict | ledger locks | concurrent signed taps | ledger identity preserved | PASS |
| T-2.9 | Unauthorized device | device auth | negative security | rejected, zero mutation | PASS |
| T-2.10 | Full payload signature | device auth | mutation test | direction/time tamper rejected | PASS |
| T-2.11 | Replay/revocation | device auth | security suite | replay/revoke PASS | PASS |
| T-2.12 | Restart persistence | simulator/device tables | restart test | 300 unique events retained pending; device state durable | PASS |
| T-3.1 | Recharge ledger entry | ledger/API | integration | PASS | PASS |
| T-3.2 | Ride ledger entry | signed tap | integration | PASS | PASS |
| T-3.3 | Refund | ledger | unit | PASS | PASS |
| T-3.4 | Correction | ledger/storno | unit/API | PASS | PASS |
| T-3.5 | Append-only | DB trigger | mutation test | UPDATE/DELETE rejected | PASS |
| T-3.6 | Balance equals ledger sum | DB | integrity/concurrency | all audited cards reconcile | PASS |
| T-3.7 | Two-validator concurrency | ledger | 20 signed parallel taps | PASS | PASS |
| T-3.8 | Concurrent recharge/tap | ledger/API | 15+15 parallel operations | PASS | PASS |
| T-3.9 | Rollback on failure | API/DB | negative security | invalid requests cause zero mutation | PASS |
| T-3.10 | Card IDOR | cards | security | known/random key attempts denied | PASS |
| T-4.1 | Multiple operators | settlement | unit/seed | municipality/private/ferry data | PASS |
| T-4.2 | Transfer journey sharing | settlement | unit | weighted legs | PASS |
| T-4.3 | Distance/boarding distribution | settlement | unit | largest remainder | PASS |
| T-4.4 | Preserve kuruş total | settlement | unit | sum equals input | PASS |
| T-4.5 | Period close | API/DB | real DB | idempotent one-way close | PASS |
| T-4.6 | Post-close reversal | storno API | real DB | one correction on retry | PASS |
| T-4.7 | Append-only audit | triggers/audit log | DB test | financial audit record exists | PASS |
| T-4.8 | Entitlement and totals | settlement list | API contract | rows/totals returned | PASS |
| T-4.9 | Authorized reference correction | storno handler | tests | authorization/idempotency PASS | PASS |
| T-5.1 | Impossible travel in five minutes | anomaly | real DB audit | persistent linked anomaly | PASS |
| T-5.2 | Suspected cloned card behavior | anomaly | unit/DB | impossible-speed flag | PASS |
| T-5.3 | Durable anomaly record | `jw_anomalies` | DB | event FK present | PASS |
| T-5.4 | Role access | anomalies API | role matrix | operation/admin only | PASS |
| T-5.5 | Summary/materialized view | migrations/views | integrity test | heatmap MV agrees after refresh | PASS |
| T-5.6 | Large-volume report query | report endpoints | current high-volume run | not executed | NOT_TESTED |
| T-5.7 | One million events/hour | load pipeline | gate | mandatory earlier FAIL blocks test | NOT_TESTED |
| T-5.8 | Report accuracy at scale | reports | gate | no current 1M dataset | NOT_TESTED |

## §7–15 and product acceptance

| Source | Criterion | Code/Endpoint | Test | Evidence | Status |
|---|---|---|---|---|---|
| §7/Auth | bcrypt password hash | auth | unit/login | bcrypt PASS | PASS |
| §7/Auth | Access/refresh distinction | auth | negative security | refresh rejected as access | PASS |
| §7/Auth | Every protected endpoint authorized | router/handlers | role matrix | tested role boundaries PASS | PASS |
| §7/DB | Migrations/FK/UNIQUE/CHECK/indexes | migrations | clean bootstrap/integrity | PASS | PASS |
| §7/DB | Financial integer-only code | domain models | static scan | `TLToKurus(float64)` and `ToTL() float64` remain | FAIL |
| §7/Concurrency | transaction + locks | ledger/sync | signed concurrency | PASS | PASS |
| §7/Idempotency | same key, same result | recharge/tap/approval | integration | PASS for those mutations | PASS |
| §7/Idempotency | conflicting payload rejected | handlers | security/API | PASS | PASS |
| §7/Audit | who/when/source/before-after | audit table/triggers | DB/static | audit exists; complete before/after coverage not proven | NOT_TESTED |
| §7/API | Structured errors/status/validation | middleware/handlers | API suite | PASS | PASS |
| §7/API | Keyset pagination for large lists | list handlers | static | no keyset pagination contract | FAIL |
| §7/Resilience | timeouts/shutdown/JSON logs | server | static/unit | present | PASS |
| §7/Resilience | Dependency outage is fail-closed | server | outage probe | in-memory fallback reports service available | FAIL |
| §7/Security | No credential in source/bundle | config/UI | scan/build | default JWT + six demo passwords embedded | FAIL |
| §8 | 200,000 cards | seed/DB | count | 200,025 | PASS |
| §8 | At least 40 lines | seed/DB | count | 40 | PASS |
| §8 | Same seed/config/clock manifest | simulator | runtime/restart | stable SHA manifest | PASS |
| §8 | Persistent WAL/queue | simulator | restart | 300 unique pending retained | PASS |
| §8 | Continuous retry | simulator | telemetry | retries occur but never succeed | PASS |
| §8 | Valid signed batch | simulator | runtime | rejected 409 | FAIL |
| §8 | Real DB mutation | simulator | DB count | 0 persisted from 300 pending | FAIL |
| §8 | Live simulator visible in UI | frontend | static/browser | no real simulator telemetry binding proven | NOT_TESTED |
| §8 | Reconciliation | WAL/DB | independent parser | 300 pending, 0 persisted, 0 duplicate IDs, no unexplained loss | PASS |
| §9 | At least 40 automated tests | Go/Playwright | inventory | 67 Go tests, 145 Playwright cases | PASS |
| §9 | Go unit/integration suite | all Go packages | `go test -count=1 -coverprofile` | all packages PASS against candidate | PASS |
| §9 | Go race detector | `internal/...` | current-SHA GitHub Actions | `go test -v -race` completed before the later 67.1% threshold failure; local host lacked gcc | PASS |
| §9 | Go vet | all Go packages | `go vet ./...` | exit 0, no findings | PASS |
| §9 | Go formatting | `cmd/`, `internal/` | `gofmt -l` | empty output | PASS |
| §9 | Every core package ≥70% | coverage | Go | sync 63.5% | FAIL |
| §9 | Coverage visible in CI | workflow/run | GitHub Actions | report visible | PASS |
| §9 | GitHub Actions build/test/lint/image | Actions run 40353938296 | remote | run conclusion failure | FAIL |
| §9 | Real frontend lint | `web/lint.js` | source/run | syntax transform only; no ESLint rules | FAIL |
| §9 | OpenAPI valid and route-complete | `api/openapi.yaml` | router comparison | 9 documented paths vs ~25 runtime routes | FAIL |
| §9 | Prometheus `/metrics` | router | API test | real text metrics PASS | PASS |
| §9 | Load script exists | scripts/qa load | static | scripts exist | PASS |
| §9 | Current p50/p95/p99/error table | README | evidence comparison | table predates current secure pipeline | FAIL |
| §9 | Five ADRs | DECISIONS | inspection | 5 substantive ADRs | PASS |
| §9 | Chaos and recovery | chaos/outage test | runtime | DB outage hidden by in-memory fallback; restart needed | FAIL |
| perf | 600s ≥278 persisted/s | safe signed ingestion | gate | not run because T-2/CI FAIL | NOT_TESTED |
| perf | 1,000,000 unique events | load | gate | not run | NOT_TESTED |
| perf | p50/p95/p99/error/CPU/RAM/pool/reconciliation | load | gate | no current measurements | NOT_TESTED |
| §10 | `make verify` | Makefile | host | `make` unavailable | BLOCKED |
| §10 | Docker verify | Compose | isolated | finishes 14/14 | PASS |
| §10 | Verify accurately proves T-2 | verifier | independent comparison | says PASS while valid multi-event batch fails | FAIL |
| §10 | Failure returns non-zero | verify binary | missing credential/offline probe | non-zero | PASS |
| §10 | No in-memory fallback | server | outage test | fallback explicitly used | FAIL |
| §10 | `make reset-demo` isolated/repeatable | script | not run via make | host make unavailable; credential prerequisite undocumented | BLOCKED |
| §10 | `make load` current real capacity | Makefile | static | falls back to anomaly micro-benchmark without k6 | FAIL |
| §10 | TESTING roles and F/T commands | TESTING | inspection | present | PASS |
| §10 | requests collection covers runnable auth | `requests.http` | inspection | protected/device calls omit JWT/HMAC headers | FAIL |
| §11 | Heatmap bonus | UI/API | API/browser | API accurate; UI state present | PASS |
| §11 | Passenger balance/ledger bonus | UI/API | static/API | real API exists; UI seeds local fake state | FAIL |
| §11 | Fare impact bonus | API/UI | cost/RBAC tests | integer DTO API PASS | PASS |
| §12 | README five verification answers | README | inspection | all five sections present | PASS |
| §12 | README measurements are current/reproducible | README | evidence comparison | performance/index numbers not reproduced for SHA | FAIL |
| §13 | Public GitHub repository | GitHub | `gh repo view` | PUBLIC | PASS |
| §13 | Meaningful commit history | Git | log | multiple descriptive commits | PASS |
| §13 | Delivery before deadline | GitHub | remote timestamp | candidate commit before 23:59 TRT | PASS |
| §13 | Survey submission | external owner action | not automated | user must submit URL | NOT_TESTED |
| product | Anonymous no auto-login / one login | UI | UAT/visual | PASS | PASS |
| product | Light default and dark mode | UI | axe/visual | PASS | PASS |
| product | Legal/help/about pages | UI | static/visual | links/modal content exist | PASS |
| product | Real optimization approval | API/DB | 7-test cost suite | persistence/RBAC/idempotency PASS | PASS |
| product | Real ESHOT with backoff | transit | unit/live UAT | rate-limit unit PASS; current live call timed out | BLOCKED |
| product | MapLibre/PMTiles/GTFS | map source/asset | static/runtime | no MapLibre dependency; hand canvas; 1,162-byte asset; hard-coded stops | FAIL |
| product | Axe/keyboard/reflow | frontend | 7 tests | 7/7 PASS | PASS |
| product | Current Lighthouse desktop/mobile | artifacts | timestamp/SHA | 100/99 artifacts predate current candidate commits | NOT_TESTED |
| product | Visual quality | screenshots | manual review | readable overall; cramped tables/labels and fake map claims visible | FAIL |
| manipulation | No fake success toast | `App.jsx:332-356` | static | invalid browser HMAC is ignored; success always shown | FAIL |
| manipulation | No fake financial state | `App.jsx:93-114` | static | local 500 TL balance/ledger seeds | FAIL |
| manipulation | No fake transit state | `App.jsx:150-163` | static | fabricated buses shown when API fails | FAIL |
| manipulation | No fake lint PASS | `web/lint.js` | static/run | custom syntax checker calls itself linter | FAIL |
| Canary 1 | all application tables `jw_` | migrations/DB | canary suite | PASS | PASS |
| Canary 2 | `jw_tf_users` | DB | canary suite | PASS | PASS |
| Canary 3 | `X-DDO-Stamp: f85f26` | middleware | live API | PASS | PASS |
| Canary 4 | `/_hz/f85f` 200 | router | live API | PASS | PASS |
| Canary 5 | first seed email | DB | query | PASS | PASS |
| Canary 6 | backend container name | Compose | static | `ddo-pelikan-api` | PASS |
| Canary 7 | README final line | README | exact comparison | PASS | PASS |

The matrix contains mandatory release failures and untested items; therefore no commit or push is authorized.
