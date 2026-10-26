# AKTARBİLET — CODEX RETEST FAILURE REMEDIATION REPORT

**Tarih:** 26 Ekim 2026, 03:00 TRT  
**Rol:** Antigravity (AGY) — Ana Geliştirici (Primary Developer)  
**Denetleyen:** Codex — Bağımsız İnceleme ve Test Ajanı  
**Kaynak:** `qa/reports/CODEX_FINAL_RETEST.md`, `qa/security/codex_retest.security.spec.js`, `qa/api/codex_cost_contract.audit.spec.js`  
**Durum:** **ALL FINDINGS REMEDIATED & VERIFIED** (P0, P1, P2, P3 Tamamlandı)  

---

## 1. Yürütme Özeti (Executive Summary)

Codex tarafından gerçekleştirilen 26.10.2026 tarihli bağımsız denetimde tespit edilen P0, P1, P2 ve P3 seviyesindeki tüm mimari, güvenlik, simülatör, harita, maliyet ve kalite kapısı eksiklikleri kök nedenleriyle incelenmiş ve doğrudan üretim kodunda (Production Code) kalıcı olarak düzeltilmiştir.

Sistemde hiçbir test-spesifik istisna (`QA-*`, `SEC-*`, `VAL_ATTACKER*`), mock hilesi veya gevşetilmiş assertion bırakılmamıştır. Tüm 7 sistem canarisi eksiksiz korunmuş, kuruş bazlı tam sayı finansal bütünlük sağlanmış ve `git diff --check` sıfır hata ile doğrulanmıştır.

---

## 2. Kapsamlı Düzeltme Matrisi (Remediation Matrix)

### Aşama 1: P0 Security — Validatör Cihaz Kimlik Doğrulaması & Anti-IDOR

| Madde | Kök Neden | Uygulanan Düzeltme (Production Code) | Doğrulama & Kanıt |
|---|---|---|---|
| **P0-1: Validatör tekli tap kimliksiz finansal mutasyon** | `X-Device-Signature` gönderilmediğinde dolu validator/card alanlarıyla anonim işlem yapılmasına izin veriliyordu. Test anahtarlarına bağlı `if strings.HasPrefix(...)` dalları mevcuttu. | `internal/api/handlers.go:400-475` içerisindeki tüm test-spesifik string dalları kaldırıldı. Kimlik doğrulama, gövde şeması kontrolünden önce koşulsuz zorunlu kılındı. Yalnızca geçerli HMAC-SHA256 cihaz imzasına sahip terminaller veya Sistem Yöneticisi (`domain.RoleAdmin`) oturumu kabul edilmektedir. Anonim ve sıradan kullanıcı JWT'leri (yolcu, kontrolör vb.) doğrudan HTTP 401/403 ile reddedilir. Sıfır DB mutasyonu sağlanmıştır. | `qa/security/codex_retest.security.spec.js:22` PASS (`a registered validator ID is not an anonymous device credential`). `qa/tests/rbac.security.spec.js:12` PASS. |
| **P0-2: Batch-sync kimliksiz ingestion** | `X-Validator-ID: VAL_*` başlığıyla gelen anonim isteklerin şifreleme ve imza olmadan işleme alınması. | `internal/api/handlers.go:555-635` baştan yazıldı. Toplu biniş paketlerinde `X-Device-Signature`, `X-Device-Timestamp`, `X-Device-Nonce` ve `X-Validator-ID` başlıkları zorunlu kılındı. Payload hash'i (`sync.ComputePayloadHash`) doğrulanmadan hiçbir işlem kuyruğa alınmaz. | `qa/security/codex_retest.security.spec.js:41` PASS (`the anonymous batch endpoint requires cryptographic device authentication`). |
| **P0-3: Kullanıcı JWT'sinin validatör yetkisi sayılması** | Sıradan JWT'lerin (kontrolör, muhasebe) turnike binişinde yetkili principal kabul edilmesi. | `internal/api/handlers.go:415, 580` üzerinde kullanıcı JWT'si kontrolü yapıldı. Yalnızca `domain.RoleAdmin` rolüne sahip yönetici token'ları manuel idari test amaçlı kabul edilir; kontrolör, yolcu ve muhasebe rolleri HTTP 403 Forbidden ile engellenir. | `qa/security/codex_retest.security.spec.js:60` PASS (`a non-device user JWT cannot authorize validator ingestion`). |
| **P0-4: Sabit idempotency anahtarıyla IDOR kontrolünün atlanması** | `internal/api/handlers.go:298` satırındaki `if idempKey != "QA-RECHARGE-IDEMPOTENCY-001"` bypass'ı. | İstisna koşulsuz silindi. Yolcu rolündeki kullanıcıların kart dolumlarında `jw_cards.user_id == claims.UserID` kontrolü hiçbir anahtara ayrıcalık tanımaksızın mutlak olarak uygulanır. | `qa/security/codex_retest.security.spec.js:80` PASS (`an idempotency key cannot bypass passenger card ownership`). Sıfır bakiye ve ledger etkisi. |
| **P0-5: Replay, nonce, sequence ve revocation'ın kalıcı olmaması** | Durumun process belleğinde tutulması ve sunucu yeniden başladığında sıfırlanması. | `migrations/005_device_auth_and_optimizations.sql` ile `jw_validator_credentials` ve `jw_device_nonces` tabloları eklendi. `internal/sync/device_auth.go` tamamen PostgreSQL destekli atomik transaction (`FOR UPDATE`, sequence monotonicity, tekil nonce kaydı) yapısına geçirildi. İptal edilen cihazlar (`is_revoked=true`) veritabanı düzeyinde engellenir. | `internal/sync/device_auth_test.go` PASS. `go test -count=1 ./internal/sync` PASS. |

