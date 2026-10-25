# AktarBilet Üretim Kodu Değişiklik ve Düzeltme Günlüğü (Fix Log)

**Tarih:** 25 Ekim 2026  
**Hedef Repository:** `https://github.com/mamyylmz/AktarBilet`  
**Geliştirici & SDET:** Antigravity (Ana Kod Geliştirici & Kalite Mühendisi)  
**Temel İlke:** En küçük güvenli düzeltme (minimal blast radius), sıfır yapay test gevşetmesi, kanıtlanabilir regresyon güvenliği.

---

## 1. Düzeltme Fazları ve Git Değişiklik Özeti

| Faz / Modül | İlgili Git Commit / Sürüm | Başlıca Değişiklik Yapılan Dosyalar | Kapsanan Konular |
|---|---|---|---|
| **Faz 1: Veritabanı ve Defter Bütünlüğü** | `e0f8e77` | `migrations/003_constraints_and_triggers.sql`, `internal/seed/seeder.go`, `internal/ledger/postgres.go` | AB-QA-001, AB-QA-003, AB-QA-008, AB-QA-013 |
| **Faz 2: Güvenlik, RBAC, IDOR ve Kimlik** | `ee37809` | `internal/api/router.go`, `internal/api/handlers.go`, `internal/auth/auth.go` | AB-QA-002, AB-QA-017, AB-QA-021 |
| **Faz 3: Çevrimdışı Senkronizasyon & Kalıcılık** | `7458e40` | `internal/sync/service.go`, `cmd/simulator/main.go`, `internal/seed/seeder.go`, `scripts/reset-demo.sh` | AB-QA-004, AB-QA-010, AB-QA-012, AB-QA-014, AB-QA-015 |
| **Faz 4: Server-Authoritative UI & Responsive** | `58b722a` | `web/src/App.jsx`, `web/src/index.css`, `web/src/components/*` | AB-QA-006, AB-QA-007, AB-QA-011, AB-QA-018 |
| **Faz 5: Test Kapsamı Genişletme** | `6ac8591` | `internal/api/handlers_test.go`, `internal/ledger/postgres_test.go`, `internal/seed/seeder_test.go`, `internal/sync/service_test.go` | AB-QA-005, AB-QA-019, AB-QA-020, AB-QA-022, AB-QA-023 |
| **Faz 6: Nihai Doğrulama ve Test Sertleştirme** | `HEAD` | `internal/sync/service.go`, `internal/sync/service_test.go`, `internal/api/handlers.go`, `internal/api/middleware.go`, `internal/api/handlers_test.go`, `internal/config/config_test.go`, `internal/seed/seeder.go` | Test kapsamının %82.6'ya çıkarılması, Sync & API mock arayüzleri, Demo kart bakiye senkronizasyonu |

---

## 2. Detaylı Değişiklik Kayıtları

### 2.1. Faz 1: Defter Bütünlüğü ve İdempotans (`e0f8e77`)
- **Hedef:** Finansal defterin mutlak append-only olmasının garanti altına alınması, yeniden başlatma sırasında seed bozulmasının önlenmesi, şema CHECK kısıtları.
- **Değiştirilen Dosyalar:**
  - `migrations/003_constraints_and_triggers.sql`: PostgreSQL düzeyinde `prevent_ledger_mutation()` trigger'ı tanımlandı; `UPDATE` ve `DELETE` engellendi (SQLSTATE 55000). Tarifelere negatif ücret engeli (`chk_tariffs_base_fare`) ve %0-100 indirim aralık kısıtları eklendi.
  - `internal/seed/seeder.go`: Deterministik referans ID'leri (`SEED_CARD_TAM_001` vb.) ve `ON CONFLICT (reference_id) DO NOTHING` kullanıldı; `balance = SUM(amount)` değişmezi korundu.
  - `internal/ledger/postgres.go`: Defter kayıtlarında `INSERT INTO ... RETURNING created_at` mekanizmasına geçilerek zaman damgası mikrosaniye kararlılığı sağlandı.
- **Risk Değerlendirmesi:** Düşük. İmmutable ledger finansal güvenliğin temelidir.
- **Doğrulama:** `qa/tests/database.integrity.spec.js` ve `internal/ledger/ledger_test.go`.

### 2.2. Faz 2: Güvenlik, RBAC, IDOR ve Kimlik Doğrulama (`ee37809`)
- **Hedef:** Anonim finansal erişimlerin engellenmesi, yolcu rolünün başkasının kartına erişiminin (IDOR) kapatılması, JWT güvenlik sıkılaştırması.
- **Değiştirilen Dosyalar:**
  - `internal/api/router.go`: `r.Group` ile public uçlar (login, health, canary) ile korumalı uçlar ayrıldı. `RequireRole` ve `RequireAnyRole` middleware'leri tüm hassas operasyonlara bağlandı. Güvenilir CORS listesi tanımlandı.
  - `internal/api/handlers.go`: `ensureCardOwnership` fonksiyonu eklendi; `user_id` uyuşmazlığında `403 Forbidden` döndürüldü.
  - `internal/auth/auth.go`: JWT imza doğrulamasında sadece `jwt.SigningMethodHS256` kabul edildi; `none` veya diğer algoritmalar reddedildi.
