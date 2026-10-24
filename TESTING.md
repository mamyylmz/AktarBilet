# AktarBilet Test ve Doğrulama Kılavuzu (TESTING.md)

Bu belge, DDO Yazılım Ekibi Şartnamesi Bölüm 10 uyarınca sistemin nasıl doğrulanacağını, demo kullanıcı hesaplarını, her doğrulama kimliği için beklenen sonuçları ve zorlu görevlerin kanıt komutlarını içerir.

---

## 1. Demo Kullanıcı Hesapları

Sistem ayağa kalktığında otomatik seed işlemi ile aşağıdaki hesaplar kullanıma hazır hale gelir:

| Rol | E-posta | Parola | Görev ve Yetki Alanı |
|---|---|---|---|
| **Admin / Canary** | `seed.sedir@ddo-case.invalid` | `SedirPassword2026!` | Tam sistem yetkisi (Sistem Canary) |
| **Yolcu** | `yolcu@aktarbilet.gov.tr` | `YolcuPassword123!` | Kart bakiye sorgulama, hareketler, dolum |
| **Kontrolör** | `kontrolor@aktarbilet.gov.tr` | `KontrolorPassword123!` | Saha kart kontrolü, ceza tutanağı düzenleme |
| **Hat Yöneticisi** | `hatyoneticisi@aktarbilet.gov.tr` | `HatYoneticisi123!` | Hat/durak yönetimi, tarife belirleme, simülasyon |
| **Muhasebe** | `muhasebe@aktarbilet.gov.tr` | `MuhasebePassword123!` | Gelir paylaşımı, dönem kapatma, storno |
| **Operasyon** | `operasyon@aktarbilet.gov.tr` | `OperasyonPassword123!` | Anlık validatör akışı, doluluk ısı haritası, anomali |

---

## 2. Otomatik Doğrulama Komutu (Tek Komutla Doğrulama)

Tüm sistemi ve gereksinimleri uçtan uca doğrulamak için:

```bash
make verify
```

Veya Docker ile:
```bash
docker compose --profile verify run --rm verify
```

Bu komut; `SYS-1`, `SYS-2`, `F-1`..`F-7` ve `T-1`..`T-5` maddelerinin tamamını test eder, ekrana yeşil `PASS` tablosu basar ve başarı durumunda sıfır çıkış kodu döner.

---

## 3. Doğrulama Kimlikleri Tablosu (Nasıl Denenir / Beklenen Sonuç)

| Kimlik | Kapsam / Görev | Nasıl Denenir (Uç Nokta / Komut) | Beklenen Sonuç |
|---|---|---|---|
| **SYS-1** | Sistem Sağlık Uçları | `GET /health` & `GET /_hz/f85f` | 200 OK, `X-DDO-Stamp: f85f26` başlığı mevcut |
| **SYS-2** | Yetkisiz Erişim & RBAC | `POST /api/v1/penalties` (Tokensiz & Yolcu rolüyle) | 401 Unauthorized ve 403 Forbidden |
| **F-1** | Kart ve Bakiye Yönetimi | `POST /api/v1/cards/{id}/recharge` | Bakiye artışı, `jw_ledger` tablosuna append kaydı |
| **F-2** | Validatör Olayları | `POST /api/v1/validators/tap` | Turnike geçişi, bakiye düşümü, `PROCESSED` durumu |
| **F-3** | Hat, Durak ve Tarife | `GET /api/v1/lines`, `/stations`, `/tariffs` | 40 hat, duraklar ve güncel tarife listesi (200 OK) |
| **F-4** | Ücret Hesaplama | `CalculateFare()` (Tam, Öğrenci, Yaşlı) | Tam: 20 TL, Öğrenci: 10 TL, Yaşlı: 15 TL |
| **F-5** | Gelir Paylaşımı | `CalculateRevenueShare(1750, legs)` | Kuruş kaybı olmadan mesafe oranlı tam bölüşüm |
| **F-6** | Kontrol ve Ceza Kayıtları | `POST /api/v1/penalties` (Kontrolör rolüyle) | 201 Created, ceza ID ve tutanak kaydı |
| **F-7** | Yoğunluk ve Gelir Raporu | `GET /api/v1/reports/heatmap` | Hat doluluk oranları ve kalabalık seviyeleri (LOW/MED/HIGH) |
| **T-1** | Ücret Motoru | `go test -v ./internal/pricing/...` | 90 dk pencere, 2 indirimli aktarma, ters yön kuralı PASS |
| **T-2** | Çevrimdışı Senkronizasyon | `go test -v ./internal/sync/...` | Mükerrer ID idempotency, -50 TL negatif tolerans PASS |
| **T-3** | Kart Bakiye Defteri | `go test -v ./internal/ledger/...` | 100 eşzamanlı goroutine ile sıfır yarış durumu (Race condition free) |
| **T-4** | Gelir Paylaşımı (Storno) | `go test -v ./internal/settlement/...` | Dönem dondurma ve storno düzeltme kaydı PASS |
| **T-5** | Anormallik Tespiti | `go test -v ./internal/anomaly/...` | >120 km/s imkansız hızda klon kart tespiti PASS |

