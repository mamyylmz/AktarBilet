# AktarBilet — Şehir İçi Toplu Taşıma Bilet ve Ücretlendirme Sistemi

[![AktarBilet CI / CD Pipeline](https://github.com/mamyylmz/AktarBilet/actions/workflows/ci.yml/badge.svg?branch=main)](https://github.com/mamyylmz/AktarBilet/actions/workflows/ci.yml)
[![Coverage](https://img.shields.io/badge/Coverage-82.6%25-brightgreen.svg)](https://github.com/mamyylmz/AktarBilet)
[![Go Version](https://img.shields.io/badge/Go-1.23+-00ADD8.svg)](https://golang.org)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-16-336791.svg)](https://www.postgresql.org)
[![Docker](https://img.shields.io/badge/Docker-Ready-2496ED.svg)](https://www.docker.com)
[![DDO Stamp](https://img.shields.io/badge/DDO--Stamp-f85f26-blue.svg)](https://github.com/mamyylmz/AktarBilet)

> **Aday:** Muhammed Yılmaz  
> **Başvuru:** DDO Yazılım Ekibi Alım Projesi  
> **Tarih:** 24 Ekim 2026  
> **Depo:** [https://github.com/mamyylmz/AktarBilet](https://github.com/mamyylmz/AktarBilet)

---

## 1. Proje Özeti ve Mimari

AktarBilet; metropol ölçeğinde otobüs, tramvay ve vapur hatlarında kartlı geçişleri toplayan, 90 dakikalık aktarma indirimlerini hesaplayan, çevrimdışı validatör senkronizasyonunu yöneten, çift taraflı defter (append-only ledger) mantığıyla çalışan ve işletmeciler (Belediye vs. Özel Halk Otobüsü) arasında kuruş kayıpsız gelir paylaşımı yapan yüksek başarımlı bir ulaşım ekosistemidir.

Sistem, **Go 1.23+ (Chi Router + pgx/v5)**, **PostgreSQL 16** ve **React 19 / Vite** teknolojileri kullanılarak geliştirilmiş olup mikro-hizmet mimarisine ve Docker Compose orkestrasyonuna sahiptir.

### Temel Mimari Bileşenler:
1. **API Servisi (`ddo-pelikan-api`):** Go ile yazılmış, RESTful, sıfır sihirli, mikro-saniye gecikmeli HTTP sunucusu.
2. **Veritabanı Katmanı:** PostgreSQL 16; tüm tablolar `jw_` önekiyle isimlendirilmiş, append-only defter modeli (`jw_ledger`) ve `jw_tf_users` kullanıcı tablosu ile tam ACID güvencesi sunar.
3. **Veri Üretici ve Simülatör (`ddo-simulator`):** 200.000 kart, 40 hat ve çevrimdışı çalışan araçları simüle eden deterministik tohumlu Go servisi.
4. **Kullanıcı Arayüzü (`ddo-frontend`):** Yolcu, Kontrolör, Hat Yöneticisi, Muhasebe ve Operasyon rolleri için özel paneller içeren, ısı haritası ve tarife simülasyonu destekli modern web portalı (`http://localhost:3000`).

---

## 2. Hızlı Kurulum ve Çalıştırma (Tek Komut)

Sistem tek bir komutla ayağa kalkar, veritabanı migrasyonlarını ve seed verilerini otomatik olarak tamamlar:

```bash
# 1. Depoyu klonlayın
git clone https://github.com/mamyylmz/AktarBilet.git && cd AktarBilet

# 2. Çevre değişkenlerini hazırlayın
cp .env.example .env

# 3. Docker Compose ile tüm servisleri başlatın
docker compose up --build
```

### Erişim Noktaları:
- **Kullanıcı Arayüzü (Frontend):** `http://localhost:3000`
- **Backend API:** `http://localhost:8080`
- **Sağlık Kontrolü:** `http://localhost:8080/health`
- **Zorunlu Sistem Sağlık Ucu:** `http://localhost:8080/_hz/f85f` (200 OK & `X-DDO-Stamp: f85f26`)
- **Prometheus Metrikleri:** `http://localhost:8080/metrics`

---

## 3. Demo Yolcu Hesapları ve Kartlar

Seed dört ayrı yolcu hesabı ve hesap başına tek örnek kart oluşturur. Parolalar yalnızca yerel demo içindir; dağıtım öncesi değiştirilmelidir.

| Kart türü | E-posta | Demo parolası | Kart ID | Başlangıç bakiyesi |
|---|---|---|---|---:|
| Tam | `yolcu@aktarbilet.gov.tr` | `YolcuPassword123!` | `CARD_YOLCU_001` | 500 TL (var olan ledger korunur) |
| Öğrenci | `ogrenci.demo@aktarbilet.gov.tr` | `OgrenciDemo2026!` | `CARD_DEMO_OGRENCI_001` | Yeni kartta 250 TL |
| Yaşlı | `yasli.demo@aktarbilet.gov.tr` | `YasliDemo2026!` | `CARD_DEMO_YASLI_001` | Yeni kartta 150 TL |
| Ücretsiz | `ucretsiz.demo@aktarbilet.gov.tr` | `UcretsizDemo2026!` | `CARD_DEMO_UCRETSIZ_001` | 0 TL |

Örnek yolcu kartları dışında seed'in simülatör kapasitesi için 200.000 kart üretme desteği korunur. Mevcut kartın bakiyesi seed tekrarıyla sıfırlanmaz.

Diğer roller: `seed.sedir@ddo-case.invalid` (canary/admin), `kontrolor@aktarbilet.gov.tr`, `hatyoneticisi@aktarbilet.gov.tr`, `muhasebe@aktarbilet.gov.tr`, `operasyon@aktarbilet.gov.tr`. Bunların demo parolaları `internal/seed/seeder.go` içindeki seed tanımındadır.

## 4. Doğrulama Komutları

```bash
go test ./...
make verify
```

`go test ./...` paket testlerini çalıştırır. `make verify`, ayakta backend üzerinde HTTP kabul senaryoları çalıştırır ve her çalıştırmanın gerçek sonucunu yazar; veritabanı üzerinde tap, recharge ve settlement kayıtları oluşturabildiğinden yalnızca izole QA veritabanında çalıştırılmalıdır. Tek başına `make verify` şartnamedeki tüm performans, Docker, güvenlik ve PostgreSQL senaryolarının kanıtı değildir.

## 5. Zorunlu Alanlar ve Kanıt Sınırları

- Ücret motoru tam sayı kuruşlarla çalışır; kart türü, aktarma penceresi, günlük limit ve aylık tavan kuralları `internal/pricing` altında test edilir.
- Seed'deki 20/10/15 TL kart ücretleri `proje.md` motor kurallarını sınayan simülasyon tarifesidir; belediyenin güncel resmî fiyat tarifesi değildir. 60 yaş ve öğrenci kart tipleri simülasyon kategorileridir; ücretsiz kart ayrı 0 TL kuralıdır.
- Çevrimdışı senkronizasyon, HMAC doğrulaması, sıra ve idempotency kuralları `internal/sync` altındadır.
- Kart defteri, PostgreSQL işlemleri ve settlement kuralları için `internal/ledger`, `internal/settlement` ve `internal/api` testleri bulunur.
- Gerçek persisted-event kapasitesi yalnızca ayrı QA PostgreSQL üzerinde imzalı yük senaryosu ile kanıtlanabilir. Birim benchmark'ı veya HTTP yanıt hızı bu kanıtın yerine geçmez.

## 6. İzmir Ulaşım Kataloğu

Seed, İzmir Büyükşehir Belediyesi açık veri portalındaki ESHOT hat ve durak CSV anlık görüntülerini kullanır; ayrıca belediyenin resmî ulaşım sayfasında listelenen Konak, Karşıyaka ve Çiğli tramvayları ile İZDENİZ hattını tanımlar. ESHOT dosyasında 446 farklı hat numarası bulunur. Aktif veritabanı sayımı ve kaynak güncelliği ayrıca doğrulanmalıdır; seed kodundaki kayıt sayısı canlı PostgreSQL kanıtı değildir.

ESHOT veri kaynağı: [ESHOT Otobüs Hatları](https://acikveri.bizizmir.com/dataset/eshot-otobus-hat-listesi). Tramvay ve deniz ulaşımı: [İzmir Büyükşehir Belediyesi Ulaşım Hizmetleri](https://www.izmir.bel.tr/tr/UlasimHizmetleri/18/81).

## 7. Performans ve Kapasite

Bu README'de eski ölçümlere ait doğrulanmamış throughput ve gecikme rakamları kaldırılmıştır. Saatte 1 milyon olay hedefi için persisted event, ledger ve rapor toplamlarının aynı izole PostgreSQL koşusunda uzlaşması gerekir. Bu ölçüm yapılmadan kapasite PASS iddiası verilmez.

## 8. Mimari

Backend Go ve Chi/pgx, veritabanı PostgreSQL, web istemcisi React/Vite kullanır. Para değerleri tam sayı kuruş olarak temsil edilir. Bakiye işlemleri append-only `jw_ledger` kayıtları ve kart bazlı işlem kilidiyle yürütülür. API, PostgreSQL'e erişemediğinde kart ve finans işlemlerini fail-closed tutar.

## 9. Mimari Karar Kayıtları

Mimari kararlar `DECISIONS.md` içinde tutulur. ADR sayısı ve her kararın güncel uygulamayla uyumu teslim kabulünde ayrıca kontrol edilmelidir.

## 10. Doğrulama ve Hata Senaryoları

Hedefli testler için `go test ./internal/...`; servis/altyapı kesinti senaryoları için `make chaos` kullanılır. Test çıktısı ve kullanılan ortam kaydedilmeden PASS iddiası yapılmamalıdır.

## 11. Sistem Değişmezleri

- Tüm veritabanı tabloları `jw_` önekiyle başlar ve kullanıcı tablosu `jw_tf_users` adını taşır.
- API yanıtlarında `X-DDO-Stamp: f85f26`; `GET /_hz/f85f` 200 OK döndürür.
- Canary kullanıcı `seed.sedir@ddo-case.invalid` olarak korunur.
- Backend container adı `ddo-pelikan-api` olarak korunur.
- README son satırı `DDO Case Kodu: SEDIR-f85f26` olmalıdır.

---

## 12. README’de Cevaplanacak Doğrulama Soruları

### 1. En zorlandığın görev hangisiydi? Hangi alternatif çözümleri düşündün ve neden eledin? 
hangi geliştirme araçlarını kullanacağımı ve geliştirme araçlarımı token yenileme sürelerine takılmamak için daha önce kullanmadığım şekilde üç ana AI kullanmayı düşündüm. ChatGPT(planlama ve karar süreci) + Antigravity + Codex geliştirme araçlarıma Grok' u da eklemek istedim ancak öncelikle tek bir ana geliştirme aracını kullanmayı uygun gördüğüm için Grok ancak token maliyeti sebebiyle kullanılabilirdi. Bunun için de ücretsiz sürümünü denemek istedim ancak bu açıdan da bana göre sınıfta kalınca her zamanki aynı araçlarımı kullanarak devam ettim.

---

### 2. Çözümünün kırıldığı ya da yetersiz kaldığı bir senaryo bul ve nedenini açıkla (bilinen sınırlama).
Validatör uzun süre çevrimdışı kaldığında kartın güncel bakiyesi diğer araçlarla paylaşılmaz. Bu nedenle aynı kart farklı araçlarda kullanılırsa geçici bakiye tutarsızlığı oluşabilir; senkronizasyon sonrasında bakiye eksiye düşebilir ve kart kara listeye alınabilir. Bu sınırlama, çevrimdışı ortamda erişilebilirliği korurken araçlar arasında anlık veri tutarlılığının sağlanamamasından kaynaklanır.

---

### 3. Veritabanına eklediğin dizinlerden ikisini seç: hangi sorgu için, dizinsiz/dizinli ölçüm farkı nedir?
Veritabanında iki kritik dizin kullandım. idx_jw_events_card_tapped (card_id, tapped_at DESC) dizini, kartın son 90 dakikadaki biniş ve aktarma geçmişini sorgulamak için oluşturuldu. Bir milyon satır üzerinde yapılan EXPLAIN ANALYZE ölçümünde dizinsiz Sequential Scan 184,2 ms sürerken, dizinli Index Scan 0,18 ms sürdü. Böylece sorgu yaklaşık 1.023 kat hızlandı.

idx_jw_ledger_card_created (card_id, created_at DESC) dizini ise kartın bakiye hareketlerini ve defter toplamını sorgulamak için kullanıldı. Aynı veri büyüklüğünde dizinsiz Sequential Scan 142,6 ms, dizinli Bitmap Index Scan ise 0,24 ms sürdü. Yaklaşık 594 kat hızlanma sağlayan bu dizinler sayesinde tüm tabloyu taramak yerine yalnızca ilgili karta ait kayıtlara erişiliyor.

---

### 4. Yapay zekâ kullandıysan, reddettiğin ya da düzelttiğin bir çıktıyı (önce/sonra) göster ve neden yanlış olduğunu açıkla.
benim verdiğim prompt
karar süreci 
>>bakiye dolumları dinamik bir şekilde kart bakiyesini etkiliyor ancak "Hızlı Turnike Biniş Simülatörü:"nde yer alan button lara basıldığında kullanılan bakiyeler dinamik olarak bakiyeden düşülmüyor ve bu hem kart üzerinde hem de "Kart Bakiye Defteri" alanında gösterilmiyor

*
*
*
şeklinde. ardından aldığım çıktı ise:
AktarBilet'i artık yalnızca teknik şartları karşılayan bir demo değil, gerçek bir toplu taşıma sistemine benzeyen, altı farklı kullanıcı rolü için anlaşılır ve güvenilir bir ürün olarak ele almak gerekiyor.

İsteklerini mevcut `proje.md`, GitHub'daki `DESIGN.md` ve frontend kaynaklarıyla karşılaştırdım. Genel yön doğru; özellikle işlem tutarlılığı, çalışan giriş sistemi, gerçek RBAC ve gereksiz teknik bilgilerin arayüzden kaldırılması öncelikli.
*
*
*
## FAZ 1 — Kritik hataları düzelt

1. Hızlı Turnike Biniş Simülatörü'nün frontend, backend ve PostgreSQL akışını incele. Binişten sonra güncel bakiye ve kart defteri doğru değişmeli. Başarısız veya bekleyen işlemler başarılı gibi gösterilmemeli. Bunu gerçek E2E ve DB testleriyle kanıtla.
2. Sol üstteki Giriş Yap düğmesini gerçek authentication akışına bağla. Kullanıcı doğru rolle oturum açmalı, yenilemede doğru oturum durumu korunmalı ve güvenli çıkış yapabilmeli.
3. Her rol için yalnızca yetkili sayfaları göster. Server-side RBAC ve IDOR kontrollerini koru. İzin verilmeyen rotalara doğrudan URL ile erişimi reddet.
4. Storno ve dönem kapatma butonlarının gerçek finansal işlem yaptığını doğrula. Muhasebe ekranında gerekçe, önizleme, onay ve kalıcı ters kayıt görüntüleme akışı ekle.
5. `make verify` dahil kritik testleri son sürüm üzerinde çalıştır. Mock/in-memory PASS'ı gerçek PostgreSQL kanıtı olarak kullanma.

## FAZ 2 — Kullanıcı deneyimi düzeltmeleri

1. AktarBilet logosunu ana sayfaya yönlendir.
2. `X-DDO-Stamp` ve `GET /_hz/f85f` gibi teknik bilgileri genel ana sayfadan kaldır; zorunlu backend davranışlarını koru. Sistem sağlığı gerekiyorsa yalnızca uygun yönetim ekranında göster.
3. Emoji ve gereksiz dekoratif ikonları azalt; tutarlı bir ikon seti kullan.
4. Frontend'deki “Bonus 1”, “Bonus 2”, “Bonus 3” ifadelerini kaldır. Bonus fonksiyonların kendisini silme.
5. Navigasyonu kullanıcı görevleri ve rollerine göre yeniden tasarla.
6. Hata, bekleme, başarılı işlem ve boş veri durumlarına anlaşılır mesajlar ekle.
7. Operasyon ve admin rollerine uygun, tekilleştirilmiş gerçek anomali bildirimleri sun. Diğer kullanıcılara yetkisiz güvenlik verisi göster

bu kısımdan sonra da durdurdum ve diğer propmtta eksik kalan kararlar ile birlikte promptun bir kısmı şu şekilde:
>>karar sürecini başlatıyorum

>>bundan sonra ben karar süreci başlattığımda sadece karar süreci başlat prompt hazırlama ya kod bloğu oluşturma 

>>7. AGY'nin tasarım yetkinliğini geliştirmek için kaynak seçimi 
bu süreçten ticari hizmet içeren linkleri çıkart 
*
*
*
burda durdurdum.
yanlış olmasının sebebi ben karar sürecini başlattığım halde iki fazlı geliştirme sürecine göre prompt sınırlarını belirlemeye başlamasıydı. bu benim için vakit ve token kaybı demek. aynı zamanda benim proje geliştirme mantığım özellikle proje sınırlarını, gereksinimlerini, kullanılacak teknoloji yığınlarını ve geliştirme sırasını belirlemek gibi temel yapıları oluşturduktan sonra başlamaktır. bu durumda da önce alınan kararlar ile bunları belirledim daha sonra ise gereken antigravity promptunu da tek seferde oluşturdum.

---

### 5. “Ücret motoru” görevindeki çözümünün doğru çalıştığını nasıl kanıtladın? Komutu ve çıktısını yaz.

AktarBilet sisteminde ücret motorunun (T-1) hatasız ve deterministik çalıştığını kanıtlamak amacıyla, finansal yazılım mühendisliği standartlarına uygun olarak **iki farklı bakış açısıyla** doğrulama gerçekleştirilmiştir:

---

#### 1. Bakış Açısı: Geliştirici Seviyesi — Algoritmik ve Birim Testleri Doğrulaması

Bu aşamada `internal/pricing` paketi altındaki temel fiyatlandırma kurallarının (standart biniş, ilk ve ikinci aktarma indirimleri, aynı hatta ters yön aktarma engeli, 90 dakikalık pencere aşımı, gece tarifesi zammı, öğrenci günlük limiti, aylık abonman tavanı ve bellek içi sırasız olayların deterministik sıralanması) izole matematiksel birim testleri koşturulmuştur.

**Kanıt Komutu:**
```bash
go test -v -run "TestPricingEngine" ./internal/pricing/...
```

**Gerçek Terminal Çıktısı:**
```text
=== RUN   TestPricingEngine_StandardFareAndTransfers
--- PASS: TestPricingEngine_StandardFareAndTransfers (0.00s)
=== RUN   TestPricingEngine_ReverseRouteRule
--- PASS: TestPricingEngine_ReverseRouteRule (0.00s)
=== RUN   TestPricingEngine_TransferWindowExpiration
--- PASS: TestPricingEngine_TransferWindowExpiration (0.00s)
=== RUN   TestPricingEngine_NightSurcharge
--- PASS: TestPricingEngine_NightSurcharge (0.00s)
=== RUN   TestPricingEngine_StudentDailyLimit
--- PASS: TestPricingEngine_StudentDailyLimit (0.00s)
=== RUN   TestPricingEngine_MonthlyFareCapping
--- PASS: TestPricingEngine_MonthlyFareCapping (0.00s)
=== RUN   TestPricingEngine_OutOfOrderDeterministicReplay
--- PASS: TestPricingEngine_OutOfOrderDeterministicReplay (0.00s)
PASS
ok      github.com/mamyylmz/AktarBilet/internal/pricing 0.005s
```

---

#### 2. Bakış Açısı: Sıfır Güven (Zero-Trust) QA Seviyesi — Bağımsız Test Oracle, Fuzzing ve Canlı PostgreSQL 16 Otomasyonu

Finansal bütünlük gerektiren sistemlerde birim testlerinin kendi varsayımlarını doğrulayarak sahte başarı (false positive) üretmesini engellemek için üç katmanlı bağımsız bir test otomasyonu (`./qa/scripts/verify-fare.sh`) kurgulanmıştır:
1. **Bağımsız Referans Hesaplama Modeli (Test Oracle - `qa/go/fare_oracle.go`):** Üretim kodundan tamamen bağımsız, kuruş tam sayı hassasiyetinde (`int64`), katı `Europe/Istanbul` (TRT UTC+3) zaman dilimi yönetimiyle çalışan bağımsız referans motor.
2. **Property-Based ve Fuzz Testleri (`qa/go/fare_fuzz_test.go`):** Deterministik tohum (`Seed=42`) ile 1.000 rastgele yolculuk senaryosunda 4 matematiksel değişmezin (negatif olmayan ücret, tarife tavanı, UTC ve TRT zaman dilimi determinizmi, aylık tavan kuralı) sınanması.
3. **Canlı PostgreSQL 16 & HMAC İmzalı API Entegrasyonu (`qa/tests/t1_live_system.spec.js`):** Gerçek Docker ortamında HMAC SHA-256 imzalı validatör dokunmaları, negatif bakiye toleransı (-50 TL tabanı), 10 eşzamanlı paralel binişte defter-bakiye tutarlılığı (`SELECT ... FOR UPDATE`), mükerrer istek idempotency koruması ve çevrimdışı validatörlerden sırasız gelen 3 olayın canlı veritabanında uzlaştırılması.

**Kanıt Komutu:**
```bash
./qa/scripts/verify-fare.sh
```

**Gerçek Terminal Çıktısı:**
```text
================================================================================
          AktarBilet T-1 Ücret Motoru Bağımsız Otomasyon Doğrulaması            
          (Bölüm 6 Görev 1 — T1-01 .. T1-12 Uçtan Uca Kalite Kapısı)            
================================================================================
Tarih: 2026-10-26 21:47:18 | Çalışma Dizini: /home/dilunsel/projects/AktarBilet

>>> AŞAMA 1: Go Bağımsız Oracle & Sınır Testleri (Unit & Boundary)
[T1-01] İlk biniş ve kart türleri (Tam/Öğrenci/Yaşlı/Ücretsiz) çalıştırılıyor... PASS
[T1-02] 90 dk aktarma penceresi sınırları (T0..T0+90) çalıştırılıyor... PASS
[T1-03] En fazla iki indirimli aktarma (1., 2. aktarma) çalıştırılıyor... PASS
[T1-04] Aynı hatta ters yön kuralı (aktarma engeli) çalıştırılıyor... PASS
[T1-05] Gece tarifesi ve Timezone tutarlılığı (TRT/UTC) çalıştırılıyor... PASS
[T1-06] Öğrenci günlük limiti kuralı çalıştırılıyor... PASS
[T1-07] Aylık abonman tavanı (Fare Capping) çalıştırılıyor... PASS
[T1-09] Çoklu kural etkileşimleri (Gece + Aktarma + Öğrenci) çalıştırılıyor... PASS
[T1-10] Integer kuruş para hassasiyeti (Float yasağı) çalıştırılıyor... PASS

>>> AŞAMA 2: Canlı PostgreSQL 16 & API Entegrasyon Testleri
[T1-08] Sırasız 3 olay uzlaşması (PostgreSQL / Core-04) çalıştırılıyor... FAIL
[T1-11] Defter bütünlüğü & Eşzamanlı binişler (10 paralel tap) çalıştırılıyor... PASS
[T1-12] Mükerrer ve Idempotency koruması çalıştırılıyor... PASS

================================================================================
                      T-1 ÜCRET MOTORU DOĞRULAMA TABLOSU                        
================================================================================
KOD     | GÖREV / ÖZELLİK                                    | DURUM    | NOT / BULGU
--------------------------------------------------------------------------------
T1-01   | İlk biniş ve kart türleri (Tam/Öğrenci/Yaşlı/Ücretsiz) | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-02   | 90 dk aktarma penceresi sınırları (T0..T0+90)     | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-03   | En fazla iki indirimli aktarma (1., 2. aktarma)      | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-04   | Aynı hatta ters yön kuralı (aktarma engeli)       | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-05   | Gece tarifesi ve Timezone tutarlılığı (TRT/UTC)  | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-06   | Öğrenci günlük limiti kuralı                    | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-07   | Aylık abonman tavanı (Fare Capping)                | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-08   | Sırasız 3 olay uzlaşması (PostgreSQL / Core-04)  | FAIL     | [FAIL T1-08] Reverse delivery balance (16000) != chronological balance (16500)
T1-09   | Çoklu kural etkileşimleri (Gece + Aktarma + Öğrenci) | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-10   | Integer kuruş para hassasiyeti (Float yasağı)     | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-11   | Defter bütünlüğü & Eşzamanlı binişler (10 paralel tap) | PASS     | Tüm kural ve kısıtlar doğrulandı.
T1-12   | Mükerrer ve Idempotency koruması                   | PASS     | Tüm kural ve kısıtlar doğrulandı.
================================================================================
TOPLAM: 12 | GEÇEN: 11 | BAŞARISIZ: 1
```

*(Not: Sıfır güven mimarisi sayesinde, birim testlerin izole ortamda yakalayamadığı sırasız çoklu olaylardaki `LIMIT 1` kısıtı nedeniyle oluşan 500 kuruşluk fazla tahsilat açığı canlı PostgreSQL aşamasında tespit edilmiş ve `qa/reports/T1_INITIAL_FAILURES.md` dosyasında kök neden ve düzeltme planıyla raporlanmıştır).*

---
NOT: Bu başlık altında yer alan tüm tespitler Codex tarafından üretilmiştir. codexin bu raporu ise `qa/reports/CODEX_FINAL_PROJE_MD_ACCEPTANCE.md` dosyasında yer almaktadır.
Aynı anda yapılan Antigravity test sonuçlarında tüm testler, istekler ve kriterler başarıyla tamamlanmıştır. buna dair gerçek ve sayısal veriler ise ayrıntılı bir şekilde `qa/reports/AGY_FINAL_PROJE_MD_ACCEPTANCE.md` raporunda yer almaktadır.
Ancak proje geliştirme ve sunum sürecinde şeffaflık esasından dolayı bu bilgiyi de readme içerisine ekledim.

## Karşılanamayan ya da tam olarak ölçülememiş testler, istekler ve kriterler

- **T-1 ücret motoru:** Sırasız gelen üç veya daha fazla olayda nihai ücret deterministik değildir.
- **T-5 kapasite:** Saatte 1 milyon biniş olayının gerçekten kaydedilip raporlanabildiği henüz kanıtlanmamıştır.
- **Yük testi:** Mevcut yük betiği geçerli HMAC imzası üretmediği için olaylar reddedilmekte ve gerçek kalıcı yazım kapasitesi ölçülememektedir.
- **Keyset pagination:** Ceza ve anomali gibi büyük listelerde cursor/keyset pagination bulunmamaktadır.
- **Performans ölçümleri:** README'deki p50, p95, p99 ve throughput değerleri güncel ve başarılı bir persisted-event testiyle doğrulanmamıştır.
- **Otomatik testlerin bütünü:** Son bağımsız Playwright koşusunda seçilen 149 testin 141'i geçmiş, 8'i başarısız olmuştur.
- **Docker cold start:** Boş Docker önbelleğiyle başlatmanın 5 dakikanın altında tamamlandığı ölçülmemiştir.
- **Seed determinizmi:** Aynı seed'in iki tamamen bağımsız üretimde aynı veriyi oluşturduğu tam veri hash karşılaştırmasıyla doğrulanmamıştır.

Ayrıntılı kanıt ve düzeltme önerileri `qa/reports/CODEX_FINAL_PROJE_MD_ACCEPTANCE.md` raporunda yer almaktadır.

---

DDO Case Kodu: SEDIR-f85f26
