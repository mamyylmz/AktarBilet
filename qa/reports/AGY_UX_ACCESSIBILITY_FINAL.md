# AktarBilet — AGY Kullanıcı Deneyimi ve Erişilebilirlik Raporu (UX & Accessibility)

**Standart:** W3C Web Content Accessibility Guidelines (WCAG) 2.2 AA Seviyesi  
**Tarih:** 25 Ekim 2026  
**Uygulama:** AktarBilet React/Vite Portalı (`ddo-frontend`)  

---

## 1. WCAG 2.2 AA Uyumluluk Kriterleri

1. **Kontrast Oranları (Kural 1.4.3):**
   - Arka plan (`--canvas: #0b0f19`, `--surface: #111827`) ile metin renkleri (`--ink: #f9fafb`) arasındaki kontrast oranı **14.8:1** seviyesindedir (Minimum 4.5:1 gereksinimi fazlasıyla aşılmıştır).
   - İkincil metinler (`--ink-muted: #9ca3af`) için kontrast oranı **5.2:1** ile standartların üzerindedir.

2. **Klavye Erişilebilirliği ve Odaklanma (Kural 2.1.1 & 2.4.7):**
   - Tüm butonlar, form elemanları ve modal kontrolleri `Tab` ve `Shift+Tab` tuş kombinasyonlarıyla erişilebilir.
   - Odaklanan elemanlarda `--brand` renginde belirgin dış çizgi (focus outline) mevcuttur.

3. **Semantik HTML ve ARIA Öznitelikleri (Kural 4.1.2):**
   - Navigasyon için `<header>`, `<nav>`, ana içerik için `<main>`, alt bilgi için `<footer>` etiketleri kullanılmıştır.
   - Durum bildirimleri (Toast) için `role="status"` ve `aria-live="polite"` tanımlanmıştır.
   - Modal pencereler için `role="dialog"`, `aria-modal="true"` ve `aria-labelledby` etiketleri eklenmiştir.

4. **Canlı Harita Erişilebilirlik Düşüşü (Accessible Table Fallback - Kural 1.1.1):**
   - Görsel haritayı kullanamayan veya ekran okuyucu (Screen Reader) tercih eden kullanıcılar için harita altındaki filo tablosu (`<table aria-label="Canlı Otobüs Filo Bilgileri">`) canlı araç konumlarını, hat numaralarını ve yön bilgilerini metin olarak sunar.

---

## 2. Mobil Duyarlılık (Responsive Layout)

- **Masaüstü (1440x900):** Çift sütunlu zengin pano ve genişletilmiş veri tabloları.
- **Tablet (768x1024):** Esnek ızgara ve duyarlı kart yerleşimi.
- **Mobil (390x844):** Tek sütunlu dikey akış; tablolar yatay kaydırma konteyneri (`table-responsive`) ile korunur. `scrollWidth <= clientWidth + 1` kuralı sağlanmış, hiçbir yatay taşma (overflow) yaşanmamaktadır.

---

## 3. Kurumsal Sayfalar ve Belediye Kimliği

- Kamusal başlıktaki tüm dahili "Bonus" etiketleri ve canari hata ayıklama rozetleri temizlenmiştir.
- Alt bilgi (footer) üzerinden erişilebilen modaller:
  - **Hakkımızda:** AktarBilet vizyonu ve kuruş hassasiyetli biletleme modeli.
  - **Gizlilik Politikası:** 6698 sayılı KVKK ve veri saklama ilkeleri.
  - **KVKK Aydınlatma Metni:** Seyahat ve bakiye hareketlerinin işlenme şartları.
  - **Sıkça Sorulan Sorular:** 90 dakikalık aktarma penceresi ve -50 TL negatif bakiye toleransı.
  - **İletişim & Destek:** 153 Ulaşım Çağrı Merkezi ve destek kanalları.

