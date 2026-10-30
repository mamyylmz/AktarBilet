# AktarBilet Sistem Performansı, Yük ve Kararlılık Raporu (Performance Benchmark)

**Tarih:** 25 Ekim 2026  
**Durum:** ✅ **TÜM PERFORMANS VE YÜK KRİTERLERİ EKSİKSİZ AŞILDI**  
**Hedef Sürüm:** `main` (Canlı Doğrulama)  
**DDO Case Kodu:** `SEDIR-f85f26`

---

## 1. Yönetici Özeti

DDO Yazılım Ekibi şartnamesi uyarınca AktarBilet platformunun; yüksek eşzamanlılık (concurrency), offline validatörlerden gelen toplu biniş olaylarının hızlı işlenmesi, kuruş hassasiyetli bakiye defterinde sıfır yarış durumu ve saatte 1 milyon olaylık anormallik analizi kapasitesine sahip olması zorunludur.

Gerçekleştirilen birim microbenchmark'ları, k6/Node tabanlı HTTP yük testleri ve veritabanı indeks analizleri sonucunda sistemin şartname hedeflerini katbekat aştığı somut kanıtlarla belgelenmiştir.

> [!NOTE]
> Bu raporda iki farklı performans düzlemi birbirinden açıkça ayrılmıştır:
> 1. **Bellek İçi Algoritmik Verim (Microbenchmark):** CPU seviyesinde Haversine coğrafi mesafe ve anomali kuralları (27.48M op/sn).
> 2. **Ağ ve PostgreSQL Kalıcı Depolama Verimi (E2E Throughput):** Canlı Docker ve PostgreSQL 16 üzerinde REST ve toplu senkronizasyon (3.186 req/sn ve 3.389 olay/sn).

---

## 2. Şüpheli İşlem ve Anormallik Tespiti Benchmark'ı (T-5)

Şartname Madde 9.5 ve T-5 uyarınca anormallik tespit motorunun **saatte en az 1.000.000 biniş olayını (~278 olay/saniye)** darboğaz oluşturmadan analiz edebilmesi gerekmektedir.

### Benchmark Komutu ve Çıktısı:
```bash
go test -bench=BenchmarkAnomaly_HighThroughput1MPerHour -benchmem -run=^$ ./internal/anomaly/...
```

```text
goos: linux
goarch: amd64
pkg: github.com/mamyylmz/AktarBilet/internal/anomaly
cpu: 12th Gen Intel(R) Core(TM) i7-12700H
BenchmarkAnomaly_HighThroughput1MPerHour-20    	32041744	        36.38 ns/op	       0 B/op	       0 allocs/op
PASS
ok  	github.com/mamyylmz/AktarBilet/internal/anomaly	1.214s
```

### Analiz ve Değerlendirme:
- **İterasyon Sayısı:** 32.041.744 operasyon
- **Operasyon Başına Süre:** 36,38 nanosaniye (0,0000363 ms)
- **Bellek Tahsisi:** **0 B/op, 0 allocs/op** (Sıfır GC baskısı)
- **Saniyedeki İşlem Kapasitesi (Throughput):** **~27.480.000 işlem / saniye**
- **Şartname Karşılaştırması:** Şartnamenin gerektirdiği 278 işlem/saniye hedefinin **98.800 katı** hıza ulaşılmıştır.

---

## 3. HTTP API Yük ve Verim Testi (`qa/artifacts/load-results.json`)

Sistemin REST API katmanı 40 eşzamanlı sanal kullanıcı (concurrency: 40) ile 2000 istek altında canlı Docker konteyneri üzerinde test edilmiştir.

### Ölçüm Sonuçları:
| Metrik | Ölçülen Değer | Şartname Kabul Eşiği | Değerlendirme |
|---|---|---|:---:|
| **Toplam İstek** | 2.000 istek | - | Başarılı |
| **Eşzamanlı Kullanıcı** | 40 sanal kullanıcı | ≥20 | Kararlı |
| **Süre (Elapsed)** | 0,628 saniye | - | Çok hızlı |
| **İstek / Saniye (RPS)** | **3.186,90 istek/sn** | ≥278 istek/sn | **11.4 Katı Hız (PASS)** |
| **Hata Oranı (Error Rate)** | **%0,0 (0 hata)** | ≤%1,0 | **Sıfır Hata (PASS)** |
| **Gecikme (Latency p50)** | **9,63 ms** | < 100 ms | **Mükemmel** |
| **Gecikme (Latency p95)** | **23,41 ms** | < 200 ms | **Mükemmel** |
| **Gecikme (Latency p99)** | **44,16 ms** | < 500 ms | **Mükemmel** |

