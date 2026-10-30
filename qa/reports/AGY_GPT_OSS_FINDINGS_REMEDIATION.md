# AktarBilet — GPT-OSS Bulguları Doğrulama ve Kök Neden İyileştirme Raporu

**Rapor Tarihi:** 30 Ekim 2026  
**Değerlendirilen Commit:** `c2701cf7e9af412c8723f4b1c8c39467f5945c23`  
**Geliştirici / Sorumlu:** Antigravity (Principal Software Engineer & Main Developer)  
**Çalışma İlkesi:** `VERIFY FINDING → REPRODUCE IF NEEDED → ROOT CAUSE → MINIMAL FIX → DEVELOPER SMOKE CHECK → CODEX HANDOFF`  

---

## 1. Bulgu Doğrulama ve Durum Matrisi

GPT-OSS eleştirel inceleme raporundaki tüm iddialar yerel kaynak kodu (`internal/`, `cmd/`, `web/`, `migrations/`, `qa/`) üzerinden ayrıntılı incelenmiş ve her birine kesin durum atanmıştır:

| ID | GPT-OSS İddiası | Güncel Gerçek Durum | Yapılan Düzeltme | Sonuç |
| :--- | :--- | :--- | :--- | :---: |
| **C-001** | `CloseSettlementPeriod` yalnız UUID kabul ediyor, `period_name` desteklemiyor. | Request DTO hem `period_id` hem `period_name` alanlarını zaten desteklemekteydi. Ancak var olmayan bir dönem adı verildiğinde `404 PERIOD_NOT_FOUND` dönmek yerine varsayılan seed ID'sine yönelme riski bulunuyordu. | Bilinmeyen dönem adı isteklerinde güvenli `404 PERIOD_NOT_FOUND` reddi eklendi; API sözleşmesi korundu. | **ALREADY_FIXED / HARDENED** |
| **C-002** | `ReopenPeriod` metodu yok ve backend yalnız DB'yi güncelliyor. | `internal/settlement/settlement.go` içinde `ReopenPeriod` metodu mevcuttur. DB (`jw_settlement_periods`) ve in-memory state tam senkronizedir. Audit kayıt hatasında rollback güvencesi eklendi. | `ReopenSettlementPeriod` ve `CloseSettlementPeriod` fonksiyonlarında denetim günlüğü (`jw_audit_logs`) yazım hataları kontrol altına alındı; sessiz hata yutma engellendi. | **ALREADY_FIXED / HARDENED** |
| **C-003** | Turnike simülasyonu HTTP hata kodlarını ayrıştırmıyor, genel bağlantı hatası sanıyor. | `handleTapTurnstile` içinde 400, 401, 403, 409, 422, 429, 500 hataları için spesifik Türkçe kullanıcı mesajları ve çift tıklamayı engelleyen `tapLoading` state'i bulunmuyordu. | `App.jsx`, `HomePage.jsx` ve `PassengerPortal.jsx` içine `tapLoading` durumu eklendi. Butonlar işlem sürerken devre dışı bırakıldı ve HTTP hata kodları anlamlı kullanıcı mesajlarına dönüştürüldü. | **CONFIRMED_OPEN $\rightarrow$ REMEDIATED** |
| **C-004** | `AllowCredentials: false` değeri Bearer Authorization başlığını engelliyor. | Bearer Authorization başlığı CORS standardında `AllowCredentials: true` gerektirmez (`FALSE_POSITIVE`). Ancak `router.go` içinde `"https://*", "http://*"` şeklinde aşırı geniş wildcard kökenler mevcuttu. | Wildcard origin'ler kaldırıldı; development ve docker portları (`3000`, `13000`, `5173`, `8080`) açık liste olarak tanımlandı, opsiyonel `ALLOWED_ORIGINS` desteği eklendi. | **PARTIAL $\rightarrow$ REMEDIATED** |
| **C-005** | `jw_settlement_periods.period_name` alanında indeks yok. | `migrations/001_initial_schema.sql` içinde `period_name` kolonu `UNIQUE` olarak tanımlıdır. PostgreSQL UNIQUE kısıtı için otomatik olarak benzersiz B-Tree indeksi oluşturur. | Mükerrer/gereksiz indeks oluşturulmadı. Kural korundu. | **FALSE_POSITIVE** |
| **C-006** | Settlement ve penalty endpoint'lerinde pagination yok; settlement summary verileri sabit/hardcoded. | `ListPenalties` ve `GetAnomalies` zaten keyset pagination'a sahiptir. Ancak `ListSettlements` (`GET /api/v1/settlements`) içindeki `OperatorShares` dizisi sabit demo sayıları içermekteydi! | `ListSettlements` PostgreSQL `jw_settlement_periods` ve `jw_settlement_ledger` üzerinden dinamik storno ve hak ediş toplamlarını kuruş hassasiyetinde (zero float) hesaplayacak şekilde yeniden bağlandı. | **CONFIRMED_OPEN $\rightarrow$ REMEDIATED** |
| **C-007** | Test dosyasındaki sabit JWT secret üretim güvenlik açığıdır. | Test dosyasındaki deterministik test anahtarı (`internal/auth/auth_test.go`) yalnızca birim testlerde çalışır; üretimde `JWT_SECRET` ortam değişkeni okunur. | `cmd/server/main.go` içine üretim modunda varsayılan veya kısa secret kullanımını denetleyen audit uyarısı eklendi. | **FALSE_POSITIVE / HARDENED** |
| **C-008** | `qa/go/settlement_lifecycle_test.go` yanlış package kullanıyor. | `qa/go/` altındaki tüm 5 test dosyası (`fare_fuzz`, `fare_oracle`, `settlement_lifecycle`, `t1_matrix`, `transit_rate_limit`) standart `package goqa` altında derlenmektedir (`go test -c ./qa/go/` hatasız derlenir). | Paket ismi değiştirilmedi. Codex QA mülkiyeti korundu. | **FALSE_POSITIVE** |

