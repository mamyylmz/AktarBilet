# T-1 Ücret Motoru — Tespit Edilen Kritik Hatalar ve Kök Neden Analizi (QA Audit Raporu)

**Proje:** AktarBilet Şehir İçi Toplu Taşıma Bilet ve Ücretlendirme Sistemi  
**Bağımsız Denetçi:** Codex (Zero-Trust SDET & Financial Systems QA)  
**Tarih:** 26 Ekim 2026  
**Hedef Context:** Ana Geliştirici Context (Antigravity)  
**Kapsam:** `proje.md` Bölüm 6 Görev 1 (T-1 Ücret Motoru)  

---

## 🚨 YÖNETİCİ ÖZETİ VE BULGU SEVİYELERİ

| ID | Seviye | Etkilenen Dosya & Satır | Hata Tanımı | Durum |
| :--- | :---: | :--- | :--- | :---: |
| **BUG-01** | **P0 (Kritik)** | `internal/sync/service.go:225-235` | Sırasız Olay Uzlaşmasında `LIMIT 1` Nedeniyle Çoklu Olaylarda Zincir Kırılması (500 Kr Fazla Tahsilat) | **AÇIK (FAIL)** |
| **BUG-02** | **P1 (Önemli)** | `internal/pricing/engine.go:41, 127-134` | `currentDirection` Parametresinin Kullanılmaması (Yön Denetimi Eksikliği) | **KOD KOKUSU** |
| **BUG-03** | **P1 (Önemli)** | `internal/sync/service.go:303` / `jw_cards` | Günlük ve Aylık Sayaçların Gece Yarısı / Ay Dönümünde Sıfırlanmaması | **MİMARİ EKSİK** |
| **SPEC-01** | **BİLGİ** | `proje.md` / `jw_tariffs` | Şartnamedeki Politika Belirsizlikleri (`SPEC_UNDEFINED`) | **BELGELENDİ** |

---

## 1. BUG-01 (P0): Sırasız Olay Uzlaşmasında `LIMIT 1` Nedeniyle Zincir Kırılması

### 1.1. Hata Özeti ve Finansal Etki
Canlı PostgreSQL 16 ve backend API üzerinde çalıştırılan `qa/tests/t1_live_system.spec.js` (Test 3) testinde, aynı 90 dakikalık aktarma penceresi içerisindeki 3 biniş olayı **ters sırada ($E_3 \to E_2 \to E_1$)** sisteme ulaştığında karttan toplam **4000 kuruş** tahsil edilmektedir. 

Olaylar normal kronolojik sırayla ($E_1 \to E_2 \to E_3$) ulaştığında ise toplam tahsilat **3500 kuruş** olmaktadır.
**Sonuç: Yolcudan 500 kuruş haksız fazla tahsilat yapılmakta ve bakiye determinizmi ihlal edilmektedir.**

### 1.2. Matematiksel Kanıt (Matematiksel Akış)

#### Senaryo:
- $E_1$: 10:00:00 — Hat A (Tam: 2000 kr)
- $E_2$: 10:20:00 — Hat B (1. Aktarma: 1000 kr)
- $E_3$: 10:40:00 — Hat C (2. Aktarma: 500 kr)
- **Doğru Kronolojik Toplam: 3500 kr | Kalan Bakiye: 20000 - 3500 = 16500 kr.**

#### Ters Teslimat Sırasında Gerçekleşen Akış ($E_3 \to E_2 \to E_1$):
1. **$E_3$ Gelir (10:40):**
   - Geçmişte hiçbir biniş yoktur.
   - İlk biniş olarak işlenir: **2000 kr** tahsil edilir.
2. **$E_2$ Gelir (10:20):**
   - Geçmişte biniş yoktur. İlk biniş olarak **2000 kr** tahsil edilir.
   - Gelecek olay sorgusu (`tapped_at > 10:20 LIMIT 1`) çalışır. $E_3$ bulunur.
   - $E_3$ yeniden hesaplanır: $E_2$'den sonraki 1. Aktarma olarak görülür (Ücret: 1000 kr).
   - $E_3$ için Storno iadesi yapılır: $2000 - 1000 = +1000\text{ kr}$.
   - $E_3$ veritabanında `fare_charged_kurus = 1000`, `transfer_level = 1` olarak güncellenir.
