# AktarBilet — Bağımsız Mobil Uyumluluk, Responsive Tasarım ve A11y Denetim Raporu

**Denetçi:** Antigravity (Kıdemli QA Mimarı & UI/UX Erişilebilirlik Uzmanı)  
**Tarih:** 26 Ekim 2026  
**Test Ortamı:** Playwright Headless Chromium, Lighthouse v12, axe-core v4.10  
**Görünüm Matrisi:** 7 Farklı Çözünürlük (320px - 1920px), Açık ve Koyu Tema (Toplam 14 Konfigürasyon)  
**Görsel Kanıtlar:** `qa/artifacts/agy-final-qa/screenshots/`  

---

## 1. YÖNETİCİ ÖZETİ

Kullanıcı talebi ve şartname gereksinimleri doğrultusunda, AktarBilet web arayüzü; mobil cihazlar (dar ekranlar), tabletler ve masaüstü monitörlerde hem açık hem koyu renk paletinde baştan sona denetlenmiştir.

### Başlıca İyileştirmeler ve Doğrulamalar:
1. **Mobil Başlık (Header) Daraltması:** Önceki sürümlerde 360px mobilde 215px yer kaplayarak ekranın yarısını işgal eden navbar; hamburger menü ve açılır çekmece (drawer) yapısına dönüştürülerek **65 piksele** düşürülmüştür.
2. **Yatay Kaydırma (Horizontal Overflow) İhlali Sıfırlandı:** 320px'den 1920px'e kadar hiçbir ekranda sayfa seviyesinde yatay kaydırma çubuğu (`scrollWidth > clientWidth`) oluşmamaktadır (`hasOverflow = false`).
3. **WCAG 2.2 AA Erişilebilirlik Uyumu:** axe-core motoruyla taranan 7 erişilebilirlik senaryosunun tamamı (kontrast, odak yönetimi, klavye gezintisi, ARIA rolleri) **sıfır ihlal ile tam not (PASS)** almıştır.
4. **Lighthouse Skorları:**
   - **Masaüstü (Desktop):** Performans **100**, Erişilebilirlik **98**, CLS **0**, TBT **0 ms**.
   - **Mobil (Mobile):** Performans **84**, Erişilebilirlik **98**, CLS **0**, TBT **520 ms**.

---

## 2. ÇÖZÜNÜRLÜK MATRİSİ VE EKRAN ÖLÇÜMLERİ

`qa/tests/viewport_audit.mjs` aracıyla her çözünürlükte tam sayfa ekran görüntüleri alınmış ve ölçülmüştür:

| Cihaz Tipi | Çözünürlük | Tema | Header Yüksekliği | Yatay Taşma | .grid Hesaplanmış Stil | Ekran Görüntüsü Dosyası |
|---|:---:|:---:|:---:|:---:|:---:|---|
| **iPhone SE** | 320 x 568 | Açık | 65 px | ❌ Yok | `block` (CSS Grid eksik) | `iphone_se_320x568_light.png` |
| **iPhone SE** | 320 x 568 | Koyu | 65 px | ❌ Yok | `block` | `iphone_se_320x568_dark.png` |
| **Modern Android** | 360 x 800 | Açık | 65 px | ❌ Yok | `block` | `android_360x800_light.png` |
| **Modern Android** | 360 x 800 | Koyu | 65 px | ❌ Yok | `block` | `android_360x800_dark.png` |
| **iPhone 14/15** | 390 x 844 | Açık | 65 px | ❌ Yok | `block` | `iphone_390x844_light.png` |
| **iPhone 14/15** | 390 x 844 | Koyu | 65 px | ❌ Yok | `block` | `iphone_390x844_dark.png` |
| **Tablet Dikey** | 768 x 1024 | Açık | 57 px | ❌ Yok | `block` | `tablet_port_768x1024_light.png` |
| **Tablet Dikey** | 768 x 1024 | Koyu | 57 px | ❌ Yok | `block` | `tablet_port_768x1024_dark.png` |
| **Tablet Yatay** | 1024 x 768 | Açık | 150 px* | ❌ Yok | `block` | `tablet_land_1024x768_light.png` |
| **Tablet Yatay** | 1024 x 768 | Koyu | 106 px* | ❌ Yok | `block` | `tablet_land_1024x768_dark.png` |
| **Masaüstü Standart**| 1440 x 900 | Açık | 61 px | ❌ Yok | `block` | `desktop_1440x900_light.png` |
| **Masaüstü Standart**| 1440 x 900 | Koyu | 61 px | ❌ Yok | `block` | `desktop_1440x900_dark.png` |
| **Full HD Monitör** | 1920 x 1080 | Açık | 61 px | ❌ Yok | `block` | `desktop_fhd_1920x1080_light.png` |
| **Full HD Monitör** | 1920 x 1080 | Koyu | 61 px | ❌ Yok | `block` | `desktop_fhd_1920x1080_dark.png` |

