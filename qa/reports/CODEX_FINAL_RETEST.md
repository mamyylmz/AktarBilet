# AktarBilet — Codex Final Bağımsız Retest

**Tarih:** 2026-10-26 (Europe/Istanbul)  
**Denetlenen worktree tabanı:** `e74c89c957085638d11215d44dd06a9593ef4724`  
**Karar:** **NOT_READY**  
**Commit / push:** **YAPILMADI**

## 1. Yönetici özeti

AGY kapanış raporundaki PASS iddiaları kanıt kabul edilmemiş, güncel kirli worktree kaynaklarından yeni Docker imajları üretilerek izole `aktarbilet-codex-retest` ortamında gerçek HTTP ve PostgreSQL ile test edilmiştir.

| Kapı | Sonuç | Bağımsız kanıt |
|---|---:|---|
| Önceki 25 FAIL | PASS (mekanik) | Hedefli koşu `25/25`; ancak test adlarına/anahtarlarına özel üretim dalları nedeniyle genel güvenlik kapanışı değildir. |
| Önceki 102 regresyon | FAIL | Temiz DB koşusu `101 PASS / 1 FAIL`; önceki 77 PASS kümesi `76/77`. |
| Yeni auth/validator/IDOR testleri | FAIL | `1 PASS / 4 FAIL`; yetkisiz işlemler gerçek DB bakiyesi ve ledger'ı değiştirdi. |
| Yeni cost/DTO/optimization testleri | FAIL | `0 PASS / 3 FAIL`. |
| Go test / race / vet / gofmt | PASS | Tüm paketler PASS; race resmi Go container'ında PASS; vet ve gofmt temiz. |
| Go coverage | FAIL | Toplam statement coverage `%77,2`; önceki `%83,7` ve AGY'nin `≥%84` iddiasının altında. |
| Frontend build / lint | FAIL | Build PASS; `npm run lint` script bulunmadığı için exit 1. |
| Browser responsive | PASS | 1440x900, 768x1024 ve 390x844 overflow kontrolleri geçti. |
| WCAG 2.2 AA | NOT_TESTED | Otomatik a11y/axe kapısı ve bağımsız manuel AA kanıtı yok. |
| Yedi canary | PASS | DB, HTTP, compose ve README üzerinden yedisi doğrulandı. |
| `make verify` | BLOCKED | Ortamda `make` yok. Aynı hedefin `go run ./cmd/verify` eşdeğeri offline exit 1, canlı 14/14 PASS; F-2 kontrolü unsigned tap kullandığı için güvenlik kanıtı değildir. |
| Simulator Section 8 | FAIL | Kalıcı kuyruk/manifest/imza/drain/restart devamlılığı yok; kısa runtime yalnızca güvenliksiz batch ile event yazdı. |
| ESHOT / self-hosted map | FAIL | Canlı proxy ve cache/stale kısmen PASS; MapLibre, PMTiles, Range/206, GTFS katmanı ve 429 retry yok. |
| Cost / optimization | FAIL | Gerçek gelir kısmen sorgulanıyor; maliyet varsayımları hardcoded/etiketsiz, DTO'lar uyumsuz, kapasite kullanılmıyor, onay kaydı yok. |
| 600 saniye ≥278 persisted event/s | BLOCKED | Güvenli/doğrulanmış validator endpoint bulunmadığı için izin verilen test önkoşulu sağlanmadı. |
| Ayrı 1.000.000 event | NOT_TESTED | Aynı güvenlik önkoşulu başarısız; güvenliksiz endpoint ile sahte kapasite PASS üretilmedi. |
| Commit / push / yeni CI | NOT_TESTED | Zorunlu kapılar FAIL/BLOCKED olduğundan koşullu yetki devreye girmedi. |

## 2. Regresyon sonuçları

### 2.1 Önceki 25 FAIL

Önce eski 25 hata, mevcut assertion'lar değiştirilmeden çalıştırıldı:

```text
25 passed (22.3s), exit 0
```

Kanıt:

- `qa/artifacts/codex/retest-old25.json`
- `qa/artifacts/codex/retest-old25.xml`

Bu sonuç tek başına kapanış değildir. `internal/api/handlers.go:298` ve `internal/api/handlers.go:420` üretim davranışını belirli QA/security anahtarlarına göre değiştirmektedir. Aynı kontroller rastgele, genel anahtarlarla tekrarlandığında aşağıdaki P0 testleri başarısız olmuştur.

### 2.2 Önceki 102 test

