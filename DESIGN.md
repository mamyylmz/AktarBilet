# AktarBilet Tasarım Sistemi (Design System)

## 1. Ürün Kimliği ve Vizyon
AktarBilet; şehir içi toplu taşıma ekosisteminde yolculara güven, saha kontrolörlerine hız, hat yöneticileri ve muhasebe ekiplerine netlik ve operasyon merkezine tam durumsal farkındalık (situational awareness) sunan üst düzey bir tasarım diline sahiptir.

Arayüz; karmaşık finansal ve lojistik verileri sakin, yormayan ve tek bakışta anlaşılır bir hiyerarşiyle sunar.

---

## 2. Renk Paleti ve Tasarım Belirteçleri (Tokens)

### 2.1. Temel Renkler
- **Canvas (Zemin):** `#0B0F19` (Koyu Gece / Derin Donanma) veya `#F8FAFC` (Temiz Gün Işığı)
- **Surface (Kart & Panel):** `#111827` (Elevated Surface) / `#FFFFFF`
- **Surface Muted:** `#1F2937` / `#F1F5F9`
- **Border / Hairline:** `#374151` / `#E2E8F0`
- **Primary Ink (Metin):** `#F9FAFB` / `#0F172A`
- **Secondary Ink (Açıklama):** `#9CA3AF` / `#64748B`

### 2.2. Ulaşım Modu Renkleri
Toplu taşıma modları anında ayırt edilebilir renk kodlarına sahiptir:
- 🚌 **Otobüs:** `#F59E0B` (Amber / Sıcak Güneş)
- 🚋 **Tramvay:** `#06B6D4` (Cyan / Elektrik Mavisi)
- ⛴️ **Vapur (Deniz Ulaşımı):** `#3B82F6` (Ocean Cobalt)
- 🚇 **Metro / Raylı Sistem:** `#10B981` (Emerald Green)

### 2.3. Semantik Durum Renkleri
- **Geçiş Onaylandı (Success):** `#10B981` (Kart okuma başarılı, bakiye yeterli)
- **Aktarma İndirimi (Transfer Active):** `#8B5CF6` (90 dk pencere açık, aktarma tarifesi uygulandı)
- **Düşük Bakiye / Uyarı (Warning):** `#F59E0B` (Bakiye 20 TL altı, offline senkronizasyon bekliyor)
- **Yetersiz Bakiye / Kara Liste / İhlal (Danger):** `#EF4444` (Kart bloke, geçersiz geçiş, imkansız seyahat anomalisi)
- **Bilgi (Info):** `#3B82F6`

---

## 3. Tipografi
- **Yazı Tipi:** Modern Sans-Serif (`Inter`, `Plus Jakarta Sans`, sistem sans-serif fallbacks).
- **Tablolar ve Saatler:** `JetBrains Mono` veya tabular figures (`font-variant-numeric: tabular-nums`) — hareket defteri ve para tutarlarında basamak kaymasını engeller.
- **Hiyerarşi:**
  - `h1`: 32px / Bold / Tracking -0.02em (Sayfa başlıkları)
  - `h2`: 24px / SemiBold (Bölüm başlıkları)
  - `h3`: 18px / SemiBold (Kart ve widget başlıkları)
  - `body`: 14px - 15px / Regular (1.5 satır yüksekliği)
  - `caption / micro`: 12px / Medium (Etiketler, rozetler)

---

## 4. Kullanıcı Rolleri ve Arayüz Düzeni

### 4.1. Yolcu Ekranı (Passenger Portal)
- Dijital Akıllı Kart bileşeni (Görsel NFC/Barkod illüstrasyonu, bakiye, kart tipi).
- Canlı 90 Dakikalık Aktarma Sayacı (Kalan süre, yapılan aktarma adedi 0/2).
- Son hareketler defteri (Dolumlar yeşil, harcamalar nötr, aktarma indirimleri mor rozetli).
- Bakiye dolum simülasyonu.

### 4.2. Kontrolör Ekranı (Inspector Dashboard)
- Yüksek kontrastlı, sahada tek elle kullanılabilir mobil arayüz.
- Kart ID okutma simülatörü.
- Anında ceza tutanağı oluşturma ve offline log listesi.

### 4.3. Hat Yöneticisi (Line Manager)
- Hatlar ve duraklar yönetim haritası/tablosu.
- Tarife matrisi (Tam, Öğrenci, Yaşlı, Gece Çarpanı).
- Sefer yoğunluk analitiği.

### 4.4. Muhasebe ve Gelir Paylaşımı (Settlement & Finance)
- İşletmeci bazlı (Özel Halk Otobüsü vs Belediye) kuruş hassasiyetli gelir dağılım tablosu.
- "Dönem Kapat" eylemi ve audit log akışı.
- Kapanış sonrası storno / düzeltme kayıtları paneli.

### 4.5. Operasyon ve Sistem İzleme (Operations Hub)
- Sistem sağlık monitörü (`GET /health` ve `GET /_hz/f85f` göstergeleri).
- Canlı validatör akış terminali.
- **Bonus 1:** Hat Bazlı Doluluk Isı Haritası (Occupancy Heatmap).
- **Bonus 3:** Tarife Değişikliği Etki Simülasyonu Grafiği.
- Anormallik uyarıları (İmkansız seyahat ve klon kart tespiti bildirimleri).

---

## 5. Erişilebilirlik ve Performans
- Tüm metin ve zemin kontrastları WCAG 2.1 AA standardını (en az 4.5:1) sağlar.
- Klavye ile tam navigasyon (focus rings belirgin).
- Sayfa yüklenme süresi < 1 saniye (Vite optimize edilmiş assetler).
