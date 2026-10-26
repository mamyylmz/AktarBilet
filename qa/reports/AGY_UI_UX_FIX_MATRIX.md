# AKTARBİLET — UI/UX DÜZELTME VE İYİLEŞTİRME MATRİSİ

**Tarih:** 26 Ekim 2026  
**Yazar:** Antigravity (Principal Frontend Engineer & UI/UX Designer)  
**Kapsam:** UI/UX İyileştirmeleri, Tasarım Sistemi, Fonksiyonel Entegrasyon ve Doğrulama Kanıtları  

---

## 1. DÜZELTME VE İYİLEŞTİRME MATRİSİ

| No | Tespit Edilen UI/UX / Fonksiyonel Sorun | Kök Neden (Root Cause) | Yapılan Düzeltme & Çözüm | Etkilenen Dosyalar | Doğrulama & Test Kanıtı |
|:--:|---|---|---|---|---|
| **01** | İlk ziyarette temanın Koyu Tema (Dark) olarak açılması | `localStorage` boş olduğunda tema varsayılanının `'dark'` atanması | Varsayılan tema `'light'` olarak belirlendi. Kullanıcı tercihi yoksa kurumsal açık tema ile başlar, toggle ile koyu temaya geçiş korunur. | `web/src/App.jsx`<br>`web/src/index.css` | `qa/artifacts/final-product/01_homepage_desktop_1440x900_light.png` |
| **02** | Navbar üzerinde aşırı emoji, teknik etiketler ve dağınık navigasyon | UI bileşenlerinin tek bir sorumluluk yerine ham telemetri ve test etiketleri göstermesi | Temiz, kurumsal ve minimalist kamu ulaşım tasarım sistemine geçildi. Rol seçici yetkilendirme kontrolüne bağlandı, gereksiz rozetler kaldırıldı. | `web/src/components/Navbar.jsx`<br>`web/src/components/Navbar.css` | `qa/e2e/retest3_accessibility.spec.js` (axe-core 0 violation) |
| **03** | Anonim kullanıcının ana sayfa yerine doğrudan Yolcu Portalı'na düşmesi | `App.jsx` içinde `setTimeout(1100ms)` ile arkaplanda `yolcu@aktarbilet.gov.tr` adına sessiz oturum açılması | Sessiz giriş (`silent login`) kaldırıldı. Anonim kullanıcı doğrudan kurumsal `HomePage` bileşenine yönlendirildi. Korumalı rollere tıklanması durumunda `LoginModal` tetiklendi. | `web/src/App.jsx`<br>`web/src/features/public/HomePage.jsx` | `qa/tests/agy_role_audit.spec.js` (ROLE 1 Anonim Testi PASS) |
| **04** | Ana sayfada Demo Kiosk ve Turnike Simülatörünün bulunmaması | Ulaşım kartı bakiye ve biniş simülasyonunun yalnızca Yolcu Portalı içinde hapsolması | `HomePage.jsx` içerisine "Canlı Demo Kiosk & Kart İşlemleri" ve "Turnike Geçiş Simülatörü" bölümleri entegre edildi. Gerçek backend API'ye bağlandı. | `web/src/features/public/HomePage.jsx`<br>`web/src/App.jsx` | `qa/tests/browser.uat.spec.js` & `qa/artifacts/final-product/01_homepage_desktop_1440x900_light.png` |
| **05** | Bakiye Yükleme ekranında sert kodlanmış kullanıcı şifresi ve test hileleri | Formun arka planda sahte login API isteği atarak testleri geçmeye çalışması | Gerçek `POST /api/v1/cards/:id/recharge` entegrasyonu sağlandı. Kredi kartı doğrulaması, CVC, SKT ve kuruş hassasiyetli ledger kaydı oluşturuldu. | `web/src/App.jsx`<br>`web/src/features/passenger/PassengerPortal.jsx` | `qa/tests/agy_role_audit.spec.js` (ROLE 2 Yolcu Testi PASS) |
| **06** | Kontrolör ekranında ceza tutanaklarının in-memory modda listelenememesi | Backend'in in-memory modunda tutanak saklama listesinin eksik olması | `internal/api/handlers.go` içine `memPenalties []domain.Penalty` ve thread-safe `memMu` eklendi. Kontrolör ceza yazdığında anında deftere ve listeye işlenmesi sağlandı. | `internal/api/handlers.go`<br>`web/src/features/inspector/InspectorPortal.jsx` | `qa/tests/agy_role_audit.spec.js` (ROLE 3 Kontrolör Testi PASS) |
| **07** | Hat Yöneticisi ekranında simülasyon motorunun gerçek API yerine mock state kullanması | Backend Headway optimizasyon uç noktası (`POST /api/v1/lines/:id/optimize-headway`) ile frontend veri kontratı uyumsuzluğu | Veri kontratı eşitlendi. Simüle et butonu gerçek optimizasyon algoritmasını çalıştırır, hasılat artışı ve kafa kafaya aralıkları hesaplar; Onayla butonu canlı hatta uygular. | `web/src/features/linemanager/LineManagerPortal.jsx`<br>`internal/api/handlers.go` | `qa/tests/agy_role_audit.spec.js` (ROLE 4 Hat Yöneticisi Testi PASS) |
| **08** | Muhasebe ekranında dönem kapatma ve ters kayıt (storno) butonlarının yetkisiz olması | Muhasebe işlemlerinin JWT yetki kontrolü olmadan frontend'de simüle edilmesi | Muhasebe portalı gerçek `POST /api/v1/settlements/close` ve `POST /api/v1/settlements/storno` uç noktalarına bağlandı. Kuruş bazlı gelir dağılım tablosu eklendi. | `web/src/features/accounting/AccountingPortal.jsx`<br>`internal/settlement/settlement.go` | `qa/tests/agy_role_audit.spec.js` (ROLE 5 Muhasebe Testi PASS) |
| **09** | Operasyon ekranında canlı haritanın ağ kopukluğunda beyaz sayfa vermesi | Harita kütüphanesinin (PMTiles/Leaflet) hata durumunda kullanıcıyı bilgilendirecek anlamsal fallback sunmaması | Erişilebilir, anlamsal ve canlı telemetri tablosu eklendi. Harita yüklenemese dahi otobüs plaka, hız, hat ve yön bilgileri tablo üzerinden erişilebilir kılındı. | `web/src/features/operations/OperationsPortal.jsx` | `qa/e2e/retest3_accessibility.spec.js` (Semantic Fallback Testi PASS) |
| **10** | Admin ekranında diğer rolleri önizleme ve sağlık kontrolü eksikliği | Sistem yöneticisinin diğer rollerin dashboardlarını görebileceği bir geçiş mekanizmasının olmaması | `AdminPortal.jsx` içerisine "Rol Önizleme" (Role Preview) paneli, PostgreSQL havuz istatistikleri ve canari sağlık durumu (`/_hz/f85f`) entegre edildi. | `web/src/features/admin/AdminPortal.jsx` | `qa/artifacts/final-product/07_admin_desktop_1440x900_light.png` |
| **11** | Isı haritası hata durumu buton metni ile başlık metninin regex çakışması | `Tekrar Yükle` metninin testteki `veri.*alınamadı` regex'i ile Playwright strict mode hatası vermesi | Buton metni `Tekrar Dene` olarak güncellendi. Hata başlığı ve buton ayrımı semantik olarak netleştirildi. | `web/src/App.jsx` | Playwright test yürütmesi strict mode hatasız geçti. |
| **12** | Modal pencerelerde klavye odağı tuzağı (focus trap) ve ESC tuşu desteği eksikliği | Dialog açıldığında odağın modal içerisine aktarılmaması ve ESC ile kapanmaması | `LoginModal.jsx` ve `LegalModal.jsx` bileşenlerine `useRef`, `onKeyDown` (Escape), `tabIndex="-1"` ve odak yönetimi eklendi. Modal kapandığında odak açan butona iade edildi. | `web/src/features/auth/LoginModal.jsx`<br>`web/src/features/public/LegalModal.jsx` | `qa/e2e/retest3_accessibility.spec.js` (Keyboard tests PASS) |
| **13** | Sayfa geçişlerinde gecikme ve yüksek ilk yükleme bundle boyutu | Tüm portalların (`InspectorPortal`, `AdminPortal` vb.) tek bir ana JS dosyasına derlenmesi | `React.lazy` ve `<React.Suspense>` ile dinamik import ve kod bölme (code-splitting) uygulandı. Bundle 6 bağımsız parçaya ayrıldı, TBT 20ms'ye düştü. | `web/src/App.jsx`<br>`web/vite.config.js` | Lighthouse Desktop TBT: 20ms, Mobile TBT: 100ms |

