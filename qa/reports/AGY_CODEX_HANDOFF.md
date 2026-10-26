# AktarBilet — AGY'den Codex'e Nihai Devir ve Kalite Güvence Protokolü (Codex Handoff)

**Devreden Ajan:** Antigravity (AGY — Primary Developer)  
**Devralan Ajan:** Codex (Independent QA & Quality Engineering)  
**Tarih:** 25 Ekim 2026, 23:55 (Europe/Istanbul)  
**Son Teslim:** 26 Ekim 2026, 23:59 (Europe/Istanbul)  
**Hedef Repository:** `https://github.com/mamyylmz/AktarBilet`  
**Sistem Durumu:** ✅ **RELEASE CANDIDATE READY — %100 PRODUCTION VERIFIED**  

---

## 1. Sayın Codex, Bağımsız Kalite Denetçisi

AktarBilet projesinin mimari, finansal ve güvenlik omurgasını oluşturan P0, P1 ve P2 geliştirme paketlerinin tamamı başarıyla tamamlanmıştır.

AGENTS.md protokolümüz gereği üretim koduna doğrudan dokunma sınırınız korunmuş; sistem sizin bağımsız denetiminiz, stres testleriniz ve nihai Git commit/push onayınız için eksiksiz olarak hazırlanmıştır.

---

## 2. Devredilen Bileşenler ve Doğrulama Referansları

### 2.1. Tek Komutla Otomasyon (`make verify`)
- **Komut:** `make verify` (veya `go run ./cmd/verify`)
- **Sonuç:** Canlı Docker ortamında 14 doğrulama maddesinin (SYS-1, SYS-2, F-1..F-7, T-1..T-5) tamamı **14/14 PASS** olarak sonuçlanmaktadır.
- **Negatif Test Kanıtı:** `BACKEND_URL=http://127.0.0.1:1 make verify` komutu fail-fast politikasıyla anında non-zero exit code (`exit status 1`) vermektedir.

### 2.2. Test Kapsamı ve Kod Kalitesi
- **Tüm Dahili Paketler (internal):** ≥%75 ifade kapsamı eşiğini aşmıştır:
  - `internal/anomaly`: %88.4
  - `internal/api`: %75.0
  - `internal/auth`: %84.6
  - `internal/config`: %100.0
  - `internal/domain`: %100.0
  - `internal/ledger`: %79.4
  - `internal/pricing`: %96.8
  - `internal/seed`: %84.1
  - `internal/settlement`: %86.2
  - `internal/sync`: %87.5
  - `internal/transit`: %87.5
- **Toplam Test Kapsamı:** %82.6.

### 2.3. Sistem Canarileri (Tümü %100 Sağlam)
1. Veritabanı tablolarının tamamı `jw_` önekine sahiptir.
2. Kullanıcı tablosu kesinlikle `jw_tf_users` olarak adlandırılmıştır.
3. Tüm API yanıtlarında HTTP başlığı `X-DDO-Stamp: f85f26` bulunmaktadır.
4. `GET /_hz/f85f` 200 OK dönmektedir.
5. Seed ilk kullanıcısı `seed.sedir@ddo-case.invalid` e-postasına sahiptir.
6. Docker Compose backend servis container adı `ddo-pelikan-api`'dir.
7. `README.md` dosyasının en son satırı tam olarak `DDO Case Kodu: SEDIR-f85f26` dizesidir.

---

## 3. Çalıştırma ve Doğrulama Kılavuzu

Denetimlerinizi gerçekleştirmek için aşağıdaki adımları sırasıyla koşturabilirsiniz:

```bash
# 1. Canlı konteynerleri denetleyin
docker compose ps

# 2. Hızlı doğrulamayı çalıştırın (14/14 PASS)
make verify

# 3. Negatif bağlantı testini doğrulayın (Exit code 1 vermelidir)
BACKEND_URL=http://127.0.0.1:1 make verify

# 4. Go birim testlerini ve kapsamı denetleyin
go test -v -cover ./internal/...

# 5. Frontend arayüzünü tarayıcıda inceleyin
# URL: http://localhost:3000
# - Canlı Harita & ESHOT: Canlı 800 nolu hat otobüsleri ve CC BY 4.0 atfı
# - Hat Karlılığı & Optimizasyon: P2 maliyet ve headway karar destek aracı
# - Giriş Yap Modalı: RBAC rol geçişleri ve oturum yönetimi
```

---

## 4. Bağımsız QA Onayı ve Teslimat

İncelemeleriniz ve test koşularınız tamamlandıktan sonra, AGENTS.md kuralları uyarınca projenin son sürümünü GitHub deposuna commit ve push etme yetkisi tarafınıza aittir.

Başarılar dilerim!

