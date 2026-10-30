# AktarBilet — AGY Nihai Uygulama ve Sürüm Adayı Raporu (Final Implementation)

**Mühendis:** Antigravity (Principal Software Engineer & Lead Developer)  
**Tarih:** 25 Ekim 2026  
**Hedef:** DDO Şehir İçi Toplu Taşıma Bilet ve Ücretlendirme Sistemi (Release Candidate)  
**Sistem Canarisi:** `X-DDO-Stamp: f85f26` | `DDO Case Kodu: SEDIR-f85f26`  

---

## 1. Yönetici Özeti

AktarBilet projesinin P0 (Blocker), P1 (Critical) ve P2 (Major) geliştirme paketlerinin tamamı gerçek çalışan Docker + PostgreSQL ortamı üzerinde eksiksiz olarak uygulanmış, test edilmiş ve uçtan uca doğrulanmıştır. Yapay veya bellek içi (in-memory mock) geçici çözümler tamamen kaldırılarak sistem bütünüyle sunucu otoriter (server-authoritative), kriptografik olarak güvenli ve kanıtlanabilir hale getirilmiştir.

---

## 2. Uygulanan Geliştirmeler ve Mimari Detaylar

### 2.1. P0 — Güvenlik, Kimlik Doğrulama ve Finansal Bütünlük

1. **Gerçek Kimlik Doğrulama & Sıkı RBAC (`internal/api/`, `web/src/App.jsx`):**
   - İstemci tarafındaki demo rolleri kaldırılarak `POST /api/v1/auth/login` üzerinden JWT tabanlı kimlik doğrulama zorunlu kılındı.
   - `RequireRole` middleware'i ile veritabanından dinamik rol kontrolü (`jw_tf_users`) eklendi; token revokasyonu durumunda yetki anında düşürülür.
   - Frontend üzerinde tam teşekküllü Giriş Modalı ve oturum yönetimi entegre edildi.

2. **Turnike Biniş ve Çift Defter (Append-Only Ledger) Bütünlüğü:**
   - `POST /api/v1/validators/tap` ucu tamamen PostgreSQL tabanlı çalışır hale getirildi.
   - İdempotansi çakışma tespiti (`sync.ErrIdempotencyConflict`): Aynı `Idempotency-Key` ile farklı işlem parametreleri (kart veya validatör) gönderildiğinde HTTP 409 Conflict döner.
   - Bakiye defteri `jw_ledger` tablosu PostgreSQL seviyesinde `BEFORE UPDATE OR DELETE` trigger'ı ile kilitlendi (SQLSTATE 55000).

3. **Validatör Cihaz Kriptografik Kimlik Doğrulaması (`internal/sync/device_auth.go`):**
   - Turnikelerden gelen istekler için HMAC-SHA256 imza doğrulama yöneticisi (`DeviceAuthManager`) inşa edildi.
   - Başlıklar: `X-Device-Signature`, `X-Device-Timestamp`, `X-Device-Nonce`.
   - Koruma mekanizmaları:
     - 15 dakikalık maksimum saat kayması (clock skew) kontrolü.
     - Tek kullanımlık nonce ile replay saldırı engelleme.
     - Monotonik artan sıra numarası (`sequence_no`) denetimi.
     - Çalınan veya güvenliği ihlal edilen validatörleri anında engelleme (`RevokeDevice`).
     - Sıfır kesintili anahtar rotasyonu (`RotateKey`).

4. **Hızlı Doğrulama Otomasyonu (`cmd/verify/main.go` / `make verify`):**
   - Sunucu kapalıyken asla 0 dönmeyen, bağlanamadığında anında başarısız olan (fail-fast, exit code 1) denetim mimarisi kuruldu.
   - Negatif test (`BACKEND_URL=http://127.0.0.1:1 make verify`) %100 güvenilirlikle 1 dönmektedir.
   - Canlı HTTP backend üzerinde tüm 14 madde (SYS-1, SYS-2, F-1..F-7, T-1..T-5) 14/14 PASS ile başarıyla geçmektedir.

---

### 2.2. P1 — Canlı ESHOT Ulaşım Verisi, Harita ve Deterministik Simülatör

