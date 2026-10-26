# AktarBilet Tasarım Sistemi (Design System & UI Architecture)

## 1. Ürün Kimliği ve Vizyon
AktarBilet; İzmir şehir içi toplu taşıma ekosisteminde yolculara güven ve şeffaflık, saha denetim personeline çeviklik, hat yöneticileri ve muhasebe ekiplerine kuruş hassasiyetli matematiksel netlik ve operasyon merkezine kesintisiz durumsal farkındalık (situational awareness) sunan kurumsal düzeyde bir tasarım diline ve modüler mimariye sahiptir.

Arayüz; karmaşık çift taraflı finansal defter hareketlerini, ESHOT canlı telemetri verilerini ve çok rollü yetkilendirme akışlarını sakin, gözü yormayan ve tek bakışta anlaşılır bir görsel hiyerarşiyle sunar.

---

## 2. Renk Paleti ve Tasarım Belirteçleri (Tokens)

Tüm renk paleti ve kontrast oranları WCAG 2.2 AA standartlarına (normal metin için minimum 4.5:1, büyük metin ve grafik bileşenler için minimum 3:1) tam uyumlu olarak belirlenmiştir.

### 2.1. Temel Arayüz Zemin ve Metin Renkleri
- **Canvas (Genel Sayfa Zemini):** `#0B0F19` (Koyu Gece / Derin Donanma)
- **Surface (Kart ve Panel Zeminleri):** `#111827` (Elevated Container)
- **Surface Muted (İkincil / İç Zeminler):** `#1F2937`
- **Surface Elevated (Modal & Popover):** `#283548`
- **Border / Hairline:** `#374151` (Yumuşak ayırıcı hatlar)
- **Border Light:** `#4B5563`
- **Primary Ink (Birincil Metin):** `#F9FAFB` (Kontrast oranı `#111827` üzerinde **14.5:1**)
- **Secondary Ink (Açıklama / İkincil Metin):** `#D1D5DB` (Kontrast oranı `#111827` üzerinde **9.6:1**)
- **Muted Ink (Tarih / Alt Bilgi Metni):** `#9CA3AF` (Kontrast oranı `#111827` üzerinde **5.3:1** — WCAG 4.5:1 barajını aşar)

### 2.2. Kurumsal Marka ve Vurgu Renkleri
- **Brand Primary:** `#0F766E` (İzmir Körfezi Zümrüt Yeşili)
- **Brand Hover:** `#115E59`
- **Brand Light / Focus Ring:** `#14B8A6` (Erişilebilirlik odak çerçevesi: `2px solid #14B8A6`, `offset 2px`)
- **Action Primary (Mavi Vurgu):** `#2563EB`
- **Action Primary Hover:** `#1D4ED8`

### 2.3. Semantik Durum ve Geri Bildirim Renkleri
- **Başarılı (Success):** `#10B981` (Bakiye yükleme, turnike geçiş onayı, senaryo kaydı)
  - Arka Plan: `rgba(16, 185, 129, 0.15)`
- **Aktarma Penceresi / Özel Durum:** `#8B5CF6` (90 dakika aktarma sayacı açık)
  - Arka Plan: `rgba(139, 92, 246, 0.15)`
- **Uyarı / Beklemede (Warning):** `#F59E0B` (Offline validatör, kapanmış dönem uyarısı)
  - Arka Plan: `rgba(245, 158, 11, 0.15)`
- **Hata / İhlal (Danger):** `#EF4444` (Yetersiz bakiye, idari ceza, anomali tespiti)
  - Arka Plan: `rgba(239, 68, 68, 0.15)`

---

## 3. Tipografi ve 8px Izgara (Spacing Grid)

