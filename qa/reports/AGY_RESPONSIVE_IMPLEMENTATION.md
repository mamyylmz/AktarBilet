# AktarBilet — Responsive Tasarım, Mobil Optimizasyon ve WCAG 2.2 AA Uyumluluk Raporu

**Tarih:** 26 Ekim 2026  
**Hazırlayan:** Antigravity (Principal Frontend Engineer & UI/UX Architect)  
**Denetleyen:** Codex (Reviewer & Quality Engineering)  
**Durum:** **TAMAMLANDI (WCAG 2.2 AA ve Mobil Testleri %100 Başarılı)**

---

## 1. Giriş ve Hedefler

Bu rapor, AktarBilet web arayüzünün mobil (360px / 390px), tablet (768px) ve masaüstü (1440px / 1920px) ekran çözünürlüklerindeki görsel hiyerarşisini, erişilebilirliğini ve performansını belgelemektedir. Codex tarafından tespit edilen mobil başlık taşması (215px sticky header) ve ekran okuyucu uyumluluğu eksiksiz giderilmiştir.

---

## 2. Mobil Sticky Header Optimizasyonu (360px Viewport)

### 2.1. Sorun Tespiti
Mobil ekranlarda (360px genişlik) navigation bar içindeki tüm butonlar (`Yolcu Portalı`, `Canlı Harita`, `Isı Haritası`, `Nasıl Çalışır?`, `Hakkımızda`, `Yardım`, tema butonu ve giriş butonu) alt alta dizilerek sticky header yüksekliğini **215 piksele** çıkarmakta ve sayfa içeriğinin neredeyse yarısını kaplamaktaydı.

### 2.2. Mimari Çözüm
1. **Header Yüksekliği Kısıtlaması:**  
   `web/src/index.css` içerisinde `@media (max-width: 768px)` altında `.navbar-container`:
   ```css
   .navbar-container {
     padding: 0.5rem 0.75rem;
     flex-direction: row;
     align-items: center;
     justify-content: space-between;
     min-height: 56px;
     max-height: 64px;
   }
   ```
   olarak sınırlandırıldı. Mobil ekranlarda başlık sabit 56–64px yüksekliğindedir.
2. **Erişilebilir Hamburger Menü:**  
   - `web/src/components/Navbar.jsx` içine `mobileMenuOpen` durumu ve `aria-expanded={mobileMenuOpen}` etiketli SVG hamburger ikonu entegre edildi.
   - Masaüstü gezinti bağlantıları (`.navbar-nav`) 768px altında gizlendi (`display: none`).
   - Hamburger butonuna tıklandığında ekranın üstünden açılan ve tüm işlevsel portallara erişim sağlayan responsive bir drawer (`.mobile-nav-drawer`) sunuldu.
   - Herhangi bir bağlantıya tıklandığında veya Escape tuşuna basıldığında menü otomatik olarak kapanır.

---

## 3. Tablo ve Veri Sunumu Duyarlılığı (Responsive Data Tables)

1. Tüm veri tabloları (`web/src/components/ui/Table.jsx`), `-webkit-overflow-scrolling: touch` ve `overflow-x: auto` özelliklerine sahip `.table-scroll-container` sarmalayıcısına alınmıştır.
2. Tablo sarmalayıcısı `role="region"`, `tabIndex={0}` ve anlamlı bir `aria-label` (örn. "Canlı Otobüs Sefer Listesi", "Güvenlik Anomalileri Tablosu") içerir.
3. 360px ve 390px ekranlarda tablolar yatayda taşma yapmadan kaydırılabilir durumdadır; sayfa bütününde yatay kaydırma çubuğu oluşmaz.

---

## 4. Sanal Toplu Taşıma Kartı Tasarımı

1. Yolcu ve ana sayfa üzerinde yer alan sanal kart (`.transit-virtual-card`), modern bankacılık ve toplu taşıma standartlarında tasarlanmıştır.
2. Kart yüzeyi; metalik NFC çip ikonu (`.card-chip-icon`), kart türü rozeti ("DEMO KART" / "TAM KART"), yüksek kontrastlı bakiye alanı (`.card-balance-value`), maskelenmiş kart ID'si ve temassız dalga sembolünü içerir.
3. Bakiye metni `formatKurus(bal)` üzerinden doğrudan kuruş tam sayısından Türk Lirası para birimi formatına (`500.00 ₺`) çevrilir.

---

## 5. WCAG 2.2 AA Erişilebilirlik ve Kontrast Güvencesi

1. **Kontrast Oranları:**
   - Açık tema (varsayılan): Birincil metin `--ink: #0f172a` beyaz zemin üzerinde **16.5:1** (AAA standardı) kontrast sunar.
   - İkincil metin `--ink-secondary: #334155` **9.5:1** (AAA), zayıf metin `--ink-muted: #64748b` **4.6:1** (AA standardı 4.5:1 gereksinimini aşar).
   - Koyu tema: Yüksek kontrastlı slate paleti (`#0b0f19` ve `#f9fafb`) ile 14:1 kontrast sağlar.
2. **Klavye Odak Yönetimi (Focus Management):**
   - Modal pencereler açıldığında odak otomatik olarak modal içine aktarılır.
   - Tab tuşu modal dışına taşmaz (focus trap).
   - Escape tuşuna basıldığında modal kapanır ve odak, modalı açan tetikleyici butona (`triggerRef`) iade edilir.
3. **200% Zoom / Sayfa Reflow Testi:**
   - Sayfa %200 yakınlaştırıldığında hiçbir yatay taşma (horizontal scrollbar) oluşmaz.

---

## 6. Playwright Axe & Erişilebilirlik Test Sonuçları

`qa/e2e/retest3_accessibility.spec.js` test dosyasının bağımsız koşturulma sonuçları:

| Test Senaryosu | Test Türü | Süre | Sonuç |
|---|---|---|---|
| Retest 3 WCAG 2.2 AA axe audit: desktop passenger portal | Automated Axe Core | 1.9s | **PASS** |
| Retest 3 WCAG 2.2 AA axe audit: desktop login dialog | Automated Axe Core | 1.9s | **PASS** |
| Retest 3 WCAG 2.2 AA axe audit: mobile passenger portal | Automated Axe Core | 1.3s | **PASS** |
| Retest 3 WCAG 2.2 AA axe audit: map view semantic fallback | Automated Axe Core | 1.3s | **PASS** |
| Retest 3 WCAG 2.2 AA: keyboard opens login dialog & moves focus | Keyboard Traversal | 877ms | **PASS** |
| Retest 3 WCAG 2.2 AA: login dialog Escape closes & restores focus | Keyboard Focus Return | 992ms | **PASS** |
| Retest 3 WCAG 2.2 AA: 200% zoom does not introduce overflow | CSS Reflow Audit | 923ms | **PASS** |
| **Toplam** | **7 Senaryo** | **10.7s** | **%100 PASS** |

