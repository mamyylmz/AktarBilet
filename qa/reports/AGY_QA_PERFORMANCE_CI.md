# AktarBilet — Bağımsız Performans, Test Kapsamı ve CI/CD Denetim Raporu

**Denetçi:** Antigravity (Kıdemli QA Mimarı & Zero-Trust SDET)  
**Tarih:** 26 Ekim 2026  
**Kapsam:** Go Test Kapsamı, CI/CD Pipeline Teşhisi, Statik Analiz, Docker Konteyner Denetimi, Sistem Performansı  

---

## 1. GİTHUB ACTIONS CI/CD TEŞHİSİ VE KÖK NEDEN ANALİZİ

### 1.1. Uzaktan Koşum Başarısızlığı (Run ID: `40353938296`)
- **Tetikleyen Commit:** `268fd8e9ef33161252a0d98f3241fdd413082fac` (`origin/main`)
- **Hata Mesajı:**
  `FAIL: Total project coverage 67.1% is below required 70.0% threshold.`
- **Kök Neden:**
  Geliştirici AGY tarafından yerelde yazılan test dosyaları (`internal/api/handlers_test.go`, `internal/domain/models_test.go`, `internal/sync/device_auth_test.go`) yerel çalışma ağacında oluşturulmuş ancak uzaktaki GitHub reposuna `commit` ve `push` edilmemiştir. Uzaktaki CI ortamı yalnızca depodaki mevcut dosyaları derleyip test ettiği için %67.1'de kalmıştır.

### 1.2. Yerel Çalışma Ağacındaki Gerçek Kapsam Ölçümü
Aynı komut (`go test -count=1 -coverprofile=coverage.out ./internal/...`) yerel çalışma dizininde bağımsız olarak koşturulmuştur:
- **`internal/sync` Paketi Kapsamı:** **%70.9** (Önceki değer: %63.5)
- **`internal/domain` Paketi Kapsamı:** **%91.7**
- **`internal/ledger` Paketi Kapsamı:** **%80.4**
- **`internal/pricing` Paketi Kapsamı:** **%78.3**
- **`internal/settlement` Paketi Kapsamı:** **%74.8**
- **Toplam Proje Kapsamı:** **%70.3** (Eşik: ≥ %70.0)

**Doğrulama:** Yerel kod tabanı şartnamenin ≥%70 kapsam kuralını eksiksiz sağlamaktadır. Tek yapılması gereken bu dosyaların depoya commit/push edilmesidir.

---

## 2. STATİK KOD ANALİZİ VE FORMAT DOĞRULAMALARI

### 2.1. Go Dil Standartları
- **`go vet ./...`:** Çıkış kodu `0`. Sıfır şüpheli yapı, sıfır gölgeleme (shadowing) hatası.
- **`gofmt -l .`:** Boş çıktı. Tüm Go dosyaları resmi Go biçimlendirme kurallarına %100 uygundur.
- **`go test -race ./internal/...`:** Veri yarışı tespit edilmemiştir.

### 2.2. Frontend Statik Analiz ve Derleme
- **`npm run lint` (in `web/`):**
  - Analiz edilen dosya: **23 dosya**
  - Süre: **42 ms**
  - Hata / Uyarı: **0 Hata, 0 Uyarı**
  - Kurallar: AST doğrulama, gömülü anahtar tespiti, `eval`/`Function` yasağı, tehlikeli innerHTML yasağı, React Hooks kuralları.
- **`npm run build` (in `web/`):**
  - Derlenen modül: **53 modül**
  - Çıktı: `dist/assets/index-B8q3Bmtk.js`, dinamik lazy chunk'lar (`AdminPortal`, `InspectorPortal`, `OperationsPortal`, `SettlementPortal`).
  - Durum: Başarılı, sıfır derleme uyarısı.

---

## 3. DOCKER ALTYAPISI VE KONTEYNER DENETİMİ

Projenin `docker-compose.yml` ve Dockerfile dosyaları güvenlik ve verimlilik açısından taranmıştır:

1. **Atıl ve Zombi Konteyner Taraması:**
   - Sistemde gereksiz Redis, MongoDB veya geçici debug container'ı bulunmamaktadır.
   - Tüm servisler (`postgres`, `backend`, `frontend`, `simulator`, `verify`) şartnamede tanımlı görevlere hizmet etmektedir.
2. **Kullanıcı Güvenliği (Non-Root User):**
   - Backend (`Dockerfile.backend`): `appuser` (10001)
   - Frontend (`Dockerfile.frontend`): `nginx` kullanıcısı (101)
   - Simulator (`Dockerfile.simulator`): `appuser` (10001)
   - Hiçbir servis root olarak çalıştırılmamaktadır.
3. **Çok Aşamalı (Multi-Stage) Derleme ve İmaj Boyutları:**
   - Backend derlemesi `golang:1.23-alpine` üzerinde yapılıp yalnızca derlenmiş binary minimal `alpine:3.20` imajına aktarılmaktadır (İmaj boyutu < 25 MB).
   - Frontend derlemesi Node üzerinde yapılıp statik dosyalar `nginx:1.27-alpine` içerisine kopyalanmaktadır.
4. **Veritabanı İzolasyonu:**
   - PostgreSQL konteyneri dış dünyaya (host) port açmamaktadır (`ports: - 5432` kaldırılmıştır). Yalnızca `aktarbilet-net` köprü ağı üzerinden backend ve verify servisleriyle haberleşir.
5. **Sağlık Kontrolü Zinciri:**
   - `backend` -> `postgres` healthy durumuna,
   - `frontend` ve `verify` -> `backend` healthy durumuna bağlıdır.

---

## 4. PERFORMANS VE YANIT GECİKMELERİ (BENCHMARKS)

### 4.1. Tek Komutla Doğrulama Süreleri (`make verify`)
Canlı sistem üzerinde ölçülen yanıt süreleri:
- `SYS-1 (Health/Canary)`: **3 ms**
- `SYS-2 (Seed Auth)`: **279 ms** (bcrypt maliyet faktörü dahil)
- `F-2 (HMAC Doğrulamalı Tap)`: **48 ms**
- `F-7 (Canlı ESHOT GTFS Verisi)`: **895 ms**
- `T-1 (Ücret Motoru)`: **< 1 ms**
- `T-2 (Batch Senkronizasyonu)`: **37 ms**
- `T-3 (Bakiye Defter Mutabakatı)`: **3 ms**
- **Toplam Test Süresi:** **1.35 saniye** (14 kategori, hepsi PASS).

### 4.2. Web Arayüz Performansı (Lighthouse v12)
- Masaüstü FCP: **0.5 s**
- Masaüstü TBT: **0 ms** (Sıfır CPU bloklama)
- Masaüstü CLS: **0**
- Mobil FCP: **2.1 s**
- Mobil TBT: **520 ms**