### 3.1. Tipografi Hiyerarşisi
- **Yazı Ailesi:** `Inter`, `-apple-system`, `BlinkMacSystemFont`, `Segoe UI`, `Roboto`, `sans-serif`.
- **Finansal ve Sayısal Veriler:** `JetBrains Mono`, `font-variant-numeric: tabular-nums` (Kuruş defteri ve para tutarlarında basamak kayması kesinlikle engellenir).
- **Ölçekler:**
  - `Hero Display`: 40px (2.5rem) / Extra Bold (800) / Tracking -0.03em
  - `Portal Title (h2)`: 24px (1.5rem) / Bold (700) / Tracking -0.02em
  - `Card Title (h3)`: 18px (1.125rem) / SemiBold (600)
  - `Section Subtitle`: 14px (0.875rem) / Regular (400)
  - `Body Text`: 15px (0.9375rem) / Regular (400) / Line-height 1.6
  - `Caption / Badge`: 12px (0.75rem) / Medium (500)

### 3.2. 8px Bazlı Spacing Sistemi
- `--space-1`: 0.25rem (4px)
- `--space-2`: 0.5rem (8px)
- `--space-3`: 0.75rem (12px)
- `--space-4`: 1rem (16px)
- `--space-5`: 1.25rem (20px)
- `--space-6`: 1.5rem (24px)
- `--space-8`: 2rem (32px)

---

## 4. Kullanıcı Rolleri ve Görev Odaklı Arayüzler

Sistemde rol geçişleri ve yetkilendirme gerçek JWT oturumu üzerinden (`CurrentUser`) yönetilir.

### 4.1. Anonim / Herkese Açık Sayfalar (Public Portal)
- **Hero Bölümü:** İzmir Şehir İçi Akıllı Ulaşım vizyonu, 90 dakika aktarma güvencesi ve hızlı giriş çağrısı.
- **90 Dakika Aktarma Şeması:** İlk biniş ve sonraki aktarmaların adım adım işleyişi.
- **Şeffaf Tarife Tablosu:** Tam Kart, Öğrenci, Öğretmen/60+ Yaş ve Ceza bedelleri listesi.
- **Sıkça Sorulan Sorular (SSS):** Klavye ve ekran okuyucu uyumlu (`aria-expanded`) akordeon bileşeni.
- **Kurumsal Footer:** KVKK, Gizlilik, Kullanım Koşulları ve ESHOT açık veri atıf bağlantıları.

### 4.2. Yolcu Portalı (Passenger Portal)
- **Dijital Akıllı Kart Bileşeni:** Bakiye göstergesi (`500.00 ₺`), kart tipi ve maskeli kart numarası.
- **90 Dakika Aktarma Sayacı:** Biniş anında tetiklenen canlı geri sayım sayacı (`5400s -> 0s`).
- **Demo Dolum Eylemleri:** `+50 TL Yükle`, `+100 TL Yükle`, `+200 TL Yükle` (Yetkisiz çağrılarda login modalını tetikler, float yuvarlama yapmaz).
- **Hızlı Turnike Biniş Simülasyonu:** 500T, 14ES, T1 ve Vapur hatlarında validatör tetikleme.
- **Defter ve Biniş Geçmişi Tablosu:** Append-only çift taraflı hareket defteri (`jw_ledger`).

### 4.3. Saha Kontrolörü (Inspector Portal)
- **Kart Geçerlilik Sorgusu:** Kart ID üzerinden biniş ve bakiye durumu doğrulaması.
- **İdari Para Cezası Tutanağı:** Kart ID, ihlal gerekçesi (`kartsiz_binis`, `sahte_kart`) ve ceza tutarı (200,00 TL) ile sunucuya kayıt.
- **Düzenlenen Tutanaklar Listesi:** Ceza numarası, tarih, kart ID ve tahsilat durumu.

### 4.4. Hat Yöneticisi (Line Manager Portal)
- **Hat Finansal Performans Tablosu:** Gelir, yolcu sayısı, km başı maliyet ve karlılık marjı analizi.
- **2 Kademeli Sefer Aralığı (Headway) Simülatörü:**
  - 1. Kademe: Sefer aralığı (dk), araç kapasitesi ve yoğun saat simülasyonu.
  - 2. Kademe: Filo ihtiyacı, ek araç maliyeti ve yolcu kapasite projeksiyonu.
- **Kalıcı Senaryo Onayı:** Simülasyon çıktısını `POST /api/v1/optimization/approve` ile kalıcı plana dönüştürme.