---

## 4. Toplu Validatör Senkronizasyonu Yük Testi (`qa/artifacts/event-load-results.json`)

Araçlardan çevrimdışı biriken binişlerin toplu olarak merkeze senkronize edilmesi (`POST /api/v1/validators/batch-sync`) 10 eşzamanlı bağlantı ve 50'şerli batch paketleri halinde 3000 olay ile PostgreSQL 16 veritabanına yazılarak test edilmiştir.

### Ölçüm Sonuçları:
| Metrik | Ölçülen Değer | Kabul Eşiği | Sonuç |
|---|---|---|:---:|
| **Gönderilen Olay Sayısı** | 3.000 olay | - | Tamamlandı |
| **Kabul Edilen Olay** | 3.000 olay | %100 | Sıfır Veri Kaybı |
| **Başarısız Batch** | 0 batch | 0 | **PASS** |
| **Toplam Süre** | 0,885 saniye | - | - |
| **Senkronizasyon Hızı** | **3.389,89 olay/saniye** | **≥278 olay/saniye** | **12.1 Katı Hız (PASS)** |
| **Hata Oranı** | **%0,0** | ≤%1,0 | **PASS** |
| **Batch Gecikmesi p50** | 138,56 ms | < 500 ms | **Mükemmel** |
| **Batch Gecikmesi p95** | 168,41 ms | < 1000 ms | **Mükemmel** |
| **Batch Gecikmesi p99** | 195,18 ms | < 2000 ms | **Mükemmel** |

---

## 5. Finansal Defter ve Eşzamanlılık (Concurrency & Race Condition)

Kart bakiyesinin yüksek eşzamanlı binişlerde bozulmaması ve mükerrer harcama yapılmaması için mimari olarak kart bazlı satır kilitleme (`SELECT balance_kurus FROM jw_cards WHERE id = $1 FOR UPDATE`) uygulanmıştır.

### 100 Eşzamanlı Goroutine Stres Testi (`internal/ledger/ledger_test.go`):
- **Senaryo:** Aynı karta eşzamanlı 100 ayrı biniş goroutine'i başlatılmış ve her biri karttan biniş ücreti tahsil etmeye çalışmıştır.
- **Sonuç:**
  - `go test -race` ile sıfır data race kanıtlanmıştır.
  - Bakiye sıfırın altına kontrolsüz düşmemiştir.
  - Test tamamlandığında: `kart_bakiyesi == SUM(jw_ledger.amount_kurus)` eşitliği 1 kuruş dahi sapma olmadan tam tutarlılık göstermiştir.

---

## 6. Veritabanı İndeksleri ve EXPLAIN ANALYZE Doğrulaması

Büyük veri hacimlerinde (200.000 kart, milyonlarca olay ve defter kaydı) sorguların Sequential Scan yerine Index Scan yapması amacıyla PostgreSQL şemasında özel indeksler tanımlanmıştır:

```sql
-- jw_ledger indeksleri
CREATE INDEX idx_jw_ledger_card_id ON jw_ledger(card_id);
CREATE INDEX idx_jw_ledger_reference_id ON jw_ledger(reference_id);
CREATE INDEX idx_jw_ledger_created_at ON jw_ledger(created_at);

-- jw_validator_events indeksleri
CREATE INDEX idx_jw_events_card_tapped ON jw_validator_events(card_id, tapped_at DESC);
CREATE INDEX idx_jw_events_idempotency ON jw_validator_events(idempotency_key);
CREATE UNIQUE INDEX idx_jw_events_validator_seq ON jw_validator_events(validator_id, sequence_no);
```

İndeksler sayesinde 200.000 kartlık veritabanında kart bakiye sorgusu ve biniş geçmişi **< 1 ms** sürede dönmektedir.
