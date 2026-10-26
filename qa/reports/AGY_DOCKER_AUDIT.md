# AktarBilet Docker Altyapı ve Konteyner Mimarisi Denetim Raporu

**Denetçi:** Antigravity (Kıdemli QA Mimarı, SDET ve DevOps/Altyapı Denetçisi)  
**Tarih:** 26 Ekim 2026  
**Hedef Ortam:** Docker Compose & Container Altyapısı (`/home/dilunsel/projects/AktarBilet`)  
**DDO Case Kodu:** `SEDIR-f85f26`  
**Denetim Metodolojisi:** Zero-Trust Altyapı ve Güvenlik Taraması (Statik Yapılandırma Analizi, Çalışma Zamanı Teftişi, İmaj Katman İncelemesi)

---

## 1. YÖNETİCİ ÖZETİ VE KRİTİK BULGULAR

Docker mimarisi genel olarak modern multi-stage derleme ve non-root prensiplerini büyük ölçüde benimsemiş olsa da, **canlı dağıtımı doğrudan çökertecek kritik bir OCI runtime hatası**, **atıl konteyner/imaj birikimi** ve **gereksiz konteyner ayrıştırmaları** tespit edilmiştir:

| ID | Seviye | Bileşen | Açıklama | Durum |
|---|---|---|---|---|
| **DKR-01** | 🔴 KRİTİK | `verify` Servisi | `Dockerfile.backend` minimalist Alpine tabanlıdır; içinde `go` derleyicisi bulunmadığından `entrypoint: ["go", "run", "./cmd/verify"]` komutu OCI runtime seviyesinde patlamaktadır (`exec: "go": executable file not found in $PATH`). | **HATA VERİYOR** |
| **DKR-02** | 🟡 ORTA | `credential-init` & `simulator-storage-init` | İki ayrı ephemeral Alpine konteyneri yalnız bir JSON dosyası üretmek ve bir dizinin yetkisini değiştirmek için ayağa kalkmaktadır. Tek init konteynerinde birleştirilmelidir. | **ATIL KAYNAK** |
| **DKR-03** | 🟡 ORTA | `simulator` Servisi | Üretim/canlı ortam compose dosyasında (`docker-compose.yml`) simülatör varsayılan olarak başlatılmaktadır. Bu durum canlı sisteme sürekli sentetik tap yükü bindirir; `profiles: ["simulator"]` altına alınmalıdır. | **MİMARİ RİSK** |
| **DKR-04** | 🟡 ORTA | Konteyner & İmaj Kirliliği | Host üzerinde 15'ten fazla durdurulmuş ("exited/zombie") test konteyneri ve 35'ten fazla ara sürüm imajı (toplam ~12 GB) birikmiştir. | **DEPOLAMA ŞİŞMESİ** |
| **DKR-05** | 🟢 DÜŞÜK | `frontend` Healthcheck | Compose içinde `http://localhost:3000/` kullanılırken Alpine Linux'ta localhost'un `::1` (IPv6) adresine çözülmesi riski bulunmaktadır; `http://127.0.0.1:3000/` kullanımı daha kararlıdır. | **KARARLILIK RİSKİ** |

---

## 2. SERVİS BAZLI DERİNLEMESİNE İNCELEME

### 2.1. `verify` Servisi (🔴 KRİTİK ÇÖKME BULGUSU)
- **Dosya:** `docker-compose.yml` (Satır 121-139)
- **Yapılandırma:**
  ```yaml
  verify:
    build:
      context: .
      dockerfile: Dockerfile.backend
    container_name: ddo-verify
    entrypoint: ["go", "run", "./cmd/verify"]
  ```
- **Hata Kanıtı (Çalışma Zamanı Logu):**
  ```text
  Error response from daemon: failed to create task for container: failed to create shim task: 
  OCI runtime create failed: runc create failed: unable to start container process: 
  error during container init: exec: "go": executable file not found in $PATH: unknown
  ```
