# AktarBilet — Codex Release Retest 3

**Denetim tarihi:** 26 Ekim 2026  
**Rol:** Bağımsız Security/SDET/Performance/Release Gate  
**Karar:** **NOT_READY**

## P0 → P1 bulgular

1. **P0 — Repo validator master secret ile dış caller başarılı tap/debit yapabiliyor.** HTTP 200; kart 10.000→7.000 kuruş; ledger/event/nonce/sequence değişti. `qa/artifacts/codex/retest3/security-summary.json`.
2. **P0 — Canonical signature fallback gövdenin tümünü bağlamıyor.** `direction` ve `tapped_at` değiştirilmiş istek HTTP 200 ve 3.000 kuruş debit üretti. `internal/api/handlers.go:449-455`.
3. **P1 — Simulator persistence gerçekte çalışmıyor.** Non-root process root-owned volume'a WAL/sequence yazamıyor; restart sonrası **730 event kayboldu**. `Dockerfile.simulator:16-23`, `qa/docker-compose.qa.yml:62-72`.
4. **P1 — Aynı validator sequence/farklı payload HTTP 200.** Exact payload conflict kontrolü yok. `internal/sync/device_auth.go:142-151`, `internal/sync/service.go:111-129`.
5. **P1 — Yanlış line/station assignment HTTP 200 ve debit.** Aktiflik kontrolü atama kontrolü değil. `internal/api/handlers.go:486-530`.
6. **P1 — Temiz 102 regresyonu 100/102.** CORE-04 reverse delivery 6.000 vs 5.500 kuruş; browser gömülü credential ile 3 login çağrısı.
7. **P1 — Harita teslimi sentetik.** Gerçek MapLibre/PMTiles/GTFS yok; 1.162 bayt dummy archive ve manuel canvas var.
8. **P1 — Frontend kalite kapıları başarısız.** Intentional bad lint fixture exit 0; axe/keyboard/zoom 1 PASS / 6 FAIL; OpenAPI yalnız 9 path ve eksik auth/HMAC/contracts.
9. **P1 — Finansal API float kullanıyor.** TL/revenue ve oran DTO/hesaplarında `float64`; fixed-point/basis-point değişmezi karşılanmıyor.
10. **P1 — Secret eksikken servis fail-fast değil.** Ayrı container exit 0 ile HTTP listener açtı.

Ayrıntı ve reproduction: `qa/reports/CODEX_RETEST3_SECURITY.md`, `qa/reports/CODEX_RETEST3_RELEASE_GATE.md`.

## Zorunlu özet tablo

Sayılar bağımsız alt kontrol/test sayılarını gösterir; tek bir zorunlu FAIL tüm alanı FAIL yapar.

| Alan | PASS | FAIL | BLOCKED/NOT_TESTED |
|---|---:|---:|---:|
| Security / Auth | 11 | 4 | 4 |
| Bootstrap / Migration | 4 | 1 | 0 |
| Simulator Section 8 | 3 | 4 | 4 |
| ESHOT / MapLibre / PMTiles / GTFS | 8 | 5 | 1 |
| Cost / Optimization | 5 | 5 | 0 |
| 102 Regression | 100 | 2 | 0 |
| Go / Race / Coverage | 8 | 1 | 0 |
| Frontend / Lint / A11y | 3 | 9 | 0 |
| Load / Stress | 0 | 0 | 2 |
| F/T / Canary / Verify | 23 | 1 | 0 |

## Bağımsız test sonuçları

### Security

- Önceki bağımsız paket: **8/8 PASS**.
- Retest 3 genişletilmiş device negatif paketi: **3 PASS / 4 FAIL**.
- PASS: cross-device key reject; exact retry tek finansal etki; revoked cihaz state tüketmiyor.
- FAIL: repo master impersonation; partial canonical body tampering; same-sequence different payload; line/station mismatch.
- JWT testi gerçek signature byte değiştirerek 401 doğruladı; Base64URL gösterim false-negative'i kullanılmadı.

### PostgreSQL bootstrap/migration

- Fresh 001–005 + seed: PASS.
- İkinci açılış idempotansı: PASS.
- Invalid 005: backend exit 1, version/probe rollback: PASS.
- Versionsuz eski 001–004 şemadan upgrade: PASS.
- Secret boş fail-fast: FAIL.
- Approval API: genişletilmiş paket **5 PASS / 2 FAIL**. Persistence, approver/time ve RBAC geçti; aynı `Idempotency-Key` iki row yazdı, negatif DTO HTTP 201 ile `current_headway=-15` ve `delta_cost_kurus=-1` kaydetti.

### 102 regression

