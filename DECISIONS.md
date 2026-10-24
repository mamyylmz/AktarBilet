# AktarBilet Mimari Karar Kayıtları (Architecture Decision Records - ADR)

Bu belge, AktarBilet sisteminin mimari omurgasını oluşturan temel teknik kararları, değerlendirilen alternatifleri ve seçilme gerekçelerini içerir.

---

## ADR-001: Go (Chi) ve pgx/v5 ile Standart Uyumlu, Yüksek Verimli Mikro/Modüler Backend Mimarisi

- **Durum:** Kabul Edildi (Accepted)
- **Tarih:** 2026-10-24
- **Kapsam:** Backend HTTP ve Veritabanı Erişim Katmanı

### Bağlam (Context)
Toplu taşıma bilet sistemlerinde saniyede binlerce validatör olayı işlenmekte ve gecikme süresinin (latency) p99 seviyesinde < 15ms olması beklenmektedir. Ağır ORM'ler (GORM vb.) karmaşık SQL optimizasyonlarını, kilit kontrolünü ve batch yüklemeleri örtmekte ve performans darboğazı yaratabilmektedir.

### Karar (Decision)
Backend çatısı olarak standart `net/http` ile %100 uyumlu, sıfır harici bellek yükü getiren **Chi Router**; veritabanı sürücüsü olarak ise doğrudan PostgreSQL binary protokolünü destekleyen, bağlantı havuzu (`pgxpool`) yönetimi üstün ve bellek tahsisatı minimum olan **`jackc/pgx/v5`** seçilmiştir.

### Değerlendirilen Alternatifler
1. *Gin / Fiber + GORM:* Popüler olmakla birlikte, GORM'un reflection maliyeti yüksek concurrency altında bellek tüketimini artırmakta ve `SELECT ... FOR UPDATE` kilitlerinde istenmeyen davranışlara yol açabilmektedir.
2. *Go Standart `database/sql` + `lib/pq`:* `lib/pq` artık bakım modundadır ve pgx/v5'in binary format, copy protocol ve statement caching avantajlarını sunmamaktadır.

### Sonuçlar
- **Olumlu:** Mikrosaniye seviyesinde yönlendirme, transaction seviyesinde tam kontrol, pgx.Batch ile yüksek throughput.
- **Olumsuz:** SQL sorgularının elle yazılması veya sqlc benzeri kod üretim araçlarına ihtiyaç duyulması (kontrol mimaride kalmaktadır).

---

## ADR-002: Append-Only Defter (Ledger) Modeli ve Pessimistic Locking (SELECT ... FOR UPDATE)

- **Durum:** Kabul Edildi (Accepted)
- **Tarih:** 2026-10-24
- **Kapsam:** Görev 3 (Kart Bakiye Defteri) ve Finansal Tutarlılık

### Bağlam (Context)
Aynı kartın eşzamanlı olarak iki farklı validatörde okutulması (örneğin iki kapılı bir metro turnikesinde milisaniyelik farkla) klasik `UPDATE jw_cards SET balance = balance - 15` yaklaşımında kayıp güncelleme (lost update) yarış durumuna (race condition) neden olur.

### Karar (Decision)
1. **Append-Only Defter:** Kart bakiyesi doğrudan güncellenen bir sayı değil, `jw_ledger` tablosundaki tüm hareketlerin (dolum, biniş, iade, ceza, düzeltme) cebirsel toplamıdır (`balance = SUM(amount)`).
2. **Kart Bazlı Serileştirme:** Kart üzerinde işlem yapılırken ilgili kart kaydı `SELECT ... FOR UPDATE` ile transaction süresince kilitlenir. Eşzamanlı gelen ikinci işlem kilit bırakılana kadar kuyrukta bekler ve güncel bakiye üzerinden reddedilir veya onaylanır.

