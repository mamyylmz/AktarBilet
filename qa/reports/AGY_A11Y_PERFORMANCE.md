# AktarBilet Performans ve WCAG 2.2 AA Erişilebilirlik (A11y) Denetim Raporu

**Denetçi:** Antigravity (Kıdemli QA Mimarı & Performans Test Uzmanı)  
**Tarih:** 26 Ekim 2026  
**Test Araçları:** Google Lighthouse 13.5.0, Deque axe-core 4.10, Playwright Chromium  
**Test Edilen Adres:** `http://localhost:3000/` (Canlı Docker Frontend & Nginx)  
**DDO Case Kodu:** `SEDIR-f85f26`

---

## 1. YÖNETİCİ ÖZETİ VE HEDEF SKOR KARŞILAŞTIRMASI

| Kategori | Ölçülen Değer | Şartname Hedefi | Durum |
|---|---|---|---|
| **Lighthouse Performance Skoru** | **81 / 100** | **≥ 90** | 🔴 **BAŞARISIZ (HEDEF ALTINDA)** |
| **Lighthouse Accessibility Skoru** | **100 / 100** (Ana Sayfa) | **≥ 90** | ✅ **BAŞARILI** |
| **axe-core Derin WCAG 2.2 AA Taraması** | **6 İhlal / 7 Senaryo** | **0 İhlal** | 🔴 **BAŞARISIZ (KRİTİK İHLALLER)** |
| **LCP (Largest Contentful Paint)** | **2.2 s** | **≤ 2.5 s** | ✅ **BAŞARILI** |
| **CLS (Cumulative Layout Shift)** | **0.045** | **≤ 0.1** | ✅ **BAŞARILI** |
| **TBT (Total Blocking Time)** | **590 ms** | **≤ 200 ms** | 🔴 **BAŞARISIZ (YÜKSEK CPU BLOKAJI)** |
| **FCP (First Contentful Paint)** | **2.1 s** | **≤ 1.8 s** | 🟡 **İYİLEŞTİRME GEREKİR** |
| **Speed Index** | **2.1 s** | **≤ 2.0 s** | 🟡 **İYİLEŞTİRME GEREKİR** |

---

## 2. LIGHTHOUSE PERFORMANS ANALİZİ

Lighthouse performans skoru **81** olarak ölçülmüş ve şartnamedeki **≥90** barajını geçememiştir. Başarısızlığın ana etkeni yüksek Toplam Engelleme Süresi'dir (**TBT: 590 ms**).

### 2.1. Metrik Kırılımları
```text
Performance: 81
├── First Contentful Paint (FCP): 2.1 s (Puan katkısı: Düşük)
├── Speed Index: 2.1 s
├── Largest Contentful Paint (LCP): 2.2 s (Kabul edilebilir sınırda)
├── Total Blocking Time (TBT): 590 ms (Kırmızı bölge: > 200ms)
└── Cumulative Layout Shift (CLS): 0.045 (İyi: < 0.1)
```

### 2.2. Performans Darboğazları ve Kök Nedenler
1. **Büyük JavaScript Bundle ve Ana İş Parçacığı (Main Thread) Yükü:**
   - React ve ilişkili kütüphaneler tek parça olarak yüklenmekte, `code-splitting` (dinamik `React.lazy` importları) kullanılmamaktadır.
   - Portal bileşenleri (`OperationsPortal`, `AdminPortal`, `MapLibreView`) ana sayfa açılışında doğrudan derlenip çalıştırıldığı için sayfa ilk açılışta 590 ms boyunca ana iş parçacığını dondurmaktadır.
2. **Statik Varlık Optimizasyonu:**
   - CSS ve font yüklemelerinde render-blocking kaynaklar bulunmaktadır.

---

## 3. AXE-CORE WCAG 2.2 AA ERİŞİLEBİLİRLİK DENETİMİ

Lighthouse'un ana sayfa yüzeysel kontrolü 100 puan vermiş olsa da, kurumsal ve tavizsiz standart olan **axe-core derin DOM denetiminde (`e2e/retest3_accessibility.spec.js`) 6 kritik/ciddi ihlal** tespit edilmiştir.

### 3.1. Tespit Edilen WCAG İhlalleri

