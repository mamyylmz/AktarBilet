# AktarBilet Final Bağımsız QA Denetimi

## Karar

**NOT_READY.** P0 verifier, validatör güvenliği/atomikliği, PostgreSQL fail-open health ve recharge IDOR blokajları açıktır. AGY'nin `READY FOR PRODUCTION`, `49/49`, `14/14 make verify` ve “zero race” ifadeleri release kanıtı olarak kullanılamaz.

## Denetim kimliği

| Alan | Değer |
|---|---|
| Zaman | 2026-10-25 04:56 TRT |
| Branch / HEAD | `main` / `6ac85910e3cc8312b1759827b87342adc4d08b41` |
| `origin/main` | `aceaef8483619f73076c913ba50dfa10f901afd8` |
| Worktree | Yerel branch remote'dan 5 commit ileride; 6 değiştirilmiş production dosyası + untracked `qa/` |
| Değişmiş production | `internal/api/handlers.go`, `handlers_test.go`, `middleware.go`, `internal/seed/seeder.go`, `internal/sync/service.go`, `service_test.go` |
| Ortam | WSL2 Linux 6.6.87.2, Go 1.23.6, Node 24.19.0, npm 11.17.0, Docker 28.3.3 |
| Host araç eksiği | `make` ve `gcc` yok; resmi `golang:1.23` image kullanıldı |
| Kaynak stabilitesi | Denetim başı/sonu altı production dosyası SHA-256 değerleri değişmedi |

## Bulgular — P0 → P3

| ID | Sev. | Sonuç | Özet | Ana konum |
|---|---:|---|---|---|
| CX-P0-01 | P0 | FAIL | Offline verifier 14/14 PASS; live F/T in-memory | `cmd/verify/main.go:48-62,410-416` |
| CX-P0-02 | P0 | FAIL | Tap/event/ledger tek transaction değil; kısmi debit | `internal/sync/service.go:211-280` |
| CX-P0-03 | P0 | FAIL | Prefix veya normal kullanıcı JWT'si validator credential | `internal/api/handlers.go:351-356,415-440` |
| CX-P0-04 | P0 | FAIL | DB yokken in-memory startup ve health 200 | `cmd/server/main.go:47-53`, `internal/api/handlers.go:71-88` |
| CX-P0-05 | P0 | FAIL | Yolcu sahipsiz karta dolum yapabiliyor; test-tailored auth | `internal/api/handlers.go:267-280` |
| CX-P1-01 | P1 | FAIL | Out-of-order aynı olay kümesi farklı bakiye | `internal/sync/service.go:182-209,406-421` |
| CX-P1-02 | P1 | FAIL | Duplicate close/storno idempotent değil | `internal/api/handlers.go:581-600,645-675` |
| CX-P1-03 | P1 | FAIL | T5 anomali yalnız memory, DB satırı 0 | `internal/api/handlers.go:367-375,853-855` |
| CX-P1-04 | P1 | FAIL | Refresh token access olarak geçerli | `internal/auth/auth.go:48-105` |
| CX-P1-05 | P1 | FAIL | Conflicting idempotency payload 200 | `internal/sync/service.go:103-123` |
| CX-P1-06 | P1 | FAIL | Ceza/simülasyon RBAC geniş; F-3 mutation yok | `internal/api/router.go:67-85` |
| CX-P1-07 | P1 | FAIL | UI client-role, embedded credential, sahte success | `web/src/App.jsx:15-23,114-274,557-559` |
| CX-P1-08 | P1 | FAIL | Audit write yok; event direction CHECK yok | `migrations/001_initial_schema.sql:110-126` |
| CX-P1-09 | P1 | FAIL | Para float dönüşümleri; refund `dolum` tipi | `internal/domain/models.go:9-17`, `internal/ledger/ledger.go:468-475` |
| CX-P1-10 | P1 | FAIL | Ana Compose güncel worktree'de build olmuyor | `Dockerfile.frontend:6-10` |
| CX-P2-01..04 | P2 | FAIL | Migration fail-open/destructive, fake report/error leak, simulator ID, test izolasyonu | `qa/reports/CODEX_BUGS.md` |
| CX-P3-01 | P3 | FAIL | OpenAPI/Swagger sözleşmesi yok | Repository envanteri |

