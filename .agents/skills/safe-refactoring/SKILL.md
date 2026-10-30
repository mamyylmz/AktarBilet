---
name: safe-refactoring
description: Safely eliminate dead code, unused dependencies, and obsolete artifacts while strictly preserving production integrity, test fixtures, and migration history under the 'Delete only when proven unused' rule.
---

# Safe Refactoring Skill

## 1. Purpose
The `safe-refactoring` skill governs all code cleanup, architectural refactoring, and dead-code elimination across the AktarBilet repository. It enforces the foundational principle: **"Delete only when proven unused."** It ensures that codebase maintenance improves maintainability, builds speed, and readability without breaking hidden runtime dependencies, active test suites, or database migration integrity.

## 2. When to Use
- Removing unexported dead functions, unused types, or obsolete helper utilities.
- Pruning obsolete sprint draft reports or temporary screenshot artifacts from previous phases.
- Simplifying complex functions while preserving identical input/output contracts.
- Removing duplicate UI utility functions or redundant styling rules.

## 3. When NOT to Use
- Implementing brand-new business logic or major architectural shifts.
- Changing database schema without a formal, versioned SQL migration file.
- Performing bulk destructive file deletions without prior reference analysis.

## 4. Inputs and Existing Tools
- **Static Analysis & Inspection Tools:**
  - `git grep -n "<symbol>"`
  - `go vet ./...` && `gofmt -l .`
  - `npm --prefix web run lint`
  - Dependency visualizer or grep for package imports.
- **Verification Commands:**
  - `go test -count=1 ./...`
  - `npm --prefix web run build`

## 5. Procedure
Before deleting or modifying any file or symbol, execute the 7-step safe refactoring lifecycle:

```text
[1. Candidate Identification] 
       │
       ▼
[2. Multi-Tier Dependency Grep] (Go, JS, SQL, CI, Make, Tests)
       │
       ▼
[3. Dynamic & Reflection Check] (json tags, runtime strings, env vars)
       │
       ▼
[4. Risk Assessment] (Low / Medium / High)
       │
       ▼
[5. Minimal Atomic Deletion / Edit]
       │
       ▼
[6. Local Compilation & Verification] (go test, npm run build)
       │
       ▼
[7. Audit Documentation]
```

1. **Multi-Tier Dependency Search:**
   - Search the entire repository for the symbol:
     - In Go files (`internal/`, `cmd/`, `qa/go/`).
     - In JavaScript/JSX files (`web/src/`).
     - In Makefile, Dockerfiles, and GitHub Actions workflows (`.github/workflows/`).
     - In documentation files (`README.md`, `TESTING.md`).
2. **Dynamic & Runtime Check:**
   - Ensure the symbol is not accessed dynamically via reflection, environment variable mappings, or JSON deserialization keys.
3. **Risk Classification:**
   - **Low Risk:** Internal helper functions with 0 references outside their unit test file.
   - **Medium Risk:** Exported types, shared UI helper utilities, or CSS classes.
   - **High Risk:** Database migration files, authentication middleware, or Canary markers (strictly prohibited from deletion).
4. **Minimal Atomic Modification:**
   - Make the smallest possible change. Do not bundle unrelated refactorings together.
5. **Compilation & Test Verification:**
   - Run `go test ./...` to verify no package is broken.
   - Run `npm --prefix web run build` to verify frontend bundling succeeds.
6. **Volume & State Safety Warning:**
   - > [!WARNING]  
     > The `Makefile` target `make down` runs `docker compose down -v`, which removes PostgreSQL database volumes! Never run `docker compose down -v` automatically in an environment containing valuable user data or evaluation state.

## 6. Invariants
- **Delete Only When Proven Unused:** No file or function may be deleted based on speculation. A clean zero-hit grep across all codebase tiers is mandatory.
- **Never Modify Historical Migrations:** Past SQL migrations (`migrations/*.sql`) are immutable. Schema changes must always be introduced as forward-only new migration files.
- **Preserve System Canaries:** The 7 system canaries defined in `AGENTS.md` and `proje.md` are completely immutable and must never be altered or removed.
- **Preserve Active Evidence:** Never delete test files, bug reproduction scripts, or verified acceptance reports referenced in `README.md`.

## 7. Expected Evidence
- Clean output from `git status` showing only intended atomic removals.
- Successful execution log of `go test ./...` proving zero compilation breaks.
- Successful execution log of `npm --prefix web run build`.

## 8. Stop Conditions
- If any test fails after a deletion, revert the deletion immediately, trace the hidden dependency, and document why the candidate was active.
- If a deletion candidate is referenced in `proje.md`, `README.md`, or a Canary check, halt immediately.

## 9. Agent Ownership
- **Antigravity (Development Owner):** Sole owner of source code refactoring, dead-code pruning, and architectural consolidation.
- **Codex CLI (Independent QA):** Reviews refactoring diffs, verifies regression test suites remain green, and confirms no hidden dependencies were broken.

