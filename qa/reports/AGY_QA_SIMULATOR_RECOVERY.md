# AktarBilet — Bağımsız Simülatör, Çevrimdışı WAL ve Kurtarma Denetimi

**Denetçi:** Antigravity (Kıdemli QA Mimarı & Altyapı Denetçisi)  
**Tarih:** 26 Ekim 2026  
**Bileşen:** `cmd/simulator/main.go`, `Dockerfile.simulator`, `docker-compose.yml`, `migrations/009`  
**Referans:** DDO Şartnamesi Bölüm 8 (Büyük Ölçekli Simülatör ve Dayanıklılık)  

---

## 1. MİMARİ VE ÇALIŞMA PRENSİBİ

AktarBilet Simülatörü, sahadaki 4 ana hat validatörünü (500T, 14ES, T1 Tramvay, Vapur) temsil eden ve ağ bağlantısı kopsa dahi biniş olaylarını yerel disk günlüğüne (WAL - Write-Ahead Log) kaydeden deterministik bir arka plan servisidir.

### Temel Özellikler:
- **Deterministik Tohum (Seed=42):** Aynı tohumla başlatıldığında, zaman ve rastgelelikten bağımsız olarak her zaman aynı olay dizisini ve SHA-256 manifest özetini üretir.
- **Sınırlı Bellek Kuyruğu (Bounded Queue):** `queue-cap: 100` ile bellek tüketimini sınırlar; ağ kesintisinde kuyruk dolsa dahi yeni olayları diske (WAL) yazar.
- **Disk Destekli WAL Günlüğü:** `simulator_wal.jsonl` dosyasında onaylanmamış (unacked) paketleri saklar. Servis veya konteyner çöktüğünde veriler kaybolmaz.
- **Monotonik Sıra Numarası (Sequence Counter):** `simulator_seq.json` dosyasında her validatörün son sıra numarasını saklayarak yeniden başlatma sonrasında sayaç sıfırlanmasını engeller.

---

## 2. GEÇMİŞ CODEX TESTİNDE TESPİT EDİLEN ARIZALAR VE KÖK NEDENLERİ

Codex'in önceki `simulator-runtime.json` denetiminde simülatörün **FAIL** almasının iki kritik kök nedeni bulunmaktaydı:

### 2.1. İzin Hatası (Permission Denied) — Docker Named Volume
- **Belirti:** `simulator-runtime.json` içinde `"wal_open_error": "permission denied"`, `"wal_file_present": false`.
- **Kök Neden:**
  1. `Dockerfile.simulator` içinde güvenlik gereği `USER appuser` (UID 10001, non-root) tanımlanmıştır.
  2. `docker-compose.yml` içinde `simulator_data` adında bir Docker named volume `/var/lib/aktarbilet-simulator` dizinine bağlanmıştır.
  3. Docker daemon, named volume ilk oluşturulduğunda ana dizini varsayılan olarak `root:root (0755)` izinleriyle bağlar.
  4. Non-root `appuser` (10001) bu dizin altında dosya oluşturamadığı için `os.OpenFile` aşamasında `permission denied` hatası alıyordu.
- **Çözüm Mimarisi:**
  - `docker-compose.yml` dosyasına eklenen `infra-init` servisi, simülatör başlamadan önce `/data` volume'ünü `chown -R 10001:10001 /data && chmod 0700 /data` komutuyla yetkilendirmektedir (`simulator-storage-init`).

### 2.2. Batch-Sync Sırasında HTTP 409 Conflict
- **Belirti:** Simülatör çevrimdışı biriken paketleri backend'e iletmek istediğinde `POST /api/v1/validators/batch-sync` çağrısı HTTP 409 Conflict dönüyordu.
- **Kök Neden:**
  - Backend tarafında `jw_device_requests` tablosundaki `uq_jw_device_requests_nonce` kısıtı, tek bir HTTP isteği içinde gelen 10'luk paket olaylarının hepsinde aynı istek nonce'ını gördüğü için 2. olayda tekillik ihlali veriyordu.
- **Çözüm Mimarisi:**
  - `migrations/009_fix_batch_device_nonce.sql` ile bu kısıt kaldırılarak paketlerin tek bir veritabanı transaction'ında atomik işlenmesi sağlandı.

---

## 3. BAĞIMSIZ DOĞRULAMA VE KURTARMA TESTLERİ

### 3.1. Ağ Kesintisi ve WAL Üretimi
1. Backend servisi durdurulduğunda (`docker compose stop backend`), simülatörün HTTP 503 aldığı ve istekleri düşürmediği görülmüştür.
2. Olaylar `simulator_wal.jsonl` dosyasına eklenmeye devam etmiştir.
3. Bellek kuyruğu dolduğunda backpressure devreye girmiş ve bellek taşması engellenmiştir.

### 3.2. Konteyner Yeniden Başlatma (Crash Recovery)
1. Simülatör çalışırken `docker compose restart simulator` komutu verilmiştir.
2. Simülatör ayağa kalktığında:
   - `simulator_seq.json` dosyasını okuyarak kaldığı sıra numarasını yüklemiştir.
   - `simulator_wal.jsonl` dosyasını tarayarak `unacked` durumdaki paketleri tespit etmiştir.
   - Backend hazır olduğunda `wal.ReplayPending()` metoduyla onaylanmamış paketleri yeniden iletmiştir.
3. Tekrarlanan paketler `idempotency_key` koruması sayesinde veritabanında çift tahsilat oluşturmamıştır.

---

## 4. DOCKER YAPILANDIRMA VE OPTİMİZASYON TAVSİYELERİ

1. **Bağımlılık Zinciri Sıralaması:**
   - `simulator` servisi mutlaka `depends_on: { infra-init: { condition: service_completed_successfully } }` koşuluna bağlı kalmalıdır. Aksi halde volume izinleri root kalır ve WAL açılamaz.
2. **Graceful Shutdown Süresi:**
   - Compose dosyasında `simulator` servisine `stop_grace_period: 30s` eklenmesi tavsiye edilir; böylece SIGTERM sinyali alındığında çalışan worker'lar kuyruktaki mevcut paketi diske güvenle yazıp çıkış yapabilir.

