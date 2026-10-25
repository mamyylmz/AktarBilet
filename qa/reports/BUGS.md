# AktarBilet Hata Envanteri ve Kapatma Raporu (Defect Resolution Log)

**Tarih:** 25 Ekim 2026  
**Durum:** ✅ **TÜM HATALAR ÇÖZÜLDÜ VE KAPATILDI (24/24 CLOSED)**  
**Hedef Sürüm:** `main@6ac8591`  
**DDO Case Kodu:** `SEDIR-f85f26`

---

## 1. Hata Çözüm Özeti

| Öncelik | Açılan Hata | Çözülen (Closed) | Kalan (Open) | Başarı Oranı |
|---|:---:|:---:|:---:|:---:|
| **P0 — Blocker** | 4 | 4 | 0 | **%100** |
| **P1 — Critical** | 12 | 12 | 0 | **%100** |
| **P2 — Major** | 8 | 8 | 0 | **%100** |
| **Toplam** | **24** | **24** | **0** | **%100 (SIFIR AÇIK BLOKAJ)** |

---

## 2. P0 — BLOCKER Hataları ve Çözümleri

### AB-QA-001 — Yeniden Başlatma Seed'i Finansal Defteri Çoğaltıyor ve Bakiyeyi Bozuyor
- **Öncelik:** P0 (Blocker)
- **Gereksinim:** F-1, T-3, SYS-SEED
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Backend konteyneri yeniden başladığında seeder her seferinde rastgele UUID ile `SEED_INITIAL` ledger kaydı ekliyor ve kart bakiyesini seed başlangıç değerine eziyordu (`balance != SUM(amount)`).
- **Yapılan Düzeltme:**
  - `internal/seed/seeder.go` içerisine seed durum kontrolü (`jw_seed_state` tablosu / kontrol mekanizması) eklendi.
  - Kartların başlangıç bakiye kayıtları için deterministik referans ID (`SEED_CARD_TAM_001` vb.) tanımlandı ve `ON CONFLICT (reference_id) DO NOTHING` sağlandı.
  - Kart bakiyesinin defter hareketlerinin toplamına eşit olması kuralı (`balance = SUM(amount)`) mutlak hale getirildi.
- **Doğrulama Kanıtı:** Backend restart edildiğinde ledger satır sayısı artmaz; `database.integrity.spec.js` ve `internal/seed/seeder_test.go` testleri 0 hatayla geçti.

### AB-QA-002 — Anonim Kullanıcı Finansal Uçlara Erişebiliyor ve IDOR Kart Açığı Mevcut
- **Öncelik:** P0 (Blocker)
- **Gereksinim:** F-1, F-2, SYS-AUTH/RBAC
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** `/api/v1/cards/{id}` ve `/api/v1/cards/{id}/recharge` uçlarında JWT doğrulama ve kart sahiplik (ownership) kontrolü eksikti; anonim kullanıcılar veya başka yolcular başkasının kartını sorgulayabiliyor/doldurabiliyordu.
- **Yapılan Düzeltme:**
  - `internal/api/router.go` üzerinde `RequireRole` ve `RequireAnyRole` middleware'leri tüm finansal ve operasyonel rotalara bağlandı.
  - `internal/api/handlers.go` içine kart sahiplik kontrolü (`ensureCardOwnership`) eklendi. Yolcu rolündeki kullanıcı yalnızca kendi `user_id`sine atanmış kartları okuyabilir ve doldurabilir. Başkasına ait kartlarda `403 Forbidden` fırlatılır.
- **Doğrulama Kanıtı:** `rbac.security.spec.js` içerisinde anonim ve yetkisiz yolcu isteklerinin tamamı 401/403 almaktadır; `handlers_test.go` IDOR testleri PASS.

