---
name: spec-compliance
description: Verify end-to-end traceability between proje.md and the AktarBilet codebase, enforcing evidence-backed PASS/FAIL status across all functional (F-1..F-7) and technical (T-1..T-5) requirements.
---

# Spec Compliance Skill

## 1. Purpose
The `spec-compliance` skill establishes a rigorous, bidirectional traceability bridge between the primary specification ([proje.md](file:///home/dilunsel/projects/AktarBilet/proje.md)) and the AktarBilet codebase. It verifies that every functional requirement (F-1 through F-7), technical requirement (T-1 through T-5), architectural criterion, performance target, and system canary is backed by reproducible automated tests, live execution artifacts, or verified source implementations.

## 2. When to Use
- Pre-release gate verification and milestone acceptance reviews.
- Auditing spec completeness before handover to evaluation committees or clients.
- Verifying requirement coverage across backend, frontend, database, and CI/CD pipelines.
- Resolving disputes between conflicting QA reports by tracing directly to code and test execution.

## 3. When NOT to Use
- Routine feature implementation or local refactoring turns.
- Debugging isolated unit test failures that have no specification-level ambiguity.
- Modifying production business logic (this skill is strictly non-destructive and analytical).

## 4. Inputs and Existing Tools
- **Specification Source:** [proje.md](file:///home/dilunsel/projects/AktarBilet/proje.md), [DESIGN.md](file:///home/dilunsel/projects/AktarBilet/DESIGN.md), [DECISIONS.md](file:///home/dilunsel/projects/AktarBilet/DECISIONS.md).
- **Execution & Test Runners:**
  - `go test -v -count=1 ./...`
  - `go test -coverprofile=coverage.out ./internal/...`
  - `npm --prefix web run build` && `npm --prefix web run lint`
  - Local shell verification: `sh qa/scripts/verify-fare.sh`
  - Docker Compose runner: `docker compose ps` / `docker compose exec`
- **Existing Reports:** [AGY_FINAL_PROJE_MD_ACCEPTANCE.md](file:///home/dilunsel/projects/AktarBilet/qa/reports/AGY_FINAL_PROJE_MD_ACCEPTANCE.md), [CODEX_FINAL_PROJE_MD_ACCEPTANCE.md](file:///home/dilunsel/projects/AktarBilet/qa/reports/CODEX_FINAL_PROJE_MD_ACCEPTANCE.md), [T1_INITIAL_FAILURES.md](file:///home/dilunsel/projects/AktarBilet/qa/reports/T1_INITIAL_FAILURES.md).

## 5. Procedure
1. **Clause Mapping:** For each requirement in `proje.md`, locate the corresponding production implementation file and line range:
   - F-1..F-7: Routing, handlers, domain models, database migrations, React components.
   - T-1..T-5: Fare engine, offline sync, ledger, settlement, anomaly engine.
2. **Evidence Identification:** Locate the automated test, fixture, or benchmark covering the clause:
   - Check unit tests in `internal/<pkg>/<pkg>_test.go`.
   - Check integration tests in `qa/go/` and API tests in `internal/api/handlers_test.go`.
   - Check frontend accessibility and E2E in `qa/tests/`.
3. **Execution Verification:** Run the minimum required test command natively:
   - Do not accept claims without command execution output.
   - Record exit codes, durations, coverage percentages, and assertions.
4. **Status Assignment:** Assign one of four immutable states to each item:
   - `PASS`: Proven with automated test execution output and source code link.
   - `FAIL`: Failing assertion, regression, or unmet requirement.
   - `BLOCKED`: Precondition not met (e.g., database unavailable in offline container).
   - `NOT_TESTED`: Implementation exists but lacks an automated test proof.
5. **Canary & Boundary Verification:** Verify the 7 mandatory system canaries:
   - All DB tables prefixed with `jw_` (DDL inspection).
   - Users table named strictly `jw_tf_users`.
   - Header `X-DDO-Stamp: f85f26` on all HTTP responses.
   - Endpoint `GET /_hz/f85f` returns 200 OK.
   - Initial seed user email: `seed.sedir@ddo-case.invalid`.
   - Backend Docker container name: `ddo-pelikan-api`.
   - README.md last line strictly: `DDO Case Kodu: SEDIR-f85f26`.

## 6. Invariants
- **No Unevidenced PASS:** Never mark a criterion as `PASS` based on assumption or code presence alone. A verifiable command, test log, or deterministic inspection is mandatory.
- **Specification Faithfulness:** Do not introduce arbitrary extra-spec quality gates as mandatory failure reasons (e.g., demanding external cloud SaaS when local open-source tools satisfy the requirement).
- **Strictly Zero Float:** Financial calculations must never use IEEE 754 floats; kuruş integer integrity is mandatory.
- **Non-Destructive:** Running this skill must never mutate production code or alter git history.

## 7. Expected Evidence
A structured compliance matrix or verification report containing:
- Specific requirement ID (e.g., `§3 F-1`, `§10 T-1`).
- Source code link (`internal/...#L...`).
- Automated test location (`internal/..._test.go#L...`).
- Verification command and real execution status.
- Final verdict (`PASS` / `FAIL` / `BLOCKED` / `NOT_TESTED`).

## 8. Stop Conditions
- If any of the 7 system canaries fails, stop immediately with status `FAIL` and pinpoint the offending file.
- If a financial calculation returns floating-point values or breaks integer conservation, mark `FAIL`.
- If an assertion fails, do not weaken the assertion to force a pass; document the gap and hand over to Antigravity for remediation.

## 9. Agent Ownership
- **Codex CLI (Independent QA):** Primary user of this skill for conducting independent acceptance audits, running retests, and issuing compliance matrices.
- **Antigravity (Development Owner):** Consumer of compliance findings to execute root-cause code fixes in production files.

