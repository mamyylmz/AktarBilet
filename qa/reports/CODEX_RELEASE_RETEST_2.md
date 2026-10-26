# AktarBilet — Security-First Bağımsız Release Retest 2

**Tarih:** 2026-10-26 (Europe/Istanbul)  
**Denetçi:** Codex — bağımsız SDET / güvenlik / backend QA  
**Denetlenen taban:** `e74c89c957085638d11215d44dd06a9593ef4724`  
**Ortam:** İzole Docker Compose projesi `aktarbilet-codex-retest`, PostgreSQL 16  
**Nihai karar:** **NOT_READY**  
**Commit / push:** **YAPILMADI**

## 1. Bulgular — P0 → P3

### P0-1 — Repository'de gömülü validator credential gerçek finansal işlem yetkisi veriyor

- **Konum:** `migrations/005_device_auth_and_optimizations.sql:34-42`, `internal/sync/device_auth.go:120-133`, `internal/api/handlers.go:69`
- **Kök neden:** Dört validator anahtarı migrasyonda düz metin ve deterministik olarak bulunuyor. `NewAPIHandler`, yüklenmiş `Config.ValidatorHMACSecret` yerine `NewDeviceAuthManager("", pool)` çağırıyor; bu da repository'deki sabit QA master secret'a düşüyor. Doğrulayıcı ayrıca stored key, master-derived key ve doğrudan master key olmak üzere üç ayrı anahtar biçimini kabul ediyor.
- **Tekrar üretim:** `qa/security/codex_retest.security.spec.js` içindeki `a repository-embedded validator credential cannot authorize an external caller` testi, migrasyonda bulunan `VAL_500T_01` credential'ı ile raw body'yi imzaladı.
- **Beklenen:** HTTP 401/403; kart bakiyesi, ledger ve event değişmemeli.
- **Gerçek:** HTTP **200**; bakiye `10000 → 7000`, ledger `1 → 2`, validator event `0 → 1`.
- **Kanıt:** `qa/artifacts/codex/retest2-exposed-device-active.json`, `.xml`; PostgreSQL olayı `CODEX_EXPOSED_*`, ücret `3000` kuruş.
- **Önerilen düzeltme:** Credential'ları migration/repository'den kaldır; cihaz başına dış secret provisioning/KMS kullan; secret yoksa bütün ortamlarda fail-fast; handler'a config'teki secret'ı açıkça geçir; geçiş amaçlı fallback anahtarlarını kaldır ve sızmış anahtarları döndür.

### P0-2 — Batch imzalayan validator başka validator adına olay yazabiliyor

- **Konum:** `internal/api/handlers.go:574-617`
- **Kök neden:** İmza `X-Validator-ID` principal'ı (`valHeader`) üzerinden doğrulanıyor; fakat batch içindeki her `event.validator_id` değerinin aynı principal'a eşit olduğu doğrulanmıyor.
- **Tekrar üretim:** `VAL_500T_01` credential'ı ile imzalanan batch içine `validator_id=VAL_14ES_01` olayı kondu.
- **Beklenen:** HTTP 401/403 ve sıfır DB etkisi.
- **Gerçek:** HTTP **200**; PostgreSQL'e `VAL_14ES_01` adına olay/ledger yazıldı, kart `10000 → 7000` oldu.
- **Kanıt:** `qa/artifacts/codex/retest2-exposed-device-active.json`, `.xml`; event anahtarı `CODEX-CONFUSED-BATCH-*`.
- **Önerilen düzeltme:** Header principal'ı ile batch içindeki bütün validator kimliklerini sabit-zaman gerektirmeyen normal eşitlik kontrolüyle birebir bağla; karışık-validator batch'i reddet; bu kontrolü imza doğrulaması ile aynı authorization transaction sınırında yap.

### P1-1 — Boş PostgreSQL bootstrap'ta 005 migrasyonu rollback oluyor, servis yine healthy kalıyor