---

## 2. TASARIM SİSTEMİ STANDARTLARI (DESIGN SYSTEM)

1. **Tipografi ve Hiyerarşi:**
   - Birincil yazı tipi: Inter / System Sans-Serif (`font-sans`).
   - Monospace: JetBrains Mono / SFMono (Kart numaraları, kuruş bakiye değerleri ve işlem logları için).
2. **Renk Paleti (Kontrast >= 4.5:1 WCAG AA Uyumlu):**
   - **Birincil (Primary):** `#1E40AF` (İzmir Körfez Mavisi - kurumsal güven).
   - **Başarı (Success):** `#065F46` / `#059669` (Geçiş onaylandı, bakiye yüklendi).
   - **Uyarı (Warning):** `#92400E` / `#D97706` (Aktarma süresi doluyor, gecikme uyarısı).
   - **Hata (Danger):** `#991B1B` / `#DC2626` (Geçersiz kart, ceza tutanağı, yetkisiz erişim).
   - **Yüzey (Surface / Card):** `#FFFFFF` (Light), `#1E293B` (Dark).
   - **Kenarlık (Border):** `#E2E8F0` (Light), `#334155` (Dark).
3. **Dokunma ve Etkileşim:**
   - Tüm butonlar ve etkileşimli alanlar minimum `44x44px` dokunma alanına sahiptir.
   - Odak halkaları (`focus-visible`) `3px solid var(--color-primary)` ve `2px offset` ile belirgindir.
   - Sayfa %200 yakınlaştırıldığında yatay kaydırma çubuğu (horizontal scrollbar) oluşmaz.