Ayrıntılı tekrar üretim ve düzeltmeler: `qa/reports/CODEX_BUGS.md`.

## Go test, race, vet ve coverage

| Kontrol | Sonuç | Bağımsız gerçek |
|---|---|---|
| `go test -count=1 ./internal/...` — DB yok | PASS* | 58 top-level testten 53 PASS, 5 SKIP; `*` live PostgreSQL yolları çalışmadı |
| Aynı komut — izole PostgreSQL | PASS | 58 PASS / 0 FAIL / 0 SKIP |
| `go vet ./...` | PASS | Çıktı yok, exit 0 |
| Coverage | PASS threshold | Toplam 82.6%; bütün core paketler ≥70% |
| `go test -race` — Docker, DB yok | PASS* | Race bulunmadı; 5 live test SKIP |
| `go test -race` — Docker + live DB | **FAIL** | Race raporu yok; `TestSeed_LiveDatabase` `context deadline exceeded` (`internal/seed/seeder_test.go:116-129`) |

58/53 farkı: repository'de 58 top-level `Test...` fonksiyonu vardır. DATABASE_URL olmadan beş live DB testi skip edildiği için 53 başarı görülür; 58 “passing” yalnız DB sağlanan koşuda doğrudur. Subtest ekleyerek şişirilmiş bir 58 sayımı bulunmadı.

### Package coverage

| Paket | Coverage |
|---|---:|
| anomaly | 88.4% |
| api | 78.3% |
| auth | 84.6% |
| config | 100.0% |
| domain | 100.0% |
| ledger | 79.4% |
| pricing | 96.8% |
| seed | 84.1% |
| settlement | 86.2% |
| sync | 88.8% |
| **Toplam** | **82.6%** |

Yüksek riskli boşluklar: API HealthCheck 46.2%, GetCard 53.3%, GetCardLedger 54.2%, BatchSync 57.7%; PostgreSQL `ledger.GetCardLedger` 0%. Coverage artifact: `qa/artifacts/coverage.out`; özet: `qa/artifacts/go-tests.log`.

## Docker ve canlı PostgreSQL

| Kontrol | Sonuç | Kanıt |
|---|---|---|
| İzole QA stack | PASS | Backend `:18080`, frontend `:13000`, PostgreSQL `:15432`; health healthy |
| PostgreSQL sürümü | PASS | 16.15 |
| Seed hacmi | PASS | 200.021 kart, 40 hat |
| Backend restart persistence | PASS | Recharge restart sonrası korundu, retry tek etki |
| DB outage while backend alive | PASS | Health 503; recovery 200; yeni mismatch yok (`qa/artifacts/final-chaos-results.json`) |
| Fresh startup while DB absent | **FAIL** | Backend memory moda geçti ve health 200 |
| Ana Compose tek komut | **FAIL** | Frontend build `node_modules/@vitejs/plugin-react` çakışması; ana runtime kurulamadı |

QA stack ana Dockerfile'ları ve gerçek backend'i kullanır; frontend orkestrasyonunda yalnız healthcheck/COPY izolasyonu için `qa/Dockerfile.frontend.qa` vardır. Bu, ana Compose build başarısızlığını PASS yapmaz.

## Playwright / API / DB / browser sonucu