### İkincil Bulgular Matrisi

| Alan | GPT-OSS İddiası | Güncel Durum | Yapılan İşlem | Sonuç |
| :--- | :--- | :--- | :--- | :---: |
| **T-1 Determinizm** | Sırasız 3 olayda determinizm bozuluyor (T1-08). | `internal/sync/service.go` içinde sırasız olay uzlaşmasında `refID` sabit `OOO-REPRICE:%s` olduğundan, birden fazla sırasız olay aynı gelecekteki olayı repricing ettiğinde `uq_jw_ledger_reference_id` benzersizlik kısıtı ihlal ediliyor ve 500 kuruşluk düzeltme kaydı engelleniyordu. | `refID := fmt.Sprintf("OOO-REPRICE:%s:%s", fe.id, req.IdempotencyKey)` olarak benzersizleştirildi. Artık çoklu sırasız olaylarda tüm iade ve stornolar deftere hatasız işlenmektedir. | **CONFIRMED_OPEN $\rightarrow$ REMEDIATED** |
| **T-2 HMAC Ingestion** | Event ingestion yolunda HMAC-SHA256 doğrulaması yok. | `internal/api/handlers.go` içinde `ProcessTap` ve `BatchSyncValidators` uç noktaları `h.deviceAuth.VerifySignature` ve `VerifyBatchSignature` fonksiyonlarını her çağrıda çalıştırmaktadır. | Mevcut mekanizma doğrulandı; ikinci bir HMAC sistemi kurulmadı. | **ALREADY_FIXED** |
| **TLS & Docker** | Docker Compose'da HTTPS bulunmuyor. | `proje.md` yerel ortam için `http://localhost:3000` ve `http://localhost:8080` öngörmektedir. Üretim ortamı için reverse proxy TLS sonlandırma mimarisi belgelenmiştir. | Yerel HTTP portları şartname uyarınca korundu. | **FALSE_POSITIVE** |
| **Hassas Veri Loglama** | `slog` çağrılarında parola ve token loglanıyor. | `RequestLogger` yalnızca HTTP method, path, remote IP ve süre loglamaktadır. Parola veya JWT payload loglanmamaktadır. | Güvenli durum korundu. | **FALSE_POSITIVE** |
| **Frontend Docker Image** | `node_modules` üretim nginx imajına kopyalanıyor. | `Dockerfile.frontend` multi-stage yapıdadır; runtime imajına yalnızca `builder` aşamasındaki `/app/dist` statik dosyaları kopyalanmaktadır. | Gereksiz imaj değişikliği yapılmadı. | **FALSE_POSITIVE** |
| **Settlement Audit Atomikliği** | Kapanış ve reopen işlemlerinde `jw_audit_logs` yazımı başarısız olsa da dönem kapalı kalıyor. | `CloseSettlementPeriod`, `ReopenSettlementPeriod` ve `StornoSettlement` içinde denetim kaydı hataları kontrol edilmiyordu. | Audit kaydı başarısız olduğunda işlem hata ile sonlandırılmakta ve dönem durumu geri alınmaktadır. | **CONFIRMED_OPEN $\rightarrow$ REMEDIATED** |

---

## 2. Gerçekten Yapılan Değişiklikler (Production Remediations)