- **Konum:** `cmd/server/main.go:143-198`, `migrations/005_device_auth_and_optimizations.sql:34-42`
- **Kök neden:** 005, `jw_validators` seed edilmeden önce FK'li credential satırlarını ekliyor. Tek `Exec` içindeki migration rollback oluyor. Migration hataları yalnız WARN loglanıyor ve üst çağrıya döndürülmüyor; seed ve HTTP sunucu devam ediyor.
- **Tekrar üretim:** Yalnız `aktarbilet-codex-retest` volume'ü silindi; güncel kaynak yeniden build edilerek boş PostgreSQL başlatıldı.
- **Beklenen:** Tüm migration'lar atomik PASS veya process non-zero/fail-unhealthy.
- **Gerçek:** `SQLSTATE 23503`; `jw_validator_credentials` ve `jw_optimization_approvals` yokken backend healthy oldu. `/app/server -reset` ikinci geçişi, seed artık mevcut olduğu için 005'i sonradan başarılı göstererek kusuru maskeliyor.
- **Kanıt:** Bootstrap logu: `Migrasyon 5 çalıştırma uyarısı ... jw_validator_credentials_validator_id_fkey`; `to_regclass(...)` iki tablo için NULL.
- **Önerilen düzeltme:** Şema DDL ile seed credential verisini ayır; validator seed'inden sonra credential provisioning yap; migration hatasında process'i durdur; uygulanan migration'ları version tablosunda transaction ile takip et.

### P1-2 — Optimization approval şeması ile INSERT sözleşmesi uyuşmuyor; kalıcı onay çalışmıyor

- **Konum:** `migrations/005_device_auth_and_optimizations.sql:20-31`, `internal/api/handlers.go:1483-1497`
- **Kök neden:** Tablo `scenario_name`, `fleet_capacity_per_vehicle`, `proposed_vehicles` zorunlu kolonlarını tanımlıyor. Handler bunları sağlamadan tabloda bulunmayan `fleet_capacity`, `vehicle_delta`, `projected_net_margin_kurus`, `status`, `notes`, `approved_at` kolonlarına INSERT yapıyor.
- **Tekrar üretim:** Yetkili `hat_yoneticisi` ile `POST /api/v1/optimization/approve`, ardından gerçek DB row kontrolü.
- **Beklenen:** HTTP 201 ve kalıcı, kullanıcıya bağlı audit row.
- **Gerçek:** HTTP **500**, row sayısı değişmedi. Yolcu negatif kontrolü 403 ve sıfır row ile PASS.
- **Kanıt:** `qa/artifacts/codex/retest2-cost.json`, `.xml`: `4 PASS / 1 FAIL`.
- **Önerilen düzeltme:** Tek bir versioned approval DTO/DDL sözleşmesi oluştur; migration upgrade'i ekle; handler ve OpenAPI contract testini aynı kolon/alan isimlerine bağla.

### P1-3 — Simulator WAL restart/reconnect ve container kalıcılığı release şartını karşılamıyor

- **Konum:** `cmd/simulator/main.go:118-155`, `:158-170`, `:196-215`, `:341-364`; `docker-compose.yml:66-78`; `qa/docker-compose.qa.yml:61-70`
- **Kök nedenler:**
  - WAL pending kayıtları `map` üzerinden rastgele sırayla replay ediliyor; monotonik sequence kontrolü daha eski paketleri 409'a düşürüyor.
  - Failed batch yalnız başlangıç replay'inde yeniden deneniyor; backend reconnect sonrası periyodik replay yok.
  - WAL ve sequence `/tmp` altında; Compose'ta simulator volume yok.
  - Sequence state `os.WriteFile` ile atomik rename/fsync olmadan ve yalnız her 10 tick/graceful shutdown'da yazılıyor.
  - Queue dolduğunda channel enqueue başarısız oluyor; WAL kalsa da runtime retry scheduler yok.
  - Deterministik manifest/hash komutu veya çıktısı yok.
- **Runtime kanıtı:**
  - İmzalı kısa koşu: `50 generated`, `5 batch sent`, WAL 10 satır.
  - Backend offline: `300 generated`, `15 sent`, `4 failed`, queue `8`, WAL 45 satır.
  - Simulator restart + reconnect: `7 batch replayed`, fakat `17 failed`; WAL ayrıştırmasında **25 pending batch / 250 pending event** kaldı.
  - Backend healthy iken pending kayıtlar otomatik kapanmadı.
  - `--force-recreate simulator` sonrası önceki WAL/sequence tamamen kayboldu; yeni container yalnız 5 yeni WAL satırı ve sequence dosyası olmadan başladı.
- **Kısa koşu DB sonucu:** Durdurma anında `1871` kalıcı `SIM:*` event; `(validator_id,sequence_no)` distinct `1871`, idempotency distinct `1871`; duplicate `0`; durum `1869 NEGATIVE_TOLERANCE / 2 REJECTED`; global kart/ledger mismatch `0`.
- **Yorum:** Duplicate ve ledger reconciliation kısa koşuda temizdir; fakat 250 pending event'in container recreation ile kaybolması loss=0 şartını ihlal eder. Bu kısa fonksiyonel koşu performans kanıtı değildir.
- **Önerilen düzeltme:** Named volume, atomik sequence snapshot, validator+sequence sıralı replay, sürekli retry/DLQ scheduler, item-level result doğrulama, WAL compaction ve deterministic manifest ekle; offline→process restart→container recreation→reconnect entegrasyon testi yaz.

