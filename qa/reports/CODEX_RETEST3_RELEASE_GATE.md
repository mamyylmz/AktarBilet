# AktarBilet — Codex Retest 3 Release Gate

**Tarih:** 26 Ekim 2026  
**Nihai karar:** **NOT_READY**  
**Commit/push:** Yapılmadı

## Findings-first karar özeti

| Gate | Durum | Bağımsız kanıt |
|---|---|---|
| A — Security | **FAIL P0** | Repo master impersonation ve eksik alanlı signature fallback gerçek debit üretti. |
| B — Bootstrap/Migration/Approval | **FAIL P1** | Migration/seed/rollback/upgrade PASS; approval persistence PASS, fakat tekrar idempotansı ve DTO sınır doğrulaması implementasyonda yok. |
| C — Simulator Section 8 | **FAIL P1** | WAL permission denied; restart sonrası 730 event kaybı, manifest yok. |
| D — MapLibre/PMTiles/GTFS | **FAIL P1** | HTTP 206 PASS; gerçek MapLibre/PMTiles/GTFS yok, 1.162 bayt sentetik archive. |
| E — Financial/Cost/Transit | **FAIL P1** | Cost paketi 5 PASS / 2 FAIL ve ESHOT canlı/retry PASS; approval idempotansı/validation, float finans DTO'ları ve CORE-04 sırasız ücret FAIL. |
| F — Regression/Toolchain/Frontend | **FAIL P1** | 100/102; go/race/vet/coverage/build PASS; gofmt, lint, OpenAPI, WCAG FAIL. |
| G — Load/Stress | **NOT_TESTED** | A–F PASS olmadığı için politika gereği başlatılmadı. |
| Git release | **PROHIBITED** | P0, credential sızıntısı, 102 FAIL, simulator loss, fake map, lint/OpenAPI/WCAG FAIL. |

## Gate B — Bootstrap ve migration

Gerçek PostgreSQL ölçümleri:

- Fresh bootstrap: migration `001..005`, `jw_schema_migrations=5`, `cards=200005`, `lines=40`, `credentials=4`, distinct credential `4` — **PASS**.
- İkinci backend açılışı: migration/card/user/credential sayıları değişmedi — **PASS**.
- Kasıtlı bozuk 005: backend exit `1`; 005 version satırı `0`, transaction içindeki probe table `0`, önceki 4 version korunmuş — **PASS**.
- Versionsuz eski 001–004 şemadan upgrade: 5 version, 4 credential ve approval table — **PASS**.
- Secret boş bootstrap: backend exit etmek yerine listener açtı — **FAIL**, güvenlik raporunda ele alındı.
- Approval paketi: **5 PASS / 2 FAIL**. Yetkili HTTP 201 + `approved_by`/`approved_at` içeren kalıcı row ve yolcu HTTP 403 geçti. Aynı `Idempotency-Key` iki row oluşturdu; negatif headway/cost HTTP 201 ile kalıcılaştı.
- Approval endpoint'i `Idempotency-Key` veya doğal uniqueness kullanmıyor. Yalnız `line_id` boşluğu kontrol ediliyor; negatif/uygunsuz headway/cost değerleri için sınır doğrulaması yok (`internal/api/handlers.go:1477-1528`) — **FAIL P1**.

Kanıt: `qa/artifacts/codex/retest3/bootstrap-runtime.json`, `qa/artifacts/codex/retest3/cost-contract.json`, `qa/artifacts/codex/retest3/approval-runtime.json`.

## Gate C — Simulator

