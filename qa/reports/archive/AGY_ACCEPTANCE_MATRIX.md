# AktarBilet — AGY Kabul Testi Matrisi (Acceptance Matrix)

**Doğrulama Aracı:** `cmd/verify/main.go` (`make verify`)  
**Tarih:** 25 Ekim 2026  
**Hedef Ortam:** Canlı Docker Compose (`ddo-pelikan-api`, `ddo-postgres`, `ddo-frontend`, `ddo-simulator`)  
**Genel Durum:** ✅ **14 / 14 TEST GEÇTİ (%100 PASS)**  

---

## 1. Doğrulama Özeti

| Kod | Kategori | Özellik / Gereksinim | Doğrulama Yöntemi | Durum | Yanıt Süresi |
|---|---|---|---|:---:|:---:|
| **SYS-1** | Sistem Canarisi | `GET /health` & `GET /_hz/f85f` (Header `X-DDO-Stamp: f85f26`) | Canlı HTTP Çağrısı | **PASS** | 2 ms |
| **SYS-2** | Kimlik / Güvenlik | Seed Kullanıcı Oturumu (`seed.sedir@ddo-case.invalid`) & Yanlış Parola Reddi | Canlı HTTP Çağrısı | **PASS** | 138 ms |
| **F-1** | Fonksiyonel | Hat, Durak ve Tarife Yönetimi (Katalog API) | `GET /lines`, `/stations`, `/tariffs` | **PASS** | 6 ms |
| **F-2** | Fonksiyonel | Turnike Geçiş ve Doğrulama (`POST /validators/tap`) | Canlı Validatör HTTP Çağrısı | **PASS** | 14 ms |
| **F-3** | Fonksiyonel | Bakiye Yükleme ve Ödeme Entegrasyonu (`POST /cards/{id}/recharge`) | Yetkili Yolcu HTTP Çağrısı | **PASS** | 5 ms |
| **F-4** | Güvenlik / RBAC | Kart Sahibi Güvenliği (Anti-IDOR & 403 Forbidden Denetimi) | Negatif IDOR Testi | **PASS** | 4 ms |
| **F-5** | Saha / Denetim | Saha Kontrol ve Ceza Takip Mekanizması (`/penalties`) | Kontrolör HTTP Çağrısı | **PASS** | 2 ms |
| **F-6** | Finansal | Gelir Paylaşımı ve Muhasebe Takas (`/settlements/close`) | Muhasebe HTTP Çağrısı | **PASS** | 6 ms |
| **F-7** | Analitik / Canlı | Doluluk Isı Haritası & Gerçek ESHOT Ulaşım Verisi | `GET /heatmap`, `GET /transit/live-buses` | **PASS** | 669 ms |
| **T-1** | Zorlu Görev | Ücret Motoru (90 dk pencere, 2 aktarma, ters yön kuralı) | PricingEngine Algoritması | **PASS** | <1 ms |
| **T-2** | Zorlu Görev | Çevrimdışı Validatör Senkronizasyonu & İdempotansi (409 Çakışma) | Canlı Replay & Conflict Testi | **PASS** | 15 ms |
| **T-3** | Finansal Değişmez | Kart Bakiye Defteri (Append-Only Ledger & Bakiye Eşitliği) | `balance == SUM(amount)` | **PASS** | 1 ms |
| **T-4** | Finansal Değişmez | Gelir Paylaşımı (Hare-Niemeyer Kuruş Koruma, Kapanış, Storno) | Settlement Algoritması | **PASS** | <1 ms |
| **T-5** | Güvenlik / Kaos | Anormallik Tespiti (İmkansız Seyahat Hızı >300 km/s & Klon Kart) | Anomaly Detector Engine | **PASS** | <1 ms |

---

## 2. Negatif Test Doğrulaması

- **Komut:** `BACKEND_URL=http://127.0.0.1:1 make verify`
- **Beklenen Sonuç:** Fail fast politikası gereği hemen durmalı ve `exit code 1` dönmelidir.
- **Gerçekleşen Sonuç:**
  ```text
  [HATA] Arka uç servisine bağlanılamadı (http://127.0.0.1:1): Get "http://127.0.0.1:1/health": dial tcp 127.0.0.1:1: connect: connection refused
  Lütfen backend servisinin çalıştığından emin olun (docker compose up veya go run ./cmd/server).
  exit status 1
  ```
- **Doğrulama:** Negatif test %100 başarıyla onaylandı.