### Değerlendirilen Alternatifler
1. *Optimistic Locking (Versiyon Numarası ile):* Çakışma durumunda işlemin istemciye hata dönüp yeniden denenmesini gerektirir. Validatör turnikesinde yolcunun 1 saniyeden az sürede geçmesi gerektiğinden retry fırtınası yolcuyu bekletir.
2. *Redis Tabanlı Dağıtık Kilit (Redlock):* Ek bir altyapı bağımlılığı ve ağ atlaması (network hop) getirir; PostgreSQL transaction bütünlüğünü ikiye böler.

### Sonuçlar
- **Olumlu:** Sıfır yarış durumu, %100 kanıtlanabilir bakiye tutarlılığı, tam denetlenebilirlik (auditability).
- **Olumsuz:** Kilit tutma süresi minimum tutulmalıdır (transaction içinde harici HTTP isteği yapılmaz).

---

## ADR-003: Deterministik Sırasız Olay İşleme ve Çevrimdışı İdempotansi Stratejisi

- **Durum:** Kabul Edildi (Accepted)
- **Tarih:** 2026-10-24
- **Kapsam:** Görev 1 (Ücret Motoru) ve Görev 2 (Çevrimdışı Validatör Senkronizasyonu)

### Bağlam (Context)
Toplu taşıma araçları tünellerde veya kırsal bölgelerde internetsiz kalır. Araç merkeze bağlandığında son 2 saatin biniş olayları topluca, sırasız ve bazen mükerrer (duplicate) olarak sisteme iletilir. Bu durum aktarma hakkının (90 dk penceresi) yanlış hesaplanmasına veya mükerrer ücret tahsiline yol açabilir.

### Karar (Decision)
1. Her validatör olayı `validator_id` ve araç tarafından artan `sequence_no` ile damgalanır. `jw_validator_events` tablosunda `UNIQUE(validator_id, sequence_no)` ve `Idempotency-Key` kısıtıyla mükerrer kayıtlar anında elenir.
2. Ücret motoru, olay işleme anındaki sistem saatini değil, validatörün kaydettiği `tapped_at` zaman damgasını referans alır. Kartın son 90 dakikalık yolculuk geçmişi `tapped_at` sırasına göre deterministik bir durum makinesinden (state machine) geçirilir.
3. Çevrimdışı okumada kart bakiyesi yetersizse sistem esnek bakiye politikası (-50 TL avans hakkı) tanır; bakiye eksiye düştüğü an kart çevrimiçi senkronizasyonda kara listeye alınır.

### Değerlendirilen Alternatifler
1. *Olayları Geliş Sırasına Göre İşleme:* Sırasız gelen paketlerde ilk binişten önce aktarma gelmiş gibi algılanıp hatalı tam ücret veya geçersiz aktarma uygulanır. Kesinlikle reddedilmiştir.

### Sonuçlar
- **Olumlu:** Ağ gecikmelerinden bağımsız deterministik ve matematiksel olarak doğru ücret hesabı.
- **Olumsuz:** Gecikmeli gelen eski bir olay geçmiş aktarma zincirini etkileyebileceğinden düzeltme (storno) işlemi tetiklenebilir.

---

## ADR-004: Kuruş Bazlı Tamsayı Muhasebesi ve Yuvarlama Kayıpsız Gelir Paylaşımı

- **Durum:** Kabul Edildi (Accepted)
- **Tarih:** 2026-10-24
- **Kapsam:** Görev 4 (Gelir Paylaşımı) ve Finansal Hesaplamalar

### Bağlam (Context)
Birden çok işletmecinin (Özel Halk Otobüsü, Belediye Otobüsü, Şehir Hatları Vapuru) dahil olduğu aktarmalı yolculuklarda ücret (örneğin 17.50 TL) biniş ve mesafe oranlarına göre paylaştırılır. Ondalıklı sayılar (float/double) IEEE-754 kayan nokta yuvarlama hatalarına neden olur; 1 kuruşluk farklar dahi muhasebe denetiminde açığa yol açar.

