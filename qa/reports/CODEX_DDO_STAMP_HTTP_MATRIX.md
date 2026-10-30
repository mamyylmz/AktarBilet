# X-DDO-Stamp HTTP matrix

**Checkout:** `c2701cf7e9af412c8723f4b1c8c39467f5945c23`  
**Environment:** isolated Compose project `aktarbilet-codex-final`; backend `127.0.0.1:18080`, frontend `127.0.0.1:13000`.  
**Test:** `qa/tests/codex_ddo_stamp_http_matrix.spec.js` (included in the 151-test Playwright run; passed).

Every sampled response contained exactly the expected normalized header value `f85f26`:

| Request class | Status |
|---|---:|
| `GET /health` | 200 |
| `GET /_hz/f85f` | 200 |
| catalog API and frontend proxy `/api/v1/tariffs` | 200 |
| unknown route | 404 |
| wrong method | 405 |
| CORS preflight | 200 |
| bad password, missing auth, malformed token, refresh token as access | 401 |
| passenger role denied from settlement list | 403 |
| passenger and accounting login | 200 |
| invalid settlement-close body | 400 |

**Result: sampled matrix PASS; universal claim remains incomplete.** This is a broad representative matrix, not an exhaustive request over every API route or all proxy/intermediary failures. It did not force a backend-generated 500, database outage, duplicate/conflicting header value, or proxy-generated error. Canary and ordinary API success/error middleware paths are covered. Run the same matrix plus explicit outage/error-path probes before claiming every possible backend response is covered.

Reproduction (from `qa/`):

```sh
QA_PROJECT_NAME=aktarbilet-codex-final QA_API_URL=http://127.0.0.1:18080 QA_FRONTEND_URL=http://127.0.0.1:13000 npx playwright test tests/codex_ddo_stamp_http_matrix.spec.js
```