### P1-4 — Temiz state 102 regresyonu 99/102

- **Koşu:** Eski 102 test dosyası, resetlenmiş gerçek PostgreSQL üzerinde ve yeni bağımsız testler hariç çalıştırıldı.
- **Sonuç:** **99 PASS / 3 FAIL**, `70.6s`, exit 1.
- **Hatalar:**
  1. `qa/tests/api.integration.spec.js:90`: signed tap beklenen 200, gerçek **409**. Önceki DB testleri aynı validator credential sequence'ını büyük değerlere ilerletiyor; test sabit `900001` kullanıyor.
  2. `qa/tests/browser.uat.spec.js:79`: başarılı recharge UI'da `550.00 ₺` yerine `500.00 ₺` kaldı.
  3. `qa/tests/rbac.security.spec.js:22`: anonim boş batch beklenen 401/403 yerine **400**; handler body'yi authorization'dan önce parse ediyor (`internal/api/handlers.go:562-570`).
- **Kanıt:** `qa/artifacts/codex/retest2-102-clean.json`, `.xml`.
- **Önerilen düzeltme:** Testler arası ortak device sequence bağımlılığını kaldır veya DB'den atomik next sequence kullan; demo UI'yi gerçek login/session sözleşmesine bağla; batch authentication'ı body parse öncesi uygula.

### P1-5 — MapLibre/PMTiles/GTFS teslimi gerçek değil; 206 sonucu HTML fallback

- **Konum:** `web/package.json:12-18`, `web/src/map/MapLibreView.jsx:1-159`, `web/src/map/pmtiles.js:10-46`, `Dockerfile.frontend:6-12`, `qa/Dockerfile.frontend.qa:5-10`
- **Kök neden:** MapLibre ve PMTiles dependency/import/instance yok. `MapLibreView` manuel `<canvas>` çiziyor. “GTFS” durakları JSX içinde hardcoded. Her iki frontend Dockerfile yalnız `web/src` kopyalıyor; `web/public/tiles/izmir.pmtiles` image'a girmiyor.
- **Tekrar üretim:** `Range: bytes=0-126` ile `/tiles/izmir.pmtiles` istendi.
- **Beklenen:** 206, PMTiles MIME/body ve `PMTiles` magic.
- **Gerçek:** HTTP 206 fakat `Content-Type: text/html`, `Content-Range: bytes 0-126/604`; body SPA `index.html`. Kaynaktaki `.pmtiles` dosyası yalnız 1162 bayt.
- **Ek kusur:** İstemci 200'ü de kabul ediyor ve yalnız header okuyup gerçek tile decode/render yapmıyor.
- **Önerilen düzeltme:** Resmî `maplibre-gl` + `pmtiles` istemcisi, gerçek lisanslı İzmir archive/style/sprite/font, Docker public copy, doğru MIME/cache/range ve gerçek GTFS feed pipeline ekle; magic/content-type E2E testi yaz.

### P1-6 — Integer-money değişmezi karşılanmıyor

- **Konum:** `internal/domain/models.go:12-17`, `internal/api/handlers.go:1154-1205`, `:1272`, `:1399-1415`
- **Kök neden:** `TLToKurus(float64)` ve `ToTL() float64` para dönüşümü yapıyor; fare-impact gelir DTO'su TL cinsinden float; projected revenue ve incremental passenger/revenue yollarında float kullanılıyor.
- **Beklenen:** Tüm para ve finansal sonuçlar kuruş `int64`; oranlar fixed-point/basis-point.
- **Gerçek:** Persisted money kolonları BIGINT olsa da API/hesap yolu float içeriyor.
- **Önerilen düzeltme:** Float para DTO ve dönüşümlerini kaldır; kuruş/basis-point kullan; açık yuvarlama politikası ve sınır testleri ekle.

### P1-7 — `Retry-After` saniyesi 100 ms olarak uygulanıyor

