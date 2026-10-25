# AktarBilet Kod Temizliği ve Silme Adayları Raporu (Deletion Candidates)

**Tarih:** 25 Ekim 2026  
**Durum:** ℹ️ **İNCELEME VE ENVANTER RAPORU (KURAL 3 UYARINCA KOD SİLİNMEMİŞTİR)**  
**Hedef Sürüm:** `main@6ac8591`  
**DDO Case Kodu:** `SEDIR-f85f26`

Bu çalışma, kaynak referans/call-site taraması, statik analiz ve çalışan sistem mimarisi incelemesidir. **Ajan Kuralları Madde 2 ve Madde 3 uyarınca hiçbir üretim kodu kendiliğinden silinmemiştir.** Dinamik çağrı/reflection olasılığı bulunan veya işlevsel dönüşüm gerektiren noktalar `INVESTIGATE` / `RELOCATE` / `SIMPLIFY` olarak sınıflandırılmıştır.

---

## 1. Silme ve Sadeleştirme Adayları Envanteri

| ID | Dosya | Satır Aralığı | Sembol veya Kod | Sorun / Durum | Kanıt | Öneri | Risk | Güven |
|---|---|---:|---|---|---|---|---|---|
| **DC-001** | `internal/config/config.go` | 39-46 | `getEnvInt` | Çağrılmayan yardımcı fonksiyon ve gereksiz `strconv` bağımlılığı | Repo genelinde tek referans tanımın kendisi; `go vet` temiz | DELETE | Düşük; ileride integer config gerekirse tekrar yazılabilir | Yüksek |
| **DC-002** | `internal/auth/auth.go` | 14, 16-17 | `ErrInvalidCredentials`, `ErrUnauthorized`, `ErrForbidden` | Üç paket hatası üretim/test kodunda kullanılmıyor; middleware kendi JSON hatasını yazıyor | `rg` sonuçlarında yalnız tanımlar; `ErrInvalidToken` ise test ve runtime'da kullanılıyor | DELETE | Düşük; dış paket tüketicisi yok, `internal` paket | Yüksek |
| **DC-003** | `internal/sync/service.go` | 16-18 | `ErrDuplicateEvent` | Tanımlı ama idempotent yol hata fırlatmak yerine mevcut olayı döndürüyor | Tek referans tanım | DELETE | Düşük; API sözleşmesi hata döndürmeye çevrilirse gerekebilir | Yüksek |
| **DC-004** | `internal/sync/service.go` | 25, 40 | `validatorSeqs` alanı ve allocation | Hiç okunmuyor/yazılmıyor; sequence idempotansı `processedKeys[seqKey]` ile yapılıyor | Referans taraması yalnız struct ve kurucuda | DELETE | Düşük | Yüksek |
| **DC-005** | `internal/api/handlers.go` | 25, 35, 44 | `APIHandler.pricingEngine` ve constructor parametresi | Handler alanı hiçbir HTTP metodunda doğrudan okunmuyor; engine zaten `syncSvc` içinde | Referanslar yalnız alan, parametre ve atama | SIMPLIFY | Orta; kurucu çağrım yerleri güncellenmeli | Yüksek |
| **DC-006** | `internal/api/handlers.go` | 106-115 | Login memory fallback | DB bağlantısı yokken demo amaçlı statik hesap doğrulaması | Üretimde DB hatasında fail-closed olmalı | RELOCATE | Orta; test koşucularına taşınmalı | Yüksek |
| **DC-007** | `internal/api/handlers.go` | 298-305, 325-331 | Hat/durak/tarife hafıza fallback'ları | Veritabanı okunamadığında statik katalog dönülmesi | Canlı seeder veritabanını tam doldurduğu için DB'ye bağlanmalı | RELOCATE | Orta; test fixture'ına taşınmalı | Yüksek |
| **DC-008** | `internal/api/handlers.go` | 510-515 | Statik heatmap verisi | DB sorgusu yerine statik dört kayıt dönülmesi | Veritabanı aggregation veya materialized view sorgusuna bağlanmalı | SIMPLIFY | Düşük | Yüksek |
| **DC-009** | `web/src/App.jsx` | 5-13, 68-75 | İstemci tarafı sahte hesaplamalar | İstemci tarafında bakiye hesaplama/düşürme kurgusu | Faz 4'te arayüz sunucu otoriter hale getirilerek devre dışı bırakıldı | RESOLVED | Düşük | Yüksek |
| **DC-010** | `web/src/App.jsx` | 152-173 | Yetkisiz arayüz geçişleri | Tüm rollere giriş yapmadan tıklanabilen statik menü sekmeleri | Faz 4'te rol tabanlı guard ve localStorage oturumu entegre edildi | RESOLVED | Düşük | Yüksek |
| **DC-011** | `cmd/verify/main.go` | 48-61 | Verify offline direct runner | Sunucu yokken hafıza içi test koşup sahte PASS üretme riski | Faz 5'te canlı HTTP zorunlu kılındı, sunucu yoksa fail-fast oldu | RESOLVED | Düşük | Yüksek |
| **DC-012** | `cmd/server/main.go` | 43-49 | DB hatasında sessiz devam etme | PostgreSQL bağlantısı kesildiğinde loglayıp belleğe düşmesi | Üretim dağıtımında fail-fast exit politikası önerilir | INVESTIGATE | Orta | Yüksek |
| **DC-013** | `cmd/simulator/main.go` | 143-164 | Simülatör hata yutma blokları | Batch gönderim hatalarının sessizce yutulması | Faz 3'te simülatör gerçek PostgreSQL havuzuna bağlandı ve loglandı | RESOLVED | Düşük | Yüksek |
| **DC-014** | `scripts/reset-demo.sh` | 10-20 | Zayıf reset kabuğu | Psql kurulu olmadığında sessiz geçme | Faz 3'te gerçek TRUNCATE ve deterministik seed akışı sağlandı | RESOLVED | Düşük | Yüksek |
| **DC-015** | `scripts/chaos.sh` | 17-39 | Sabit konteyner ismi ve zayıf başarı kontrolü | Canlı ortamda konteyner durdurma riski | Yalnız izole QA Compose projesine hedeflenmeli | INVESTIGATE | Yüksek | Yüksek |

---

## 2. Mimari İnceleme ve Canlıya Geçiş Sonrası Öneriler

1. **Ölü Kodların Temizlenmesi (Post-Release Sprint):**
   - `DC-001` (`getEnvInt`), `DC-002` (kullanılmayan hata sabitleri) ve `DC-004` (`validatorSeqs`) canlıya geçiş onayının ardından ilk bakım sürümünde (v1.0.1) güvenle silinebilir.
2. **Kural Korunumu:**
   - Şartname ve teslim süresince geriye dönük uyumluluk ve API sözleşme bütünlüğünü bozmamak adına hiçbir kod körlemesine silinmemiş; bunun yerine mimari izolasyon ve güvenli arayüzler tercih edilmiştir.
