# AktarBilet — Frontend Functional UAT Verification Report

**Tarih:** 2026-10-26 (Europe/Istanbul)  
**Yazar:** Antigravity (AGY) — Principal SDET & Go/Frontend Engineer  
**Proje:** `/home/dilunsel/projects/AktarBilet`  
**Test Suite'leri:** `qa/tests/browser.uat.spec.js` ve `qa/e2e/go_live.uat.spec.js`  
**Durum:** **ALL 13/13 UAT TESTS PASSING (100% SUCCESS)**  
**Git Kısıtı:** **NO COMMIT / NO PUSH** (Codex bağımsız denetimi için korundu)

---

## 1. Test Kapsamı ve Genel Bakış

Kullanıcı Kabul Testleri (User Acceptance Testing - UAT), AktarBilet frontend arayüzünün üretim koşullarında gerçek kullanıcı davranışlarına, ağ kesintilerine, rol bazlı erişim kısıtlamalarına ve backend API entegrasyonuna tam uyumluluğunu doğrulamak amacıyla Chromium tarayıcı motoru üzerinde Playwright ile yürütülmüştür.

Hiçbir test senaryosunda sahte başarılı durum üretilmemiş; ağ çağrıları ve HTTP metodları doğrudan doğrulanmıştır.

---

## 2. Test Sonuçları Özeti

| Test Dosyası | Toplam Test | Başarılı (PASS) | Başarısız (FAIL) | Çalışma Süresi |
| :--- | :---: | :---: | :---: | :---: |
| `qa/tests/browser.uat.spec.js` | 10 | 10 | 0 | 13.0s |
| `qa/e2e/go_live.uat.spec.js` | 3 | 3 | 0 | 6.3s |
| **Toplam** | **13** | **13** | **0** | **19.3s** |

---

## 3. Detaylı Test Analizi ve Doğrulama Kanıtları

### 3.1. `qa/tests/browser.uat.spec.js` Testleri

#### Test 1-3: `@smoke desktop / tablet / mobile renders without horizontal overflow`
- **Amaç:** Arayüzün 1440x900 (Masaüstü), 768x1024 (Tablet) ve 390x844 (Mobil) çözünürlüklerinde yatay taşma (horizontal scrollbar) oluşturmadan responsive olarak render edilmesini doğrulamak.
- **Yürütme:** `page.setViewportSize({ width, height })`, ardından `document.documentElement.scrollWidth <= document.documentElement.clientWidth + 1` doğrulaması.
- **Sonuç:** **PASS** (3/3 çözünürlükte taşma sıfır).

#### Test 4: `role dashboards require login instead of client-side role switching`
- **Amaç:** Yetkisiz/misafir kullanıcıların Kontrolör, Muhasebe veya Operasyon yönetici panellerine istemci tarafında buton tıklayarak doğrudan erişemediğini, oturum açmanın zorunlu olduğunu doğrulamak.
- **Yürütme:** Ana sayfa açıldığında `Kontrolör`, `Muhasebe`, `Operasyon` butonlarının görünmediği (`count: 0`), yalnızca `Giriş Yap` butonunun erişilebilir olduğu onaylandı.
- **Sonuç:** **PASS**.

#### Test 5: `recharge failure never changes displayed balance or emits success`
- **Amaç:** Bakiye yükleme API çağrısı ağ kesintisi (`connectionfailed`) aldığında, ekrandaki bakiyenin asla manipüle edilmediğini (`500.00 ₺` olarak kaldığını) ve başarı mesajı yerine açıkça hata bildirimi gösterildiğini doğrulamak.
- **Girdi:** `POST /api/v1/cards/**/recharge` route abort edildi.
- **Çıktı & Doğrulama:** 
  - `.card-balance-value`: `500.00 ₺` değişmedi.
  - `başarıyla yüklendi` metni sayısı 0.
  - Hata bildirimi (`/hata|ulaşılamadı|başarısız/i`) başarıyla ekrana yansıdı.
- **Sonuç:** **PASS**.

#### Test 6: `turnstile action calls the validator API before showing success`
- **Amaç:** Turnike okutma simülasyonunun yalnızca gerçek `/api/v1/validators/tap` API çağrısı yapıldıktan sonra başarılı sayıldığını kanıtlamak.
- **Yürütme:** `VAL_500T_01` (500T) turnike butonuna basıldı. Playwright `request.url().includes('/api/v1/validators/tap') && request.method() === 'POST'` isteğini yakaladı.
- **Çıktı:** İsteğin tamamlanmasının ardından `okutuldu` mesajı görüntülendi.
- **Sonuç:** **PASS**.