- **Konum:** `internal/transit/eshot.go:180-198`
- **Kök neden:** `Retry-After` integer değeri saniye olmasına rağmen `100 * time.Millisecond` ile çarpılıyor ve 1 saniyede cap ediliyor.
- **Tekrar üretim:** QA-only `qa/go/transit_rate_limit.audit_test.go`, ilk cevap `429 Retry-After: 1`, sonra 200.
- **Beklenen:** En az yaklaşık 1 saniye bekleme.
- **Gerçek:** Retry **101.38 ms** sonra yapıldı; test FAIL.
- **Önerilen düzeltme:** Delta-seconds için `time.Second`, HTTP-date desteği, bounded exponential backoff+jitter ve elapsed-time assertion kullan.

### P1-8 — Validator sequence tasarımı normal test/verify akışını kırıyor

- **Konum:** `internal/sync/device_auth.go:146-178`, `cmd/verify/main.go:185-221`, `qa/tests/api.integration.spec.js:90-104`
- **Kök neden:** Tek global validator sequence monotonik tutuluyor; verifier/testler sabit veya modulo sequence üretiyor. Daha yüksek sequence'lı başka test/işlem sonrası meşru istekler kalıcı 409 alıyor.
- **Gerçek:** `go run ./cmd/verify` sonucu F-2 FAIL; T-2 üç çağrı da 409. Aynı kök neden clean 102'de signed tap'i de düşürdü.
- **Önerilen düzeltme:** Cihazın durable next-sequence state'ini kullanan istemci sözleşmesi oluştur; testlerde ayrılmış validator fixture veya DB'den güvenli next value kullan; idempotent retry ile out-of-order event semantiğini birbirinden ayır.

### P2-1 — Kalite ve sözleşme kapıları eksik/yanıltıcı

- `web/package.json:9` lint bir linter çalıştırmıyor; yalnız “passed” metni basıyor. Sonuç release açısından **FAIL**.
- OpenAPI/Swagger belgesi bulunamadı: **FAIL**.
- axe/WCAG 2.2 AA otomasyon kapısı ve bağımsız manuel kanıt yok: **NOT_TESTED**.
- Ortamda `make` yok; `make verify` exit 127: **BLOCKED**. Makefile hedefinin eşdeğeri `go run ./cmd/verify` ayrıca çalıştırıldı ve F-2/T-2 nedeniyle exit 1.
- Tam `pnpm audit`: `2 high / 6 moderate / 1 low` dev dependency açığı; production-only audit `0` bulgu.
- Secret taraması gerçek private-key/cloud-token bulmadı; ancak deployable varsayılan JWT/HMAC secret'ları, dört validator credential'ı ve frontend bundle kaynaklarında master secret bulundu. Security gate **FAIL**.

## 2. Güvenlik retest matrisi

| Kontrol | Sonuç | Kanıt |
|---|---:|---|
| Anonim registered validator tap | PASS | 401/403, DB etkisi yok |
| Anonim batch | PASS | 401/403, DB etkisi yok |
| Normal kullanıcı JWT ile tap | PASS | 401/403, DB etkisi yok |
| Bilinen QA idempotency key ile IDOR | PASS | 403/404, DB etkisi yok |
| Rastgele idempotency key ile IDOR | PASS | 403/404, DB etkisi yok |
| JWT gerçek signature byte XOR | PASS | 401; Base64URL kanonik false-negative giderildi |
| Gömülü validator credential ile dış çağrı | **FAIL** | HTTP 200 ve finansal mutasyon |
| Batch principal / event validator eşleşmesi | **FAIL** | Başka validator adına HTTP 200 |
| Test adına/özel anahtara bağlı production branch statik tarama | PASS | Eski `QA-RECHARGE`, `SEC-*`, `VAL_ATTACKER`, `HasPrefix` istisnaları bulunmadı |

İlk altı testin artefaktı: `qa/artifacts/codex/retest2-security-fresh.json/.xml` (`6/6 PASS`). İki kritik negatif test, 005 tablosunun ikinci migration geçişiyle gerçekten aktif olduğu durumda çalıştırıldı; artefakt `retest2-exposed-device-active.json/.xml` (`0/2`, beklenmeyen HTTP 200).

## 3. Fonksiyonel ve kalite sonuçları

