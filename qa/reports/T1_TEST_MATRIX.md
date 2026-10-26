# T-1 Ücret Motoru Kapsamlı Test ve Doğrulama Matrisi (T1-01 .. T1-12)

**Proje:** AktarBilet Şehir İçi Toplu Taşıma Bilet ve Ücretlendirme Sistemi  
**Bağımsız Denetçi:** Codex (Zero-Trust SDET & Financial Systems QA)  
**Tarih:** 26 Ekim 2026  
**Şartname Referansı:** `proje.md` — Bölüm 6, Zorlu Görev 1 (T-1)  
**Çalıştırılan Otomasyon:** `./qa/scripts/verify-fare.sh`  

---

## 1. Genel Özet ve Durum Tablosu

| Kod | Özellik / Kural | Test Tipi | Test Dosyası | Durum | Oracle Uyumu |
| :--- | :--- | :--- | :--- | :---: | :---: |
| **T1-01** | İlk Biniş ve Kart Türleri (Tam, Öğrenci, Yaşlı, Ücretsiz) | Go Unit / Boundary | `qa/go/t1_matrix_test.go` | **PASS** | %100 Uyumlu |
| **T1-02** | 90 Dakikalık Aktarma Penceresi Sınırları ($T_0$ .. $T_0+90\text{ dk}$) | Go Unit / Boundary | `qa/go/t1_matrix_test.go` | **PASS** | %100 Uyumlu |
| **T1-03** | En Fazla İki İndirimli Aktarma Kuralı | Go Unit / Invariant | `qa/go/t1_matrix_test.go` | **PASS** | %100 Uyumlu |
| **T1-04** | Aynı Hatta Ters Yön / Geri Dönüş Aktarma Engeli | Go Unit & Live E2E | `qa/tests/t1_live_system.spec.js` | **PASS** | %100 Uyumlu |
| **T1-05** | Gece Tarifesi ve Timezone Tutarlılığı (TRT vs UTC) | Go Property/Fuzz | `qa/go/fare_fuzz_test.go` | **PASS** | %100 Uyumlu |
| **T1-06** | Öğrenci Günlük Limit Kuralı | Go Unit / Boundary | `qa/go/t1_matrix_test.go` | **PASS** | %100 Uyumlu |
| **T1-07** | Aylık Abonman Tavanı (Fare Capping) | Go Unit / Boundary | `qa/go/t1_matrix_test.go` | **PASS** | %100 Uyumlu |
| **T1-08** | Sırasız Olay Uzlaşması (Out-of-Order Reconciliation) | Live PostgreSQL 16 | `qa/tests/t1_live_system.spec.js` | **FAIL** | **Açık Tespit Edildi (500 Kr Fazla Tahsilat)** |
| **T1-09** | Çoklu Kural Etkileşimleri (Gece + Aktarma + Öğrenci) | Go Unit / Combo | `qa/go/t1_matrix_test.go` | **PASS** | %100 Uyumlu |
| **T1-10** | Kuruş Para Hassasiyeti (Integer Kuruş, Float Yasağı) | Go Static & Unit | `qa/go/t1_matrix_test.go` | **PASS** | %100 Uyumlu |
| **T1-11** | Defter Tutarlılığı & Eşzamanlı Binişler (Race Conditions) | Live API Parallel | `qa/tests/t1_live_system.spec.js` | **PASS** | %100 Uyumlu |
| **T1-12** | Mükerrer Dokunma ve İdempotency Koruması | Live API Replay | `qa/tests/t1_live_system.spec.js` | **PASS** | %100 Uyumlu |

---

## 2. Ayrıntılı Kriter Analizi ve Kanıtlar

### T1-01: İlk Biniş ve Kart Türleri
- **Kural:** Tam kart taban ücreti (2000 kr), Öğrenci kartı (%50 indirim = 1000 kr), Yaşlı/65+ kartı (%50 indirim = 1000 kr), Ücretsiz kart (0 kr).
- **Test Senaryosu:** `TestT1_01_CardTypesAndInitialFares`
- **Girdi:** 4 farklı kart türüyle boş geçmişle ilk biniş.
- **Beklenen Sonuç:** Tam: 2000 kr, Öğrenci: 1000 kr, Yaşlı: 1000 kr, Ücretsiz: 0 kr.
- **Gerçekleşen:** Tam: 2000 kr, Öğrenci: 1000 kr, Yaşlı: 1000 kr, Ücretsiz: 0 kr.
- **Sonuç:** **PASS**.

