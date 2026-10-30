---
name: security-gate
description: Enforce OWASP ASVS 5.0.0 security standards across authentication, session management, RBAC, input sanitization, and cryptographic device integrity.
---

# Security Gate Skill

## 1. Purpose
The `security-gate` skill establishes an objective, standard-compliant security verification gate for AktarBilet based on **OWASP Application Security Verification Standard (ASVS) 5.0.0**. It systematically evaluates authentication strength, password hashing, session lifecycle, role-based access control (RBAC), IDOR defenses, SQL injection prevention, idempotency guarantees, and device cryptography.

## 2. When to Use
- Pre-release gate verification and code review for security-sensitive paths.
- Evaluating changes to authentication ([internal/auth/auth.go](file:///home/dilunsel/projects/AktarBilet/internal/auth/auth.go)) or authorization middleware ([internal/api/middleware.go](file:///home/dilunsel/projects/AktarBilet/internal/api/middleware.go)).
- Verifying device signature validation and replay defense ([internal/sync/device_auth.go](file:///home/dilunsel/projects/AktarBilet/internal/sync/device_auth.go)).
- Auditing dependencies for known CVEs using official tools (`govulncheck`).

## 3. When NOT to Use
- Normal feature styling, layout adjustments, or minor copy updates.
- Running destructive penetration tests or fuzzing attacks on live production databases.
- Circumventing security checks or adding auth bypasses for testing convenience.

## 4. Inputs and Existing Tools
- **Normative Reference:** [asvs-5.0-map.md](./references/asvs-5.0-map.md) (Official ASVS v5.0.0 mappings).
- **Security Tools:**
  - Official Go vulnerability checker: `govulncheck ./...` (Documentation: https://go.dev/doc/security/vuln/)
  - Go data race detector: `go test -race ./...`
  - Automated security specs: `qa/security/codex_retest.security.spec.js`, `qa/tests/rbac.security.spec.js`.
- **Backend Components:** `internal/auth/`, `internal/api/middleware.go`, `internal/sync/device_auth.go`.

## 5. Procedure
1. **Scope & Reference Check:**
   - Consult [asvs-5.0-map.md](./references/asvs-5.0-map.md) for the target security requirement (using the canonical format `v5.0.0-<id>`).
2. **Static & Dependency Audit:**
   - Run `govulncheck ./...` to detect known vulnerabilities in Go dependencies.
   - Run `npm --prefix web audit` to inspect frontend packages.
3. **Authentication & Password Review (ASVS Chapter V6):**
   - Verify that passwords in `jw_tf_users` are stored using `bcrypt` (work factor $\ge 10$).
   - Confirm that authentication failure messages are generic (`v5.0.0-6.3.8`).
4. **Session & Token Verification (ASVS Chapter V7):**
   - Verify that JWTs are signed using HMAC-SHA256 with a strong secret key.
   - Confirm that tokens have an explicit `ExpiresAt` boundary and are rejected when expired.
5. **Authorization & IDOR Verification (ASVS Chapter V8):**
   - Run RBAC test suites to ensure non-privileged roles cannot access administrative or accounting routes.
   - Test IDOR boundaries: User A must never access User B's card balance or ledger entries (`v5.0.0-8.2.2`).
   - Confirm real-time role checks: When user roles change in the DB, stale JWT elevation is rejected (`v5.0.0-8.3.2`).
6. **Injection & Parameterization Audit (ASVS Chapter V1):**
   - Verify that all database queries use pgx parameterized arguments (`$1, $2`).
   - Confirm zero string interpolation (`fmt.Sprintf` directly inside SQL strings is prohibited).
7. **Device Cryptography & Replay Protection (ASVS Chapter V4 & App. C):**
   - Verify that offline batches require HMAC-SHA256 signatures with valid timestamp, sequence number, and fresh nonce.

## 6. Invariants
- **No Auth Bypass:** Never introduce backdoor headers, test bypass flags, or mock authenticators into production handlers.
- **No Hardcoded Production Secrets:** JWT secrets and cryptographic keys must be loaded from environment variables.
- **Strict Server-Side Enforcement:** Frontend checks are convenience only; authorization must be strictly enforced in Go middleware.
- **Accurate ASVS Claims:** Never claim "ASVS Certified" or "100% ASVS Compliant". Report strictly against evaluated requirement IDs.

## 7. Expected Evidence
- Security scan output from `govulncheck ./...`.
- Test execution output from `internal/api/handlers_test.go:TestAPI_RoleAuthorization` and `TestAPI_GetCard_And_IDOR`.
- Test execution output from `internal/sync/device_auth_test.go`.
- Traceability rows in [asvs-5.0-map.md](./references/asvs-5.0-map.md) updated with test execution evidence.

## 8. Stop Conditions
- Discovery of any unparameterized SQL query executing untrusted user input halts the gate immediately (`FAIL`).
- Any horizontal privilege escalation (IDOR) allows reading other users' private data halts the gate immediately (`FAIL`).
- If hardcoded private keys or secrets are committed to the codebase, halt immediately and trigger credential rotation.

## 9. Agent Ownership
- **Codex CLI (Independent QA):** Formulates negative security test vectors, runs automated security suites, conducts IDOR fuzzing, and maintains the ASVS compliance map.
- **Antigravity (Development Owner):** Implements server-side authorization middleware, password hashing, and parameterized queries in production source code.