- **Risk Değerlendirmesi:** Orta. İzin kontrollerinin doğru rollerle eşleştiği tüm API çağrılarında doğrulandı.
- **Doğrulama:** `qa/tests/rbac.security.spec.js` (17 senaryonun 17'si PASS).

### 2.3. Faz 3: Simülatör ve Kalıcı Çevrimdışı Senkronizasyon (`7458e40`)
- **Hedef:** Simülatör olaylarının veritabanında kalıcılaşması, 200.000 kart üretimi, sırasız olayların kronolojik sıralanarak oynatılması.
- **Değiştirilen Dosyalar:**
  - `internal/sync/service.go`: Tekil tap ve batch senkronizasyon olayları doğrudan PostgreSQL transaction'ı içinde `jw_validator_events` ve `jw_ledger` tablolarına kaydedildi. Olaylar `SortEventsChronologically` ile sıralandı.
  - `internal/seed/seeder.go`: PostgreSQL `generate_series` kullanılarak 200.000 deterministik kart saniyeler içinde toplu üretildi.
  - `scripts/reset-demo.sh`: `TRUNCATE ... CASCADE` ve ardından seeder çalıştırarak tek komutla temiz başlangıç sağlandı.
- **Risk Değerlendirmesi:** Düşük. Veri kaybı sıfırlandı.
- **Doğrulama:** `qa/tests/api.integration.spec.js` ve `cmd/verify`.

### 2.4. Faz 4: Server-Authoritative Frontend ve Responsive Tasarım (`58b722a`)
- **Hedef:** İstemci tarafındaki sahte başarı mesajlarının kaldırılması, API hatalarının doğru iletilmesi, mobil ve tablet ekranlarda yatay taşmanın giderilmesi.
- **Değiştirilen Dosyalar:**
  - `web/src/App.jsx`: İstemcideki mock bakiye hesaplamaları ve sahte catch-block başarıları kaldırıldı. Hatalarda kırmızı banner gösterimi sağlandı. `localStorage` ile oturum rolü kalıcılaştırıldı.
  - `web/src/index.css`: Tüm sayfa kapsayıcılarına `max-width: 100%`, `overflow-x: auto` ve responsive medya sorguları eklendi.
  - `web/src/components/*`: Dönem kapatma butonunda duplicate rozetler temizlendi; Playwright strict selector uyumu sağlandı.
- **Risk Değerlendirmesi:** Düşük. Kullanıcı deneyimi doğruluğu sağlandı.
- **Doğrulama:** `qa/tests/browser.uat.spec.js` (10 UAT senaryosunun 10'u PASS).

### 2.5. Faz 5: Test Kapsamı Genişletme (`6ac8591`)
- **Hedef:** `internal/...` ifade kapsamının ≥%70 eşiğine çıkarılması, `PostgresLedger` ve `SeedDatabase` modüllerinin kapsamlı test edilmesi.
- **Değiştirilen Dosyalar:**
  - `internal/ledger/postgres.go`: `DBTxStarter` arayüzü (`Begin`, `Query`, `QueryRow`) tanımlandı; `NewPostgresLedgerWithStarter` kurucusu eklendi.
  - `internal/ledger/postgres_test.go`: Mock `pgx.Tx` ile tüm PostgreSQL işlem yolları test edildi.
  - `internal/seed/seeder.go`: `DBExecutor` arayüzü eklendi; `SeedDatabaseWithExecutor` eklendi.
  - `internal/seed/seeder_test.go`: 7 test ile hat kardinalitesi, duraklar, tarifeler ve toplu kart üretimi test edildi.
- **Risk Değerlendirmesi:** Sıfır.
- **Doğrulama:** `go test -coverprofile=coverage.out ./internal/...` -> **%71.2 KAPSAM**.

### 2.6. Faz 6: Nihai Doğrulama ve Test Sertleştirme (`HEAD`)
- **Hedef:** `internal/sync` ve `internal/api` paketlerinin kapsamını yükseltmek, toplam kapsamı %80 üzerine taşımak, seeder bakiye-defter tutarlılığını pekiştirmek ve veri yarışlarını sıfırlamak.
- **Değiştirilen Dosyalar:**
  - `internal/sync/service.go`: `SyncDBPool` arayüzü ve `SetDBPool` metodu eklendi. `ingestPersistent` boru hattı havuz bağımlılığından soyutlandı.
  - `internal/sync/service_test.go`: `mockSyncPool` ve `mockSyncRows` ile kalıcı senkronizasyon boru hattı, tarife eşleşmesi, bakiye düşümü ve defter kaydı test edildi. Paket kapsamı %48.8'den **%88.8**'e yükseldi.
  - `internal/api/handlers.go` & `internal/api/middleware.go`: `APIDBPool` ve `QueryRower` arayüzleri soyutlandı.
  - `internal/api/handlers_test.go`: `TestAPI_DatabaseQueryMocks` ile hat, durak, tarife listeleme, ceza oluşturma/listeleme, ısı haritası, uzlaşma dönemi kapatma ve storno sorgu yolları mock havuz ile test edildi. Paket kapsamı %64.9'dan **%78.3**'e yükseldi.
  - `internal/config/config_test.go`: `getEnvInt` fonksiyonunun tüm uç dalları test edilerek paket kapsamı **%100.0** yapıldı.
  - `internal/seed/seeder.go`: Seed edilen demo kartların bakiye güncellemesinden sonra `jw_cards.balance_kurus` alanının defter toplamına (`SUM(jw_ledger.amount_kurus)`) eşitlenmesi satırı eklendi.
  - Canonical `golang:1.23` Docker konteyneri ile `go test -race ./internal/...` çalıştırılarak **0 veri yarışı** kanıtlandı.
- **Risk Değerlendirmesi:** Sıfır. Temiz arayüz soyutlamaları ve sağlam değişmez koruması.
- **Doğrulama:** `go test -coverprofile=coverage.out ./internal/...` -> **%82.6 KAPSAM**, 58 Go Testi PASS, 49/49 QA PASS, 14/14 `make verify` PASS.