### T1-02: 90 Dakikalık Aktarma Penceresi Sınırları
- **Kural:** Aktarma penceresi ilk biniş anı ($T_0$) üzerinden başlar. $T_0 \le T \le T_0 + 90\text{ dk}$ arası aktarma kapsamındadır; $T_0 + 90\text{ dk} + 1\text{ sn}$ aktarma dışıdır.
- **Test Senaryosu:** `TestT1_02_TransferWindow_Boundaries`
- **Sınır Değerleri:** $T_0+1\text{s}$, $T_0+1\text{m}$, $T_0+30\text{m}$, $T_0+89\text{m}59\text{s}$, $T_0+90\text{m}00\text{s}$ (İçinde); $T_0+90\text{m}01\text{s}$, $T_0+91\text{m}$, $T_0+120\text{m}$ (Dışında).
- **Sonuç:** **PASS**. $T_0+90\text{ dk}$ anında aktarma indirimi uygulanırken $T_0+90\text{ dk} + 1\text{ sn}$ anında yeni tam biniş başlatılmaktadır.

### T1-03: En Fazla İki İndirimli Aktarma Kuralı
- **Kural:** 90 dakikalık bir aktarma zincirinde yalnızca 1. aktarma (%50) ve 2. aktarma (%75) indirimli olabilir. 3. ve sonraki aktarmalar tam ücrettir.
- **Test Senaryosu:** `TestT1_03_MaxTwoDiscountedTransfers`
- **Zincir:** 
  - Biniş 1 ($T_0$): 2000 kr (İlk biniş)
  - Biniş 2 ($T_0+15\text{m}$): 1000 kr (1. Aktarma)
  - Biniş 3 ($T_0+30\text{m}$): 500 kr (2. Aktarma)
  - Biniş 4 ($T_0+45\text{m}$): 2000 kr (Aktarma kotası doldu, tam ücret)
  - Biniş 5 ($T_0+60\text{m}$): 2000 kr (Pencere içi ancak kota doldu, tam ücret)
- **Sonuç:** **PASS**.

### T1-04: Aynı Hatta Ters Yön / Geri Dönüş Aktarma Engeli
- **Kural:** Aynı hatta ters yöne veya aynı yöne tekrar dokunma durumunda aktarma indirimi uygulanmaz; tam ücret tahsil edilir.
- **Test Senaryoları:**
  1. `TestT1_04_SameLineReverseDirection` (Go Unit)
  2. `tests/t1_live_system.spec.js` (Live API & PostgreSQL 16)
- **Kanıt (Live API):**
  - Biniş 1: LINE_101, Direction 1 -> 2000 kr (İlk)
  - Biniş 2: LINE_101, Direction 2 (Ters yön) -> 2000 kr (Aktarma engellendi)
  - Biniş 3: LINE_102, Direction 1 (Farklı hat) -> 1000 kr (1. Aktarma indirimi uygulandı)
- **Sonuç:** **PASS**.

### T1-05: Gece Tarifesi ve Timezone Tutarlılığı
- **Kural:** 00:00 - 05:59 Europe/Istanbul (TRT UTC+3) saatleri arasında %50 gece zammı uygulanır. İstemci aynı fiziksel anı UTC (`...Z`) veya TRT (`...+03:00`) gönderdiğinde ücret değişmemelidir.
- **Test Senaryoları:**
  - 1000 vakalık Property & Fuzz Testi (`qa/go/fare_fuzz_test.go`)
  - `TestT1_05_NightFareAndTimezone` boundary testleri (23:59:59, 00:00:00, 00:00:01, 03:30:00, 05:59:59, 06:00:00).
- **Sonuç:** **PASS**. `engine.go:86` üzerinde `tappedAt.In(locTRT)` dönüşümü sayesinde tüm timezone'larda determinizm sağlandı.