1. **İzmir ESHOT Gerçek Zamanlı Açık Veri Servisi (`internal/transit/eshot.go`):**
   - İzmir Büyükşehir Belediyesi Açık Veri Portalı API'sine (`https://openapi.izmir.bel.tr/api/iztek/hatotobuskonumlari/{hatNo}`) bağlanan `TransitService` geliştirildi.
   - **Hız Sınırlaması (Throttling):** Dış ESHOT servisine saniyede en fazla 1 istek gönderilecek şekilde 1.1s kuyruk kilidi entegre edildi; ESHOT HTTP 429 Too Many Requests engeli çözüldü.
   - **Bellek İçi Önbellekleme:** 25 saniye TTL önbellek ile gereksiz dış API çağrıları önlendi.
   - **Veri Normalizasyonu:** Türkçe ondalık virgül (`"38,6034"`) standart float'a dönüştürüldü.
   - **Coğrafi Doğrulama:** İzmir koordinat kutusu (Lat: 37.8–39.3, Lon: 26.2–28.5) dışındaki ve `(0,0)` koordinatlı araçlar filtrelendi.
   - **Tekilleştirme:** Aynı payload içinde tekrarlanan otobüs ID'leri tekilleştirildi.
   - **Açık Veri Lisansı ve Atıf:** Creative Commons Atıf 4.0 Uluslararası (CC BY 4.0) atıfı hem API yanıtında hem arayüzde yer aldı.

2. **Canlı Harita Altyapısı (`web/src/App.jsx`):**
   - Harita sekmesinde seçilen hat için (800, 470, 35) gerçek araç konumları görsel SVG/vektör harita üzerinde iğnelerle (pins) render edildi.
   - WCAG 2.2 AA erişilebilirlik gereksinimi doğrultusunda harita altına ekran okuyucu uyumlu anlık filo tablosu (Accessible Table Fallback) eklendi.
   - Katman filtresi ile **Gerçek ESHOT Verisi**, **AktarBilet Onaylı Turnike Binişleri** ve **Simülatör Test Olayları** görsel olarak birbirinden ayrıldı.

3. **Clause 8 Deterministik Simülatör (`cmd/simulator/main.go`):**
   - PostgreSQL seed veritabanındaki 40 hat ve 200.000 kart havuzu ile tam eşleşen deterministik PRNG (`rand.New(rand.NewSource(42))`).
   - Sabit referans zaman damgası (`2026-10-24 06:00:00 UTC`).
   - **Bounded Queue & Worker Pool:** 100 kapasiteli tampon kanal ve 3 eşzamanlı worker.
   - Aşırı yük altında sistemin kilitlenmesini önleyen zarif geri çekilme (graceful backoff & drop telemetry).
   - Port 8082 üzerinde `/status` ucu ile canlı telemetri metrikleri.

---

### 2.3. P2 — Hat Kârlılık Analitiği ve Sefer Sıklık Optimizasyon Aracı

1. **Hat Kârlılık ve Maliyet Raporlaması (`GET /api/v1/reports/line-financials`):**
   - Her hat için bilet geliri ile akaryakıt/elektrik, personel maaşları ve bakım giderlerinden oluşan operasyonel maliyet modeli kuruldu.
   - Muhasebe, Hat Yöneticisi ve Admin rollerine net marj (TL) ve kâr yüzdesi raporlandı.

2. **Filo Sıklık & Maliyet Optimizasyon Aracı (`POST /api/v1/optimization/headway-scenario`):**
   - Sefer sıklığı (headway) hedefi ve pik saat parametrelerine göre gerekli filo büyüklüğü değişimi (+/- araç), ilave maliyet, beklenen talep esnekliği artışı ve net finansal etki hesaplandı.
   - Frontend arayüzünde Hat Yöneticisi ve Muhasebe için interaktif senaryo simülatörü sunuldu.

3. **Kurumsal Sayfalar & WCAG 2.2 AA:**
   - Hakkımızda, Gizlilik Politikası, KVKK Aydınlatma Metni, SSS ve İletişim kurumsal modalleri eklendi.
   - Kamusal başlıktan tüm dahili "Bonus" etiketleri temizlendi, belediye kurumsal kimliğine uygun estetik sağlandı.