| İhlal Kodu | Etki Seviyesi | Etkilenen Öğeler | WCAG Kriteri | Açıklama |
|---|---|---|---|---|
| **color-contrast** | 🔴 Ciddi (Serious) | `.active`, `.btn-primary`, `button[type="submit"]` | WCAG 1.4.3 Contrast (Minimum) | Birincil butonlardaki arka plan ve yazı rengi arasındaki kontrast oranı 4.5:1 eşiğini karşılamamaktadır. |
| **label** | 🔴 Kritik (Critical) | `input[type="email"]`, `input[type="password"]` | WCAG 4.1.2 Name, Role, Value | Giriş modalındaki e-posta ve şifre inputlarında programatik `<label for="...">` veya `aria-label` etiketi bulunmamaktadır. Ekran okuyucular alanı tanıyamamaktadır. |
| **keyboard-focus** | 🔴 Ciddi (Serious) | `LoginModal` Dialog | WCAG 2.1.2 No Keyboard Trap & 2.4.3 Focus Order | Giriş modalı açıldığında klavye odağı (focus) modal içine aktarılmamakta, arka planda kaybolmaktadır. |
| **escape-key** | 🔴 Ciddi (Serious) | `LoginModal` Dialog | WCAG 2.1.1 Keyboard Nav | Modal açıkken `Escape` tuşuna basıldığında modal kapanmamakta ve tetikleyici butona odak dönmemektedir (`expect(dialog).toBeHidden()` zaman aşımına uğramıştır). |
| **map-fallback** | 🟡 Orta (Moderate) | Canvas ESHOT Haritası | WCAG 1.1.1 Non-text Content | HTML5 `<canvas>` harita öğesi görme engelli kullanıcılar için erişilebilir metin alternatifi (aria-live veya semantic table fallback) sunmamaktadır. |

---

## 4. RESPONSIVE VE VIEWPORT KIRILMA ANALİZİ

Uygulamanın mobil ve masaüstü çözünürlüklerindeki davranışı Playwright ile taranmış ve tam sayfa ekran görüntüleri kaydedilmiştir.

### 4.1. Viewport Ölçümleri

| Viewport | Genişlik x Yükseklik | scrollWidth | clientWidth | Yatay Taşıma (Overflow) | Görsel Durum |
|---|---|---|---|---|---|
| **Mobil** | 360 x 800 px | 360 px | 360 px | ❌ Yok (0 px) | ✅ Responsive Uyumlu |
| **Desktop** | 1440 x 900 px | 1440 px | 1440 px | ❌ Yok (0 px) | ✅ Masaüstü Uyumlu |

- **Kanıt Dosyaları:**
  - Mobil Görünüm: `qa/artifacts/test-evidence/viewport_360x800.png`
  - Masaüstü Görünüm: `qa/artifacts/test-evidence/viewport_1440x900.png`
- **Gözlem:** Sayfa gövdesinde yatay kaydırma çubuğu (horizontal scrollbar) oluşmamaktadır; CSS grid ve flex yapısı 360px genişliğe başarıyla uyum sağlamaktadır.

---

## 5. GELİŞTİRİCİYE YÖNELİK PERFORMANS VE A11Y DÜZELTME TALİMATLARI

1. **TBT Süresini 200 ms Altına Düşürün (Performans Skoru ≥ 90 Hedefi):**
   - `web/src/App.jsx` içinde portal bileşenlerini dinamik import ile yükleyin:
     ```javascript
     const OperationsPortal = React.lazy(() => import('./features/operations/OperationsPortal.jsx'));
     const AdminPortal = React.lazy(() => import('./features/admin/AdminPortal.jsx'));
     ```
2. **Form Etiketlerini Ekleyin (axe-core Critical Fix):**
   - `web/src/features/auth/LoginModal.jsx` içindeki inputlara açık `<label>` ve `id` tanımlayın:
     ```jsx
     <label htmlFor="login-email">E-posta Adresi</label>
     <input id="login-email" type="email" aria-required="true" ... />
     ```
3. **Klavye Escape ve Focus Döngüsünü Onarın:**
   - Modal açıldığında ilk inputa `.focus()` verin ve `keydown` dinleyicisinde `e.key === 'Escape'` durumunda `onClose()` çağrısını garantiye alın.
4. **Buton Kontrastını Artırın:**
   - `.btn-primary` ve `.active` sınıflarının arka plan rengi ile metin rengi arasındaki kontrastı en az **4.5:1** oranına yükseltin.