### 4.5. Muhasebe ve Gelir Paylaşımı (Accounting Portal)
- **Dönem Kapanış Kartı:** Kapanış durumu, kapanış tarihi ve `Dönemi Kapat` eylemi (`POST /api/v1/settlements/close`).
- **Kapanış Sonrası Storno Düzeltmesi:** Hatalı kesinti iadesi için çift taraflı ters kayıt (`POST /api/v1/settlements/storno`).
- **İşletmeci Gelir Paylaşım Tablosu:** ESHOT Belediye vs Özel Halk Otobüsü hak edişleri.

### 4.6. Operasyon ve Canlı Harita (Operations Portal)
- **İzmir ESHOT Canlı Araç Takip Haritası:** MapLibre GL ve yerel PMTiles vektör kiremitleri üzerinden 800 (Menemen Aktarma), 470 (Tınaztepe) ve 35 (Fahrettin Altay) hatlarının gerçek konumları.
- **Erişilebilir Tablo Desteği (Semantic Fallback):** Ekran okuyucular için canlı araç kapı no, enlem, boylam ve sefer durumunu listeleyen veri tablosu (`aria-label="Canlı Otobüs Sefer Listesi"`).
- **Yolcu Biniş Yoğunluk Isı Haritası (Heatmap):** `/api/v1/reports/heatmap` üzerinden durak yoğunlukları ve hata durumunda koruma.
- **Güvenlik ve Anomali İzleme:** İmkansız hız ve klon kart alarmları.

### 4.7. Sistem Yöneticisi (Admin Portal)
- **Sistem Sağlık Göstergeleri:** Canlı API durumu (`/health` ve `/_hz/f85f`), PostgreSQL 16 motoru ve 7/7 Canari doğrulaması.
- **Kullanıcı ve Rol Dizin Tablosu:** Kayıtlı roller, e-posta adresleri ve yetki matrisi.
- **Güvenlik Denetim Günlüğü (Audit Log):** Sistem anomalileri ve kritik güvenlik bildirimleri.

---

## 5. Modüler Bileşen Kütüphanesi (`components/ui/`)

Tüm bileşenler bağımsız, test edilebilir ve WCAG 2.2 AA kurallarına tam uyumludur:

1. **`Modal.jsx`:**
   - `role="dialog"`, `aria-modal="true"`, `aria-labelledby`.
   - Açıldığında odağı modal içindeki ilk elemana taşır.
   - Odak tuzağı (focus trap) ile `Tab` ve `Shift+Tab` döngüsünü pencere içinde tutar.
   - `Escape` tuşu ile kapanır ve odağı tetikleyici butona (`triggerRef`) eksiksiz iade eder.
2. **`Table.jsx`:**
   - Yatay kaydırma konteyneri klavyeyle erişilebilir: `tabIndex={0}`, `role="region"`, `aria-label="..."`.
   - `scrollable-region-focusable` axe kuralına %100 uyumludur.
3. **`Input.jsx`:**
   - `<label htmlFor={id}>` ile giriş alanı ID'si birebir eşleştirilmiştir.
   - Hata durumunda `aria-describedby` ve `role="alert"` ile ekran okuyucuya anında seslendirilir.
4. **`Button.jsx`:**
   - `loading` durumunda spinner gösterir, `disabled` niteliği ekler ve `aria-busy="true"` bayrağı açar.
5. **`Toast.jsx`:**
   - `role="status"`, `aria-live="polite"` ile arayüzü engellemeden bildirim sağlar.

---

## 6. Erişilebilirlik ve Doğrulama Kanıtları (WCAG 2.2 AA)

- **Axe-core Doğrulaması:** `qa/e2e/retest3_accessibility.spec.js` altındaki tüm 7 testte violations = `[]` (Sıfır İhlal).
- **Zoom %200 Uyumluluğu:** 1280px genişlikte %200 yakınlaştırmada `scrollWidth <= clientWidth` sağlanır, sayfa düzeyinde taşma oluşmaz.
- **Mobil ve Tablet Duyarlılığı:** 390px (mobil), 768px (tablet), 1440px ve 1920px (masaüstü) ekran boyutlarında esnek ızgara ve duyarlı tipografi.