| Kapı | Sonuç | Ölçüm |
|---|---:|---|
| Temiz state eski 102 | **FAIL** | `99/102`, 3 hata |
| Cost DTO + capacity | PASS | 3/3 |
| Approval RBAC | PASS | yolcu 403, row yok |
| Approval persistence | **FAIL** | yetkili çağrı HTTP 500 |
| Go internal testleri | PASS | tüm `internal/...` paketleri |
| Güncel `go test -count=1 ./...` | **FAIL** | production paketleri PASS; QA 429 audit FAIL |
| Go race | PASS | `golang:1.23`, read-only repo mount, tüm `internal/...` |
| `go vet ./...` | PASS | çıktı yok |
| `gofmt -l cmd internal` | PASS | çıktı yok |
| Go coverage | PASS/ölçüm | toplam `%74.6`; `internal/domain` `%100` |
| QA 429 audit | **FAIL** | 101.38 ms; beklenen ≥900 ms |
| Frontend build | PASS | 33 modules; bundle 194.29 kB, gzip 59.07 kB |
| Frontend lint | **FAIL** | gerçek linter yok |
| OpenAPI | **FAIL** | belge yok |
| WCAG 2.2 AA | **NOT_TESTED** | axe/manual AA kanıtı yok |
| Canary | PASS | 7/7 |
| `make verify` | **BLOCKED/FAIL** | make yok; eşdeğer live verify F-2/T-2 FAIL |
| `git diff --check` | PASS | exit 0 |

## 4. Canary sonuçları

1. Public schema'da `jw_` dışı tablo: `0` — PASS.
2. Kullanıcı tablosu `jw_tf_users` — PASS.
3. `/health` yanıtı `X-DDO-Stamp: f85f26` — PASS.
4. `GET /_hz/f85f` HTTP 200 — PASS.
5. İlk seed kullanıcı `seed.sedir@ddo-case.invalid` — PASS.
6. Compose backend container adı `ddo-pelikan-api` — PASS.
7. README son satırı `DDO Case Kodu: SEDIR-f85f26` — PASS.

## 5. Performans kapısı ve zorunlu ölçümler

P0/P1 ve zorunlu fonksiyon kapıları PASS olmadığı için talimat gereği 600 saniyelik ve ayrı 1.000.000 event güvenli yük testlerine **GEÇİLMEDİ**.

| Ölçüm | Durum |
|---|---|
| Güvenli persisted events/sec (600 s) | **NOT_TESTED — gate blocked** |
| p50 / p95 / p99 | **NOT_TESTED — gate blocked** |
| CPU / RAM | **NOT_TESTED — gate blocked** |
| 1M duplicate / loss | **NOT_TESTED — gate blocked** |
| 1M ledger reconciliation | **NOT_TESTED — gate blocked** |
| Kısa simulator duplicate | `0` (`1871/1871/1871`) |
| Kısa simulator ledger mismatch | `0` |
| Restart/recreation queue loss | **FAIL: 250 pending event WAL ile birlikte kayboldu** |

Kısa Section 8 çalışması yalnız fonksiyonel teşhistir; throughput/latency kapasite sonucu olarak kullanılmamıştır.

## 6. Git, secret ve CI durumu

- Yerel `HEAD`: `e74c89c957085638d11215d44dd06a9593ef4724`
- Yerel `origin/main`: `e74c89c957085638d11215d44dd06a9593ef4724`
- Branch: `main...origin/main`
- Worktree: çok sayıda AGY production değişikliği ve untracked dosya ile kirli; Codex yalnız QA testleri/raporu değiştirdi.
- Yeni commit: **YOK**.
- Push: **YOK**; force kullanılmadı.
- Yeni remote SHA / GitHub Actions: **NOT_APPLICABLE**; release koşulu sağlanmadığı için push yetkisi devreye girmedi.
- Secret sonucu: Cloud/private-key paterni bulunmadı; fakat repository/deployable kaynakta validator/JWT/HMAC varsayılanları bulunduğu için PASS verilmedi.

## 7. Karar ve exact next action

**NOT_READY.** İki P0 cihaz yetkilendirme açığı, temiz bootstrap migration kusuru, approval persistence hatası, 99/102 regresyon, kalıcı simulator queue kaybı ve sahte harita/lint kapıları release'i engelliyor.

Tam sonraki işlem:

1. Antigravity P0 credential ve principal-binding açıklarını kapatmalı; tüm sızmış validator anahtarlarını döndürmeli.
2. 005 migration boş DB'de fail-fast ve tek geçişte uygulanabilir hale getirilmeli; approval DDL/handler sözleşmesi eşitlenmeli.
3. Simulator WAL volume+sıralı replay+continuous retry+manifest ve MapLibre/PMTiles/GTFS gerçek entegrasyonu tamamlanmalı.
4. Integer-money, Retry-After, gerçek lint/OpenAPI/WCAG kapıları ve üç regresyon düzeltilmeli.
5. Aynı temiz-DB bağımsız retest yeniden çalıştırılmalı. Yalnız tüm zorunlu kapılar PASS olursa 600 s ve 1M testleri, ardından commit/push ve yeni SHA GitHub Actions doğrulaması yapılmalı.