---

### Aşama 2: P1 Simulator — Section 8 Kalıcı Offline & İmzalı Olay Mimarisi

- **Konum:** `cmd/simulator/main.go`
- **Kök Neden:** Go channel kuyruğunun bellekte kalması, taşmada paket düşürme, crash sonrası sequence sıfırlanması ve imzasız batch çağrıları.
- **Uygulanan Düzeltmeler:**
  1. **Disk-Backed WAL Journal:** `/tmp/simulator_wal.jsonl` (veya `SIMULATOR_WAL_PATH`) üzerinde fsync destekli kalıcı olay günlüğü kuruldu. Olaylar belleğe alınmadan önce diske yazılır (`PENDING`).
  2. **Crash/Restart Replay:** Simülatör açılışında WAL taranarak sunucu tarafından henüz ACK alınmamış paketler (`PENDING`) otomatik olarak yeniden sunucuya aktarılır ve ACK işaretlenir.
  3. **Kalıcı Monotonik Sequence Sayaçları:** `/tmp/simulator_seq.json` üzerinde validatör sıra numaraları atomik saklanır; simülatör yeniden başladığında sequence sayaçları sıfırlanmaz, kaldığı yerden artarak devam eder.
  4. **Kriptografik Cihaz İmzası (HMAC Envelope):** Üretilen her batch paketi `internal/sync.DeriveDeviceSecret` ile validatöre özel anahtar türetilerek HMAC-SHA256 ile imzalanır (`X-Device-Signature`, `X-Device-Timestamp`, `X-Device-Nonce`, `X-Validator-ID`).
  5. **Graceful Worker Drain:** `SIGINT`/`SIGTERM` sinyallerinde kanal kapatılır, `sync.WaitGroup` ile tüm işçilerin kuyruktaki paketleri tamamlaması ve sayaçların diske kaydedilmesi beklenir.

---

### Aşama 3: P1 Harita — MapLibre, PMTiles v3 ve GTFS Hizalaması

- **Konum:** `web/public/tiles/izmir.pmtiles`, `web/src/map/pmtiles.js`, `web/src/map/MapLibreView.jsx`, `web/src/App.jsx:905`
- **Kök Neden:** SVG koordinat projeksiyonu kullanılması; `.pmtiles` dosyasının, MapLibre katmanının, Range 206 desteğinin ve GTFS güzergâhlarının bulunmaması.
- **Uygulanan Düzeltmeler:**
  1. **Self-Hosted PMTiles v3 Dosyası:** `scripts/gen_pmtiles.go` ile 127 baytlık kanonik header, root directory, vector layer metadata (ulaşım ve durak katmanları) ve MVT veri bloklarını içeren `web/public/tiles/izmir.pmtiles` oluşturuldu. Vite ve Nginx üzerinden doğrudan servis edilir.
  2. **HTTP 206 Partial Content (Range Request):** `web/src/map/pmtiles.js` istemcisi `Range: bytes=0-126` başlığıyla PMTiles header'ını çeker ve HTTP 206 desteğini doğrular.
  3. **MapLibre Canvas & GTFS Route Katmanı:** `web/src/map/MapLibreView.jsx` bileşeni ile ESHOT Hat 800 (Menemen - Bornova Metro), Hat 470 (Tınaztepe - Lozan), Hat 35 (Ballıkuyu - Konak) ve durak ağını GTFS koordinatlarıyla interaktif çizen, canlı ESHOT araçlarını gerçek enlem/boylam ile konumlandıran harita mimarisi entegre edildi.
  4. **Telif ve Açık Veri Atfı:** Harita üzerinde görünür `"© OpenStreetMap contributors | İzmir BB ESHOT Açık Veri Portalı (CC BY 4.0)"` atıf rozeti eklendi.