- Compose'ta named volume mevcut; ancak volume root sahipli, image `USER appuser` (`Dockerfile.simulator:16-23`, `qa/docker-compose.qa.yml:62-72`).
- Runtime log: WAL open `permission denied`; uygulama hatayı yutup memory queue ile devam etti.
- Restart/recreate sonrası `/var/lib/aktarbilet-simulator` boş, telemetry sayaçları ve queue sıfır.
- `1.000 generated`, `270 persisted`; **730 loss**. Duplicate `0`, ledger mismatch `0`.
- Manifest/export/hash komutu yok. `BatchID` `time.Now().UnixNano()` içeriyor (`cmd/simulator/main.go`), aynı seed/config/clock manifesti üretilemiyor.
- Telemetry `batches_sent`, duplicate idempotent 200 yanıtlarını yeni persistence gibi sayabiliyor; per-item ACK sonucu doğrulanmıyor.

Kanıt: `qa/artifacts/codex/retest3/simulator-runtime.json`.

## Gate D — ESHOT/MapLibre/PMTiles/GTFS

Başarılı alt kontroller:

- `/tiles/izmir.pmtiles`: HTTP 206, doğru MIME ve `Content-Range`, `PMTiles` v3 magic.
- ESHOT 800/470/35: 14/5/8 canlı araç, `is_stale=false`; attribution/license mevcut.
- Cache hit ve Retry-After seconds/HTTP-date/context bound PASS.

Release engelleri:

- `web/package.json` içinde `maplibre-gl` ve `pmtiles` dependency yok.
- `web/src/map/MapLibreView.jsx:1-169` MapLibre Map instance kurmuyor; manuel `<canvas>` çiziyor.
- `web/src/map/MapLibreView.jsx:21-53` GTFS yerine hardcoded JSX durakları kullanıyor; `routes.txt`, `stops.txt`, `shapes.txt` yok.
- `scripts/gen_pmtiles.go:52-102` sahte directory metni ve 0..255 byte örnek tile bloğu üretiyor. Dosya yalnız **1.162 bayt**; gerçek tile decode/render yok.
- PMTiles client yalnız header/metadata okuyor, MapLibre protocol kaydetmiyor (`web/src/map/pmtiles.js:1-52`).
- Backoff jitter implementasyonu yok.

Kanıt: `qa/artifacts/codex/retest3/pmtiles-headers.txt`, `pmtiles-range.bin`, `eshot-runtime.json`.

## Gate E — Financial/Cost/Transit

- Cost/optimization API paketi: **5 PASS / 2 FAIL**; capacity 50/200 farklı sonuç, approval persistence/approver/timestamp ve RBAC doğru; tekrar idempotansı ile negatif DTO reddi başarısız.
- Finansal API değişmezi FAIL: `FareImpactResponse` TL alanları `float64`; margin ve headway/demand oranları da float (`internal/api/handlers.go:1183-1213,1253-1265,1301-1303,1353-1370,1429-1444`). Basis-point/fixed-point sözleşmesi yok.
- Maliyet varsayımları kaynak etiketi olmadan hardcoded: roundtrip, demand, günlük araç maliyeti ve ortalama ücret (`internal/api/handlers.go:1402-1444`). Minimum hizmet/personel sınırı yeterli doğrulanmıyor.
- CORE-04 temiz regresyonda FAIL: reverse delivery balance `6000`, chronological `5500`.
- Ledger append-only, concurrency ve reconciliation testleri geçti.

## Gate F — Test ve kalite

| Kontrol | Sonuç |
|---|---|
| Eski 102, temiz PostgreSQL | **100 PASS / 2 FAIL**, 57,6 sn |
| `go test -count=1 ./...` | PASS, resmi Go 1.23 container |
| `go test -race -count=1 ./internal/...` | PASS |
| `go vet ./...` | PASS |
| Çekirdek coverage | domain %100; ledger %77,6; pricing %96,8; sync %71,7 — PASS |
| `git diff --check` | PASS |
| `gofmt` | **FAIL**: `scripts/gen_pmtiles.go` |
| Frontend build | PASS |
| Frontend lint temiz tree | Exit 0 |
| Kasıtlı bozuk lint fixture | **Yanlış exit 0** — lint gate FAIL |
| OpenAPI YAML parse | PASS, 3.0.3 |
| OpenAPI route/contract | **FAIL**: yalnız 9 path; auth/HMAC/errors/DTO ve birçok gerçek route eksik |
| axe/keyboard/zoom WCAG 2.2 AA | **1 PASS / 6 FAIL** |
| `make verify` live/offline | **PASS**: geçici random secret ile live SYS-1/2 + F-1..F-7 + T-1..T-5; backend kapalı exit 2 |