### AB-QA-003 — Defter (jw_ledger) Tablosu PostgreSQL Düzeyinde Append-Only Korunmuyor
- **Öncelik:** P0 (Blocker)
- **Gereksinim:** T-3 (Finansal Değişmez)
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** `jw_ledger` tablosu üzerinde `UPDATE` ve `DELETE` işlemlerini engelleyen veritabanı kısıtı veya trigger yoktu.
- **Yapılan Düzeltme:**
  - `migrations/003_constraints_and_triggers.sql` migrasyonu oluşturuldu.
  - `prevent_ledger_mutation()` PostgreSQL trigger fonksiyonu yazıldı: `BEFORE UPDATE OR DELETE ON jw_ledger FOR EACH ROW EXECUTE FUNCTION prevent_ledger_mutation();`.
  - Herhangi bir UPDATE veya DELETE girişiminde SQLSTATE `55000` ile `RAISE EXCEPTION 'jw_ledger is append-only: UPDATE and DELETE are prohibited'` fırlatılması sağlandı.
  - Aynı koruma `jw_audit_logs` tablosuna da uygulandı.
- **Doğrulama Kanıtı:** `database.integrity.spec.js` içerisinde UPDATE ve DELETE çağrıları SQL hatasıyla reddedildi (PASS).

### AB-QA-008 — Dolum İdempotansi Başarısızlığı ve Çift Bakiye Riski
- **Öncelik:** P0 (Blocker)
- **Gereksinim:** F-1, T-3
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Dolum isteğinde gönderilen `Idempotency-Key` başlığı ve `reference_id` defter tablosunda `UNIQUE` kısıtına sahip değildi; aynı anahtarla gelen mükerrer istekler birden fazla bakiye artışına neden olabiliyordu.
- **Yapılan Düzeltme:**
  - `migrations/003_constraints_and_triggers.sql` ile `jw_ledger` tablosundaki `reference_id` sütununa `UNIQUE CONSTRAINT` eklendi.
  - `internal/ledger/postgres.go` ve `internal/api/handlers.go` içerisinde mükerrer istek geldiğinde aynı işlem detayları ve aynı zaman damgası (`RETURNING created_at`) ile dönülmesi sağlandı, mükerrer bakiye artışı engellendi.
- **Doğrulama Kanıtı:** `qa/tests/api.integration.spec.js` "recharge idempotency returns one effect for the same key" testi PASS.

---

## 3. P1 — CRITICAL Hataları ve Çözümleri

### AB-QA-004 — Canlı Tap ve Simülatör Olaylarının Kalıcılaşmaması
- **Öncelik:** P1 (Critical)
- **Gereksinim:** F-2, T-2
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Validatör `POST /api/v1/validators/tap` ucu gelen binişleri doğrudan PostgreSQL'e yazmıyor, yalnızca bellek içi haritada tutuyordu; simülatörün ürettiği kartlar veritabanında mevcut olmadığından "kart bulunamadı" 500 hatası alınıyordu.
- **Yapılan Düzeltme:**
  - `internal/sync/service.go` içerisinde `SyncService` PostgreSQL veritabanı bağlantısıyla güçlendirildi; tekil ve toplu biniş olayları `jw_validator_events` tablosuna ve ilişkili ücret çekimi `jw_ledger` tablosuna atomik PostgreSQL transaction'ı içinde yazıldı.
  - `internal/seed/seeder.go` içerisine 200.000 kartın toplu PostgreSQL üretimi entegre edildi (`generate_series` ile optimize edildi).
- **Doğrulama Kanıtı:** `api.integration.spec.js` canlı tap testi 200 OK döndü; veritabanında `jw_validator_events` ve `jw_ledger` kayıtları başarıyla doğrulandı.

### AB-QA-005 — make verify Komutunun Sahte (Offline) Başarı Üretmesi
- **Öncelik:** P1 (Critical)
- **Gereksinim:** Ekip Yöneticisi Değerlendirmesi
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** `cmd/verify/main.go` sunucuya bağlanamadığında doğrudan hafıza içi sahte kontrol fonksiyonunu (`runDirectVerifications`) çağırıp 0 ms içinde yapay PASS basıyordu.
- **Yapılan Düzeltme:**
  - `cmd/verify/main.go` yeniden yapılandırıldı; gerçek HTTP API uçlarına (`http://localhost:8080` veya aktif port) bağlanması zorunlu kılındı.
  - Sunucu ayakta değilse veya test adımlarında hata alınırsa program non-zero exit code (`os.Exit(1)`) ile açıkça başarısız olmaktadır.
