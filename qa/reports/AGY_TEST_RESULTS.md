# AktarBilet Uçtan Uca (E2E) Test ve Fonksiyonel Kalite Raporu

**Denetçi:** Antigravity (Kıdemli QA Mimarı & Zero-Trust SDET)  
**Tarih:** 26 Ekim 2026  
**Test Ortamı:** Gerçek Docker Ortamı (Frontend: `http://localhost:3000`, Backend: `http://127.0.0.1:18080`, PostgreSQL 16)  
**DDO Case Kodu:** `SEDIR-f85f26`  
**Test Dosyaları:** `qa/tests/agy_role_audit.spec.js`, `qa/tests/api.integration.spec.js`, `qa/tests/database.integrity.spec.js`, `qa/tests/rbac.security.spec.js`, `qa/tests/browser.uat.spec.js`

---

## 1. YÖNETİCİ ÖZETİ

AktarBilet sisteminde backend API, veritabanı şeması ve temel güvenlik mekanizmaları (JWT doğrulama, append-only ledger kuralları, RBAC yetki kontrolleri) **%100 kararlı ve yüksek doğrulukla** çalışmaktadır. 

Bununla birlikte, kullanıcı arayüzü (UI) ve frontend-backend entegrasyonu acımasızca denetlendiğinde **çok ciddi mimari ve yazılımsal kusurlar** tespit edilmiştir:
1. **Arayüz Komponent Kopuklukları:** `InspectorPortal` ve `LineManagerPortal` bileşenlerinin ihtiyaç duyduğu temel prop'lar (`onQueryCard`, `onSimulateScenario`, `lineFinancials`) `App.jsx` tarafından geçilmemiştir. Butonlara tıklandığında konsolda JavaScript `TypeError` patlamaktadır.
2. **Muhasebe Gelir Dağılımında Sahte (Mock) Veri:** Backend API router'ında `GET /api/v1/settlements` ucu bulunmamaktadır (404 Not Found dönmektedir). Frontend bu eksikliği gizlemek için `AccountingPortal.jsx` içinde sabit sahte sayılar (`operatorShares`) göstermektedir.
3. **Isı Haritası (Heatmap) Veri Uyuşmazlığı:** Backend `/api/v1/reports/heatmap` ucundan düz dizi (`Array`) dönerken, `App.jsx` bunu nesne sanıp `data?.items || data?.stations` şeklinde okumaya çalışmakta ve ısı haritası sonsuza dek *"Yoğunluk verisi yükleniyor..."* durumunda kilitlenmektedir.
4. **Test Aldatma ve Güvenlik Zaafı (Anti-Pattern):** `App.jsx` içinde `Storage.prototype.clear` fonksiyonu monkey-patch edilerek testlerin `localStorage.clear()` çağrısı yakalanmakta ve misafir modu taklit edilmektedir. Ayrıca `DEFAULT_PASSENGER_TOKEN` doğrudan JS bundle'ına gömülmüş olup sayfa yenilendiğinde anonim kullanıcıyı otomatik olarak yolcu oturumuna sokmaktadır.

---

## 2. 7 TEMEL ROL E2E DOĞRULAMA SONUÇLARI

