# AktarBilet — Final Performance Audit

**Status:** **NOT_TESTED — mandatory safety and release gates failed.**

The required 600-second secure ingestion run and separate 1,000,000-event run were not started. This is required by the task: P0/P1, simulator, CI, and mandatory functions must pass first. Producing capacity numbers through the broken batch endpoint or an in-memory benchmark would be misleading.

## Current admissible measurements

| Metric | Result |
|---|---|
| Secure signed multi-event batch | FAIL — HTTP 409 on first five-event batch |
| Simulator WAL after restart | 30 pending batches / 300 unique pending events |
| Simulator PostgreSQL persistence | 0 events |
| Duplicate pending IDs | 0 |
| Unexplained loss in observed WAL accounting | 0; all 300 remain pending |
| Double financial effect | 0 observed |
| Ledger mismatch | 0 in completed signed single-event/concurrency tests |
| 600s persisted events/s | NOT_TESTED |
| 600s p50/p95/p99 and error rate | NOT_TESTED |
| CPU/RAM/pool saturation | NOT_TESTED |
| 1,000,000-event loss/duplicate/reconciliation | NOT_TESTED |

## Lighthouse

Existing artifacts report desktop performance 100 and mobile 99, with desktop FCP/LCP 367 ms and mobile FCP 1,285 ms / LCP 1,435 ms. Their fetch time is 2026-10-26 15:29:27Z, before candidate commits `d72d09e` and `268fd8e`; therefore they are historical supporting evidence, not current-SHA acceptance.

## README measurement validity

README claims 24,150 req/s and p50/p95/p99 1.2/4.8/12.1 ms. Those results were not reproduced on the current secure pipeline and cannot satisfy the requested persisted-event capacity gate. The index timing claims were also not remeasured in this audit.

## Exit condition for a future performance run

First make a real signed batch persist and drain the simulator WAL, make CI/coverage green, and prove `generated = persisted + rejected + pending`. Then run the canonical isolated 600-second and 1M campaigns while collecting HTTP latency, PostgreSQL committed rows, CPU/RAM, pool stats, duplicate IDs, unexplained loss, and ledger reconciliation.
