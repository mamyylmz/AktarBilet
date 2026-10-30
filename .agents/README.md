# AktarBilet — Lean Agent & Skill Ecosystem

This directory contains the operational skills, evaluation prompts, and security standards that power the pair-programming and quality engineering ecosystem for **AktarBilet**.

The objective of this ecosystem is not to add more artificial intelligence frameworks or runtime bloat, but to structure existing, permanently free engineering tools into repeatable, modular, and evidence-backed verification procedures.

---

## 1. Skill Catalog & Core Mandates

| Skill Name | Canonical Path | Core Mandate (One-Sentence Summary) |
| :--- | :--- | :--- |
| **`spec-compliance`** | [skills/spec-compliance/SKILL.md](./skills/spec-compliance/SKILL.md) | Maps and verifies 100% bidirectional traceability between `proje.md` requirements (F-1..F-7, T-1..T-5) and automated test execution results. |
| **`financial-invariants`** | [skills/financial-invariants/SKILL.md](./skills/financial-invariants/SKILL.md) | Enforces integer-kuruş math, append-only ledger accounting, transfer discount limits, monthly capping, and zero-sum revenue sharing. |
| **`offline-recovery`** | [skills/offline-recovery/SKILL.md](./skills/offline-recovery/SKILL.md) | Verifies offline turnstile tap buffering, HMAC-SHA256 signatures, replay defense, and fault-tolerant batch sync into PostgreSQL. |
| **`security-gate`** | [skills/security-gate/SKILL.md](./skills/security-gate/SKILL.md) | Audits authentication, session management, RBAC, IDOR, and parameterized queries against verified **OWASP ASVS 5.0.0** requirements. |
| **`safe-refactoring`** | [skills/safe-refactoring/SKILL.md](./skills/safe-refactoring/SKILL.md) | Eliminates dead code, unused dependencies, and temporary artifacts under the strict principle: *"Delete only when proven unused."* |
| **`accessibility-audit`** | [skills/accessibility-audit/SKILL.md](./skills/accessibility-audit/SKILL.md) | Ensures WCAG 2.2 AA accessibility, semantic HTML, keyboard-only traversal, responsive mobile reflow, and text fallbacks for transit maps. |
| **`performance-evidence`** | [skills/performance-evidence/SKILL.md](./skills/performance-evidence/SKILL.md) | Measures throughput, latency percentiles (p50/p95/p99), and query execution plans (`EXPLAIN ANALYZE`) using open-source local tooling. |

---

## 2. Agent Ownership & Role Division

AktarBilet operates with two primary agents and one on-demand evaluation profile:

```text
               ┌────────────────────────────────────────────────────────┐
               │              AKTARBİLET AGENT ROLES                    │
               └────────────────────────────────────────────────────────┘
                                 │
         ┌───────────────────────┴────────────────────────┐
         ▼                                                ▼
┌─────────────────────────┐                      ┌─────────────────────────┐
│       ANTIGRAVITY       │                      │        CODEX CLI        │
│    Development Owner    │                      │  Independent QA Owner   │
│  (Production Code Only) │                      │   (Test & Audit Only)   │
└─────────────────────────┘                      └─────────────────────────┘
         │                                                │
         │ • safe-refactoring                             │ • spec-compliance
         │ • Security remediation                         │ • security-gate
         │ • Fare engine implementation                   │ • financial-invariants
         │ • Database migrations                          │ • offline-recovery
         │ • Performance optimization                     │ • accessibility-audit
         │                                                │ • performance-evidence
         └───────────────────────┬────────────────────────┘
                                 │
                                 ▼
               ┌──────────────────────────────────┐
               │        TECHNICAL REVIEWER        │
               │      (On-Demand Prompt Only)     │
               │   Simulated Committee Defense    │
               │      Read-Only Architecture      │
               └──────────────────────────────────┘
```

- **Antigravity (Main Developer):** Holds exclusive write access to production code (`internal/`, `cmd/`, `web/`, `migrations/`). Never runs destructive QA or tests.
- **Codex CLI (Independent QA):** Holds exclusive authority over independent test suites (`qa/`), load tests, negative security tests, and compliance validation. Never modifies production code.
- **Technical Reviewer:** An on-demand, read-only evaluation persona defined in [prompts/technical-reviewer.md](./prompts/technical-reviewer.md) for simulating technical defense interviews.

---

## 3. Invocation Examples

Skills are loaded progressively (on-demand) based on task context or explicit prompt triggers:

```bash
# Example 1: Triggering financial invariant verification
"Verify transfer discounting, same-line return rule, and settlement storno under financial-invariants skill."

# Example 2: Triggering OWASP ASVS security audit
"Evaluate authentication, RBAC, and IDOR defenses against security-gate skill and asvs-5.0-map.md."

# Example 3: Simulating committee defense
"Assume the role defined in .agents/prompts/technical-reviewer.md and conduct a 20-minute architecture defense of AktarBilet."
```

---

## 4. Conscious Omissions (Zero-Cost & Lean Infrastructure)

To preserve software engineering simplicity, zero operational costs, and low context overhead, the following technologies were **intentionally omitted**:

- **No Heavy Agent Orchestrators:** No LangChain, LangGraph, CrewAI, or AutoGen frameworks.
- **No Complex Distributed Middleware:** No Kafka, RabbitMQ, Redis, or ZooKeeper. PostgreSQL 16's ACID transactions, connection pooling, and append-only ledger fulfill all throughput and reliability goals without distributed split-brain risks.
- **No Paid Observability or SaaS Platforms:** No Datadog, New Relic, or Grafana Cloud dependencies. Metrics are exported locally via `/metrics` (Prometheus format) and tested via open-source k6.
- **No Paid Security Scanners:** Security verification relies on Go's official `govulncheck`, race detector (`-race`), and targeted OWASP ASVS 5.0.0 automated specifications.

---

## 5. Evidence Standards

> [!IMPORTANT]  
> The presence of a skill definition does **not** grant automatic compliance or a `PASS` rating. All claims must be backed by reproducible terminal commands, automated test logs, or code references.

