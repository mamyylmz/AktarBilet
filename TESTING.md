# AktarBilet Test ve Doğrulama Kılavuzu (TESTING.md)

Bu belge, DDO Yazılım Ekibi Şartnamesi Bölüm 10 uyarınca sistemin nasıl doğrulanacağını, demo kullanıcı hesaplarını, her doğrulama kimliği için beklenen sonuçları ve zorlu görevlerin kanıt komutlarını içerir.

---

## 1. Demo Kullanıcı Hesapları

Sistem ayağa kalktığında otomatik seed işlemi ile aşağıdaki hesaplar kullanıma hazır hale gelir:

| Rol | E-posta | Parola | Görev ve Yetki Alanı |
|---|---|---|---|
| **Admin / Canary** | `seed.sedir@ddo-case.invalid` | `SedirPassword2026!` | Tam sistem yetkisi (Sistem Canary) |
| **Yolcu** | `yolcu@aktarbilet.gov.tr` | `YolcuPassword123!` | Kart bakiye sorgulama, hareketler, dolum |
| **Yolcu / Öğrenci kartı** | `ogrenci.demo@aktarbilet.gov.tr` | `OgrenciDemo2026!` | Yalnız kendisine bağlı öğrenci kartı |
| **Yolcu / Yaşlı kartı** | `yasli.demo@aktarbilet.gov.tr` | `YasliDemo2026!` | Yalnız kendisine bağlı yaşlı kartı |
| **Yolcu / Ücretsiz kart** | `ucretsiz.demo@aktarbilet.gov.tr` | `UcretsizDemo2026!` | Sıfır bakiyede ücretsiz biniş politikası |
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

Bu komut canlı HTTP ve domain kabul senaryolarını çalıştırır, her çalıştırmanın sonucunu yazar ve FAIL varsa sıfır dışı çıkar. Test; kart, tap, ceza ve settlement kayıtları oluşturduğundan yalnızca izole QA veritabanında çalıştırılmalıdır. Tek başına `make verify` bütün `proje.md` maddelerini, PostgreSQL concurrency/capacity veya CI durumunu kanıtlamaz.

---

## 3. Doğrulama Kimlikleri Tablosu (Nasıl Denenir / Beklenen Sonuç)

| Kimlik | Kapsam / Görev | Nasıl Denenir (Uç Nokta / Komut) | Beklenen Sonuç |
|---|---|---|---|
| **SYS-1** | Sistem Sağlık Uçları | `GET /health` & `GET /_hz/f85f` | 200 OK, `X-DDO-Stamp: f85f26` başlığı mevcut |
| **SYS-2** | Yetkisiz Erişim & RBAC | `POST /api/v1/penalties` (Tokensiz & Yolcu rolüyle) | 401 Unauthorized ve 403 Forbidden |
| **F-1** | Kart ve Bakiye Yönetimi | Dört demo hesapla `GET /api/v1/cards/{id}` ve recharge | Kart tipi, sahiplik/IDOR, bakiye ve ledger |
| **F-2** | Validatör Olayları | `POST /api/v1/validators/tap` | Turnike geçişi, bakiye düşümü, `PROCESSED` durumu |
| **F-3** | Hat, Durak ve Tarife | `GET /api/v1/lines`, `/stations`, `/tariffs` | En az 40 aktif hat, sıfır eski İstanbul hattı, aktif İzmir tarifesi |
| **F-4** | Ücret Hesaplama | `CalculateFare()` (Tam, Öğrenci, Yaşlı, Ücretsiz) | Simülasyon tarifesi: Tam 20 TL, Öğrenci 10 TL, Yaşlı 15 TL; ücretsiz kart 0 TL |
| **F-5** | Gelir Paylaşımı | `CalculateRevenueShare(1750, legs)` | Kuruş kaybı olmadan payların tam uzlaşması |
| **F-6** | Kontrol ve Ceza Kayıtları | `POST /api/v1/penalties/` (Kontrolör rolüyle) | 201 Created, ceza ID ve tutanak kaydı |
| **F-7** | Yoğunluk ve Gelir Raporu | `GET /api/v1/reports/heatmap` | Persisted olaylardan türetilen rapor; olay yoksa yoğunluk sonucu yok |
| **T-1** | Ücret Motoru | `go test -v ./internal/pricing/...` | 90 dk pencere, 2 indirimli aktarma, ters yön kuralı PASS |
| **T-2** | Çevrimdışı Senkronizasyon | `go test -v ./internal/sync/...` | Mükerrer ID idempotency, -50 TL negatif tolerans PASS |
| **T-3** | Kart Bakiye Defteri | `go test -v ./internal/ledger/...` | 100 eşzamanlı goroutine ile sıfır yarış durumu (Race condition free) |
| **T-4** | Gelir Paylaşımı (Storno) | `go test -v ./internal/settlement/...` | Dönem dondurma ve storno düzeltme kaydı PASS |
| **T-5** | Anormallik Tespiti | `go test -v ./internal/anomaly/...` | >120 km/s imkansız hızda klon kart tespiti PASS |

F-4 ücretleri `proje.md` kuralları için kullanılan simülasyon tarifesidir; belediyenin güncel resmî fiyatları olarak yorumlanmamalıdır. `öğrenci`, `yaşlı` ve `ücretsiz` simülasyon kart sınıfları ayrı iş kurallarına sahiptir.

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
- Anomali birim testleri sentetik koordinatlarla imkânsız hız kuralını kontrol eder.
- Benchmark yalnız hesaplama maliyetini ölçer; PostgreSQL persisted-event kapasitesi veya 1 milyon olay/saat hedefini kanıtlamaz.

---

## 5. Arayüzde Elle Denenecek Senaryolar (User Journey)

Frontend arayüzü (`http://localhost:3000`) üzerinden aşağıdaki adımları sırayla deneyebilirsiniz:

1. **Yolcu Deneyimi:**
   - `yolcu@aktarbilet.gov.tr` ile giriş yapın.
   - Dijital kartvizitte anlık bakiyeyi görün.
   - "Bakiye Yükle (+50 TL)" düğmesine basın; bakiyenin anında güncellendiğini ve hareket defterine yeşil dolum kaydı düştüğünü izleyin.
2. **Turnike Biniş ve Aktarma Deneyimi:**
   - 5 numaralı ESHOT otobüsüne binin (tarifeye göre tam ücret kesilir).
   - Canlı 90 dakikalık aktarma sayacının geri saymaya başladığını görün.
   - 10 numaralı ESHOT hattına aktarma yapın (tarifeye göre indirim uygulanır).
3. **Kontrolör Deneyimi:**
   - `kontrolor@aktarbilet.gov.tr` ile kontrolör ekranına geçin.
   - `CARD_KARA_LISTE_001` ID'sini sorgulayın; ekran kırmızı alarm verir. "Kaçak Geçiş Cezası Kes" butonuyla tutanak tutun.
4. **Operasyon & Isı Haritası (Bonus 1):**
   - Operasyon paneline geçin; yalnız kaydedilmiş ESHOT olaylarına dayanan hat raporunu inceleyin. Olay yokken örnek yoğunluk verisi gösterilmez.
5. **Tarife Simülasyonu (Bonus 3):**
   - Hat yöneticisi panelinde "Öğrenci zammı %20 olursa" senaryosunu çalıştırın; gelir ve yolcu kaybı projeksiyon grafiğini gözlemleyin.
