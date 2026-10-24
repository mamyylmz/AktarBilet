package pricing

import (
	"testing"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func getTestTariff() domain.Tariff {
	return domain.Tariff{
		ID:                       "tariff_std",
		Name:                     "Standart Tarife",
		Mode:                     domain.ModeBus,
		BaseFareKurus:            2000,   // 20.00 TL
		StudentFareKurus:         1000,   // 10.00 TL
		ElderlyFareKurus:         1500,   // 15.00 TL
		Transfer1DiscountPct:     50,     // %50 (10.00 TL)
		Transfer2DiscountPct:     75,     // %75 (5.00 TL)
		NightSurchargePct:        50,     // %50 (30.00 TL)
		StudentDailyLimit:        10,     // 10 indirimli biniş
		MonthlyCappingLimitKurus: 150000, // 1500 TL
		IsActive:                 true,
	}
}

func TestPricingEngine_StandardFareAndTransfers(t *testing.T) {
	engine := NewPricingEngine()
	tariff := getTestTariff()
	card := domain.Card{
		ID:           "CARD_001",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	}

	now := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC) // 14:00 (Gündüz)

	// 1. İlk Biniş (Hat 1) -> 20.00 TL (2000 kuruş)
	res1 := engine.CalculateFare(card, tariff, "LINE_1", "STAT_A", 1, now, nil)
	if res1.FareChargedKurus != 2000 || res1.IsTransfer || res1.TransferLevel != 0 {
		t.Fatalf("Beklenen ilk biniş 2000 kuruş, alınan: %d, transfer: %v", res1.FareChargedKurus, res1.IsTransfer)
	}

	history := []TripHistoryItem{
		{LineID: "LINE_1", StationID: "STAT_A", Direction: 1, TappedAt: now, IsTransfer: false, TransferLevel: 0},
	}

	// 2. 1. Aktarma (Hat 2, 25 dk sonra) -> %50 indirim = 1000 kuruş
	t2 := now.Add(25 * time.Minute)
	res2 := engine.CalculateFare(card, tariff, "LINE_2", "STAT_B", 1, t2, history)
	if res2.FareChargedKurus != 1000 || !res2.IsTransfer || res2.TransferLevel != 1 {
		t.Fatalf("Beklenen 1. aktarma 1000 kuruş, alınan: %d, transferLevel: %d", res2.FareChargedKurus, res2.TransferLevel)
	}

	history = append(history, TripHistoryItem{
		LineID: "LINE_2", StationID: "STAT_B", Direction: 1, TappedAt: t2, IsTransfer: true, TransferLevel: 1,
	})

	// 3. 2. Aktarma (Hat 3, 60 dk sonra) -> %75 indirim = 500 kuruş
	t3 := now.Add(60 * time.Minute)
	res3 := engine.CalculateFare(card, tariff, "LINE_3", "STAT_C", 1, t3, history)
	if res3.FareChargedKurus != 500 || !res3.IsTransfer || res3.TransferLevel != 2 {
		t.Fatalf("Beklenen 2. aktarma 500 kuruş, alınan: %d, transferLevel: %d", res3.FareChargedKurus, res3.TransferLevel)
	}

	history = append(history, TripHistoryItem{
		LineID: "LINE_3", StationID: "STAT_C", Direction: 1, TappedAt: t3, IsTransfer: true, TransferLevel: 2,
	})

	// 4. 3. Aktarma (Hat 4, 80 dk sonra) -> Maksimum 2 aktarma aşıldığı için tam tarife = 2000 kuruş
	t4 := now.Add(80 * time.Minute)
	res4 := engine.CalculateFare(card, tariff, "LINE_4", "STAT_D", 1, t4, history)
	if res4.FareChargedKurus != 2000 || res4.IsTransfer {
		t.Fatalf("3. aktarma tam ücret olmalı, alınan: %d, transfer: %v", res4.FareChargedKurus, res4.IsTransfer)
	}
}

func TestPricingEngine_ReverseRouteRule(t *testing.T) {
	engine := NewPricingEngine()
	tariff := getTestTariff()
	card := domain.Card{
		ID:           "CARD_002",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	}

	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	history := []TripHistoryItem{
		{LineID: "LINE_1", StationID: "STAT_A", Direction: 1, TappedAt: now, IsTransfer: false, TransferLevel: 0},
	}

	// 30 dk sonra aynı hatta geri dönüş binişi yapılıyor (Ters yön kuralı)
	t2 := now.Add(30 * time.Minute)
	res := engine.CalculateFare(card, tariff, "LINE_1", "STAT_Z", 2, t2, history)

	// Kural: Aynı hatta dönüş aktarma sayılmaz, tam ücret (2000 kuruş) alınmalıdır
	if res.IsTransfer || res.FareChargedKurus != 2000 {
		t.Fatalf("Aynı hatta geri dönüş aktarma sayılmamalı, ücret 2000 kuruş olmalı. Alınan: %d, Transfer: %v",
			res.FareChargedKurus, res.IsTransfer)
	}
}

func TestPricingEngine_TransferWindowExpiration(t *testing.T) {
	engine := NewPricingEngine()
	tariff := getTestTariff()
	card := domain.Card{
		ID:           "CARD_003",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	}

	now := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	history := []TripHistoryItem{
		{LineID: "LINE_1", StationID: "STAT_A", Direction: 1, TappedAt: now, IsTransfer: false, TransferLevel: 0},
	}

	// 95 dakika sonra (90 dk penceresi bitti)
	t2 := now.Add(95 * time.Minute)
	res := engine.CalculateFare(card, tariff, "LINE_2", "STAT_B", 1, t2, history)

	if res.IsTransfer || res.FareChargedKurus != 2000 {
		t.Fatalf("90 dk sonrası aktarma sayılmamalı, alınan ücret: %d, transfer: %v", res.FareChargedKurus, res.IsTransfer)
	}
}