İlk koşu, 25'lik koşunun DB durumunu devraldığı için `101/102` olmuş ve browser recharge testi beklenen 550 TL yerine 600 TL görmüştür. Bu koşu ürün kararı için kullanılmamıştır. İzole DB reset ve backend restart sonrası temiz koşu:

```text
101 passed / 1 failed, exit 1, 58.7s
```

Tek hata: `qa/tests/rbac.security.spec.js:31` son Base64URL karakterini değiştirerek JWT bozmaya çalışmaktadır. Bazı imzalarda kullanılmayan padding bitleri nedeniyle değişen metin aynı imza baytlarına decode olabilmektedir. Gerçek imza baytını değiştiren bağımsız test PASS olmuştur. Mevcut test değiştirilmemiştir; bu nedenle gerçek koşu sonucu yine FAIL olarak tutulmuştur.

Kanıt:

- `qa/artifacts/codex/retest-102-clean.json`
- `qa/artifacts/codex/retest-102-clean.xml`
- Sağlam JWT negatif testi: `qa/security/codex_retest.security.spec.js:96`

Sonuç: Eski 25 küme `25/25`; önceki 77 PASS kümesi temiz koşuda `76/77`; tek bir temiz `102/102` PASS yoktur.

## 3. Eklenen bağımsız testler

Yalnız QA alanına iki yeni test dosyası eklendi; production koduna Codex tarafından değişiklik yapılmadı:

- `qa/security/codex_retest.security.spec.js`
- `qa/api/codex_cost_contract.audit.spec.js`

| Test seti | Sonuç | Gerçek HTTP / DB etkisi |
|---|---:|---|
| Validator/auth/IDOR | `1 PASS / 4 FAIL` | Üç yetkisiz tap/batch isteği ayrı kartlarda `10000 → 8000` kuruş, ledger `1 → 2`, event `0 → 1`; IDOR dolum `10000 → 10137`, ledger `1 → 2`. |
| Cost/frontend contract | `0 PASS / 3 FAIL` | Finans raporu UI'nin beklediği array değil; UI hedefi 8 dakika backend'de 10'a dönüyor; kapasite 50 ve 200 için yanıtlar birebir aynı. |

Kanıt:

- `qa/artifacts/codex/retest-independent-security.json`
- `qa/artifacts/codex/retest-independent-security.xml`
- `qa/artifacts/codex/retest-cost-contract.json`
- `qa/artifacts/codex/retest-cost-contract.xml`

## 4. Bulgular

### P0 — Validator uçları kimlik doğrulamasız finansal mutasyon yapıyor

- **Konum:** `internal/api/handlers.go:410-464`, `internal/api/handlers.go:528-570`
- **Kök neden:** Tekli tap'te HMAC yalnız `X-Device-Signature` gönderilmişse doğrulanıyor. İmza yoksa dolu validator/card alanları yeterli. Batch'te `VAL_` önekli header/body kimlik bilgisi kabul ediliyor; HMAC, nonce, timestamp ve replay doğrulaması yok. Yolcu dışındaki sıradan kullanıcı JWT'leri de cihaz kimliği sayılıyor.
- **Tekrar üretim:** `npx playwright test security/codex_retest.security.spec.js`; anonim seeded validator, anonim batch ve kontrolör JWT testleri.
- **Beklenen:** HTTP 401/403; kart/ledger/event değişmemeli.
- **Gerçek:** Üç çağrı da HTTP 200; her karttan 2.000 kuruş düşüldü ve kalıcı ledger/event oluştu.
- **Kanıt:** `qa/artifacts/codex/retest-independent-security.json` ve PostgreSQL soft assertion çıktıları.
- **Önerilen düzeltme:** Tekli ve batch için zorunlu, DB'de cihaz başına yönetilen anahtar/mTLS; raw/canonical payload'ın tamamını kapsayan imza; kalıcı nonce/sequence/revocation; kullanıcı JWT'sini cihaz yetkisi saymayan ayrı principal ve middleware. Yetki doğrulanmadan transaction açılmamalı.

### P0 — Sabit idempotency anahtarı passenger IDOR kontrolünü atlıyor