| Koşu | Sonuç | Açıklama |
|---|---|---|
| AGY öncesi QA paketi | 49/49 raporlanmış | Bağımsız kabul kanıtı değil; IDOR testi sabit `QA-IDOR-001` ile production tailoring'e denk geliyor |
| İlk genişletilmiş clean koşu | **53 PASS / 14 FAIL / 0 SKIP** (67) | `qa/artifacts/playwright.log`, `qa/artifacts/run-summary.json` |
| Final genişletilmiş koşu | **75 PASS / 27 FAIL / 0 SKIP** (102) | 25 ürün assertion'ı + 2 test izolasyon/state failure; exit 1, 96.5 s |
| Ek finansal workflow paketi | **6/6 PASS** | parallel tap, tap+recharge, concurrent duplicate, negative limit, batch retry, restart retry |
| Ek RBAC paketi | **18 PASS / 10 FAIL** | `qa/security/final_rbac.audit.spec.js` |
| Ek T5 persistence | **0 PASS / 1 FAIL** | API anomaly true, iki event, DB anomaly 0 |

27 final failure'ın iki tanesi harness izolasyonudur: live Go testleri beş kalıcı balance/ledger mismatch bırakır; browser recharge testi önceki security tap'lerinden sonra sabit 500 TL başlangıç varsayar. Bunlar ürünün diğer 25 failure'ını açıklamaz veya düşürmez.

Faydalı failure artifacts: `qa/artifacts/playwright/` altındaki trace, screenshot ve video dosyaları.

## PostgreSQL finansal bütünlük

### Doğrulanan başarılı yollar

- 20 paralel tap: 20 event, 20 debit, unique reference, balance=ledger.
- 15 paralel tap + 15 recharge: 30 mutation, balance=ledger.
- 100 concurrent recharge: tek-kuruş kaybı yok.
- 20 concurrent aynı tap: bir event ve bir debit.
- Negatif tolerans contention: bakiye `-5000` altına inmedi.
- Aynı batch iki kez: item başına bir event/bir finansal etki.
- Backend restart sonrası recharge kalıcı ve retry idempotent.
- Ledger UPDATE ve DELETE DB trigger ile reddedildi.

### Blokajlar

- Validator/event akışı tek transaction değil; FK hatası sonrası kısmi finansal etki.
- Conflicting idempotency payload reddedilmiyor.
- Out-of-order sonuç deterministik değil; retroaktif iade/ek ücret yok.
- Duplicate close zamanı değiştiriyor; storno retry iki satır.
- Audit log hiç yazılmıyor.
- Refund PostgreSQL'de `iade` yerine `dolum` olarak yazılıyor.
- Live test fixture'ları mevcut QA DB'sinde 5 `balance != SUM(ledger)` bırakıyor; schema doğrudan balance güncellemesini engellemiyor.

## F-1..F-7 / T-1..T-5 izlenebilirlik

| ID | Durum | Gerçek kanıt / açık |
|---|---|---|
| F-1 | **FAIL P0** | Recharge/persistence çalışıyor; ownership bypass var |
| F-2 | **FAIL P0** | Event/debit canlı; validator auth ve atomicity kırık |
| F-3 | **FAIL P1** | Listeleme var; hat/durak/tarife yönetim mutation route'u yok |
| F-4 | **FAIL P1** | Unit pricing güçlü; live out-of-order determinism başarısız |
| F-5 | **NOT_TESTED P1** | Pure unit paylaştırma var; çok-operatör live DB/API workflow kanıtı yok |
| F-6 | **FAIL P1** | API create persistence var; read RBAC ve UI failure handling kırık |
| F-7 | **FAIL P1** | DB heatmap test edildi; gelir raporu yok, DB hata fallback sahte veri |
| T-1 | **FAIL P1** | 90 dk/boundary/gece/limit/cap unit testleri var; arrival-order canlı test fail |
| T-2 | **FAIL P0** | Batch retry path pass; auth, conflict, transaction, late event fail |
| T-3 | **FAIL P1** | Concurrency/append-only pass; refund tipi, audit ve global invariant açık |
| T-4 | **FAIL P1** | Unit share var; live duplicate close/storno fail, çok-operatör live NOT_TESTED |
| T-5 | **FAIL P1** | Detection memory'de çalışıyor; DB persistence fail; heatmap DB path pass |