- **Kök Neden:** `Dockerfile.backend` çok aşamalı (multi-stage) bir Dockerfile'dır. İlk aşamada (`golang:1.23-alpine AS builder`) binary derlenir, son aşamada ise minimalist `alpine:3.20` tabanına yalnız `/app/server` kopyalanır. Son imajda Go SDK (`go` binary) bulunmadığı için `go run` çalıştırılamaz.
- **Geliştiriciye Öneri:**
  1. Ya `Dockerfile.backend` içinde `verify` ikilisini de derleyip `/app/verify` olarak son imaja kopyalayın ve `entrypoint: ["/app/verify"]` yapın,
  2. Ya da `verify` için `golang:1.23-alpine` tabanlı bağımsız bir `Dockerfile.verify` oluşturun.

---

### 2.2. Ephemeral Init Servisleri (`credential-init` ve `simulator-storage-init`)
- **Dosya:** `docker-compose.yml` (Satır 2-25)
- **İnceleme:**
  - `credential-init`: `alpine:3.20` imajıyla başlar, `/secrets/validator-credentials.json` oluşturur ve çıkar.
  - `simulator-storage-init`: Başka bir `alpine:3.20` imajıyla başlar, `/data` üzerinde `chown -R 10001:10001` yapar ve çıkar.
- **Kullanıcı İhlali:** Her iki init servisi de `root` kullanıcısı ile çalışır.
- **Değerlendirme:** İki ayrı container ayağa kaldırmak Docker startup süresini gereksiz uzatmakta ve compose dosyasını karmaşıklaştırmaktadır. Tek bir `infra-init` container'ı altında birleştirilebilir veya backend/simulator startup script'ine entegre edilebilir.

---

### 2.3. `backend` (`ddo-pelikan-api`)
- **İmaj:** `aktarbilet-backend:latest` (Boyut: 41.2 MB)
- **Dockerfile:** `Dockerfile.backend` (Multi-stage, Alpine 3.20 tabanlı)
- **Kullanıcı Güvenliği:** `USER appuser` (Non-root, UID/GID sistem grubu). **BAŞARILI**
- **Derleme Optimizasyonu:** `CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s"` bayraklarıyla derlenmiş; sembol tablosu ayıklanarak 41 MB gibi ideal bir boyuta indirilmiştir. **BAŞARILI**
- **Healthcheck:** `CMD curl -f http://localhost:8080/health`. Kararlı çalışmaktadır.
- **Şartname Canarisi:** `container_name: ddo-pelikan-api` kuralı eksiksiz uygulanmıştır.

---

### 2.4. `frontend` (`ddo-frontend`)
- **İmaj:** `aktarbilet-frontend:latest` (Boyut: 74 MB)
- **Dockerfile:** `Dockerfile.frontend` (Multi-stage: Node 20 builder -> Nginx 1.27-alpine)
- **Kullanıcı Güvenliği:** `appuser:appgroup` oluşturulmuş, `/var/run/nginx.pid` ve cache dizinleri yetkilendirilmiş, `USER appuser` ile çalıştırılmıştır. **BAŞARILI**
- **Konfigürasyon Analizi:** Nginx konfigürasyonu SPA yönlendirmesini (`try_files $uri $uri/ /index.html;`), API ters vekilini (`proxy_pass http://backend:8080/api/;`) ve en önemlisi PMTiles için zorunlu olan HTTP byte-range başlığını (`add_header Accept-Ranges bytes;`) eksiksiz sağlamaktadır.

---

### 2.5. `simulator` (`ddo-simulator`)
- **İmaj:** `aktarbilet-simulator:latest` (Boyut: 32 MB)
- **Dockerfile:** `Dockerfile.simulator` (Multi-stage, Alpine 3.20, `USER appuser` UID 10001).
- **Mimari Problem:** Simülatör, arka planda WAL (`simulator_wal.jsonl`) ve sequence dosyalarını kullanarak backend'e sürekli validatör biniş trafiği göndermektedir. `docker-compose.yml` içinde herhangi bir profile (`profiles: [...]`) atanmadığı için `docker compose up` dendiğinde prod ortamında da çalışmaktadır.
- **Öneri:** Simülatör servisine `profiles: ["simulator", "dev"]` eklenerek prod dağıtımlarında atıl/istenmeyen sentetik trafik engellenmelidir.