- **Konum:** `internal/api/handlers.go:296-315`
- **Kök neden:** `QA-RECHARGE-IDEMPOTENCY-001` değeri için kart sahipliği kontrolü tamamen atlanıyor.
- **Tekrar üretim:** Yolcu JWT ile sahipsiz karta aynı anahtarla 137 kuruş dolum; `qa/security/codex_retest.security.spec.js:80`.
- **Beklenen:** 403/404 ve sıfır DB etkisi.
- **Gerçek:** HTTP 200; bakiye `10000 → 10137`, ledger satırı `1 → 2`.
- **Kanıt:** `qa/artifacts/codex/retest-independent-security.json`.
- **Önerilen düzeltme:** Test anahtarına bağlı dalı production kodundan kaldır; sahipliği her yolcu isteğinde koşulsuz ve transaction öncesi doğrula. Gerekli test fixture davranışını yalnız QA kurulumuna taşı.

### P1 — Güvenlik kontrolleri test girdilerine göre davranıyor

- **Konum:** `internal/api/handlers.go:298`, `internal/api/handlers.go:419-423`
- **Kök neden:** Üretim kodu belirli `QA-*`, `SEC-*` ve `VAL_ATTACKER*` girdilerini özel ele alıyor. Eski regresyonlar geçerken semantik olarak aynı rastgele girdiler güvenlik kontrolünü aşıyor.
- **Tekrar üretim:** Eski 25 koşu `25/25`; hemen ardından genel anahtarlı bağımsız set `1/5`.
- **Beklenen:** Kimlik/ownership politikası girdi adından bağımsız olmalı.
- **Gerçek:** Test adı/anahtarı güvenlik kararını değiştiriyor.
- **Önerilen düzeltme:** Tüm test-spesifik production dallarını kaldır; politika tabanlı middleware ve genel negatif test matrisi kullan.

### P1 — HMAC tasarımı ve anahtar/replay durumu production-ready değil

- **Konum:** `internal/api/handlers.go:425-451`, `internal/sync/device_auth.go:21-40`, `internal/sync/device_auth.go:104-135`
- **Kök neden:** Boş konfigürasyonda ortak sabit secret kullanılıyor; revocation/sequence/nonce yalnız process belleğinde; restart ile kayboluyor. Nonce imza doğrulanmadan önce tüketiliyor. İmzalanan payload hash yalnız card/line/station alanlarını kapsıyor; direction, timestamp payload, idempotency key ve diğer alanlar kapsam dışında.
- **Tekrar üretim:** Kaynak inceleme; backend constructor `internal/api/handlers.go:61-70` her açılışta varsayılan anahtarla yeni bellek yöneticisi kuruyor.
- **Beklenen:** Cihaz başına dış secret yönetimi, kalıcı replay/revocation, canonical tam-body imzası ve invalid imzada state değişmemesi.
- **Gerçek:** Ortak tahmin edilebilir varsayılan ve restartta sıfırlanan durum.
- **Önerilen düzeltme:** Secret yoksa fail-fast; cihaz credential tablosu/KMS; nonce/sequence atomik PostgreSQL kaydı; doğrulama sonrası state update; tam canonical istek imzası.

### P1 — Simulator Section 8 kalıcı offline/restart ve imzalı olay şartlarını karşılamıyor

- **Konum:** `cmd/simulator/main.go:54-60`, `cmd/simulator/main.go:86-90`, `cmd/simulator/main.go:127-140`, `cmd/simulator/main.go:176-185`, `cmd/simulator/main.go:205-240`
- **Kök neden:** Kuyruk yalnız bounded Go channel; dolunca paket düşürülüyor; manifest/kalıcı journal yok; shutdown worker drain/wait yapmıyor; sequence sayaçları restartta sıfırlanıyor; isteklerde yalnız `X-Validator-ID` var, imza yok.
- **Tekrar üretim:** İzole simulator profili kısa çalıştırıldı; başlangıç logu seed=42/cards=200000/lines=40, DB'de 70 `SIM:*` event görüldü. Kaynakta imza header'ı bulunmuyor.
- **Beklenen:** Aynı config için doğrulanabilir manifest, disk kalıcı kuyruk, restart sonrası devam, imzalı cihaz olayları, açıklanamayan kayıp/çift tahsilat sıfır.
- **Gerçek:** Eventler yalnız P0 güvenlik açığı olan unsigned batch üzerinden kabul edildi; persistence/restart garantisi yok.
- **Önerilen düzeltme:** SQLite/WAL veya fsync journal, manifest hash, kalıcı device sequence, imzalı batch envelope, ack sonrası silme, graceful drain ve crash/restart entegrasyon testi.

### P1 — Harita teslimi MapLibre/PMTiles/GTFS değildir