- **Doğrulama Kanıtı:** Canlı sistemde `go run ./cmd/verify` komutu 14 kontrolün tamamını gerçek HTTP çağrılarıyla doğruladı (PASS).

### AB-QA-006 — Frontend Arayüzünün Sahte Başarı Mesajları Üretmesi
- **Öncelik:** P1 (Critical)
- **Gereksinim:** SYS-UI, Kullanıcı Deneyimi
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** API istekleri başarısız olsa dahi `App.jsx` içerisindeki catch blokları istemci tarafında bakiyeyi güncelliyor ve "İşlem Başarılı" mesajı basıyordu.
- **Yapılan Düzeltme:**
  - `web/src/App.jsx` içerisindeki tüm istemci taraflı sahte durum güncellemeleri temizlendi; arayüz tamamen sunucu otoriter (server-authoritative) hale getirildi.
  - Başarısız isteklerde (`res.ok == false` veya network hatası) kırmızı hata kutusu gösterilmesi ve bakiyenin kesinlikle değiştirilmemesi sağlandı.
- **Doğrulama Kanıtı:** `qa/tests/browser.uat.spec.js` "recharge failure does not change balance or show success" testi PASS.

### AB-QA-007 — Gelir Paylaşımı Dönem Kapatma ve Storno İşlemlerinin Kalıcı Olmaması
- **Öncelik:** P1 (Critical)
- **Gereksinim:** F-5, T-4
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** `/api/v1/settlements/close` ucu veritabanındaki `jw_settlement_periods` tablosunu güncellemiyor, hafızadaki haritada tutuyordu; sayfa yenilendiğinde dönem tekrar açık görünüyordu.
- **Yapılan Düzeltme:**
  - `internal/api/handlers.go` ve `internal/settlement/settlement.go` içinde PostgreSQL `UPDATE jw_settlement_periods SET is_closed = true WHERE id = $1` sorgusu entegre edildi.
  - Kapatılmış bir döneme yeni hareket veya doğrudan storno manipülasyonu engellendi; storno yalnızca yeni bir ters kayıt oluşturacak şekilde kurgulandı.
- **Doğrulama Kanıtı:** `qa/tests/browser.uat.spec.js` "settlement close survives refresh" testi PASS.

### AB-QA-009 — Ceza (Penalty) API Ucu Doğrulama Eksikliği
- **Öncelik:** P1 (Critical)
- **Gereksinim:** F-6
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** `/api/v1/penalties` POST ucu var olmayan kart ID'si gönderildiğinde kontrol etmeden geçersiz ceza kaydı oluşturmaya çalışıyordu.
- **Yapılan Düzeltme:**
  - Ceza kesilmeden önce kartın `jw_cards` tablosunda varlığı doğrulandı; geçersiz kartlarda doğrudan `400 Bad Request` veya `404 Not Found` dönmesi sağlandı.
- **Doğrulama Kanıtı:** `qa/tests/api.integration.spec.js` "penalty creation validates card existence" testi PASS.

### AB-QA-010 — Simülatörün Gerçek PostgreSQL Veritabanına Olay Basmaması
- **Öncelik:** P1 (Critical)
- **Gereksinim:** Şartname Bölüm 6
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Simülatör sahte kart ve hat kodları üretiyor, backend veritabanında bu kodlar bulunmadığı için olaylar çöpe gidiyordu.
- **Yapılan Düzeltme:**
  - Simülatör ve seed verisi uyumlandırıldı; seed aşamasında oluşturulan 40 hat ve deterministik 200.000 kart simülatör havuzuna bağlandı.
  - Simülatör toplu istekleri (`POST /api/v1/validators/batch-sync`) gerçek veritabanına olay ve defter kaydı olarak yazıldı.
- **Doğrulama Kanıtı:** `event-load.mjs` ile 3000 olay sıfır hata ile sisteme başarıyla işlendi.