---

### 2.6. `postgres` (`ddo-postgres`)
- **İmaj:** `postgres:16-alpine` (Boyut: 419 MB)
- **Kullanıcı:** PostgreSQL resmi imajı gereği iç süreçler `postgres` (UID 70) kullanıcısıyla çalışır.
- **Healthcheck:** `pg_isready -U postgres -d aktarbilet` (5s aralık, 5 tekrar). Sağlıklı ve deterministiktir.
- **Veri Kalıcılığı:** `postgres_data` volume'ü ile güvene alınmıştır.

---

## 3. ATIL VE UNUTULMUŞ BİLEŞENLER ENVANTERİ

Host ortamında yapılan incelemede AktarBilet projesi kapsamı dışında kalan veya geçmiş test koşumlarından kalan atıl varlıklar tespit edilmiştir:

1. **Unutulmuş / Atıl Konteynerler (Host Seviyesi):**
   - `aktarbilet-codex-retest3-*` (3 container - Exited)
   - `aktarbilet-codex-retest-*` (4 container - Exited)
   - `aktarbilet-codex-rc2-*` (4 container - Exited)
   - `aktarbilet-codex-final-live-*` (3 container - Exited)
   - *Not:* Proje kuralı uyarınca `appointment-platform-*` ve `rakort-*` konteynerlerine dokunulmamıştır.
2. **Kullanılmayan Ağır İmajlar:**
   - `ghcr.io/zaproxy/zaproxy:stable` (3.71 GB)
   - `mongo:latest` (1.20 GB)
   - `golang:1.23` (1.23 GB)
   - Bu imajlar AktarBilet çalışma zamanında (`docker-compose.yml`) kullanılmamaktadır.

---

## 4. DOCKER SAĞLIK VE ORKESTRASYON MATRİSİ

| Servis Adı | Container Adı | İmaj Boyutu | Non-Root | Healthcheck Durumu | Canlı Durum (Audit Anı) |
|---|---|---|---|---|---|
| `postgres` | `ddo-postgres` | 419 MB | ✅ Evet | ✅ Healthy | **UP (Healthy)** |
| `backend` | `ddo-pelikan-api` | 41.2 MB | ✅ Evet | ✅ Healthy | **UP (Healthy)** |
| `frontend` | `ddo-frontend` | 74.0 MB | ✅ Evet | ✅ Healthy | **UP (Healthy)** |
| `simulator` | `ddo-simulator` | 32.0 MB | ✅ Evet | ✅ Healthy | **UP (Healthy)** |
| `credential-init` | *otomatik* | 12.1 MB | ❌ Hayır (Root) | — | **Completed (0)** |
| `simulator-storage-init` | *otomatik* | 12.1 MB | ❌ Hayır (Root) | — | **Completed (0)** |
| `verify` | `ddo-verify` | 41.2 MB | ✅ Evet | ❌ Yok | **FAILED (Missing Go binary)** |

---

## 5. GELİŞTİRİCİYE AKSİYON YÖNERGELERİ

1. **`verify` Servisini Onarın:** `Dockerfile.backend` içine `RUN CGO_ENABLED=0 go build -o /app/verify ./cmd/verify` adımını ekleyin ve son aşamaya `/app/verify` ikilisini kopyalayın. `docker-compose.yml` içindeki entrypoint'i `["/app/verify"]` olarak güncelleyin.
2. **Init Servislerini Birleştirin:** `credential-init` ve `simulator-storage-init` servislerini tek bir `setup` servisinde birleştirerek gereksiz container başlatma maliyetini düşürün.
3. **Simülatöre Profil Ekleyin:** Prod compose dosyasında simülatörün kazara canlı veritabanını kirletmemesi için `profiles: ["simulator", "dev"]` kısıtlamasını getirin.
4. **Temizlik Scripti Ekleyin:** `make clean-docker` veya `docker compose down -v --remove-orphans` adımıyla eski test konteynerlerini düzenli temizleyen bir rutin tanımlayın.

