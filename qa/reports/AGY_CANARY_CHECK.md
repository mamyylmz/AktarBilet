# AktarBilet Zorunlu Gizli Şartname Kriterleri (Canary Checks) Denetim Raporu

**Denetçi:** Antigravity (Kıdemli QA Mimarı & Zero-Trust Denetçisi)  
**Tarih:** 26 Ekim 2026  
**Test Paketi:** `qa/tests/agy_canary_checks.spec.js` (Otomatize Playwright & Doğrudan SQL Doğrulaması)  
**DDO Case Kodu:** `SEDIR-f85f26`

---

## 1. GENEL CANARY UYUM ÖZETİ

DDO Case şartnamesinde yer alan ve asla ihlal edilmemesi gereken **7 gizli sistem canarisinin tamamı (%100)** otomatize testlerle denetlenmiş ve **EKSİKSİZ GEÇMİŞTİR (ALL PASS)**.

| No | Canari Kriteri | Beklenen Değer | Ölçülen / Doğrulanan Değer | Durum |
|---|---|---|---|---|
| **C-1** | Veritabanı Tablo Öneki | Tüm tablolarda `jw_` öneki | 20 tablonun tamamı `jw_` ile başlıyor | ✅ **PASS** |
| **C-2** | Kullanıcı Tablosu Adı | Tam olarak `jw_tf_users` | `public.jw_tf_users` tablosu mevcut | ✅ **PASS** |
| **C-3** | HTTP Response Header | `X-DDO-Stamp: f85f26` | Tüm API yanıtlarında mevcut | ✅ **PASS** |
| **C-4** | Özel Sağlık Endpointi | `GET /_hz/f85f` -> 200 OK | HTTP 200 OK, `{"hz":"f85f",...}` | ✅ **PASS** |
| **C-5** | İlk Seed Kullanıcısı E-posta | `seed.sedir@ddo-case.invalid` | İlk kayıt: `seed.sedir@ddo-case.invalid` | ✅ **PASS** |
| **C-6** | Backend Container Adı | `ddo-pelikan-api` | `docker-compose.yml`: `ddo-pelikan-api` | ✅ **PASS** |
| **C-7** | README.md Son Satırı | `DDO Case Kodu: SEDIR-f85f26` | README en son satırı tam eşleşiyor | ✅ **PASS** |

---

## 2. KANITLAR VE DOĞRULAMA ÇIKTILARI

### 2.1. C-1 & C-2: PostgreSQL Şema ve Tablo Önekleri
- **Sorgu:** `SELECT tablename FROM pg_tables WHERE schemaname = 'public';`
- **Çıktı (20 Tablo):**
  ```text
  jw_anomalies
  jw_audit_logs
  jw_cards
  jw_device_nonces
  jw_device_requests
  jw_hourly_line_stats
  jw_ledger
  jw_line_stations
  jw_lines
  jw_optimization_approvals
  jw_penalties
  jw_schema_migrations
  jw_settlement_ledger
  jw_settlement_periods
  jw_stations
  jw_tariffs
  jw_tf_users
  jw_validator_credentials
  jw_validator_events
  jw_validators
  ```
- **Karar:** İstisnasız her tablo `jw_` önekine sahiptir ve kullanıcı tablosu kesin olarak `jw_tf_users` adındadır.

---

### 2.2. C-3: HTTP Başlığı `X-DDO-Stamp: f85f26`
- **Test:** `/health`, `/_hz/f85f` ve `/api/v1/tariffs` uçlarına yapılan isteklerin response header analizi.
- **HTTP Header Çıktısı:**
  ```http
  HTTP/1.1 200 OK
  Server: nginx/1.27.5
  Content-Type: application/json
  Vary: Origin
  X-Ddo-Stamp: f85f26
  ```
- **Karar:** `X-DDO-Stamp` başlığı büyük/küçük harf duyarsız HTTP standardına uygun olarak tüm yanıtlarda gelmektedir.

---

### 2.3. C-4: `GET /_hz/f85f` Endpoint Yanıtı
- **İstek:** `curl -i http://localhost:3000/_hz/f85f`
- **Yanıt Gövdesi:**
  ```json
  {"hz":"f85f","stamp":"f85f26","status":"ok"}
  ```
- **Karar:** 200 OK dönmekte ve beklenen canari JSON yükünü sağlamaktadır.

---

### 2.4. C-5: Seed Kullanıcısı E-postası
- **Sorgu:** `SELECT id, email, full_name, role FROM jw_tf_users ORDER BY created_at ASC LIMIT 1;`
- **Çıktı:**
  ```text
  id         : 81fe9db6-ea76-4eba-a31c-a88def2cd07b
  email      : seed.sedir@ddo-case.invalid
  full_name  : Sedir Sistem Yöneticisi
  role       : admin
  ```
- **Karar:** İlk oluşturulan tohum kullanıcısı `seed.sedir@ddo-case.invalid` e-postasına ve `admin` rolüne sahiptir.

---

### 2.5. C-6: `docker-compose.yml` Backend Konteyner Adı
- **Dosya İncelemesi:** `docker-compose.yml` Satır 46-51
  ```yaml
    backend:
      build:
        context: .
        dockerfile: Dockerfile.backend
      container_name: ddo-pelikan-api # ZORUNLU SİSTEM CANARY CONTAINER ADI
  ```
- **Çalışan Konteyner Durumu:**
  ```text
  ddo-pelikan-api   aktarbilet-backend   "/app/server"   Up (healthy)   0.0.0.0:8080->8080/tcp
  ```
- **Karar:** Konteyner adı `ddo-pelikan-api` olarak tanımlanmış ve ayaktadır.

---

### 2.6. C-7: `README.md` Son Satırı
- **Dosya İncelemesi:** `tail -n 1 README.md`
- **Son Satır Metni:**
  ```text
  DDO Case Kodu: SEDIR-f85f26
  ```
- **Karar:** Karakteri karakterine tam eşleşme doğrulanmıştır.