- **100 PASS / 2 FAIL**, 57,6 saniye, gerçek temiz PostgreSQL.
- Önceki üç hedefin sonucu: signed tap PASS, recharge 550 TL PASS, anonymous empty batch PASS.
- Assertion zayıflatma: signed tap dynamic sequence değişikliği işlevi koruyor; recharge test fixture'ı kart sahipliği kurup geri alıyor. Buna karşın production frontend'e eklenen otomatik demo login, eski UAT güvenlik kabulünü bozdu.

### Simulator Section 8

- Seed cardinality ve geçerli validator/line/station seçimi: PASS.
- Signed batch sağlıklı fazda kalıcı event yazdı.
- WAL/sequence: FAIL (`permission denied`).
- Process restart + container recreation: queue/sayaçlar sıfır; replay 0.
- Generated/persisted reconciliation: 1.000/270; **loss=730**.
- Duplicate=0, ledger mismatch=0.
- Deterministik manifest/hash: yok; NOT_TESTED/FAIL.

### ESHOT ve harita

- Canlı 800/470/35: 14/5/8 araç; HTTP 200; stale false.
- Cache: 1,289 ms.
- Retry-After seconds/HTTP-date/context-bound: 3/3 PASS; jitter yok.
- Range: 206 + Content-Range + PMTiles magic PASS.
- Gerçek tile decode, MapLibre Map instance, PMTiles protocol ve resmi GTFS ingest: FAIL.

### Cost/financial

- `qa/api/codex_cost_contract.audit.spec.js`: **5 PASS / 2 FAIL**.
- Capacity 50/200 sonuçları farklı.
- Hat yöneticisi approval 201, approver/time ve kalıcı row; yolcu 403. Retry idempotansı ve negatif DTO validation FAIL.
- Integer-money/fixed-point API değişmezi, kaynak etiketleri ve minimum hizmet sınırları: FAIL.
- CORE-04 sırasız finansal determinism: FAIL.

### Go/frontend/contracts

- Go all: PASS.
- Race internal: PASS.
- Vet: PASS.
- Coverage: domain %100, ledger %77,6, pricing %96,8, sync %71,7.
- `git diff --check`: PASS.
- `gofmt`: FAIL (`scripts/gen_pmtiles.go`).
- Frontend production build: PASS.
- Custom lint: temiz tree exit 0 fakat kasıtlı hatalı fixture da exit 0 — FAIL.
- OpenAPI parse: PASS; gerçek route/auth/HMAC/DTO/error coverage: FAIL.
- Axe/keyboard/zoom: **1 PASS / 6 FAIL**. Contrast, label, focusable-scroll, select-name, modal focus transfer ve Escape/focus restore başarısız; 200% zoom yatay taşma kontrolü geçti.
- Canary: **7/7 PASS**.
- `make verify` live: geçici random secret ile SYS-1/2, F-1..F-7, T-1..T-5 **tam PASS**. Backend kapalı aynı hedef exit `2` ile doğru non-zero — PASS.

## Performance

Gate A–F zorunlu kontrolleri PASS olmadığı için 600 saniye ve 1M güvenli load başlatılmadı. Retest 3 için throughput, p50, p95, p99, CPU ve RAM değeri **N/A / NOT_TESTED**. Ayrıntı: `qa/reports/CODEX_RETEST3_PERFORMANCE.md`.

## Git ve release

- Local HEAD: `e74c89c957085638d11215d44dd06a9593ef4724`.
- Remote `origin/main`: `e74c89c957085638d11215d44dd06a9593ef4724`.
- Branch: `main`; worktree dirty.
- Secret scan: FAIL; literal/default validator credential kaynakları var.
- Commit/push: **YAPILMADI**.
- Force: kullanılmadı.
- Yeni GitHub Actions: **N/A**, yeni SHA yok.
- İzole Retest 3 container'ları durduruldu; veri koruma kuralı gereği QA volume'leri silinmedi.

## Artifact dizini

- `security-summary.json`
- `device-security-results.json`
- `prior-security-results.json`
- `bootstrap-runtime.json`
- `simulator-runtime.json`
- `eshot-runtime.json`
- `regression-102.json`, `regression-102.xml`
- `cost-contract.json`
- `approval-runtime.json`
- `accessibility.json`
- `toolchain-gates.json`
- `pmtiles-headers.txt`, `pmtiles-range.bin`

## Kesin sonraki eylem

AGY önce iki P0'ı kapatıp tüm repo credential'larını rotate/revoke etmeli. Ardından simulator volume sahipliği/WAL, CORE-04, frontend otomatik login, gerçek MapLibre/PMTiles/GTFS, integer-money, lint/OpenAPI/WCAG ve gofmt düzeltilmeli. Temiz izole ortamda bu sırayla yeniden test edilmeden commit/push yapılmamalıdır.