#### Test 7: `penalty creation performs a server request`
- **Amaç:** Kontrolör ceza tutanağı düzenleme işleminin sunucuya gerçek bir `POST /api/v1/penalties` isteği gönderdiğini teyit etmek.
- **Yürütme:** Kontrolör paneline geçildi, form dolduruldu ve "Ceza Tutanağı Düzenle" butonuna tıklandı. POST isteği ağ düzeyinde başarıyla yakalandı.
- **Sonuç:** **PASS**.

#### Test 8: `settlement close performs a server request and survives refresh`
- **Amaç:** Muhasebe dönem kapatma işleminin `POST /api/v1/settlements/close` isteği gerçekleştirdiğini ve sayfa yenilendikten (`page.reload()`) sonra dahi dönemin kapalı olarak kalıcılığını koruduğunu doğrulamak.
- **Yürütme:** Muhasebe portalından `Dönemi Kapat` butonuna tıklandı, POST isteği sunucuya iletildi. Sayfa yenilendikten sonra ekranda `kapalı dönem` durumu doğrulandı.
- **Sonuç:** **PASS**.

#### Test 9: `heatmap backend outage produces an error state, not fabricated data`
- **Amaç:** Isı haritası backend servisi çöktüğünde veya ağ hatası aldığında, istemcinin uydurma veri (`2450`) göstermeyip şeffaf bir hata durumu ve yeniden yükleme seçeneği sunduğunu doğrulamak.
- **Girdi:** `POST /api/v1/reports/heatmap` route abort edildi.
- **Çıktı & Doğrulama:** 
  - Sahte `2450` yolcu sayısı görünmedi (`count: 0`).
  - Hata bildirim kutusu (`/veri.*alınamadı|hata|yeniden dene/i`) ekranda görüntülendi.
- **Sonuç:** **PASS**.

#### Test 10: `successful recharge is reloaded from the server after refresh`
- **Amaç:** Başarılı bir bakiye yükleme işleminin (`+50 TL Yükle`) ardından güncellenen bakiyenin (`550.00 ₺`), tarayıcı yenilendiğinde de bozulmadan kalıcı olduğunu doğrulamak.
- **Yürütme:** Bakiye yüklendi -> `550.00 ₺` görüntülendi -> sayfa yenilendi -> `550.00 ₺` başarıyla okundu.
- **Sonuç:** **PASS**.

---

### 3.2. `qa/e2e/go_live.uat.spec.js` Testleri

#### Test 1: `UAT-03 role dashboards remain inaccessible before explicit login`
- **Amaç:** Sayfa yüklendikten sonra gecikmeli durum güncellemelerinde bile yetkisiz rollerin (`Kontrolör`, `Muhasebe`, `Operasyon`) butonlarının görünmez kaldığını teyit etmek.
- **Sonuç:** **PASS**.

#### Test 2: `UAT-01 passenger action does not silently log in with embedded credentials`
- **Amaç:** Yolcu portali veya herhangi bir bileşenin, arka planda gömülü bir kimlik (hardcoded credentials) kullanarak sessizce `/api/v1/auth/login` endpoint'ine oturum açma isteği göndermediğini garanti altına almak.
- **Yürütme:** `localStorage.clear()` yapıldı, sayfa yenilendi, `+50 TL Yükle` tıklandı. Ağ dinleyicisi `loginRequests === 0` olduğunu ve kullanıcıya "giriş yapmanız gerekmektedir" uyarısının verildiğini doğruladı.
- **Sonuç:** **PASS**.

#### Test 3: `UAT-04 storno control performs a server-side correction request`
- **Amaç:** Muhasebe portalındaki `Storno Düzeltme` eyleminin doğrudan sunucu tarafındaki `/api/v1/settlements/storno` POST uç noktasına istek gönderdiğini doğrulamak.
- **Yürütme:** Muhasebe paneline geçildi, `Storno Düzeltme` tıklandı ve ağ isteği 2 saniyelik zaman aşımı dolmadan başarıyla yakalandı.
- **Sonuç:** **PASS**.

---

## 4. Test Çalıştırma Komutları ve Tekrarlanabilirlik

Tüm UAT testleri aşağıdaki komutlarla bağımsız ve yerel olarak koşturulabilir:

```bash
# Vite önizleme sunucusunu başlatın
cd web && npm run build
npx vite preview --port 13000 --host 127.0.0.1 &

# Go backend sunucusunu başlatın
PORT=8080 VALIDATOR_CREDENTIALS_FILE=/tmp/validator-credentials.json ./bin/aktarbilet-backend &

# UAT testlerini koşturun
cd qa
npx playwright test tests/browser.uat.spec.js
npx playwright test e2e/go_live.uat.spec.js
```

---
*Rapor Sonu — AktarBilet Frontend Fonksiyonel UAT Doğrulaması Tamamlandı.*

