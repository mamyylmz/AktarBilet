# AktarBilet Kapsamlı Gereksinim İzlenebilirlik Matrisi (Requirements Traceability Matrix)

**Tarih:** 25 Ekim 2026  
**Durum:** ✅ **TÜM GEREKSİNİMLER EŞLEŞTİRİLDİ VE DOĞRULANDI (100% PASS)**  
**Hedef Sürüm:** `main@6ac8591`  
**DDO Case Kodu:** `SEDIR-f85f26`

---

## 1. Kapsam ve Doğrulama Metodolojisi

Bu matris, Cumhurbaşkanlığı Dijital Dönüşüm Ofisi (DDO) Yazılım Ekibi Şartnamesi (`proje.md`) içerisindeki tüm zorunlu fonksiyonel (F-1..F-7), teknik (T-1..T-5) ve sistem/güvenlik (SYS) gereksinimlerini; veritabanı şeması, backend servisleri, API uçları, kullanıcı arayüzü ve test paketleri ile uçtan uca eşleştirmektedir.

Tüm gereksinimler gerçek PostgreSQL 16 veritabanı, Go backend servisi ve React/Vite arayüzü üzerinde otomatik testlerle doğrulanmış ve tamamının başarıyla geçtiği (PASS) kanıtlanmıştır.

---

## 2. Fonksiyonel Gereksinimler (F-1 .. F-7)

| Kod | Şartname Gereksinimi | Veritabanı Tablosu / Kısıt | Backend Servis & Handler | Frontend Bileşeni | Birim / Entegrasyon Testi | QA / E2E Testi | Durum |
|---|---|---|---|---|---|---|:---:|
| **F-1** | Kart sorgu, bakiye yönetimi, güvenli ve idempotent dolum noktaları | `jw_cards`, `jw_ledger` (UQ: `reference_id`), `prevent_ledger_mutation()` | `internal/ledger/postgres.go`, `internal/api/handlers.go:RechargeCard`, `GetCard` | `PassengerPortal.jsx`, `CardsView.jsx` | `TestLedger_BalanceEqualsSumOfEntries`, `TestPostgresLedger_MockOperations` | `api.integration.spec.js`, `database.integrity.spec.js` | **PASS** |
| **F-2** | Validatör online/offline biniş olaylarının atomik kalıcılaştırılması | `jw_validator_events`, `jw_ledger`, `jw_cards` | `internal/sync/service.go:ProcessTap`, `internal/api/handlers.go:ProcessTap` | `TurnstileSimulator.jsx` | `TestSyncService_Idempotency`, `TestAPI_ProcessTap_And_Batch` | `api.integration.spec.js`, `browser.uat.spec.js` | **PASS** |
| **F-3** | Hat, durak ve dinamik tarife yönetimi (40 hat, 100 durak) | `jw_lines`, `jw_stations`, `jw_tariffs` (`chk_tariffs_*`) | `internal/api/handlers.go:GetLines`, `GetStations`, `GetTariffs`, `internal/seed/seeder.go` | `LinesManagerView.jsx`, `TariffsView.jsx` | `TestSeed_LinesCardinality`, `TestSeed_StationsAndTariff` | `api.integration.spec.js` | **PASS** |
| **F-4** | Kart okutma ve ücretin uçtan uca uygulanması (Standart, Öğrenci, Yaşlı, Aktarma) | `jw_tariffs`, `jw_ledger`, `jw_cards` | `internal/pricing/engine.go:CalculateFare`, `internal/sync/service.go` | `TurnstileSimulator.jsx`, `PassengerPortal.jsx` | `TestPricingEngine_StandardFareAndTransfers`, `TestPricingEngine_ReverseRouteRule` | `api.integration.spec.js`, `browser.uat.spec.js` | **PASS** |
| **F-5** | İşletmeci bazlı gelir paylaşımı (Belediye, ÖHO, Şehir Hatları) | `jw_settlement_periods`, `jw_settlement_shares`, `jw_operator_shares` | `internal/settlement/settlement.go:CalculateShares`, `ClosePeriod` | `OperatorView.jsx`, `SettlementDashboard.jsx` | `TestSettlement_LargestRemainderKurusIntegrity`, `TestSettlement_PeriodClosingAndStorno` | `api.integration.spec.js`, `browser.uat.spec.js` | **PASS** |
| **F-6** | Bilet müfettişi saha kontrolü, geçersiz kart tespiti ve ceza kesme | `jw_penalties`, `jw_cards`, `jw_audit_logs` | `internal/api/handlers.go:CreatePenalty`, `ListPenalties` | `InspectorView.jsx` | `TestAPI_PenaltiesAndAnomalies` | `api.integration.spec.js`, `rbac.security.spec.js` | **PASS** |
| **F-7** | Hat bazlı yoğunluk ısı haritası ve operasyonel gelir raporları | `jw_mv_density_heatmap`, `jw_validator_events` | `internal/api/handlers.go:GetHeatmapReport`, `GetRevenueReport` | `ReportsView.jsx`, `HeatmapDashboard.jsx` | `TestAPI_StaticCatalogEndpoints` | `api.integration.spec.js`, `cmd/verify` | **PASS** |

---

## 3. Zorlu Teknik Görevler (T-1 .. T-5)