102 FAIL'leri:

1. `qa/db/go_live.db.spec.js:30`: CORE-04 out-of-order final balance uyuşmuyor.
2. `qa/e2e/go_live.uat.spec.js:12`: frontend bundle gömülü credential ile 3 login isteği yaptı.

Axe bulguları:

- `color-contrast`: desktop/mobile/map.
- `label`: login email/password input'ları, critical.
- `scrollable-region-focusable`: mobile tablo alanı.
- `select-name`: map hat seçimi, critical.
- Modal açılışında focus dialog içine taşınmıyor; Escape modalı kapatmıyor ve trigger focus'u geri yüklenmiyor.
- 200% zoom page-level horizontal overflow testi PASS.

Lint fixture kullanılmayan değişken, koşullu hook, labelsız image ve `eval` içerdiği halde geçti; `web/lint.js` yalnız syntax ve iki literal pattern kontrol ediyor.

Kanıt: `qa/artifacts/codex/retest3/regression-102.json`, `.xml`, `accessibility.json`, `toolchain-gates.json`.

## Canary ve F/T

Yedi canary bağımsız kaynak/runtime kontrollerinde **7/7 PASS**:

1. Tablolar `jw_` önekli.
2. `jw_tf_users` mevcut.
3. `X-DDO-Stamp: f85f26` başarı/4xx yanıtlarda doğrulandı.
4. `GET /_hz/f85f` 200.
5. İlk seed kullanıcı `seed.sedir@ddo-case.invalid`.
6. Backend container adı `ddo-pelikan-api`.
7. README son satırı `DDO Case Kodu: SEDIR-f85f26`.

Gerçek Makefile hedefi geçici random secret ile live ortamda SYS-1/2, F-1..F-7 ve T-1..T-5 için tam PASS; backend kapalıyken exit 2 ile PASS. Bununla birlikte F/T bütünü release düzeyinde PASS değildir: 102 paketindeki CORE-04 gerçek sırasız event senaryosu başarısızdır; hızlı verify bu daha güçlü negatif senaryoyu kapsamıyor.

## Git release durumu

- Branch: `main...origin/main`.
- Local HEAD: `e74c89c957085638d11215d44dd06a9593ef4724`.
- Gerçek remote `origin/main`: `e74c89c957085638d11215d44dd06a9593ef4724` (`git ls-remote`).
- Worktree: 27 tracked dosyada **2.997 insertion / 641 deletion** ve çok sayıda untracked AGY/QA dosyası; temiz değil.
- `git diff --check`: PASS.
- Secret scan: deployable validator secret/default birden fazla production/QA/config dosyasında bulundu — FAIL. Değer rapora yazılmadı.
- Commit: **YOK**.
- Push: **YOK**; force kullanılmadı.
- Yeni remote SHA: **YOK**.
- Yeni GitHub Actions: **N/A**, push yapılmadı.
- İzole Retest 3 container'ları durduruldu. Veri koruma kuralı gereği QA volume'leri silinmedi.

## Kesin sonraki eylem

AGY önce P0 credential modelini ve signature binding'i düzeltmeli; sonra validator assignment/idempotency, simulator volume/WAL, CORE-04, gerçek harita zinciri, frontend credential/lint/a11y/OpenAPI ve integer-money sorunlarını gidermeli. Ardından Docker motoru açık temiz ortamda Retest 3 güvenlik → 102 → Makefile → simulator → yalnız tümü PASS ise 600s/1M sırasıyla yeniden çalıştırılmalıdır.