İlk/ikinci/üçüncü transfer, ters yön, gece, öğrenci günlük limit, aylık cap ve 95. dakikada pencere sonu Go unit testlerinde vardır. **Tam 90:00 sınırı NOT_TESTED**; ayrıca bu kuralların gerçek API+PostgreSQL zincirinin tümü için live entegrasyon kanıtı yoktur.

## Yedi zorunlu invariant

| # | Durum | Bağımsız kanıt |
|---:|---|---|
| 1. Tüm application tabloları `jw_` | PASS | `public` schema'da non-`jw_` tablo = 0 |
| 2. User tablosu `jw_tf_users` | PASS | `to_regclass` = `jw_tf_users` |
| 3. `X-DDO-Stamp: f85f26` | PASS | 200, 400, 401, 404 response yollarında doğrulandı |
| 4. `GET /_hz/f85f` 200 | PASS | Canlı 200 + stamp |
| 5. İlk seed user e-postası | PASS | `DefaultSeedUsers()[0]` ve DB admin kaydı doğrulandı |
| 6. Main Compose container adı | PASS-static / **BLOCKED-runtime** | `docker-compose.yml:25` doğru; ana build fail olduğu için container runtime oluşmadı |
| 7. README son içerik satırı | PASS | `DDO Case Kodu: SEDIR-f85f26` |

## Kapasite ve performans

Önceki üç sayı farklı ölçümlerdir: anomaly microbenchmark (~27.48M ops/s), 0.63 s read-only `/lines` burst (~3186 RPS), 0.89 s batch event burst (~3389 event/s). Hiçbiri tek başına 1M valid event/saat kanıtı değildir.

Bağımsız temsilî valid-event koşusu:

| Metrik | Sonuç |
|---|---:|
| Süre / event | 37.153 s / 20.000 |
| Throughput | 538.31 valid event/s |
| p50 / p95 / p99 batch latency | 408.61 / 2297.62 / 2440.26 ms |
| HTTP batch error | 0 |
| Unique persisted / status | 20.000 / 20.000 `PROCESSED` |
| Kayıp ledger / test card mismatch | 0 / 0 |
| Rapor sorgusu | 20.049 processed row üzerinde 5.375 ms |

Artifact: `qa/artifacts/event-load-results.json`. Kısa koşu 278 event/s hedefini geçti; **1 saat / 1.000.000 event sustained test NOT_TESTED**. Mevcut load script'i yalnız HTTP 200 + `processed_count` sayar; status/DB unique/ledger kontrolü yapmadığı için tek başına yanıltıcıdır. İlk 20.000-event denemesinde HTTP'nin tamamı kabul edilmesine rağmen olayların tamamı `NEGATIVE_TOLERANCE` idi.

## Git ve CI

- Yerel HEAD: `6ac85910...`; remote main: `aceaef848...`; aynı kaynak değildir.
- Son remote Actions run `40126215225`, remote SHA `aceaef848...`, conclusion `success`.
- CI yerel beş commit ve altı uncommitted production değişikliğini test etmedi.
- `.github/workflows/ci.yml:31-37` PostgreSQL service olmadan Go/race çalıştırır; 5 live test skip olur. Coverage threshold yok. `go run ./cmd/verify` backend olmadan false in-memory PASS.
- CI Playwright/RBAC/security/load/Compose runtime/OpenAPI gate çalıştırmıyor; yalnız ayrı image build'leri var.

## Açık blocker ve sonraki adım

Release yalnız `qa/reports/CODEX_RELEASE_GATE.md` içindeki P0/P1 kabul kriterleri gerçek PostgreSQL ve tarayıcı testleriyle PASS olduğunda yeniden değerlendirilebilir. Production kodu bu denetimde değiştirilmedi.