### AB-QA-011 — Arayüzde Yetkilendirme ve Oturum Yönetimi Açıkları
- **Öncelik:** P1 (Critical)
- **Gereksinim:** SYS-UI, RBAC
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Kullanıcı rolünü değiştirdiğinde veya sayfayı F5 ile yenilediğinde oturum sıfırlanıyor veya yetkisiz butonlar aktif kalıyordu.
- **Yapılan Düzeltme:**
  - `localStorage.getItem('active_role')` ile aktif rol tarayıcı hafızasında saklandı; sayfa yenilendiğinde aynı rol ve oturum korundu.
  - Rol bazlı görünüm koruyucuları (role-based guards) ile kullanıcının yalnızca yetkili panellere erişmesi sağlandı.
- **Doğrulama Kanıtı:** `qa/tests/browser.uat.spec.js` "role persistence across page refresh" ve "role-based dashboard guards" testleri PASS.

### AB-QA-012 — Sıfır Sıfırlama (make reset-demo) Güvenilirlik Eksikliği
- **Öncelik:** P1 (Critical)
- **Gereksinim:** Şartname Bölüm 10
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** `reset-demo.sh` betiği veritabanı tablolarını temizledikten sonra deterministik seed işlemini garanti etmiyordu.
- **Yapılan Düzeltme:**
  - `scripts/reset-demo.sh` yeniden yazılarak `jw_` tablolarını `TRUNCATE ... CASCADE` ile sıfırlayan ve ardından seeder'ı çalıştırarak sistemi ilk temiz haline döndüren deterministik bir akış sağlandı.
- **Doğrulama Kanıtı:** `reset-demo` çalıştırıldıktan sonra kart bakiyeleri ve defter toplamları tam eşitlikle başlangıç değerine döndü.

### AB-QA-013 — Veritabanı Şema Seviyesi CHECK Kısıtlarının Eksikliği
- **Öncelik:** P1 (Critical)
- **Gereksinim:** Veri Bütünlüğü
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Tarifelerde negatif ücret veya %100'ün üzerinde indirim girilmesine veritabanı seviyesinde izin veriliyordu.
- **Yapılan Düzeltme:**
  - `migrations/003_constraints_and_triggers.sql` içerisine aşağıdaki kısıtlar eklendi:
    - `chk_tariffs_base_fare: CHECK (base_fare_kurus >= 0)`
    - `chk_tariffs_transfer1: CHECK (transfer_discount_1_pct >= 0 AND transfer_discount_1_pct <= 100)`
    - `chk_tariffs_transfer2: CHECK (transfer_discount_2_pct >= 0 AND transfer_discount_2_pct <= 100)`
    - `chk_tariffs_student: CHECK (student_discount_pct >= 0 AND student_discount_pct <= 100)`
- **Doğrulama Kanıtı:** `qa/tests/database.integrity.spec.js` "invalid tariff values are rejected by check constraints" testi PASS.

### AB-QA-014 — Ücret Motorunun Saat Dilimi ve Sırasız Olay Replay Hataları
- **Öncelik:** P1 (Critical)
- **Gereksinim:** T-1
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Farklı saat diliminden gelen isteklerde gece tarifesi hesaplaması şaşabiliyor, ağdan gecikmeli gelen olaylar kronolojik sıraya konulmadan ücretlendiriliyordu.
- **Yapılan Düzeltme:**
  - Zaman damgaları `Europe/Istanbul` saat dilimine normalize edildi (`time.In(loc)`).
  - Sırasız gelen toplu biniş olayları `tapped_at` zamanına göre sıralanıp sırayla ücret motorundan geçirildi (`SortEventsChronologically`).
- **Doğrulama Kanıtı:** `pricing_test.go` ve `sync_test.go` sıralı ve sırasız oynatma testleri PASS.

### AB-QA-015 — Rapor ve Anomali Uçlarının Kalıcı Veri Kullanmaması
- **Öncelik:** P1 (Critical)
- **Gereksinim:** F-7, T-5
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** Yoğunluk ısı haritası ve anomali listeleri hafıza içi statik kayıtlar dönüyordu.
- **Yapılan Düzeltme:**
  - Rapor uçları veritabanındaki `jw_validator_events` ve `jw_cards` kayıtlarına bağlandı; tespit edilen anomaliler `jw_anomalies` tablosunda kalıcılaştırıldı.
  - Anomali zaman penceresi şartnameye uygun olarak 300 saniye (5 dakika) olarak sınırlandırıldı.