*\*Not: 1024px tablet yatay modunda masaüstü menü elemanları sığmaya çalışırken 2 satıra katlandığı için header yüksekliği geçici olarak 150px/106px olmaktadır. Breakpoint eşiğinin `lg: (1024px)` yerine `xl: (1280px)` olarak ayarlanması önerilir.*

---

## 3. ERİŞİLEBİLİRLİK (A11Y) VE WCAG 2.2 AA TESTLERİ

`qa/e2e/retest3_accessibility.spec.js` süiti Playwright ve axe-core ile koşturulmuş, 7 testin tamamı geçmiştir:

1. **Masaüstü Yolcu Portalı (Desktop Passenger Portal):** Sıfır kontrast hatası, doğru başlık hiyerarşisi (H1-H4).
2. **Giriş Modalı Odak Tuzağı (Focus Trap):** Giriş butonu tıklandığında odak doğrudan ilk input'a taşınır; modal dışına sekme (tab) atılamaz.
3. **Klavye ile Kapatma (Escape Key):** ESC tuşuna basıldığında modal kapanır ve odak tetikleyici butona geri döner.
4. **Mobil Yolcu Portalı:** Dokunmatik hedef boyutları (Touch targets) ≥ 44x44 piksel.
5. **Canlı Harita Semantik Geri Düşümü:** Ekran okuyucu kullanıcıları için harita yanında erişilebilir alternatif HTML tablosu sunulmaktadır.
6. **%200 Yakınlaştırma (200% Zoom Reflow):** Tarayıcı %200 yakınlaştırıldığında içerik tek sütuna düzgünce akar; yatay taşma oluşmaz.

---

## 4. LIGHTHOUSE DENETİM SONUÇLARI

### 4.1. Masaüstü Denetimi (`lighthouse_desktop.json`)
- **Performans:** **100 / 100**
- **Erişilebilirlik:** **98 / 100**
- **İlk İçerikli Boyama (FCP):** 0.5 saniye
- **En Büyük İçerikli Boyama (LCP):** 0.5 saniye
- **Toplam Engelleme Süresi (TBT):** 0 ms
- **Kümülatif Düzen Kayması (CLS):** 0

### 4.2. Mobil Denetimi (`lighthouse_mobile.json`)
- **Performans:** **84 / 100**
- **Erişilebilirlik:** **98 / 100**
- **İlk İçerikli Boyama (FCP):** 2.1 saniye
- **En Büyük İçerikli Boyama (LCP):** 2.1 saniye
- **Toplam Engelleme Süresi (TBT):** 520 ms
- **Kümülatif Düzen Kayması (CLS):** 0

---

## 5. GELİŞTİRİCİYE SUNULACAK MOBİL & CSS AKSİYON LİSTESİ

1. **CSS Grid Tanımlaması (P1):**
   - `web/src/index.css` dosyasında `.grid` için `display: grid;` kuralı yazılmalıdır. Şu anda JSX bileşenleri `className="grid grid-cols-2"` kullanmakta fakat CSS dosyasında bu kural bulunmadığı için tarayıcı bunu `display: block` olarak işlemektedir.
2. **1024px Tablet Kırılım Noktası (P2):**
   - Tablet yatay modunda menünün 2 satıra kırılmasını önlemek için `Navbar.jsx` içindeki hamburger menü görünürlüğü `md:hidden` yerine `lg:hidden` olarak genişletilmelidir.
3. **Kart ve Buton Dokunma Alanları (P2):**
   - Mobil 320px ekranda bakiye yükleme hızlı seçim butonlarının padding'leri `py-2 px-3` yapılarak dar ekranlarda birbirine yapışması önlenmelidir.

