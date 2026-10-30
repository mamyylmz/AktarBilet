---
name: performance-evidence
description: Measure, profile, and verify throughput, p95/p99 latencies, database query plans (EXPLAIN ANALYZE), and T-5 1M events/hour capacity using empirical benchmarks and open-source tooling.
---

# Performance Evidence Skill

## 1. Purpose
The `performance-evidence` skill establishes an empirical, measurement-driven methodology for performance evaluation and optimization across AktarBilet. It prevents speculative optimization by mandating before-and-after profiling, query execution analysis (`EXPLAIN (ANALYZE, BUFFERS)`), throughput measurements (T-5 1M events/hour target), latency percentiles (p50, p95, p99), and memory allocation audits using exclusively open-source, local tools.

## 2. When to Use
- Evaluating the T-5 requirement (capacity to ingest and report 1,000,000 taps per hour).
- Benchmarking batch sync throughput on `POST /api/v1/validators/batch-sync`.
- Investigating slow database queries using PostgreSQL `EXPLAIN ANALYZE`.
- Profiling Go memory allocations or CPU bottlenecks with `pprof` and `testing.B`.
- Measuring keyset cursor pagination efficiency on large table scans.

## 3. When NOT to Use
- Implementing speculative optimizations without baseline measurements.
- Running heavy load tests on active production databases or developer workspaces with unstaged changes.
- Replacing rigorous database transactions with unsafe in-memory mocks purely to inflate synthetic throughput scores.

## 4. Inputs and Existing Tools
- **Profiling & Measurement Tools:**
  - Go Benchmarks: `go test -bench=. -benchmem ./...`
  - Open-source local k6: `k6 run scripts/load/event_ingestion.js` (No cloud SaaS required).
  - PostgreSQL Query Analyzer: `EXPLAIN (ANALYZE, BUFFERS) <SQL>`
  - Go pprof: `net/http/pprof` or benchmark CPU/heap profiles.
  - Metrics Endpoint: `GET /metrics` (Prometheus exposition format).
- **Core Performance Paths:**
  - [internal/sync/service.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/service.go) (Batch ingestion)
  - [internal/anomaly/detector.go](file:///home/dilunsel/projects/AktarBilet/internal/anomaly/detector.go) (High-speed sliding window)
  - [migrations/002_views_and_indexes.sql](file:///home/dilunsel/projects/AktarBilet/migrations/002_views_and_indexes.sql) (Database composite indexes)

## 5. Procedure
```text
[1. Baseline Measurement] (Record p50, p95, p99, throughput, RAM/CPU)
           │
           ▼
[2. Bottleneck Isolation] (EXPLAIN ANALYZE, pprof, flamegraph)
           │
           ▼
[3. Targeted Remediation] (Add index, batching, keyset cursor, alloc reduction)
           │
           ▼
[4. Controlled Retest] (Run identical load in identical environment)
           │
           ▼
[5. Comparative Audit] (Side-by-side metric comparison table)
```

1. **Baseline Measurement:**
   - Execute the test in an isolated QA or local Docker environment.
   - Record baseline metrics: throughput (req/sec or events/sec), latency (p50, p95, p99), error rate (%), CPU and RAM consumption.
2. **Bottleneck Isolation:**
   - For database bottlenecks, run `EXPLAIN (ANALYZE, BUFFERS)` on the query. Check for:
     - Sequential Scans on large tables (`jw_validator_events`, `jw_ledger`).
     - Expensive Sort operations caused by offset pagination (verify index coverage for `ORDER BY (issued_at, id) DESC`).
   - For Go runtime bottlenecks, run `go test -bench=. -benchmem` to identify heap allocations and memory escapes.
3. **Targeted Remediation:**
   - Implement the minimal code or index change:
     - Example: Introduce a composite B-tree index on `(card_id, tapped_at DESC)`.
     - Example: Switch from `OFFSET` pagination to Keyset `WHERE (issued_at, id) < ($1, $2)`.
4. **Controlled Retest:**
   - Re-run the exact same benchmark or k6 script under identical hardware conditions.
5. **Comparative Verification:**
   - Verify that the optimization did not weaken any financial or security invariant (zero float, append-only ledger, and row locks must remain strictly intact).

## 6. Invariants
- **No Extrapolation Without Persistence:** In-memory benchmark numbers cannot be presented as proof of PostgreSQL database persistence. The T-5 persistence pipeline must include actual disk commits and index updates.
- **Never Trade Correctness for Speed:** Idempotency verification, signature checks, and ledger row-locking cannot be bypassed to boost benchmark figures.
- **Reproducible Local Tooling:** Benchmarks must run with standard open-source tools (local k6, standard Go test runner) without requiring paid external subscriptions.

## 7. Expected Evidence
- Comparative benchmark table:
  | Metric | Baseline | Post-Optimization | Delta |
  | :--- | :---: | :---: | :---: |
  | **Throughput** | X req/s | Y req/s | +Z% |
  | **p95 Latency** | X ms | Y ms | -Z% |
  | **p99 Latency** | X ms | Y ms | -Z% |
  | **Allocations** | X B/op | Y B/op | -Z B/op |
- PostgreSQL `EXPLAIN (ANALYZE, BUFFERS)` query plan demonstrating Index Scan replacing Seq Scan.

## 8. Stop Conditions
- If an optimization causes even a single unit test failure in `internal/pricing`, `internal/ledger`, or `internal/sync`, revert immediately.
- If error rate exceeds 0.1% during load testing, abort the test and inspect database connection pool limits.

## 9. Agent Ownership
- **Codex CLI (Independent QA):** Formulates load test scenarios, runs k6 scripts, executes benchmark suites, and gathers latency percentiles.
- **Antigravity (Development Owner):** Implements index definitions, SQL query tuning, memory pooling, and algorithmic optimizations in production code.

