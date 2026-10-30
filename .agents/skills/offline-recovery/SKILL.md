---
name: offline-recovery
description: Verify offline validator ingestion, local queue/WAL persistence, HMAC-SHA256 signature verification, replay protection, and fault-tolerant batch synchronization into PostgreSQL.
---

# Offline Recovery Skill

## 1. Purpose
The `offline-recovery` skill governs the verification and maintenance of AktarBilet's offline turnstile synchronization protocol (T-2). It guarantees that when transit vehicles (buses, trams, ferries) operate in cellular dead zones or experience network outages, local tap events are cryptographically sealed, queued with monotonic sequence numbers, and reliably synced to the central PostgreSQL database upon reconnection without duplicates, data loss, or balance corruption.

## 2. When to Use
- Auditing the offline event ingestion pipeline ([internal/sync/service.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/service.go)).
- Testing device authentication, sequence tracking, and replay attack protection ([internal/sync/device_auth.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/device_auth.go)).
- Verifying batch sync endpoints (`POST /api/v1/validators/batch-sync`) under network failure injection.
- Validating the negative balance tolerance and blacklisting policies during offline operations.

## 3. When NOT to Use
- Normal online passenger web portal operations.
- Testing non-validator endpoints (e.g., reports, heatmaps, tariff queries).
- Running disruptive restart commands on live production infrastructure.

## 4. Inputs and Existing Tools
- **Core Sync Packages:**
  - [internal/sync/service.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/service.go)
  - [internal/sync/device_auth.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/device_auth.go)
  - [cmd/simulator/main.go](file:///home/dilunsel/projects/AktarBilet/cmd/simulator/main.go)
- **Database Tables:** `jw_validator_events`, `jw_validator_credentials`, `jw_device_nonces`, `jw_cards`, `jw_ledger`.
- **Automated Tests:**
  - `go test -v -count=1 ./internal/sync/...`
  - Integration suites: `qa/tests/api.integration.spec.js`, `qa/scripts/chaos-check.mjs`.

## 5. Procedure
1. **Device Authentication Verification:**
   - Verify that all validator event batches are authenticated using HMAC-SHA256 signatures:
     `HMAC-SHA256(secret, validator_id + ":" + sequence_no + ":" + timestamp + ":" + nonce + ":" + payload_hash)`
   - Verify timestamp drift protection ($\pm 300$ seconds allowed; older or future timestamps rejected with 401).
   - Verify nonce freshness: Nonces must be tracked in `jw_device_nonces`; repeated nonces rejected as replay attempts.
   - Verify monotonic sequence numbers: Events with sequence numbers less than or equal to the committed high-water mark are recognized and rejected as duplicates.
2. **Offline Failure & Recovery Cycle:**
   In an isolated QA environment, execute the 8-step verification cycle:
   - **Step 1 (Generate Events):** Emit offline taps locally on the simulator.
   - **Step 2 (Disconnect Simulation):** Simulate network partitioning or server unreachable state.
   - **Step 3 (Persistent Local Queue):** Confirm events are written to the simulator's local SQLite/file WAL queue.
   - **Step 4 (Network Reconnection):** Restore network connection to the backend API.
   - **Step 5 (Batch Ingestion):** Trigger `POST /api/v1/validators/batch-sync` with the buffered batch.
   - **Step 6 (PostgreSQL Verification):** Query `jw_validator_events` to verify that all events have transitioned to `status = 'PROCESSED'`.
   - **Step 7 (Zero Duplicate Financial Impact):** Re-submit the identical batch payload; verify that HTTP 200/idempotent status is returned and no secondary debit is made to `jw_ledger`.
   - **Step 8 (Total Reconciliation):** Verify the fundamental balance equation:
     $$\text{generated} = \text{persisted} + \text{rejected} + \text{pending}$$
3. **Negative Balance Tolerance Enforcement:**
   - If a card has insufficient balance during offline taps:
     - Allow tap under negative tolerance limit (default: 50.00 TL / 5000 kuruş).
     - Event status marked as `NEGATIVE_TOLERANCE`.
     - If negative tolerance limit is exceeded, status marked as `REJECTED` and card flagged `is_blacklisted = true`.
   - Subsequent offline taps with a blacklisted card must be rejected immediately at turnstile.

## 6. Invariants
- **Conservation of Events:** No event can disappear without a trace; every generated tap must be deterministically accounted for as `PROCESSED`, `REJECTED`, or `PENDING`.
- **Monotonic Sequencing:** Duplicate sequence numbers per validator ID cannot generate duplicate ledger entries.
- **Strict Zero Duplicate Debit:** Idempotency keys (`SYNC:<validator_id>:<sequence_no>`) guarantee exactly-once processing.
- **Fail-Safe Integrity:** Network drops during a batch sync transaction must roll back cleanly without leaving orphan partial debits.

## 7. Expected Evidence
- Unit test logs from `TestDeviceAuthManager_ValidSignatureAndReplayProtection`, `TestDeviceAuthManager_VerifyBatchSignature`.
- Sync service test logs from `TestSyncService_Idempotency`, `TestSyncService_NegativeTolerancePolicy`, `TestSyncService_IngestBatch_ChronologicalSort`.
- SQL count queries verifying `COUNT(*) FROM jw_validator_events WHERE validator_id = $1` matches the ingested batch count.

## 8. Stop Conditions
- If any replay attack with an identical nonce succeeds, stop immediately and report security flaw.
- If an event batch causes duplicate debits in `jw_ledger`, halt testing and mark `FAIL`.
- If the event equation $\text{generated} \neq \text{persisted} + \text{rejected} + \text{pending}$ shows unexplained discrepancy, fail the recovery test.

## 9. Agent Ownership
- **Codex CLI (Independent QA):** Formulates network partitioning scenarios, generates malicious replay/duplicate payloads, verifies sequence enforcement, and audits PostgreSQL persistence logs.
- **Antigravity (Development Owner):** Implements device signature validation, database idempotency constraints, and WAL sync queue handling in backend source code.