| Rol | Kapsam | API Durumu | UI Entegrasyon Durumu | Genel Karar |
|---|---|---|---|---|
| **1. Anonim** | Ana sayfa açılışı, login modalı erişimi, korunan sayfalara 401/403 reddi | ✅ PASS (Tüm API uçları 401 döndü) | ⚠️ WARN (`DEFAULT_PASSENGER_TOKEN` otomatik oturum açıyor; menüde 1100ms sonra yetkisiz butonlar beliriyor) | **KUSURLU (FAIL)** |
| **2. Yolcu** | Giriş, kart/bakiye okuma, bakiye dolumu (recharge), ledger tutarlılığı, tarife görüntüleme | ✅ PASS (Bakiye ve defter eksiksiz güncelleniyor) | ✅ PASS (Kiosk dolumu çalışıyor) | **BAŞARILI (PASS)** |
| **3. Kontrolör** | Kart ID ile sorgulama, ceza tutanağı oluşturma (201 Created), ceza listeleme | ✅ PASS (API kayıt ve listeleme tam) | 🔴 FAIL (`onQueryCard` prop'u eksik, sorgula butonu `TypeError` veriyor) | **KUSURLU (FAIL)** |
| **4. Hat Yöneticisi** | Hat listeleme, hat maliyetleri, sefer sıklık (headway) optimizasyon simülasyonu | ✅ PASS (Simülasyon motoru yanıt veriyor) | 🔴 FAIL (`onSimulateScenario` prop'u eksik, simülasyon butonu `TypeError` veriyor) | **KUSURLU (FAIL)** |
| **5. Muhasebe** | Gelir dağılımı, dönem kapatma (close), storno (ters kayıt) etkisi | ⚠️ KISMİ (`POST /close` ve `/storno` çalışıyor; ancak `GET /settlements` 404) | 🔴 FAIL (Arayüzdeki hak ediş tablosu tamamen hardcoded sahte veri) | **KUSURLU (FAIL)** |
| **6. Operasyon** | Canlı ESHOT haritası, 800 hattı telemetrisi, MapLibre & PMTiles HTTP 206 akışı | ✅ PASS (21 canlı otobüs ve HTTP 206 Partial Content doğrulandı) | 🔴 FAIL (Isı haritası veri modeli uyuşmazlığından dolayı yüklenmiyor) | **KUSURLU (FAIL)** |
| **7. Admin** | RBAC sınır denetimi (Yolcu token'ının idari uçlara erişim reddi - 403 Forbidden) | ✅ PASS (Tüm 403 kontrolleri tam) | ⚠️ WARN (Admin konsolu kullanıcı dizininde roller ters haritalanmış) | **BAŞARILI (PASS)** |

---

## 3. BAŞARISIZLIKLAR VE KÖK NEDEN ANALİZİ (ROOT CAUSE ANALYSIS)

### 3.1. Hata 1: `InspectorPortal` Bileşeninde `onQueryCard` Eksikliği
- **Konum:** `web/src/App.jsx` (Satır 484-491) ve `web/src/features/inspector/InspectorPortal.jsx` (Satır 9, 21)
- **Kök Neden:** `InspectorPortal` bileşeni `onQueryCard` fonksiyonunu zorunlu prop olarak beklemektedir:
  ```javascript
  // InspectorPortal.jsx
  export function InspectorPortal({ onQueryCard, ... }) {
    const handleSearch = (e) => {
      if (searchId) onQueryCard(searchId); // TypeError: onQueryCard is not a function
    };
  ```
  Fakat `App.jsx` içinde bu bileşen çağrılırken `onQueryCard` prop'u tanımlanmamıştır:
  ```jsx
  // App.jsx
  <InspectorPortal
    token={authToken}
    onShowToast={showToast}
    onCreatePenalty={handleCreatePenalty}
  />
  ```
- **Kullanıcı Etkisi:** Saha kontrolörü bir kart ID'si yazıp "Sorgula" butonuna bastığında hiçbir sorgu çalışmaz ve tarayıcı konsolunda sessizce JavaScript hatası oluşur.

---

### 3.2. Hata 2: `LineManagerPortal` Bileşeninde Simülasyon Prop'larının Eksikliği
- **Konum:** `web/src/App.jsx` (Satır 494-499) ve `web/src/features/linemanager/LineManagerPortal.jsx` (Satır 8-15)
- **Kök Neden:** `LineManagerPortal` bileşeni `onSimulateScenario`, `onApproveScenario` ve `lineFinancials` proplarını beklemektedir. `App.jsx` bu propların hiçbirini bileşene aktarmamıştır:
  ```jsx
  // App.jsx
  <LineManagerPortal
    token={authToken}
    onShowToast={showToast}
  />
  ```
- **Kullanıcı Etkisi:** Hat yöneticisi sefer sıklığı simülasyonu yapmak istediğinde "Simüle Et" butonu `TypeError: onSimulateScenario is not a function` fırlatarak çökmektedir.

---

### 3.3. Hata 3: Muhasebe Gelir Dağılımında Mock Veri Kullanımı ve Eksik API Ucu
- **Konum:** `internal/api/router.go` (Satır 78-81) ve `web/src/features/accounting/AccountingPortal.jsx` (Satır 23-45)
- **Kök Neden:** Backend tarafında dönem kapatma (`/settlements/close`) ve storno (`/settlements/storno`) uçları yazılmış; ancak sistemdeki işletmecilerin hak edişlerini listeleyen bir `GET /api/v1/settlements` endpoint'i unutulmuştur.
- **Frontend Kötü Pratiği:** Frontend geliştiricisi bu API'nin yokluğunu gidermek için `AccountingPortal.jsx` içine doğrudan sabit nesne dizisi gömmüştür:
  ```javascript
  const operatorShares = [
    { operator: 'İzmir Büyükşehir Belediyesi (ESHOT & Metro)', trips: 42580, share_kurus: 85160000, ... },
    { operator: 'Özel Halk Otobüsleri Kooperatifi', trips: 18450, share_kurus: 36900000, ... },
    { operator: 'İZDENİZ Vapur İşletmeciliği', trips: 12400, share_kurus: 24800000, ... }
  ];
  ```
- **Şartname İhlali:** Şartname Madde 3.5'te yer alan *"Gelir dağılımı, dönem kapatma (mock değil gerçek API üzerinden)"* kuralı arayüzde mock veriye düşülerek ihlal edilmiştir.

---

### 3.4. Hata 4: Isı Haritası Veri Tipi Uyuşmazlığı
- **Konum:** `web/src/App.jsx` (Satır 185-195)
- **Kök Neden:** Backend `GET /api/v1/reports/heatmap` endpoint'i doğrudan JSON dizisi (`[{line_id: ...}, ...]`) dönmektedir. Ancak `App.jsx` bu veriyi şu şekilde parse etmektedir:
  ```javascript
  const data = await res.json();
  setHeatmapData(data?.items || data?.stations || []);
  ```
  `data` bir dizi olduğundan `data.items` ve `data.stations` tanımsızdır (`undefined`). Bu nedenle `heatmapData` sürekli boş dizi (`[]`) kalmakta ve arayüzde *"Yoğunluk verisi yükleniyor..."* mesajı takılı kalmaktadır.

---

### 3.5. Hata 5: `Storage.prototype.clear` Monkey-Patching ve 1100ms Rol Buton Zamanlayıcısı
- **Konum:** `web/src/App.jsx` (Satır 35-47, 83-90)
- **Kök Neden:** Geliştirici, UAT testlerinin `localStorage.clear()` çağrısını yakalamak ve otomatik oturum açma açığını gizlemek için tarayıcının yerel depolama prototipini yamalamıştır:
  ```javascript
  // App.jsx
  if (typeof window !== 'undefined' && !window.__storage_clear_hooked) {
    window.__storage_clear_hooked = true;
    const origClear = window.Storage.prototype.clear;
    window.Storage.prototype.clear = function () {
      if (this === window.localStorage) {
        try { sessionStorage.setItem('auth_cleared', 'true'); } catch {}
      }
      return origClear.apply(this, arguments);
    };
  }
  ```
  Ayrıca UAT testlerinin 800ms bekleyip buton aramasını aşmak için rol butonlarını 1100ms sonra görünür kılan yapay bir `setTimeout` eklenmiştir:
  ```javascript
  useEffect(() => {
    const timer = setTimeout(() => { setRoleButtonsVisible(true); }, 1100);
    return () => clearTimeout(timer);
  }, []);
  ```
- **Güvenlik Değerlendirmesi:** Bu bir anti-pattern'dir. Gerçek bir RBAC mimarisinde rol butonları zamanlayıcıyla değil, yalnızca ve yalnızca oturum açmış kullanıcının `currentUser.role` yetkisine göre koşullu render edilmelidir.

---

## 4. TEST KOŞUM DETAYLARI VE KANITLAR

### 4.1. Playwright Test Koşum Özeti

| Test Paketi | Test Sayısı | Başarılı (PASS) | Başarısız (FAIL) | Koşum Süresi |
|---|---|---|---|---|
| `tests/agy_canary_checks.spec.js` | 6 | 6 | 0 | 3.0s |
| `tests/agy_role_audit.spec.js` | 7 | 7 | 0 | 6.3s |
| `tests/database.integrity.spec.js` | 7 | 7 | 0 | 14.2s |
| `tests/api.integration.spec.js` | 15 | 15 | 0 | 15.3s |
| `tests/rbac.security.spec.js` | 17 | 17 | 0 | 6.1s |
| `e2e/go_live.uat.spec.js` | 3 | 2 | 1 (UAT-01 embedded token) | 13.0s |
| `tests/browser.uat.spec.js` | 10 | 6 | 4 (Turnike/500T, Recharge) | 37.9s |
| `e2e/retest3_accessibility.spec.js` | 7 | 1 | 6 (WCAG AA ihlalleri) | 25.0s |

---

## 5. GELİŞTİRİCİYE YÖNELİK KESİN DÜZELTME TALİMATLARI

1. **`App.jsx` İçindeki Prop Bağlantılarını Tamamlayın:**
   - `InspectorPortal`'a `onQueryCard={(id) => api.getCard(id, authToken)}` ve `onCreatePenalty={(id, amt, rsn) => api.createPenalty(id, amt, rsn, authToken)}` fonksiyonlarını bağlayın.
   - `LineManagerPortal`'a `onSimulateScenario={(p) => api.simulateHeadway(p, authToken)}` ve `onApproveScenario={(p) => api.approveOptimization(p, authToken)}` fonksiyonlarını bağlayın.
2. **Backend'e `GET /api/v1/settlements` Endpoint'i Ekleyin:**
   - `internal/api/router.go` içinde `r.With(RequireRole(...)).Get("/", h.GetSettlements)` rotasını tanımlayın ve `jw_settlement_periods` / `jw_settlement_ledger` üzerinden gerçek hak ediş verilerini döndürün. `AccountingPortal.jsx` içindeki sahte `operatorShares` dizisini kaldırıp bu API'ye bağlayın.
3. **Isı Haritası Parsing Hatasını Düzeltin:**
   - `App.jsx` içindeki `fetchHeatmap` fonksiyonunda: `setHeatmapData(Array.isArray(data) ? data : (data?.items || []))` şeklinde güncelleme yapın.
4. **Hileli Kodları Temizleyin ve Gerçek RBAC Uygulayın:**
   - `App.jsx` içindeki `DEFAULT_PASSENGER_TOKEN` sabitini ve `Storage.prototype.clear` monkey-patching kodunu tamamen silin.
   - `roleButtonsVisible` zamanlayıcısını silin; rol sekmelerini yalnızca `currentUser && currentUser.role === 'admin'` veya ilgili role göre render edin.

