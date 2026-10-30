# ASVS 5.0.0 and skill ecosystem audit

**Checkout:** `c2701cf7e9af412c8723f4b1c8c39467f5945c23`. The mapping was checked against the version-pinned [OWASP ASVS 5.0.0 CSV](https://github.com/OWASP/ASVS/raw/v5.0.0/5.0/docs_en/OWASP_Application_Security_Verification_Standard_5.0.0_en.csv). No certification claim is made.

## ASVS map

The map claims 18 requirements but contains 22 unique IDs. Status is assigned to the **mapping claim**, not a global product certification.

| ID(s) | Status | Audit finding |
|---|---|---|
| V6.2.1, V6.2.6, V6.2.8 | NOT_TESTED | Map topics align (password length, password input type, no transformation); this run did not test each requirement end to end. |
| V6.3.1 | NOT_TESTED | Brute-force controls are relevant; no rate-limit threshold test in this run. |
| V6.3.8 | NOT_TESTED | Generic authentication failure aligns; wrong-password status tested, response equivalence/timing not. |
| V6.5.2 | INVALID_MAPPING | Requirement concerns lookup secrets, not bcrypt password storage. |
| V7.2.1, V7.2.2 | INVALID_MAPPING | These do not establish JWT UUID/randomness, signature/expiry/issuer checks as claimed. |
| V7.3.1 | INVALID_MAPPING | The map’s access-token expiry claim does not establish inactivity timeout. No refresh/logout/revocation route was found; browser logout only clears client storage. |
| V8.1.1, V8.2.1, V8.2.2, V8.3.1 | NOT_TESTED | Appropriate authorization topics, but the mapping does not itself prove endpoint/data coverage. Existing RBAC/IDOR tests passed in the broad suite. |
| V8.3.2 | FAIL | `RequireRole` falls back to the token role if the current-role DB lookup errors; stale authority can survive a database failure/deleted user. |
| V1.2.1 | INVALID_MAPPING | Output encoding requirement, not SQL injection prevention. |
| V1.3.1 | INVALID_MAPPING | WYSIWYG/untrusted HTML sanitization; React escaping alone is not evidence for this requirement. |
| V4.1.1 | INVALID_MAPPING | Response `Content-Type`, not authentication. |
| V4.2.1, V4.2.2 | INVALID_MAPPING | HTTP request/response framing requirements, not validation/idempotency. |
| V13.1.1 | INVALID_MAPPING | Communication documentation requirement, not dependency scanning. |
| V13.2.1 | INVALID_MAPPING | Authenticated backend component communications, not merely configurable environment secrets. |
| V13.3.1 | INVALID_MAPPING | Secret management solution requirement, not HMAC algorithm choice. |

The map also points to `govulncheck` although this task did not run it. The 24-hour access-token setting (`internal/auth/auth.go`) has no demonstrated idle timeout or revocation mechanism. Refresh-as-access rejection passed; that does not establish safe refresh, logout, or account-role revocation. Smallest AGY action: correct the map to v5.0.0 normative titles, correct the count, and link each PASS only to a test that exercises that exact requirement.

## Seven skill smoke/contract review

All seven files exist, have frontmatter and the expected procedural sections, and their relative reference paths resolve. Runtime discovery: this Codex session exposed all seven repository skills. Standalone Antigravity discovery was not available to verify.

| Skill | Assessment |
|---|---|
| `spec-compliance` | PARTIAL — maps to real F/T evidence, but overstates a 171-criteria matrix and references `make`; `make verify` is not a comprehensive proof of T-1/T-5. |
| `financial-invariants` | PARTIAL — points to ledger/settlement checks; review its transfer wording against project semantics (a third boarding is full fare, not rejected). |
| `offline-recovery` | PARTIAL — correct HMAC/sequence/persistence workflow; hard-coded ±300-second drift must match actual `device_auth.go` configuration before use. |
| `security-gate` | FAIL — normative ASVS map has materially wrong titles/claims (table above), so following it can produce false PASS. |
| `safe-refactoring` | PASS — deletion proof/reference search and data/migration preservation are actionable and align with the cleanup reviewed. |
| `accessibility-audit` | PARTIAL — invokes existing Playwright/axe checks; current broad run still failed two desktop axe scenarios, so it cannot claim acceptance. |
| `performance-evidence` | PARTIAL — calls for empirical DB/load evidence, but current `qa/load/event-load.mjs` does not sign the current device batch protocol; no current T-5 persisted capacity result. |

AGENTS.md changes and `.agents/` are uncommitted user/AGY work and were not edited. The Technical Reviewer prompt is present and read-only by its stated role; no independent Antigravity runtime was available for execution.