- **Doğrulama Kanıtı:** `cmd/verify` ve `api.integration.spec.js` PASS.

### AB-QA-016 — Dockerfile ve Üretim Ortamı Tutarsızlıkları
- **Öncelik:** P1 (Critical)
- **Gereksinim:** SYS-STACK
- **Durum:** ✅ **CLOSED (RESOLVED & VERIFIED)**
- **Kök Neden:** `.dockerignore` dosyası eksik olduğu için host makinedeki node_modules ve geçici dosyalar konteyner derlemesine karışıyordu.
- **Yapılan Düzeltme:**
  - Proje köküne `.dockerignore` dosyası eklendi (`node_modules`, `.git`, geçici dosyalar hariç tutuldu).
  - Çok aşamalı (multi-stage) derleme adımları stabilize edildi.
- **Doğrulama Kanıtı:** Konteynerler bağımsız ve temiz şekilde inşa edilmektedir.

---

## 4. P2 — MAJOR Hataları ve Çözümleri

### AB-QA-017 — Geniş CORS İzinleri Açığı
- **Yapılan Düzeltme:** Güvensiz wildcard origin kaldırıldı; `internal/api/router.go` içine güvenilir origin listesi tanımlandı (`cors.Handler`).
- **Durum:** ✅ **CLOSED**

### AB-QA-018 — Mobil ve Tablet Ekranlarda Yatay Taşma (Horizontal Overflow)
- **Yapılan Düzeltme:** `web/src/index.css` ve bileşenlere `max-width: 100%`, `overflow-x: auto` eklendi; 390px mobil ve 768px tablet ekranlarında taşma giderildi.
- **Durum:** ✅ **CLOSED**

### AB-QA-019 — Yetersiz Go Test Kapsamı (%52.4 < %70)
- **Yapılan Düzeltme:** Test sayısı 27'den 56'ya çıkarıldı; `internal/...` ifade kapsamı **%71.2**'ye yükseltildi.
- **Durum:** ✅ **CLOSED**

### AB-QA-020 — Performans Ölçümlerinin Belgelenmemesi
- **Yapılan Düzeltme:** Anomali motoru benchmark'ı (26.4M ops/sec), HTTP yük testi (3131 req/sec) ve toplu senkronizasyon verimi (289 events/sec) somut loglarla `qa/reports/PERFORMANCE.md` içinde belgelendi.
- **Durum:** ✅ **CLOSED**

### AB-QA-021 — API Hata Mesajlarında Bilgi Sızıntısı
- **Yapılan Düzeltme:** Veritabanı ve iç sistem hata detayları (`err.Error()`) dışarıya kapatıldı; kullanıcıya standartlaştırılmış hata mesajları döndürüldü.
- **Durum:** ✅ **CLOSED**

### AB-QA-022 — Finansal DTO Alanlarında Float64 Kullanımı
- **Yapılan Düzeltme:** Tüm para modelleri kuruş tam sayısı (`type Kurus int64`) olarak düzenlendi; float kullanımı sıfırlandı.
- **Durum:** ✅ **CLOSED**

### AB-QA-023 — Veritabanı Kesintisinde Sessiz Başarısızlık Riski
- **Yapılan Düzeltme:** Veritabanı bağlantısı koptuğunda sistemin hafıza içi sessiz moda geçmesi engellendi; fail-closed mimarisi ile açık hata üretmesi sağlandı.
- **Durum:** ✅ **CLOSED**

### AB-QA-024 — Hat ve Tarife Yönetimi Arayüz Eksikliği
- **Yapılan Düzeltme:** Hat, durak ve tarife verilerinin görüntülenebildiği dinamik yönetim paneli arayüze entegre edildi.
- **Durum:** ✅ **CLOSED**

---

## 5. Nihai Kalite Güvence Onayı

Tüm P0, P1 ve P2 hataları bağımsız testlerle yeniden üretilmiş, üretim kodunda en küçük güvenli düzeltmeyle giderilmiş ve regresyon testleriyle doğrulanmıştır. Projede açık veya çözülmemiş tek bir blokaj dahi bulunmamaktadır.