3. **$E_1$ Gelir (10:00):**
   - Geçmişte biniş yoktur. İlk biniş olarak **2000 kr** tahsil edilir.
   - Gelecek olay sorgusu çalışır:
     ```sql
     SELECT id, fare_charged_kurus, line_id, station_id, direction, tapped_at
     FROM jw_validator_events
     WHERE card_id = $1
       AND tapped_at > $2::timestamptz
       AND tapped_at <= ($2::timestamptz + INTERVAL '90 minutes')
       AND status = 'PROCESSED'
     ORDER BY tapped_at ASC
     LIMIT 1
     ```
   - **KRİTİK HATA:** `ORDER BY tapped_at ASC LIMIT 1` sorgusu **yalnızca $E_2$'yi çeker!**
   - $E_2$ yeniden fiyatlandırılır: $E_1$'den sonraki 1. Aktarma yapılır (1000 kr). $E_2$ için 1000 kr storno iadesi yapılır.
   - **ANCAK $E_3$ ASLA YENİDEN FİYATLANDIRILMAZ!**
   - $E_3$, $E_1$'in gelişinden tamamen habersiz kalır; 2. Aktarma (500 kr) olması gerekirken 1. Aktarma (1000 kr) olarak kalır.
   - **Toplam Tahsilat:** $E_1(2000) + E_2(1000) + E_3(1000) = 4000\text{ kr}$.
   - **Kalan Bakiye:** $20000 - 4000 = 16000\text{ kr}$ (Beklenen: 16500 kr).

### 1.3. Doğrudan Tekrarlama Adımları (Reproduction)
```bash
# Canlı sistem üzerinde çalıştırma:
cd /home/dilunsel/projects/AktarBilet/qa
npx playwright test tests/t1_live_system.spec.js -g "T1-08"
```
**Alınan Hata Logu:**
```text
Error: [FAIL T1-08] Reverse delivery balance (16000) must equal chronological balance (16500)
Expected: "16500"
Received: "16000"
```

### 1.4. Geliştiriciye Önerilen Çözüm (Recommended Architecture Fix)
`internal/sync/service.go:225-278` bloğundaki `LIMIT 1` kaldırılmalı, o karta ait o penceredeki tüm gelecek olaylar (`tapped_at > req.TappedAt ORDER BY tapped_at ASC`) bir döngüde çekilmelidir. Her gelecek olay, kendisinden önceki olayların kümülatif geçmişi ile sırayla yeniden fiyatlandırılmalı ve storno düzeltmeleri zincir boyunca uygulanmalıdır:

```go
// Örnek Düzeltme Mantığı:
rows, err := pool.Query(ctx, `
    SELECT id, fare_charged_kurus, line_id, station_id, direction, tapped_at
    FROM jw_validator_events
    WHERE card_id = $1
      AND tapped_at > $2::timestamptz
      AND tapped_at <= ($2::timestamptz + INTERVAL '90 minutes')
      AND status = 'PROCESSED'
    ORDER BY tapped_at ASC
`, req.CardID, req.TappedAt)
if err != nil {
    return nil, err
}
defer rows.Close()

// req kendisi de kümülatif tarihe eklenir
rollingHistory := append([]pricing.TripHistoryItem{}, history...)
rollingHistory = append(rollingHistory, pricing.TripHistoryItem{
    LineID: req.LineID, StationID: req.StationID, Direction: req.Direction,
    TappedAt: req.TappedAt, IsTransfer: calcRes.IsTransfer, TransferLevel: calcRes.TransferLevel,
})

for rows.Next() {
    var fEvID uuid.UUID
    var fFare domain.Kurus
    var fLineID, fStationID string
    var fDir int
    var fTappedAt time.Time
    if err := rows.Scan(&fEvID, &fFare, &fLineID, &fStationID, &fDir, &fTappedAt); err != nil {
        continue
    }

    // rollingHistory ile yeniden hesapla
    fCalc := s.pricingEngine.CalculateFare(card, tariff, fLineID, fStationID, fDir, fTappedAt, rollingHistory)
    correction := fFare - fCalc.FareChargedKurus
    if correction != 0 {
        ref := fmt.Sprintf("OOO-REPRICE:%s", fEvID)
        _, _ = s.ledgerService.StornoCorrection(ctx, req.CardID, correction, ref, "Sırasız olay zincirleme uzlaşması")
    }
    _, _ = pool.Exec(ctx, `
        UPDATE jw_validator_events
        SET is_transfer = $1, transfer_level = $2, fare_charged_kurus = $3
        WHERE id = $4
    `, fCalc.IsTransfer, fCalc.TransferLevel, fCalc.FareChargedKurus, fEvID)

    // Bu olayı sonraki binişler için tarihe ekle
    rollingHistory = append(rollingHistory, pricing.TripHistoryItem{
        LineID: fLineID, StationID: fStationID, Direction: fDir,
        TappedAt: fTappedAt, IsTransfer: fCalc.IsTransfer, TransferLevel: fCalc.TransferLevel,
    })
}
```