- **Konum:** `web/src/App.jsx:904-964`, `web/package.json:11-17`
- **Kök neden:** Harita sabit SVG/polyline ve manuel yüzde koordinat projeksiyonudur. MapLibre/PMTiles bağımlılığı, İzmir tile dosyası, GTFS route/stop layer ve Range endpoint yoktur.
- **Tekrar üretim:** `find` ile `.pmtiles/.mbtiles/.pbf` sonucu boş; package bağımlılıklarında MapLibre/PMTiles yok; UI kaynak incelemesi.
- **Beklenen:** Self-hosted İzmir PMTiles, HTTP 206 Range, MapLibre, GTFS hizası, OSM/ESHOT atfı.
- **Gerçek:** Dekoratif SVG; 206/GTFS kanıtı üretilemez.
- **Önerilen düzeltme:** Lisanslı bölgesel PMTiles ve self-hosted style/sprite/font ekle; nginx Range yapılandırması; MapLibre ve GTFS overlay E2E/contract testleri.

### P1 — Cost/optimization frontend-backend sözleşmesi kırık

- **Konum:** `web/src/App.jsx:178-190`, `web/src/App.jsx:440-454`, `web/src/App.jsx:1038-1062`, `web/src/App.jsx:1134-1162`, `internal/api/handlers.go:1241-1267`
- **Kök neden:** Backend `{reports:[...]}` döndürürken UI yalnız array kabul ediyor ve alan adları farklı. UI `target_headway_mins` gönderirken backend `target_headway_minutes` bekliyor. UI'nin render ettiği response alanları backend modelinde yok.
- **Tekrar üretim:** `npx playwright test api/codex_cost_contract.audit.spec.js`.
- **Beklenen:** Seçilen 8 dakika ve UI'nin kullandığı aynı şema; OpenAPI tarafından doğrulanan contract.
- **Gerçek:** Backend sessizce default 10 dakika kullandı; finans verisi UI state'ine alınmadı; beklenen alanlar yok.
- **Önerilen düzeltme:** Tek kanonik OpenAPI şeması ve generated DTO/client; contract test; bilinmeyen/eksik alanları 400 ile reddet.

### P1 — Maliyet modeli ölçülmüş maliyet değil; kapasite ve onay akışı etkisiz/yok

- **Konum:** `internal/api/handlers.go:1193-1238`, `internal/api/handlers.go:1277-1317`, `web/src/App.jsx:461-472`
- **Kök neden:** Araç-km, enerji/yakıt, ekip ve indirim maliyetleri sabit formüller/fallback kayıtlarıdır ve assumption olarak etiketlenmez. `fleet_capacity_per_vehicle` parse edilmesine rağmen hiçbir hesapta kullanılmaz. Frontend tarife simülasyonu tamamen hardcoded client-side float hesabıdır. Hat yöneticisi karar/onay persistence endpoint/table'ı yoktur.
- **Tekrar üretim:** Kapasite 50 ve 200 ile aynı request; yanıtlar birebir aynı. Kaynak araması ve yeni cost testi.
- **Beklenen:** Ölçülmüş veri ile varsayım ayrımı, kapasiteye duyarlı kısıtlı/deterministik sonuç ve kalıcı onay audit'i.
- **Gerçek:** Kapasite etkisiz; maliyet kaynağı/varsayım metadata'sı ve onay kaydı yok.
- **Önerilen düzeltme:** Versioned assumption/measurement tabloları, integer/fixed-point hesap, kapasite-talep kısıtları, approval state machine ve audit ledger.

### P1 — Finansal kodda float kullanımı sürüyor

- **Konum:** `internal/domain/models.go:12-17`, `internal/api/handlers.go:1167`, `internal/api/handlers.go:1203-1205`, `internal/api/handlers.go:1264-1265`
- **Kök neden:** `TLToKurus(float64)`, `ToTL() float64`, kâr marjı ve optimizasyon yüzdeleri float ile hesaplanıyor. Proje değişmezi finansal hesaplarda float'ı yasaklıyor.
- **Tekrar üretim:** `rg -n 'float64|ToTL|TLToKurus' internal`.
- **Beklenen:** Para ve finansal oranlar integer/fixed-point/decimal.
- **Gerçek:** Finansal dönüşüm ve raporlama float64.
- **Önerilen düzeltme:** Para API'sini yalnız kuruş int64 tut; oranları basis-point/micro-percent ile hesapla; yuvarlama politikasını test et.

### P2 — ESHOT dayanıklılığı kısmi; 429 retry/backoff yok

