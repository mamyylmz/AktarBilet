# AktarBilet Independent QA

This directory is intentionally isolated from production source files. It runs the shipped images against a dedicated Compose project and PostgreSQL database.

## Prerequisites

- Docker Compose
- Node.js 20+
- Go 1.23+

Install once:

```bash
cd qa
npm install
npx playwright install chromium
```

Run the complete bounded acceptance suite:

```bash
npm run test:all
```

The command starts `aktarbilet-qa`, executes Go coverage plus real API/DB/browser tests, writes JUnit/JSON/HTML evidence under `qa/artifacts/`, and stops only the QA containers. The dedicated `aktarbilet-qa_qa_postgres_data` volume is retained to comply with the repository's no-volume-deletion rule.

Set `QA_PROJECT_NAME` to run against a fresh, separately named QA project and volume without deleting earlier evidence, for example `QA_PROJECT_NAME=aktarbilet-qa-final npm run test:all`.

Focused commands are available as `test:api`, `test:rbac`, `test:e2e`, `test:db`, `test:security`, `test:integration`, and `test:smoke`.