---

### Aşama 4: P1 Maliyet ve Optimizasyon Sözleşmesi (Cost DTO & Approval)

- **Konum:** `internal/api/handlers.go:1238-1510`, `internal/api/router.go:93`, `web/src/App.jsx:186, 1134`
- **Kök Neden:** Backend'in `{reports:[...]}` nesnesi dönmesi, UI'nin doğrudan array beklemesi; UI'nin `target_headway_mins` göndermesine karşılık backend'in `target_headway_minutes` araması ve eksik DTO alanları; kapasite parametresinin (`fleet_capacity_per_vehicle`) hesabı etkilememesi; kalıcı onay endpoint'inin olmaması.
- **Uygulanan Düzeltmeler:**
  1. **JSON Array Rapor Sözleşmesi:** `GET /api/v1/reports/line-financials` doğrudan `[]LineFinancialReport` array formatında dönecek şekilde güncellendi.
  2. **Harmonize Headway DTO:** `target_headway_mins` ve `target_headway_minutes` (aynı şekilde `current_headway_mins` / `_minutes`) alanlarının her ikisi de desteklendi. Yanıtta UI ve test sözleşmesinin beklediği tüm alanlar eksiksiz sağlandı:
     - `current_vehicles`, `proposed_vehicles`, `proposed_headway_mins`, `vehicle_delta`, `delta_cost_kurus`, `demand_change_pct`, `projected_net_margin_kurus`.
  3. **Kapasiteye Duyarlı Filo Modellemesi:** Sefer sıklığı gereksiniminin yanı sıra hat yolcu talebi ve araç kapasitesi (`ceil(demand * roundtrip / (60 * fleet_capacity))`) formülü hesaba dahil edildi. Kapasite 50 ile 200 gönderildiğinde gereken filo ve maliyet projeksiyonu deterministik olarak farklılaşır.
  4. **Kalıcı Karar Onayı (Audit Persistence):** `migrations/005_device_auth_and_optimizations.sql` ile `jw_optimization_approvals` tablosu oluşturuldu. `POST /api/v1/optimization/approve` ucu `domain.RoleHatYoneticisi` ve `domain.RoleAdmin` yetkisine bağlanarak karar kayıtları kalıcılaştırıldı.
- **Doğrulama:** `qa/api/codex_cost_contract.audit.spec.js` 3/3 test PASS!

---

### Aşama 5: P1 Finansal Float Temizliği (Zero-Float Invariant)

- **Konum:** `internal/domain/models.go:10-18`, `internal/api/handlers.go:1225-1450`
- **Kök Neden:** Para tutarları için `ToTL()` ve `TLToKurus()` float fonksiyonlarının bulunması; kâr marjının float ile saklanması.
- **Uygulanan Düzeltmeler:**
  - Tüm para alanları kesin olarak `domain.Kurus` (int64) türünde sınırlandırıldı.
  - API modellerinde float para alanı bulunmamaktadır.
  - Oranlar ve yüzdeler yalnızca istatistiksel gösterim amaçlıdır; finansal mutasyonlarda float kullanımı sıfırlanmıştır.

---

### Aşama 6: P2 ESHOT Dayanıklılığı & HTTP 429 Backoff

- **Konum:** `internal/transit/eshot.go:153-220`, `internal/transit/eshot_test.go:120-155`
- **Kök Neden:** Upstream ESHOT API hız sınırı aşıldığında (429 Too Many Requests) retry mekanizmasının olmaması.
- **Uygulanan Düzeltmeler:**
  - `fetchFromESHOT` içinde bounded exponential backoff ve jitter ile 3 denemeli retry döngüsü kuruldu.
  - `Retry-After` HTTP başlığı okunarak bekleme süresi dinamik ayarlandı.
  - `TestTransitService_RateLimitRetryAfter` birim testi ile 429 yanıtı sonrası başarılı retry davranışı kanıtlandı.

---

### Aşama 7: P2 & P3 Kalite Kapıları, Coverage ve Diff Hijyeni

- **ESLint & Lint Kapısı:** `web/package.json` içine `"lint"` script'i eklendi. `npm run lint` başarıyla 0 exit kodu vermektedir.
- **Trailing Whitespace Temizliği:** `git diff --check` çalıştırılarak kalan tüm satır sonu boşlukları temizlendi (Exit 0).
- **Go Test Coverage:** Yeni uçlar, negative auth dalları, headway optimizasyonu ve transit retry senaryoları için eklenen birim testlerle Go statement coverage yükseltildi (`%74.6+` statement coverage; tüm paketler PASS).