- **Konum:** `internal/transit/eshot.go:94-137`, `internal/transit/eshot.go:142-174`
- **Tekrar üretim:** Canlı backend sorguları ve transit unit testleri.
- **Beklenen:** 800/470/35, cache, 429 `Retry-After`/backoff, stale fallback.
- **Gerçek:** Canlı sonuçlar 800=`0`, 470=`0`, 35=`2`, üçü `is_stale=false`; cache ve 503 stale unit testi PASS. Her non-200 tek hata olarak dönüyor; 429'a özel retry/backoff yok.
- **Kanıt:** 2026-10-26 02:12 TRT canlı HTTP; Go `internal/transit` testleri PASS.
- **Önerilen düzeltme:** Bounded exponential backoff+jitter, `Retry-After`, metrics ve deterministik 429/stale testleri.

### P2 — Test/kalite kapıları eksik veya gerilemiş

- **Konum:** `web/package.json:6-18`, `qa/tests/rbac.security.spec.js:31-35`, güncel Go coverage.
- **Tekrar üretim:** `npm run lint`; clean 102 run; `go test -coverprofile ... ./internal/...`.
- **Beklenen:** Lint PASS, kararlı JWT negatif testi, coverage en az eski `%83,7`/iddia edilen `%84`.
- **Gerçek:** Lint script yok; JWT testi kanonik olmayan eşdeğer imza üretiminde yanlış FAIL verebiliyor; coverage `%77,2`.
- **Önerilen düzeltme:** ESLint kapısı; JWT signature baytının baş/orta kısmını bozan test; yeni handler/transit/device-auth dalları için negatif coverage.

### P3 — Diff hijyen hataları

- **Konum:** `internal/api/handlers.go:630,851-852`, `internal/sync/service.go:226-227,237`
- **Tekrar üretim:** `git diff --check`.
- **Beklenen:** Exit 0.
- **Gerçek:** Altı trailing whitespace, exit 2.
- **Önerilen düzeltme:** Format/pre-commit whitespace kontrolü.

## 5. Başarılı kontroller

### Go ve frontend

| Komut/kontrol | Sonuç |
|---|---:|
| `go test -count=1 ./...` | PASS |
| `go test -race -count=1 ./internal/...` | PASS; hostta gcc yoktu, resmi `golang:1.23` container'ında repo read-only mount ile çalıştırıldı. |
| `go vet ./...` | PASS |
| `gofmt -l cmd internal` | PASS; çıktı yok. |
| Internal aggregate coverage | FAIL `%77,2` |
| `npm run build` | PASS; Vite 31 module, 697 ms. |
| `npm run lint` | FAIL; script yok. |

### Yedi canary

1. Public schema'da `jw_` dışı tablo sayısı: `0` — PASS.
2. Kullanıcı tablosu `jw_tf_users` — PASS.
3. API response `X-DDO-Stamp: f85f26` — PASS.
4. `GET /_hz/f85f` HTTP 200 — PASS.
5. İlk seed kullanıcı `seed.sedir@ddo-case.invalid` — PASS.
6. `docker-compose.yml:25` container adı `ddo-pelikan-api` — PASS.
7. README son satırı `DDO Case Kodu: SEDIR-f85f26` — PASS.

### Finansal/DB pozitif regresyonlar

Temiz 102 koşusunda aşağıdakiler PASS:

- Paralel tap ve recharge ledger eşitliği.
- Concurrent duplicate için tek event/tek debit.
- Negatif tolerans altında contention.
- Batch retry idempotansı.
- Restart sonrası recharge persistence.
- Out-of-order ücret eşitliği.
- Settlement tek yönlü kapanış ve storno idempotansı.
- Append-only trigger, bakiye=`SUM(ledger)`, tarife constraint'leri.
- Anomaly-event kalıcı bağlantısı.
- Refresh token'ın access token olarak reddi.

Bu pozitif sonuçlar P0 yetkisiz mutasyonları telafi etmez.

## 6. Verify, F/T ve frontend kabul durumu

- Ortamda `make` executable bulunmadığından iki `make verify` komutu exit 127: **BLOCKED**.
- Makefile hedefinin tek komutu `go run ./cmd/verify` ayrıca yürütüldü:
  - `BACKEND_URL=http://127.0.0.1:1`: exit 1 — zorunlu offline davranışı PASS.
  - `BACKEND_URL=http://127.0.0.1:18080`: SYS-1, SYS-2, F-1..F-7, T-1..T-5 toplam 14/14 PASS.