### Karar (Decision)
1. Sistemdeki tüm para birimleri veritabanında ve Go modellerinde tamsayı (`int64` / `bigint`) cinsinden kuruş olarak tutulur (Örn: 20.00 TL = 2000 kuruş).
2. Gelir paylaşımında en büyük kalan algoritması (Largest Remainder Method / Hare-Niemeyer) kullanılır. Bölüşüm sonrası ortaya çıkan 1-2 kuruşluk yuvarlama kalıntıları ağırlıklı ana işletmeciye (Belediye) eklenerek toplamın her zaman girdiye eşit olması garanti edilir ($\sum Paylar = Toplam$).
3. Kapanan dönemler dondurulur (`closed_at`); geçmişe dönük düzeltmeler ters kayıt (storno) mantığıyla yeni döneme işlenir.

### Değerlendirilen Alternatifler
1. *Float/Double Kullanımı:* IEEE 754 yuvarlama hatası (0.1 + 0.2 != 0.3) nedeniyle finansal sistemlerde kabul edilemez; `proje.md` şartnamesince de açıkça yasaklanmıştır.
2. *PostgreSQL NUMERIC(10,2):* Veritabanında güvenli olsa da Go katmanında string dönüşümü gerektirir ve bellek tahsisatı yaratır. Tamsayı kuruş yaklaşımı CPU hızında çalışır.

### Sonuçlar
- **Olumlu:** %100 kuruş doğruluğu, denetimden geçen finansal tablolar.

---

## ADR-005: 1 Milyon Olay/Saat Yükü İçin PostgreSQL COPY/Batch Pipeline ve Anormallik Tespiti

- **Durum:** Kabul Edildi (Accepted)
- **Tarih:** 2026-10-24
- **Kapsam:** Görev 5 (Anormallik Tespiti ve Toplu Veri)

### Bağlam (Context)
Metropol ölçeğinde günde 15-20 milyon, saatte ise 1 milyon biniş olayı oluşabilmektedir (saniyede ~280-350 olay, yoğun saatlerde 1500+ TPS). Her biniş için tekil `INSERT` ve tetikleyici çalıştırmak diski kilitler. Aynı zamanda klon kartların tespiti için fiziksel hız sınırlarının denetlenmesi gerekir.

### Karar (Decision)
1. **Toplu Veri Yükleme:** Validatörlerden gelen toplu paketler PostgreSQL `COPY` protokolü ve `pgx.Batch` ile bellekte 1000'lik gruplar halinde diske flush edilir.
2. **Anormallik Tespiti:** Biniş anında kartın son biniş lokasyonu ve zamanı kontrol edilir:
   $$\text{Hız} = \frac{\text{Mesafe}(Durak_A, Durak_B)}{\Delta t}$$
   Eğer hız > 120 km/s ise (örneğin 5 dakika içinde Kadıköy'den Sarıyer'e biniş) işlem "İmkansız Seyahat / Klon Kart Şüphesi" olarak `jw_anomalies` tablosuna yazılır ve güvenlik merkezine bildirilir.
3. **Raporlama:** Yoğunluk ve gelir raporları için biniş anında özet sayaç tabloları ve periyodik Materialized View'lar kullanılır; böylece milyonlarca satırlık `SELECT COUNT(*)` sorguları rapor ekranlarını yavaşlatmaz.

### Değerlendirilen Alternatifler
1. *Her Biniş İçin Canlı Coğrafi GIS Sorgusu:* PostGIS ST_Distance her binişte çalıştırılırsa yüksek TPS altında CPU tavan yapar. Duraklar arası mesafe matrisi uygulama belleğinde (in-memory lookup table) önbelleklenerek $O(1)$ sürede kontrol edilir.

### Sonuçlar
- **Olumlu:** Saatte 1 milyon olayı rahatlıkla sindirebilen mimari, gerçek zamanlı klon kart tespiti.