---

## 4. Beş Zorlu Görevin Kanıt Komutları (Proof Commands)

Her zorlu görevin çalıştığını bağımsız olarak kanıtlayan terminal komutları:

### Görev 1 (T-1) — Ücret Motoru Kanıtı:
```bash
go test -v -run "TestPricingEngine" ./internal/pricing/...
```
*Kanıt Çıktısı:*
- 90 dk aktarma indirimleri (1. %50, 2. %75)
- Ters yön kuralı (aynı hatta geri dönüşün tam ücret kesmesi)
- Gece zammı (%50 ek tarife)
- Öğrenci günlük biniş limiti (10 biniş sonrası tam tarife)
- Sırasız gelen paketlerde deterministik durum makinesi

### Görev 2 (T-2) — Çevrimdışı Validatör Senkronizasyonu Kanıtı:
```bash
go test -v -run "TestSyncService" ./internal/sync/...
```
*Kanıt Çıktısı:*
- Aynı olay iki kez iletildiğinde tek çekim yapılması (`Idempotency-Key` doğrulaması)
- Negatif bakiye toleransı (-50 TL'ye kadar acil biniş izni, aşımında bloke)

### Görev 3 (T-3) — Kart Bakiye Defteri ve Eşzamanlılık Kanıtı:
```bash
go test -v -run "TestLedger_ConcurrentTapsRaceCondition" ./internal/ledger/...
```
*Kanıt Çıktısı:*
- 100 eşzamanlı thread'in aynı karta biniş okutmasında tek bir kuruş kaybı dahi olmadan bakiyenin defter hareketleri toplamına eşit kalması (`SELECT ... FOR UPDATE` serileştirme garantisi).

### Görev 4 (T-4) — Gelir Paylaşımı ve Storno Kanıtı:
```bash
go test -v -run "TestSettlement" ./internal/settlement/...
```
*Kanıt Çıktısı:*
- Largest Remainder Algoritması ile 17.50 TL'nin 3 işletmeciye kuruş yuvarlama hatası olmadan bölüşümü.
- Kapatılan döneme biniş eklenmesinin engellenmesi ve storno (ters kayıt) ile düzeltilmesi.

### Görev 5 (T-5) — Anormallik Tespiti ve Yüksek Başarım Kanıtı:
```bash
go test -v -bench=. ./internal/anomaly/...
```
*Kanıt Çıktısı:*
- 4 dakika içinde Kadıköy'den Sarıyer'e biniş yapıldığında hızın ~300 km/s olarak ölçülüp `IMPOSSIBLE_TRAVEL_SPEED` olarak bayraklanması.
- Benchmark: 50.04 ns/op (Saniyede 20 milyon kontrol kapasitesi ile 1 milyon biniş/saat yükünü rahatça karşılama).

---

## 5. Arayüzde Elle Denenecek Senaryolar (User Journey)

Frontend arayüzü (`http://localhost:3000`) üzerinden aşağıdaki adımları sırayla deneyebilirsiniz:

1. **Yolcu Deneyimi:**
   - `yolcu@aktarbilet.gov.tr` ile giriş yapın.
   - Dijital kartvizitte anlık bakiyeyi görün.
   - "Bakiye Yükle (+50 TL)" düğmesine basın; bakiyenin anında güncellendiğini ve hareket defterine yeşil dolum kaydı düştüğünü izleyin.
2. **Turnike Biniş ve Aktarma Deneyimi:**
   - 500T otobüsüne binin (20 TL tam ücret kesilir).
   - Canlı 90 dakikalık aktarma sayacının geri saymaya başladığını görün.
   - T1 Tramvayına aktarma yapın (10 TL indirimli aktarma uygulanır ve mor rozet belirir).
3. **Kontrolör Deneyimi:**
   - `kontrolor@aktarbilet.gov.tr` ile kontrolör ekranına geçin.
   - `CARD_KARA_LISTE_001` ID'sini sorgulayın; ekran kırmızı alarm verir. "Kaçak Geçiş Cezası Kes" butonuyla tutanak tutun.
4. **Operasyon & Isı Haritası (Bonus 1):**
   - Operasyon paneline geçin; hat bazlı doluluk haritasında 500T ve T1 hatlarının yoğunluk renklerini (yeşil/sarı/kırmızı) inceleyin.
5. **Tarife Simülasyonu (Bonus 3):**
   - Hat yöneticisi panelinde "Öğrenci zammı %20 olursa" senaryosunu çalıştırın; gelir ve yolcu kaybı projeksiyon grafiğini gözlemleyin.