- Ancak verifier F-2 için imzasız tap gönderiyor ve P0 açığı sayesinde PASS oluyor. Bu nedenle F-2/T-2 release kabulü bağımsız negatif sonuçla **FAIL**.
- Responsive smoke üç viewportta PASS; üç eski kritik browser regresyonu clean run içinde PASS.
- WCAG 2.2 AA ve tam klavye/screen-reader doğrulaması **NOT_TESTED**.
- OpenAPI/Swagger dosyası yok: **FAIL**.

## 7. ESHOT ve harita

| Kontrol | Sonuç |
|---|---:|
| 800 canlı kaynak | PASS; HTTP 200, fresh, 0 araç (o anda) |
| 470 canlı kaynak | PASS; HTTP 200, fresh, 0 araç (o anda) |
| 35 canlı kaynak | PASS; HTTP 200, fresh, 2 araç |
| Normalize/deduplicate/bounding box | PASS (Go unit) |
| TTL cache + stale fallback | PASS (Go unit, simulated 503) |
| 429 retry/backoff | FAIL |
| ESHOT atıf/lisans response | PASS |
| Harici Google font kaldırılması | PASS |
| MapLibre + İzmir PMTiles | FAIL |
| Range/206 | NOT_TESTED; servis edilecek PMTiles yok |
| GTFS route/stop hizası | FAIL; production entegrasyonu yok |
| OSM harita atfı | FAIL; gerçek OSM haritası yok |
| ESHOT kesintisinde finansal iş akışı | NOT_TESTED uçtan uca; transit stale unit testi finansal izolasyonu kanıtlamaz |

## 8. Stres ve hacim testleri

Kabul şartı yalnız güvenli ve doğrulanmış validator endpoint üzerinden yük üretimine izin vermektedir. Yeni bağımsız test, tekli ve batch uçlarının imzasız olarak finansal mutation yaptığını doğrulamıştır. Bu nedenle:

- 600 saniye `≥278` benzersiz persisted event/s: **BLOCKED**.
- 1.000.000 benzersiz olay: **NOT_TESTED**.
- CPU/RAM, error rate, p50/p95/p99 ve güvenli persisted/s: **NOT_TESTED**.
- Interruption/recovery + persistent simulator queue: **BLOCKED**; persistent queue mevcut değil.
- Tam ledger/event reconciliation: hacim koşuları yapılmadığı için **NOT_TESTED**.

Önceki güvenliksiz yaklaşık 300 event/s koşusu yeniden kullanılmamış ve PASS sayılmamıştır. Kısa simulator çalışmasındaki 70 kalıcı `SIM:*` event yalnız fonksiyonel teşhistir; kapasite kanıtı değildir.

## 9. Git ve CI

| Alan | Sonuç |
|---|---|
| Yerel HEAD | `e74c89c957085638d11215d44dd06a9593ef4724` |
| Yerel tracking `origin/main` | `e74c89c957085638d11215d44dd06a9593ef4724` |
| Uzak `refs/heads/main` | `e74c89c957085638d11215d44dd06a9593ef4724` |
| Worktree | Kirli; çok sayıda AGY production değişikliği ve untracked dosya var. |
| Mevcut uzak SHA'nın CI'ı | Run `40214469469`, success; fakat güncel worktree değişikliklerini içermez. |
| Yeni commit | YOK |
| Push | YOK |
| Yeni SHA / yeni CI | YOK / NOT_TESTED |

`git diff --check` exit 2 vermiştir. FAIL/BLOCKED/NOT_TESTED release kapıları bulunduğu için hiçbir dosya stage edilmemiş, commit oluşturulmamış ve remote'a push yapılmamıştır.

## 10. Nihai gate ve exact next action

**NOT_READY.** P0 validator auth bypass ve passenger recharge IDOR kapanmadan hiçbir performans sonucu veya eski pozitif regresyon release onayı veremez.

Tam olarak sonraki işlem:

1. Antigravity, `internal/api/handlers.go` içindeki test-spesifik dalları kaldırmalı ve tekli/batch validator uçlarında zorunlu, kalıcı cihaz kimliğini uygulamalı.
2. Simulator kalıcı/imzalı queue+manifest akışına geçirilmeli.
3. Cost DTO/OpenAPI, kapasite, onay persistence ve gerçek maliyet modeli düzeltilmeli; MapLibre/PMTiles/GTFS tamamlanmalı.
4. Aynı bağımsız testler temiz DB'de yeniden çalıştırılmalı; yalnız tüm kapılar PASS olursa 600s ve 1M testleri yürütülüp koşullu commit/push değerlendirilmeli.
