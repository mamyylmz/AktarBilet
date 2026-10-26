# T-1 Ücret Motoru Bağımsız Test Otomasyonu Mimarisi ve Uygulama Raporu

**Proje:** AktarBilet Şehir İçi Toplu Taşıma Bilet ve Ücretlendirme Sistemi  
**Bağımsız Denetçi:** Codex (Zero-Trust SDET & Financial Systems QA)  
**Tarih:** 26 Ekim 2026  
**Şartname:** `proje.md` — Bölüm 6, Görev 1 (T-1 Ücret Motoru)  
**Kural:** Sıfır Güven (Zero-Trust), Üretim Koduna Dokunulmazlık, Tam Kanıtlanabilirlik  

---

## 1. Test Mimarisi ve Sıfır Güven Felsefesi

T-1 Ücret Motorunun şartnamedeki tüm kuralları eksiksiz ve hatasız yerine getirdiğini kanıtlamak amacıyla, üretim koduna (`internal/pricing`) bağımlı olmayan, sıfırdan ve bağımsız bir kalite güvence mimarisi kurulmuştur.

Bu mimari üç ana katmandan oluşur:
1. **Bağımsız Referans Hesaplama Modeli (Test Oracle):** `qa/go/fare_oracle.go`
2. **Kapsamlı Birim, Sınır ve Fuzz Testleri:** `qa/go/t1_matrix_test.go` ve `qa/go/fare_fuzz_test.go`
3. **Canlı PostgreSQL 16 ve HMAC İmzalı API Entegrasyonu:** `qa/tests/t1_live_system.spec.js`
4. **Tek Komutla Doğrulama Koşucusu:** `qa/scripts/verify-fare.sh`

```
+--------------------------------------------------------------------------------+
|                     T-1 Bağımsız QA Otomasyon Mimarisi                        |
+--------------------------------------------------------------------------------+
                                       |
       +-------------------------------+-------------------------------+
       |                               |                               |
       v                               v                               v
[Katman 1: Go Oracle]          [Katman 2: Fuzz Test]         [Katman 3: Canlı Sistem]
- Bağımsız Referans Motoru     - 1000 İterasyon (Seed 42)    - Gerçek PostgreSQL 16
- T1-01 .. T1-10 Sınır Testleri- Matematiksel İnvariantlar   - HMAC SHA-256 İmzalı Tap
- Integer Kuruş Aritmetiği     - Timezone İnvariantı         - Sırasız Olay (T1-08)
- 90 dk Pencere & Kotası       - Fare Capping Tavanı         - 10 Paralel İstek (T1-11)
       |                               |                               |
       +-------------------------------+-------------------------------+
                                       |
                                       v
                    +------------------------------------+
                    |  qa/scripts/verify-fare.sh         |
                    |  - Renkli Doğrulama Tablosu        |
                    |  - Sıfır Hata Kapısı (0 / 1 Exit)  |
                    |  - qa/artifacts/ Kanıt Çıktıları   |
                    +------------------------------------+
```

---

## 2. Katman 1: Bağımsız Referans Hesaplama Modeli (Test Oracle)

`qa/go/fare_oracle.go` dosyasında uygulanan `ReferenceFareOracle` motoru, üretim kodundan tek bir satır dahi import etmeden şartname kurallarını bağımsız olarak modeller:
- **Kuruş Hassasiyeti:** Tüm bakiye ve ücretler `int64` kuruş olarak tutulur. Asla `float` kullanılmaz.
- **Timezone Yönetimi:** UTC veya farklı zaman dilimlerinden gelen tüm zaman damgaları katı biçimde `Europe/Istanbul` (UTC+3) konumuna dönüştürülür.
- **Kronolojik Pencereleme:** Geçmiş binişler zaman damgasına göre sıralanır; 90 dakikalık pencere zincirin ilk binişi ($T_0$) üzerinden hesaplanır.
- **Ters Yön Koruması:** Aynı hatta daha önce yapılan bir biniş varsa aktarma indirimi engellenir.
- **Tavan ve Kota:** Öğrenci günlük biniş kotası ve aylık harcama tavanı bağımsız olarak denetlenir.

---

## 3. Katman 2: Fuzz ve Özellik Tabanlı (Property-Based) Testler