func TestPricingEngine_NightSurcharge(t *testing.T) {
	engine := NewPricingEngine()
	tariff := getTestTariff()
	card := domain.Card{
		ID:           "CARD_004",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	}

	// Gece 02:30 (00:00 - 05:59 arası %50 gece zammı)
	nightTime := time.Date(2026, 10, 24, 2, 30, 0, 0, time.UTC)
	res := engine.CalculateFare(card, tariff, "LINE_1", "STAT_A", 1, nightTime, nil)

	// 2000 + %50 = 3000 kuruş
	if res.FareChargedKurus != 3000 || !res.IsNightFare {
		t.Fatalf("Gece tarifesi 3000 kuruş olmalı, alınan: %d", res.FareChargedKurus)
	}
}

func TestPricingEngine_StudentDailyLimit(t *testing.T) {
	engine := NewPricingEngine()
	tariff := getTestTariff()
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

	// 9 biniş yapmış öğrenci (Limit: 10) -> İndirimli tarife (1000 kuruş)
	cardUnderLimit := domain.Card{
		ID:             "CARD_STUDENT_1",
		CardType:       domain.CardTypeOgrenci,
		DailyTapsCount: 9,
	}
	res1 := engine.CalculateFare(cardUnderLimit, tariff, "LINE_1", "STAT_A", 1, now, nil)
	if res1.FareChargedKurus != 1000 {
		t.Fatalf("Öğrenci 10. biniş indirimli olmalı (1000 kuruş), alınan: %d", res1.FareChargedKurus)
	}

	// 10 biniş yapmış öğrenci (Limit dolmuş) -> Tam tarife (2000 kuruş)
	cardAtLimit := domain.Card{
		ID:             "CARD_STUDENT_2",
		CardType:       domain.CardTypeOgrenci,
		DailyTapsCount: 10,
	}
	res2 := engine.CalculateFare(cardAtLimit, tariff, "LINE_1", "STAT_A", 1, now, nil)
	if res2.FareChargedKurus != 2000 {
		t.Fatalf("Öğrenci günlük limit sonrası tam tarife ödemeli (2000 kuruş), alınan: %d", res2.FareChargedKurus)
	}
}

func TestPricingEngine_MonthlyFareCapping(t *testing.T) {
	engine := NewPricingEngine()
	tariff := getTestTariff()
	now := time.Date(2026, 10, 24, 12, 0, 0, 0, time.UTC)

	// Tavanı doldurmuş kart (1500 TL harcanmış)
	cappedCard := domain.Card{
		ID:                  "CARD_CAPPED",
		CardType:            domain.CardTypeTam,
		MonthlyCappingSpent: 150000,
	}
	resCapped := engine.CalculateFare(cappedCard, tariff, "LINE_1", "STAT_A", 1, now, nil)
	if resCapped.FareChargedKurus != 0 || !resCapped.IsCapped {
		t.Fatalf("Tavanı aşan kart için biniş ücretsiz (0 kuruş) olmalı, alınan: %d", resCapped.FareChargedKurus)
	}

	// Tavana 8 TL (800 kuruş) kalmış kart -> 20 TL yerine yalnızca 8 TL kesilmeli
	nearCapCard := domain.Card{
		ID:                  "CARD_NEAR_CAP",
		CardType:            domain.CardTypeTam,
		MonthlyCappingSpent: 149200, // 150000 - 800
	}
	resNearCap := engine.CalculateFare(nearCapCard, tariff, "LINE_1", "STAT_A", 1, now, nil)
	if resNearCap.FareChargedKurus != 800 {
		t.Fatalf("Tavanı tamamlamak için kalan 800 kuruş kesilmeli, alınan: %d", resNearCap.FareChargedKurus)
	}
}

func TestPricingEngine_OutOfOrderDeterministicReplay(t *testing.T) {
	engine := NewPricingEngine()
	tariff := getTestTariff()
	card := domain.Card{
		ID:           "CARD_OUT_OF_ORDER",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	}

	t0 := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	t1 := t0.Add(20 * time.Minute)
	t2 := t0.Add(40 * time.Minute)

	// Olaylar ters sırada (t1 önce, t0 sonra) geçmişe sunuluyor
	unorderedHistory := []TripHistoryItem{
		{LineID: "LINE_2", StationID: "STAT_B", Direction: 1, TappedAt: t1, IsTransfer: true, TransferLevel: 1},
		{LineID: "LINE_1", StationID: "STAT_A", Direction: 1, TappedAt: t0, IsTransfer: false, TransferLevel: 0},
	}

	// t2 anındaki 2. aktarma hesaplanıyor
	res := engine.CalculateFare(card, tariff, "LINE_3", "STAT_C", 1, t2, unorderedHistory)

	// Sırasız gelse bile motorun sortedHistory sayesinde deterministik olarak 2. aktarmayı tespit edip 500 kuruş kesmesi gerekir
	if res.FareChargedKurus != 500 || res.TransferLevel != 2 {
		t.Fatalf("Sırasız geçmişte deterministik hesaplama başarısız. Beklenen: 500 kuruş L2, alınan: %d (L%d)",
			res.FareChargedKurus, res.TransferLevel)
	}
}