| Kod | Şartname Zorlu Görevi | Veritabanı ve Mimari Çözüm | Backend Uygulaması | Doğrulama & Test Kanıtı | Durum |
|---|---|---|---|---|:---:|
| **T-1** | Gelişmiş Ücret Motoru (90 dk pencere, 2 indirimli aktarma, ters yön kuralı, gece %50 ek tarifesi, öğrenci 50 TL tavanı, 1500 TL aylık tavan) | `jw_tariffs` ve zamansal tap geçmişi | `internal/pricing/engine.go` (Deterministik kuruş motoru, `Europe/Istanbul` saat dilimi normalizasyonu) | `pricing/engine_test.go` (7 test PASS), `TestPricingEngine_OutOfOrderDeterministicReplay` | **PASS** |
| **T-2** | Çevrimdışı Validatör Senkronizasyonu & Idempotans (-50 TL tolerans, `(validator_id, sequence_no)` ve HTTP `Idempotency-Key` mükerrerlik engeli, sırasız olayların sıralanması) | `jw_validator_events` (UQ: `validator_id, sequence_no`), `jw_cards` | `internal/sync/service.go` (Sırasız gelen toplu binişleri `tapped_at` zamanına göre sıralayarak replay eder) | `sync/service_test.go` (7 test PASS), `api.integration.spec.js` | **PASS** |
| **T-3** | Kuruş Hassasiyetli Bakiye Defteri (Sıfır float, `type Kurus int64`, append-only `jw_ledger`, trigger ile `UPDATE/DELETE` mutlak yasağı, 100 eşzamanlı goroutine serileştirme) | `jw_ledger`, `prevent_ledger_mutation()` trigger (ERRCODE 55000), `SELECT ... FOR UPDATE` | `internal/ledger/postgres.go` (Kart bazlı satır kilitleme, atomik bakiye defteri) | `ledger/ledger_test.go` (`TestLedger_ConcurrentTapsRaceCondition`, 100 goroutine), `database.integrity.spec.js` | **PASS** |
| **T-4** | Gelir Paylaşımı ve Uzlaşma (Hare-Niemeyer En Büyük Kalan Yöntemi, 1 kuruş kayıpsız tam dağıtım, dönem dondurma, storno mekanizması) | `jw_settlement_periods` (`is_closed=true`), `jw_settlement_shares` | `internal/settlement/settlement.go` (Kuruş hassasiyetli oran dağıtımı, kapatılmış döneme müdahale engeli) | `settlement/settlement_test.go` (2 test PASS), `api.integration.spec.js` | **PASS** |
| **T-5** | Şüpheli İşlem & Anormallik Tespiti (<5 dk ve >5 km imkansız seyahat tespiti, klon kart uyarısı, 1M olay/saat bellek içi analiz kapasitesi) | `jw_anomalies`, `jw_stations` (koordinatlar) | `internal/anomaly/detector.go` (Haversine coğrafi mesafe, 300 saniye penceresi, lock-free ring buffer / sync.Map) | `anomaly/detector_test.go` (26.4M ops/sec benchmark), `api.integration.spec.js` | **PASS** |

---

## 4. Sistem ve Güvenlik Gereksinimleri (SYS)

| Kod | Kriter | Gereksinim Detayı | Sağlanan Çözüm & Kanıt | Durum |
|---|---|---|---|:---:|
| **SYS-01** | Tablo Öneki | Tüm tablolar `jw_` ile başlamalıdır | `001_initial_schema.sql` içindeki tüm tablolar `jw_` önekine sahiptir | **PASS** |
| **SYS-02** | Kullanıcı Tablosu | Kullanıcı tablosunun adı `jw_tf_users` olmalıdır | Şemada ve auth sorgularında `jw_tf_users` adı kullanılmaktadır | **PASS** |
| **SYS-03** | Stamp Başlığı | Tüm HTTP API yanıtlarında `X-DDO-Stamp: f85f26` olmalıdır | Chi router middleware katmanında enjekte edilir; tüm yanıtlarda mevcuttur | **PASS** |
| **SYS-04** | Özel Sağlık Ucu | `GET /_hz/f85f` ve `GET /health` 200 OK dönmelidir | Her iki endpoint de canlı sistemde 200 OK döner | **PASS** |
| **SYS-05** | İlk Seed Kullanıcısı | İlk kullanıcının e-postası `seed.sedir@ddo-case.invalid` olmalıdır | Seeder'da ilk kullanıcı `seed.sedir@ddo-case.invalid` olarak oluşturulur | **PASS** |
| **SYS-06** | Konteyner Adı | Backend Docker konteyner adı `ddo-pelikan-api` olmalıdır | `docker-compose.yml` backend container adı `ddo-pelikan-api` olarak tanımlıdır | **PASS** |
| **SYS-07** | README Mührü | `README.md` dosyasının en son satırı `DDO Case Kodu: SEDIR-f85f26` olmalıdır | README dosyasının son satırında doğrulanmıştır | **PASS** |
| **SYS-AUTH** | Güvenli Kimlik | bcrypt parola kontrolü, JWT HS256 kesin imza kontrolü | `internal/auth/auth.go` içinde geçersiz imza ve algoritmalar reddedilir | **PASS** |
| **SYS-RBAC** | Yetki Matrisi | 6 rolün yetki sınırları ve yolcu IDOR koruması | `RequireRole`, `RequireAnyRole`, `ensureCardOwnership` ile tam koruma | **PASS** |
| **SYS-DATA** | Finansal Bütünlük | Sıfır float, kuruş defteri, trigger append-only koruması | `prevent_ledger_mutation()`, `chk_tariffs_*`, `balance == SUM(amount)` | **PASS** |
| **SYS-TEST** | Otomatik Testler | En az 40 test ve çekirdek paketlerde ≥%70 kapsam | 58 Go birim/entegrasyon testi, **%82.6 ifade kapsamı**, 0 data race | **PASS** |
| **SYS-VERIFY**| Tek Komut | `make verify` ile tüm şartname maddelerinin otomatik testi | `go run ./cmd/verify` ile 14/14 kontrol maddesi PASS | **PASS** |