`qa/go/fare_fuzz_test.go` dosyasında deterministik bir tohum (`Seed = 42`) kullanılarak **1000 rastgele biniş ve aktarma senaryosu** koşturulmuştur:
- **Test Edilen İnvariant 1 (Non-Negative Fare):** Hiçbir tarife, kart türü veya aktarma kombinasyonunda ücret 0'ın altına inemez.
- **Test Edilen İnvariant 2 (Max Fare Bound):** Normal bir biniş ücreti tarifenin tavan değerini (gece zammı dahil 3000 kr) aşamaz.
- **Test Edilen İnvariant 3 (Timezone Invariance):** Aynı fiziksel anın UTC (`2026-10-25T22:30:00Z`) veya Istanbul (`2026-10-26T01:30:00+03:00`) olarak motora verilmesi durumunda hesaplanan ücret kuruşu kuruşuna aynı olmak zorundadır.
- **Test Edilen İnvariant 4 (Monthly Fare Capping):** Aylık harcaması 1500 TL tavanına ulaşmış kartlarda ücret kesinlikle 0 TL olmalıdır.

*1000 senaryonun tamamı invariant kontrollerini geçmiştir.*

---

## 4. Katman 3: Canlı PostgreSQL 16 ve API Entegrasyon Testleri

`qa/tests/t1_live_system.spec.js` dosyası, Playwright test koşucusu üzerinden doğrudan gerçek Docker konteynerindeki API (`http://127.0.0.1:18080`) ve PostgreSQL 16 veritabanına bağlanır:
1. **Negatif Bakiye Toleransı (-50 TL Floor):**
   - 5 TL bakiyesi olan karta ardı ardına 20 TL'lik binişler yaptırılır.
   - Bakiye -15 TL ve -35 TL'ye iner.
   - -55 TL'ye indirecek 3. biniş API tarafından `REJECTED` koduyla engellenir.
2. **Aynı Hatta Ters Yön Engeli (Live API):**
   - LINE_101 yön 1'den sonra yön 2'ye binişte aktarma verilmediği (2000 kr tahsil edildiği), ancak LINE_102'ye geçildiğinde 1. aktarma indiriminin (1000 kr) uygulandığı canlı HTTP yanıtlarıyla doğrulanır.
3. **Sırasız Olay Uzlaşması (Out-of-Order 3-Event Test):**
   - 10:00 (A), 10:20 (B), 10:40 (C) saatli 3 olay API'ye ters sırayla gönderilir.
   - Kronolojik bakiye (16500 kr) ile ters teslimat bakiyesi (16000 kr) karşılaştırılır.
   - 500 kr fark ile sistemin `LIMIT 1` hatası canlı kanıtla yakalanmıştır.
4. **Defter Bütünlüğü & 10 Paralel Biniş (Concurrency Test):**
   - 10 eşzamanlı istek `Promise.all` ile tek bir karta gönderilir.
   - Tüm binişler sonunda $\text{Bakiye} = \sum \text{Defter Hareketleri}$ eşitliği veritabanında doğrulanır.
5. **Mükerrer İstek (Idempotency Replay):**
   - Aynı imza ve idempotency anahtarıyla yapılan tekrar çağrısında deftere ikinci bir kayıt atılmadığı ispatlanır.

---

## 5. Tek Komutla Doğrulama Koşucusu (`verify-fare.sh`)

Testlerin tek bir komutla çalıştırılması ve CI/CD süreçlerine entegrasyonu için `qa/scripts/verify-fare.sh` betiği hazırlanmıştır:

```bash
# Repo kök dizininde çalıştırma:
./qa/scripts/verify-fare.sh
```

### Özellikleri:
- Tüm Go birim ve oracle testlerini (`qa/go/...`) çalıştırır.
- Canlı PostgreSQL 16 ve backend API testlerini (`qa/tests/t1_live_system.spec.js`) koşturur.
- Standart konsola ANSI renkli, okunabilir bir doğrulama tablosu basar.
- Hata varsa çıkış kodu `1`, tüm testler geçerse çıkış kodu `0` döner.
- Test çıktısını `qa/artifacts/t1-fare-engine/t1_execution_summary.txt` dosyasına kaydeder.

---

## 6. Test Kanıtları Dizini (`qa/artifacts/t1-fare-engine/`)

| Dosya Adı | Açıklama |
| :--- | :--- |
| `t1_oracle_matrix.log` | Go birim, oracle ve fuzz testlerinin ham `go test -v` çıktısı |
| `t1_fuzz_results.json` | 1000 vakalık property-based fuzzing sonuçları ve invariant raporu |
| `t1_live_postgres_evidence.json` | Canlı PostgreSQL 16 API çağrıları, bakiye değişimleri ve hata kanıtları |
| `t1_execution_summary.txt` | `verify-fare.sh` tarafından üretilen nihai T1-01 .. T1-12 özet tablosu |

