---
name: financial-invariants
description: Verify financial correctness, integer-kuruş accounting, append-only ledger, transfer discount caps, monthly fare caps, and settlement storno life-cycle. Use for finance-related changes or acceptance reviews.
---

# Financial Invariants Skill

## 1. Purpose
The `financial-invariants` skill guarantees absolute mathematical, ledger, and transactional integrity across all financial subsystems of AktarBilet. It enforces the immutable principles of kuruş-level integer math, append-only ledger accounting, transfer rule limits, row-level locking on balance mutations, out-of-order event determinism, and zero-sum revenue apportionment.

## 2. When to Use
- Whenever modifying or testing fare calculation algorithms ([internal/pricing/engine.go](file:///home/dilunsel/projects/AktarBilet/internal/pricing/engine.go)).
- When changing balance ledger mechanics, database queries, or transaction boundaries ([internal/ledger/ledger.go](file:///home/dilunsel/projects/AktarBilet/internal/ledger/ledger.go)).
- When updating operator revenue share allocations, period closure, reopenings, or storno entries ([internal/settlement/settlement.go](file:///home/dilunsel/projects/AktarBilet/internal/settlement/settlement.go)).
- When handling out-of-order validator tap reconciliation ([internal/sync/service.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/service.go)).

## 3. When NOT to Use
- Modifying UI stylesheets, layouts, or static informational content.
- Auditing non-financial endpoints (e.g., catalog queries `/api/v1/stations`, `/api/v1/lines`).
- Performing hardware signature verification without balance implications.

## 4. Inputs and Existing Tools
- **Core Domain Packages:**
  - [internal/domain/models.go](file:///home/dilunsel/projects/AktarBilet/internal/domain/models.go) (`type Kurus int64`)
  - [internal/pricing/engine.go](file:///home/dilunsel/projects/AktarBilet/internal/pricing/engine.go)
  - [internal/ledger/ledger.go](file:///home/dilunsel/projects/AktarBilet/internal/ledger/ledger.go)
  - [internal/settlement/settlement.go](file:///home/dilunsel/projects/AktarBilet/internal/settlement/settlement.go)
- **Database Tables:** `jw_cards`, `jw_ledger`, `jw_settlement_periods`, `jw_settlement_ledger`.
- **Automated Test Runners:**
  - `go test -v -count=1 ./internal/pricing/...`
  - `go test -v -count=1 ./internal/ledger/...`
  - `go test -v -count=1 ./internal/settlement/...`
  - `go test -v -run TestAPI_SettlementLifecycle_Full ./internal/api/...`
  - Shell runner: `sh qa/scripts/verify-fare.sh`

## 5. Procedure
1. **Integer Kuruş Verification:**
   - Verify that all money representations use `domain.Kurus` (`int64`).
   - Grep for `float32`, `float64`, or division without integer quotient/remainder handling in pricing and settlement packages. Ensure zero float usage in financial logic.
2. **Fare Engine Rule Validation (T-1):**
   - **First Tap:** Full base fare charged according to card type (Tam: 2000 kr, Öğrenci: 1000 kr, Öğretmen: 1500 kr, 65+: 0 kr).
   - **Transfer Window:** Valid only within 90 minutes of the first boarding.
   - **Maximum Two Transfers:** Transfer 1 (50% discount) and Transfer 2 (75% discount) are allowed. A 3rd transfer, 4th transfer, or beyond is strictly forbidden; any tap beyond 2 transfers resets to full fare.
   - **Same-Line Return Rule:** Tapping back on the same line or in reverse direction does not qualify as a transfer; full fare must be charged.
   - **Night Tariff:** Boardings between 00:00 and 05:59 apply a 100% surcharge (2x rate).
   - **Monthly Fare Capping:** Once cumulative monthly spending reaches `MonthlyCappingLimitKurus`, subsequent taps within the calendar month are charged 0 kuruş (or the exact remainder to meet the cap).
3. **Ledger & Concurrency Validation (T-3):**
   - Verify that `jw_ledger` is strictly append-only (no `UPDATE` or `DELETE` statements on `jw_ledger`).
   - Check that concurrent taps on the same card execute `SELECT ... FOR UPDATE` on `jw_cards` to prevent race conditions and double debits.
   - Ensure card balance is always mathematically identical to `COALESCE(SUM(amount_kurus), 0)` from `jw_ledger`.
4. **Out-of-Order Event Determinism:**
   - Simulate out-of-order tap arrival within the 90-minute window.
   - Ensure the reconciliation loop iterates all subsequent events, recalculating chronological pricing and applying deterministic `StornoCorrection` ledger entries with reference `OOO-REPRICE:<id>`.
5. **Settlement & Largest Remainder Method (T-4):**
   - Verify that `CalculateRevenueShare` allocates pooled revenue among operators such that `SUM(operator_shares) == total_revenue` down to the exact kuruş.
   - Kuruş rounding fractions must be allocated using the Hamilton / Largest Remainder Method without creating or destroying money.
   - Verify the period lifecycle: `OPEN -> Storno Rejected (400) -> CLOSED -> Storno Accepted (200) -> Reopen Authorized (200) -> OPEN`.

## 6. Invariants
- **Kuruş Integrity:** No IEEE floating-point arithmetic permitted in financial state or calculations.
- **Append-Only Ledger:** Balance cannot be directly edited via raw update; every balance alteration requires a corresponding row in `jw_ledger`.
- **Zero Double-Debit:** Card balance cannot be decremented twice for the same event or idempotency key.
- **Zero Money Creation/Destruction:** `Total Pool == sum(all operator shares)`.
- **Chronological Determinism:** The set of events $E = \{e_1, e_2, e_3\}$ must yield the exact same ledger balance and capping spend regardless of the arrival permutation order.

## 7. Expected Evidence
- Complete test pass output from `internal/pricing`, `internal/ledger`, `internal/settlement`.
- Output log from `verify-fare.sh` confirming integer kuruş and transfer discounting rules.
- Test log from `TestAPI_SettlementLifecycle_Full` confirming period close, reopen, and storno authorization.

## 8. Stop Conditions
- Any occurrence of `float64` in pricing or ledger calculation logic triggers immediate failure.
- If a 3rd transfer receives a discount instead of full fare, stop and report policy violation.
- If revenue apportionment total deviates from pool revenue by even 1 kuruş, stop immediately.
- If an open period accepts a storno request without returning HTTP 400 `PERIOD_NOT_CLOSED`, mark `FAIL`.

## 9. Agent Ownership
- **Codex CLI (Independent QA):** Runs existing financial unit and integration suites, constructs edge-case permutations, verifies ledger balance assertions, and reports discrepancies.
- **Antigravity (Development Owner):** Implements pricing engine algorithms, ledger transaction locks, and settlement life-cycle fixes in production source code.