### T1-06: Öğrenci Günlük Limit Kuralı
- **Kural:** Öğrenci kartı günde en fazla 10 indirimli biniş yapabilir (Tarife: `student_daily_limit = 10`). 11. binişten itibaren tam ücret (2000 kr) uygulanır.
- **Test Senaryosu:** `TestT1_06_StudentDailyLimit`
- **Sonuç:** **PASS**. (Not: Veritabanı sayaç sıfırlama mimari açığı Bölüm 3'te raporlanmıştır).

### T1-07: Aylık Abonman Tavanı (Fare Capping)
- **Kural:** Bir kartın ay içerisindeki toplam harcaması aylık tavana (1500 TL = 150000 kr) ulaştığında sonraki tüm binişler 0 TL olarak ücretlendirilir. Tavanı aşan kısmi tutarda yalnızca tavanı tamamlayacak miktar tahsil edilir.
- **Test Senaryosu:** `TestT1_07_MonthlyFareCapping`
- **Girdi:** Harcama 1490 TL olan kart, 20 TL biniş yapar.
- **Beklenen:** 10 TL tahsil edilir, harcama 1500 TL'ye tamamlanır. Sonraki binişte 0 TL tahsil edilir.
- **Sonuç:** **PASS**.

### T1-08: Sırasız 3 Olay Uzlaşması (Out-of-Order Delivery)
- **Kural:** Araçlardan ağ kopukluğu nedeniyle biniş olayları sunucuya sırasız gelse dahi nihai kart bakiyesi ve defter durumu, olaylar kronolojik sırayla gelmiş gibi deterministik olmalıdır.
- **Test Senaryosu:** `qa/tests/t1_live_system.spec.js` (Test 3)
- **Olaylar:**
  - $E_1$: 10:00:00 (Hat A) -> Kronolojik Ücret: 2000 kr
  - $E_2$: 10:20:00 (Hat B) -> Kronolojik Ücret: 1000 kr (1. Aktarma)
  - $E_3$: 10:40:00 (Hat C) -> Kronolojik Ücret: 500 kr (2. Aktarma)
  - Toplam Kronolojik Tahsilat: 3500 kr. Bakiye: 20000 - 3500 = **16500 kr**.
- **Gerçekleşen Teslimat Sırası:** $E_3 \to E_2 \to E_1$.
- **Canlı Test Çıktısı:**
  ```text
  Error: [FAIL T1-08] Reverse delivery balance (16000) must equal chronological balance (16500)
  Expected: "16500"
  Received: "16000"
  ```
- **Kök Neden:** `internal/sync/service.go:233` satırındaki `LIMIT 1` kısıtı, $E_1$ geldiğinde sadece $E_2$'yi yeniden fiyatlandırmakta, zincirin devamındaki $E_3$'ü görmezden gelmektedir.
- **Sonuç:** **FAIL** (P0 Finansal Bütünlük Hatası).

### T1-09: Çoklu Kural Etkileşimleri
- **Kural:** Gece tarifesi + Aktarma indirimi, Öğrenci indirimi + Gece zammı kurallarının matematiksel öncelik ve çarpım sırası deterministik olmalıdır.
- **Test Senaryosu:** `TestT1_09_MultiRuleInteractions`
- **Sonuç:** **PASS**.

### T1-10: Kuruş Para Hassasiyeti (Integer Kuruş)
- **Kural:** Tüm para hesaplamaları tam sayı (int64) kuruş cinsinden yapılmalı, float veya kayan noktalı yuvarlama hataları olmamalıdır.
- **Test Senaryosu:** `TestT1_10_IntegerMoneyPrecision`
- **Sonuç:** **PASS**.

### T1-11: Defter Bütünlüğü & Eşzamanlı Binişler
- **Kural:** Aynı karta aynı anda gelen 10 eşzamanlı dokunmada `SELECT ... FOR UPDATE` ile bakiye serileştirilmeli, kayıp güncelleme (lost update) olmamalı ve Kart Bakiyesi == Defter Toplamı eşitliği korunmalıdır.
- **Test Senaryosu:** `qa/tests/t1_live_system.spec.js` (Test 4 - Concurrency)
- **Sonuç:** **PASS**. 10 paralel istek sonunda bakiye tam olarak 30000 kr, defter toplamı 30000 kr olarak doğrulanmıştır.

### T1-12: Mükerrer ve İdempotency Koruması
- **Kural:** Aynı `(validator_id, sequence_no)` veya `Idempotency-Key` ile gelen mükerrer istekler defterde ikinci bir tahsilat oluşturmamalı, aynı yanıtı dönmelidir.
- **Test Senaryosu:** `qa/tests/t1_live_system.spec.js` (Test 5 - Idempotency)
- **Sonuç:** **PASS**. Replay edilen istek 200 OK ile aynı veriyi dönmüş, defterde fazladan 1 kuruş dahi hareket oluşmamıştır.

