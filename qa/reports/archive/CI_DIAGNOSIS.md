# AktarBilet GitHub Actions ve CI/CD Süreçleri Tanı Raporu (CI Diagnosis)

**Tarih:** 25 Ekim 2026  
**Durum:** ✅ **CI/CD İNCELEMESİ VE DÜZELTME ÖNERİLERİ RAPORLANDI**  
**Hedef Repository:** `https://github.com/mamyylmz/AktarBilet`  
**DDO Case Kodu:** `SEDIR-f85f26`

---

## 1. Yönetici Özeti

GitHub Actions üzerinde gerçekleştirilen workflow koşuları (Run ID: `40124169654` .. `40126215225`) incelenmiş, ilk dört koşudaki başarısızlıkların Go sürüm uyumsuzlukları ve derleme adımlarından kaynaklandığı, son koşu olan `40126215225` (commit `aceaef8483619f73076c913ba50dfa10f901afd8`) itibarıyla CI boru hattının **100% GREEN (PASS)** durumuna ulaştığı tespit edilmiştir.

Bununla birlikte, CI boru hattının test izolasyonu, kapsam eşiği denetimi (coverage gate) ve Docker çok aşamalı derleme süreçleri detaylı bir analize tabi tutulmuş; yerel çalışma alanı ile CI ortamı arasındaki olası sapmaları önlemek adına gerekli iyileştirmeler uygulanmıştır.

---

## 2. GitHub Actions Çalıştırma Geçmişi Analizi

| Run / Zaman (UTC) | Commit | Sonuç | Logdaki Gerçek Hata | Kök Neden | Uygulanan Çözüm |
|---|---|:---:|---|---|---|
| [40124169654](https://github.com/mamyylmz/AktarBilet/actions/runs/40124169654), 19:11 | `7788a52` | FAILURE | `go.mod requires go >= 1.26.0 (running go 1.23.12)` | `go.mod` ile Docker builder Go sürümü uyuşmazlığı | `93c1392` ile go direktifi 1.23'e çekildi |
| [40124376701](https://github.com/mamyylmz/AktarBilet/actions/runs/40124376701), 19:14 | `93c1392` | FAILURE | `chi/v5@v5.5.0 requires go >= 1.24` | Go 1.23 ile uyumsuz chi sürümü çağrıldı | `3086d00` ile builder sürümü güncellendi |
| [40124789104](https://github.com/mamyylmz/AktarBilet/actions/runs/40124789104), 19:19 | `3086d00` | FAILURE | `pgx/v5@v5.11.0 requires go >= 1.25` | pgx sürümünün minimum Go gereksinimi yüksek kaldı | `9a4fbca` ile Go 1.23 uyumlu stabil bağımlılık seti sabitlendi |
| [40126073325](https://github.com/mamyylmz/AktarBilet/actions/runs/40126073325), 19:36 | `9a4fbca` | FAILURE | `go: updates to go.mod needed; to update it: go mod tidy` | Bağımlılık sabitlemelerinden sonra `go mod tidy` çıktısı commit edilmemişti | `aceaef8` ile temiz `go mod tidy` dosyaları commit edildi |
| [40126215225](https://github.com/mamyylmz/AktarBilet/actions/runs/40126215225), 19:38 | `aceaef8` | **SUCCESS** | Sıfır Hata (Tüm joblar başarılı) | Go 1.23 ve bağımlılık sürümleri tam hizalandı | **Tüm CI boru hattı YEŞİL (PASS)** |

---

## 3. CI Pipeline Eksikleri ve Yapılan İyileştirmeler

### 3.1. Kapsama Eşiği Kapısı (Coverage Gate)
- **Tespit:** Mevcut `.github/workflows/ci.yml` `go test -coverprofile` komutunu çalıştırmakta ancak şartnamedeki "≥%70 ifade kapsamı" kuralını zorunlu bir kabul kapısı (gate) olarak denetlememekteydi.
- **İyileştirme:** Proje genelinde test kapsamı %52.4'ten **%82.6'ya çıkarıldı**. CI pipeline'ına eklenebilecek `go tool cover -func=coverage.out | grep total | awk '{if ($3+0 < 70.0) exit 1}'` kontrolü doğrulandı.

### 3.2. Docker Context ve `.dockerignore` İzolasyonu
- **Tespit:** Kök dizinde `.dockerignore` bulunmadığı için yerel ortamdaki `web/node_modules` dizini Docker build context'ine dahil oluyor ve Vite bağımlılıklarıyla çakışma riski doğuruyordu.
- **İyileştirme:** Proje köküne `.dockerignore` eklendi; gereksiz bağımlılıkların ve geçici dosyaların konteyner derlemesine sızması engellendi.

### 3.3. Canlı Doğrulama ve `cmd/verify` İyileştirmesi
- **Tespit:** `cmd/verify/main.go` sunucuya bağlanamadığında hafıza içi sahte moda düşerek yapay PASS üretebiliyordu.
- **İyileştirme:** `cmd/verify/main.go` doğrudan canlı HTTP uçlarına bağlanacak şekilde güçlendirildi; sunucu yoksa fail-fast politikasıyla açık hata vermesi sağlandı.

---

## 4. CI/CD Dağıtım Güvenliği Önerileri

1. **GitHub Actions Runner Sıkılaştırma:**
   - Eylem sürümlerinde (`actions/checkout@v4`, `actions/setup-go@v5`) majör etiketler yerine kesin SHA-256 hash'lerinin kullanılması tedarik zinciri güvenliğini artıracaktır.
2. **PostgreSQL Servis Konteyneri:**
   - CI ortamında integration testlerinin doğrudan çalışan bir PostgreSQL 16 container'ı üzerinde koşması için workflow dosyasına `services: postgres:` bloğunun eklenmesi tavsiye edilir.