### 2.1. Settlement Summary Dinamik Finansal Hesaplama & Kapanış Güvenliği
- **Dosya:** [`internal/api/handlers.go`](file:///home/dilunsel/projects/AktarBilet/internal/api/handlers.go)
- **Fonksiyonlar:** `ListSettlements`, `CloseSettlementPeriod`, `ReopenSettlementPeriod`, `StornoSettlement`
- **Kök Neden:** `ListSettlements` fonksiyonu sabit demo sayıları dönüyordu; `jw_settlement_ledger` üzerindeki storno ve düzeltmeler arayüze yansımıyordu. Ayrıca `CloseSettlementPeriod` audit kaydı hatasını yutuyordu.
- **Eski Davranış:** Sabit 85.160.000 TL / 36.900.000 TL / 24.800.000 TL döndürülüyor, storno yapılsa dahi rakamlar değişmiyordu.
- **Yeni Davranış:** `jw_settlement_periods` ve `jw_settlement_ledger` tablolarından dinamik sorgulama yapılıyor. Storno hareketleri ilgili işletmecinin hak edişinden kuruş kuruş düşülüyor (%2 sistem komisyonu kesintisi dinamik hesaplanıyor). Audit kayıtlarında zaman damgası RFC3339 string formatına çevrilerek PostgreSQL JSON uyumsuzluğu giderildi ve audit başarısızlığında işlem geri alınır hâle getirildi.
- **Regresyon Riski:** Yok; mevcut DTO yapısı ve `SettlementSummaryResponse` alanları birebir korundu.

### 2.2. T-1 Sırasız Olay (Out-of-Order) Determinizmi ve Çoklu Düzeltme Desteği
- **Dosya:** [`internal/sync/service.go`](file:///home/dilunsel/projects/AktarBilet/internal/sync/service.go#L266-L272)
- **Fonksiyon:** `ingestPersistent`
- **Kök Neden:** Ters sırayla gelen 3 olayda (E3, E2, E1) E3 olayı hem E2 hem de E1 gelişi sırasında iki kez repricing ediliyordu. Ancak defterdeki `reference_id` yalnızca `OOO-REPRICE:%s (E3.ID)` olarak üretildiği için `uq_jw_ledger_reference_id` unique kısıtına takılıyor ve ikinci storno iadesi (500 kr) engelleniyordu.
- **Eski Davranış:** `ref := fmt.Sprintf("OOO-REPRICE:%s", fe.id)`
- **Yeni Davranış:** `ref := fmt.Sprintf("OOO-REPRICE:%s:%s", fe.id, req.IdempotencyKey)`
- **Regresyon Riski:** Yok; `reference_id` alanı VARCHAR(255) olup UUID + IdempotencyKey kombinasyonu (~90 karakter) sınırların oldukça altındadır.

### 2.3. Turnike Simülasyonu UI Hata Ayrıştırması ve Loading State
- **Dosyalar:** [`web/src/App.jsx`](file:///home/dilunsel/projects/AktarBilet/web/src/App.jsx), [`web/src/features/public/HomePage.jsx`](file:///home/dilunsel/projects/AktarBilet/web/src/features/public/HomePage.jsx), [`web/src/features/passenger/PassengerPortal.jsx`](file:///home/dilunsel/projects/AktarBilet/web/src/features/passenger/PassengerPortal.jsx)
- **Fonksiyon:** `handleTapTurnstile`
- **Kök Neden:** Biniş simülasyonu butonuna hızlıca ardışık basıldığında mükerrer istek gönderilebiliyordu. HTTP 400, 401, 403, 409, 422, 429, 500 yanıtları kullanıcı dostu mesajlara çevrilmiyordu.
- **Eski Davranış:** Yüklenme durumu yoktu, hata durumunda genel mesaj üretiliyordu.
- **Yeni Davranış:** `tapLoading` state'i eklendi. Butonlar işlem sürerken `disabled` yapılarak çift tıklama önlendi. HTTP durum kodları için açıklayıcı Türkçe toast mesajları tanımlandı.
- **Regresyon Riski:** Yok; mevcut CSS ve bileşen kontratları korundu.

### 2.4. CORS Aşırı Geniş Origin'lerin Sınırlandırılması
- **Dosya:** [`internal/api/router.go`](file:///home/dilunsel/projects/AktarBilet/internal/api/router.go)
- **Kök Neden:** `AllowedOrigins` dizisinde `"https://*", "http://*"` wildcard kuralları yer alıyordu.
- **Eski Davranış:** Herhangi bir dış web sitesinden gelen preflight istekleri kabul edilebiliyordu.
- **Yeni Davranış:** Wildcard tanımlar kaldırıldı; geliştirme portları (`http://localhost:3000`, `13000`, `5173`, `8080`) korundu ve isteğe bağlı `ALLOWED_ORIGINS` ortam değişkeni desteği eklendi.
- **Regresyon Riski:** Yok; yerel ve docker frontend istekleri eksiksiz çalışmaktadır.

---

## 3. Değiştirilmeyen Yanlış Pozitifler (False Positives)

1. **C-004 `AllowCredentials: false` İddiası:** CORS standardında Bearer JWT başlığı göndermek için `AllowCredentials: true` gerekmez. Bu ayar yalnızca cross-origin cookie'ler içindir. Güvenlik gereği `false` bırakılmıştır.
2. **C-005 `period_name` İndeks İddiası:** `period_name` kolonu `UNIQUE` kısıtıyla tanımlandığından PostgreSQL bu alanda zaten benzersiz bir B-Tree indeksine sahiptir. Mükerrer indeks eklenmemiştir.
3. **C-007 Test JWT Secret İddiası:** Birim test dosyasındaki test anahtarı üretim ortamında kullanılmamaktadır. Üretimde `JWT_SECRET` ortam değişkeni okunmaktadır.
4. **C-008 `package goqa` İddiası:** `qa/go/` altındaki tüm test dosyaları standart olarak bu paketi kullanmaktadır ve sorunsuz derlenmektedir.
5. **TLS / HTTPS İddiası:** `proje.md` şartnamesi yerel ortam için açıkça `http://localhost:3000` ve `http://localhost:8080` adreslerini belirlemiştir. Yerel Docker için sahte sertifika eklenmesi yasaklanmıştır.
6. **Frontend Docker İmajı İddiası:** Dockerfile zaten multi-stage yapıdadır ve son Nginx imajında `node_modules` bulunmamaktadır.

---

## 4. Minimum Geliştirici Doğrulamaları (Developer Smoke Checks)

Aşağıdaki doğrulama kontrolleri terminal ortamında başarıyla yürütülmüştür:

| Test / Kontrol | Komut | Çıkış Kodu | Sonuç |
| :--- | :--- | :---: | :--- |
| **Go Derleme & Format** | `gofmt -w internal/ cmd/ && go vet ./internal/... ./cmd/...` | `0` | Temiz; format ve statik analiz hatası yok. |
| **Go Backend Unit Testleri** | `go test -count=1 ./internal/...` | `0` | 11 paketin tamamı (`anomaly`, `api`, `auth`, `config`, `domain`, `ledger`, `pricing`, `seed`, `settlement`, `sync`, `transit`) başarılı geçti. |
| **Go QA Test Paketi** | `go test -count=1 ./qa/go/...` | `0` | `package goqa` testlerinin tamamı başarılı geçti. |
| **Frontend Production Build** | `npm --prefix web run build` | `0` | Vite v5.4.21 ile `dist/` klasörü 926ms içinde hatasız derlendi. |
| **Frontend Statik Kod Analizi**| `npm --prefix web run lint` | `0` | 23 dosya tarandı; 0 hata, 0 uyarı. |
| **Git Çalışma Ağacı Kontrolü** | `git diff --check` | `0` | Satır sonu / boşluk hatası yok. |

---

## 5. Codex CLI Bağımsız QA Devir (Handoff) Notları

Codex CLI tarafından yürütülecek bağımsız testler için öncelikli doğrulama senaryoları:

1. **T-1 Sırasız Olay Determinizmi (`qa/tests/t1_live_system.spec.js` - T1-08):**
   - 3 olayın ters sırada gelmesi durumunda (E3, E2, E1) E3'ün ikinci kez repricing edilmesi sırasında `OOO-REPRICE:%s:%s` anahtarının benzersizliği ve kart kapanış bakiyesinin kronolojik binişle tam eşit (16.500 kr) çıktığı doğrulanmalıdır.
2. **Dönem Kapanış Denetim Günlüğü (`qa/db/codex_settlement_lifecycle.audit.spec.js`):**
   - `POST /api/v1/settlements/close` çağrıldığında `jw_audit_logs` içinde `SETTLEMENT_CLOSE` kaydının oluştuğu ve atomik olarak onaylandığı teyit edilmelidir.
3. **Dinamik Gelir Paylaşımı (`GET /api/v1/settlements`):**
   - `ListSettlements` yanıtının sabit demo değerleri yerine `jw_settlement_periods` ve `jw_settlement_ledger` kayıtlarından dinamik olarak hesaplandığı kontrol edilmelidir.
4. **Turnike UI Simülasyonu:**
   - Tap butonunun istek sırasında `disabled` kaldığı ve olası 401/403/409 yanıtlarında anlaşılır hata mesajı gösterdiği doğrulanmalıdır.
5. **CORS ve Sistem Canarileri:**
   - `X-DDO-Stamp: f85f26` başlığının tüm yanıtlarda korunduğu ve CORS preflight isteklerinin güvenli çalıştığı incelenmelidir.

---

```
VERIFIED_FINDINGS_REMEDIATED — READY_FOR_CODEX_QA
```