---

## 2. BUG-02 (P1): `currentDirection` Parametresinin Kullanılmaması

### 2.1. İnceleme
`internal/pricing/engine.go:41` fonksiyon imzası:
```go
func (e *PricingEngine) CalculateFare(
    card domain.Card,
    tariff domain.Tariff,
    currentLineID string,
    currentStationID string,
    currentDirection int,
    tappedAt time.Time,
    history []TripHistoryItem,
) CalculationResult
```
Ancak `engine.go:127-133` satırlarında:
```go
sameLineFound := false
for _, trip := range chainTrips {
    if trip.LineID == currentLineID {
        sameLineFound = true
        break
    }
}
```
`currentDirection` değişkeni gövdede hiçbir yerde kullanılmamaktadır.
Eğer şartnamedeki "aynı hatta geri dönüş (ters yön)" kuralı sadece `currentDirection != trip.Direction` durumunu kastediyorsa, şu anki kod aynı hatta aynı yönde binişi de ters yönde binişi de aynı şekilde engellemektedir. Fonksiyondaki parametre atıl kalmıştır.

---

## 3. BUG-03 (P1): Günlük ve Aylık Veritabanı Sayaçlarının Sıfırlanmaması

### 3.1. İnceleme
`jw_cards` tablosunda:
- `daily_taps_count`
- `monthly_capping_spent_kurus`
kolonları bulunmaktadır.

`internal/sync/service.go:303` satırında her binişte:
```sql
UPDATE jw_cards
SET balance_kurus = $1,
    daily_taps_count = daily_taps_count + 1,
    monthly_capping_spent_kurus = monthly_capping_spent_kurus + $2
WHERE id = $3
```
Bu sayaçlar arttırılmaktadır. Ancak kartın en son biniş tarihi ile bugünün tarihi farklı olduğunda `daily_taps_count` sıfırlanmamaktadır.
Aynı şekilde ay değiştiğinde `monthly_capping_spent_kurus` sıfırlanmamaktadır.
Bu durum uzun vadede bir öğrencinin 10 biniş limitini bir kez doldurduktan sonra hayatı boyunca tam bilet ödemesine sebep olur.

---

## 4. SPEC-01: Şartname Belirsizlikleri (`SPEC_UNDEFINED`)

`proje.md` şartnamesinde netleştirilmemiş ve mevcut kod/seed verilerinden tespit edilen varsayımlar:
1. **Gece Tarifesi Saatleri:** Şartnamede saat belirtilmemiştir. Sistemde `00:00:00 - 05:59:59` arası kabul edilmiş ve `%50` zam (`night_surcharge_pct = 50`) uygulanmıştır.
2. **Öğrenci Günlük Limiti:** Şartnamede adet veya bütçe belirtilmemiştir. Sistemde günde en fazla `10` indirimli biniş hakkı (`student_daily_limit = 10`) olarak modellenmiştir.
3. **Aylık Abonman Tavanı:** Şartnamede tutar belirtilmemiştir. Sistemde `150000 kuruş` (1500 TL) olarak tanımlanmıştır.

Bu politikalar test oraclemızda (`qa/go/fare_oracle.go`) aynı kurallarla bağımsız olarak referans alınmıştır.