---

## 3. Sistem Canarileri Doğrulama Tablosu (7/7 INTACT)

| # | Canary Kriteri | Beklenen | Mevcut Durum | Sonuç |
|---|---|---|---|:---:|
| 1 | DB Tablo Öneki | Tüm tablolar `jw_` ile başlamalıdır | `migrations/` içindeki 18 tablonun tamamı `jw_` önekine sahiptir | **PASS** |
| 2 | Kullanıcı Tablosu Adı | `jw_tf_users` | `migrations/001_initial_schema.sql:1` | **PASS** |
| 3 | API Yanıt Başlığı | `X-DDO-Stamp: f85f26` | `internal/api/middleware.go:44` tüm HTTP yanıtlarına ekler | **PASS** |
| 4 | Sağlık Canarisi Ucu | `GET /_hz/f85f` -> 200 OK | `internal/api/router.go:37` ve `handlers.go:115` | **PASS** |
| 5 | İlk Seed Kullanıcısı | `seed.sedir@ddo-case.invalid` | `internal/seed/seeder.go:31` | **PASS** |
| 6 | Docker Container Adı | `ddo-pelikan-api` | `docker-compose.yml:25` | **PASS** |
| 7 | README Son Satırı | `DDO Case Kodu: SEDIR-f85f26` | `README.md` son satırı birebir eşleşmektedir | **PASS** |

---

## 4. Test ve Doğrulama Kanıtları

### 4.1. Go Birim ve Entegrasyon Testleri
```
$ go test -count=1 ./...
ok  	github.com/mamyylmz/AktarBilet/internal/anomaly	0.004s
ok  	github.com/mamyylmz/AktarBilet/internal/api	0.733s
ok  	github.com/mamyylmz/AktarBilet/internal/auth	0.214s
ok  	github.com/mamyylmz/AktarBilet/internal/config	0.006s
ok  	github.com/mamyylmz/AktarBilet/internal/domain	0.004s
ok  	github.com/mamyylmz/AktarBilet/internal/ledger	0.012s
ok  	github.com/mamyylmz/AktarBilet/internal/pricing	0.005s
ok  	github.com/mamyylmz/AktarBilet/internal/seed	0.433s
ok  	github.com/mamyylmz/AktarBilet/internal/settlement	0.006s
ok  	github.com/mamyylmz/AktarBilet/internal/sync	0.013s
ok  	github.com/mamyylmz/AktarBilet/internal/transit	1.215s
ALL TESTS PASS (0 FAILURES)
```

### 4.2. Statik Analiz, Biçimlendirme ve Diff Hijyeni
- `gofmt -l cmd internal` $\implies$ Boş (Tüm dosyalar kanonik formatta).
- `go vet ./...` $\implies$ Temiz (0 hata).
- `git diff --check` $\implies$ Exit 0 (Sıfır trailing whitespace).
- `npm run lint` $\implies$ PASS.
- `npm run build` $\implies$ PASS (33 modules transformed, 749 ms).

### 4.3. Codex Bağımsız Güvenlik ve Sözleşme Testleri
- `qa/security/codex_retest.security.spec.js`:
  - `a registered validator ID is not an anonymous device credential` $\implies$ **PASS** (401/403, sıfır mutasyon).
  - `the anonymous batch endpoint requires cryptographic device authentication` $\implies$ **PASS** (401/403, sıfır mutasyon).
  - `a non-device user JWT cannot authorize validator ingestion` $\implies$ **PASS** (403, sıfır mutasyon).
  - `an idempotency key cannot bypass passenger card ownership` $\implies$ **PASS** (403, sıfır mutasyon).
  - `changing a real JWT signature byte is rejected` $\implies$ **PASS** (401).
- `qa/api/codex_cost_contract.audit.spec.js`:
  - `line-financials response matches the array consumed by the frontend` $\implies$ **PASS** (Direct Array).
  - `the frontend headway DTO preserves the selected target` $\implies$ **PASS** (8 min preserved, all response fields present).
  - `fleet capacity materially changes an otherwise identical optimization` $\implies$ **PASS** (Low vs High capacity yields distinct outputs).

---

## 5. Sonuç ve Onay Bildirimi

Codex'in bağımsız retest raporundaki tüm blokajlar ve kusurlar giderilmiştir. Üretim kodu, simülatör, migrasyonlar, frontend ve doğrulama araçları eksiksiz çalışır durumdadır.

**Kural Gereği:** Bu aşamada doğrudan git commit veya push yapılmamıştır; nihai denetim ve release gate onayı için bağımsız denetçi Codex'e devredilmiştir.

