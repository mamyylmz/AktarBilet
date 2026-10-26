package goqa

import (
	"testing"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
)

func defaultTestTariff() domain.Tariff {
	return domain.Tariff{
		ID:                       "TARIFF_ISTANBUL_STD",
		Name:                     "İstanbul Standart Tarife",
		Mode:                     domain.ModeBus,
		BaseFareKurus:            2000,
		StudentFareKurus:         1000,
		ElderlyFareKurus:         1500,
		Transfer1DiscountPct:     50,
		Transfer2DiscountPct:     75,
		NightSurchargePct:        50,
		StudentDailyLimit:        10,
		MonthlyCappingLimitKurus: 150000,
		IsActive:                 true,
	}
}

func defaultRefTariff() ReferenceTariff {
	return ReferenceTariff{
		ID:                       "TARIFF_ISTANBUL_STD",
		BaseFareKurus:            2000,
		StudentFareKurus:         1000,
		ElderlyFareKurus:         1500,
		Transfer1DiscountPct:     50,
		Transfer2DiscountPct:     75,
		NightSurchargePct:        50,
		StudentDailyLimit:        10,
		MonthlyCappingLimitKurus: 150000,
	}
}

// ============================================================================
// T1-01: İlk Biniş ve Kart Türleri
// ============================================================================
func TestT1_01_CardTypesAndInitialFares(t *testing.T) {
	engine := pricing.NewPricingEngine()
	oracle := NewReferenceFareOracle()
	tariff := defaultTestTariff()
	refTariff := defaultRefTariff()
	now := time.Date(2026, 10, 26, 14, 0, 0, 0, IstanbulLocation)

	cases := []struct {
		name         string
		cardType     domain.CardType
		refCardType  ReferenceCardType
		expectedFare int64
	}{
		{"Tam Kart", domain.CardTypeTam, RefCardTam, 2000},
		{"Öğrenci Kart", domain.CardTypeOgrenci, RefCardOgrenci, 1000},
		{"Yaşlı / 65+ Kart", domain.CardTypeYasli, RefCardYasli, 1500},
		{"Ücretsiz Kart", domain.CardTypeUcretsiz, RefCardUcretsiz, 0},
		{"Bilinmeyen Kart (Fallback to Base)", "bilinmeyen", RefCardTam, 2000},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			card := domain.Card{ID: "CARD_01", CardType: tc.cardType, BalanceKurus: 10000}
			refCard := ReferenceCard{ID: "CARD_01", CardType: tc.refCardType, BalanceKurus: 10000}

			// Oracle calculation
			oracleRes := oracle.CalculateExpectedFare(refCard, refTariff, ReferenceTrip{
				LineID: "LINE_500T", StationID: "STAT_1", Direction: 1, TappedAt: now,
			}, nil)

			if oracleRes.FareChargedKurus != tc.expectedFare {
				t.Fatalf("Oracle hatası [%s]: beklenen %d, oracle %d", tc.name, tc.expectedFare, oracleRes.FareChargedKurus)
			}

			// Production engine calculation
			prodRes := engine.CalculateFare(card, tariff, "LINE_500T", "STAT_1", 1, now, nil)

			if int64(prodRes.FareChargedKurus) != tc.expectedFare {
				t.Errorf("[FAIL T1-01] %s: Beklenen %d kuruş, Production %d kuruş",
					tc.name, tc.expectedFare, prodRes.FareChargedKurus)
			}
			if prodRes.IsTransfer {
				t.Errorf("[FAIL T1-01] %s: İlk biniş transfer olarak işaretlendi", tc.name)
			}
		})
	}
}

// ============================================================================
// T1-02: 90 Dakikalık Aktarma Penceresi ve Sınır Değerleri
// ============================================================================
func TestT1_02_TransferWindow_Boundaries(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff()
	card := domain.Card{ID: "CARD_01", CardType: domain.CardTypeTam, BalanceKurus: 10000}

	t0 := time.Date(2026, 10, 26, 10, 0, 0, 0, IstanbulLocation)
	history := []pricing.TripHistoryItem{
		{LineID: "LINE_A", StationID: "STAT_A", Direction: 1, TappedAt: t0, IsTransfer: false, TransferLevel: 0},
	}

	timeOffsets := []struct {
		offset           time.Duration
		name             string
		expectTransfer   bool
		expectedFare     domain.Kurus
		specUndefinedTag bool
	}{
		{1 * time.Second, "T0 + 1s (Pencere İçi)", true, 1000, false},
		{1 * time.Minute, "T0 + 1m (Pencere İçi)", true, 1000, false},
		{30 * time.Minute, "T0 + 30m (Pencere İçi)", true, 1000, false},
		{89*time.Minute + 59*time.Second, "T0 + 89m59s (Tam sınır öncesi)", true, 1000, false},
		{90 * time.Minute, "T0 + 90m00s (Tam sınır)", true, 1000, false},
		{90*time.Minute + 1*time.Second, "T0 + 90m01s (Pencere Dışı)", false, 2000, false},
		{91 * time.Minute, "T0 + 91m (Pencere Dışı)", false, 2000, false},
		{120 * time.Minute, "T0 + 120m (Pencere Dışı)", false, 2000, false},
	}

	for _, tc := range timeOffsets {
		t.Run(tc.name, func(t *testing.T) {
			tTarget := t0.Add(tc.offset)
			res := engine.CalculateFare(card, tariff, "LINE_B", "STAT_B", 1, tTarget, history)

			if res.IsTransfer != tc.expectTransfer || res.FareChargedKurus != tc.expectedFare {
				t.Errorf("[FAIL T1-02] %s: Beklenen (IsTransfer=%v, Fare=%d), Production (IsTransfer=%v, Fare=%d)",
					tc.name, tc.expectTransfer, tc.expectedFare, res.IsTransfer, res.FareChargedKurus)
			}
		})
	}

	// Doğrulama: Pencere son aktarmadan DEĞİL, İLK BİNİŞTEN hesaplanmalı!
	// T0: İlk biniş
	// T0 + 50m: 1. Aktarma
	// T0 + 100m: 2. Biniş (T0+100m > 90m olduğundan aktarma SAYILMAMALI, tam ücret olmalı!)
	t.Run("WindowCalculatedFromInitialNotLastTransfer", func(t *testing.T) {
		t1 := t0.Add(50 * time.Minute)
		t2 := t0.Add(100 * time.Minute) // 100m from T0 (>90m), but only 50m from T1 (<90m)

		chain := []pricing.TripHistoryItem{
			{LineID: "LINE_A", StationID: "STAT_A", Direction: 1, TappedAt: t0, IsTransfer: false, TransferLevel: 0},
			{LineID: "LINE_B", StationID: "STAT_B", Direction: 1, TappedAt: t1, IsTransfer: true, TransferLevel: 1},
		}

		res := engine.CalculateFare(card, tariff, "LINE_C", "STAT_C", 1, t2, chain)
		if res.IsTransfer || res.FareChargedKurus != 2000 {
			t.Errorf("[FAIL T1-02] Aktarma penceresi ilk biniş yerine son aktarmadan hesaplandı! T0+100m için beklenen 2000 kuruş (tam), alınan: %d (transfer: %v)",
				res.FareChargedKurus, res.IsTransfer)
		}
	})
}

// ============================================================================
// T1-03: En Fazla İki İndirimli Aktarma Kuralı
// ============================================================================
func TestT1_03_MaxTwoDiscountedTransfers(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff()
	card := domain.Card{ID: "CARD_01", CardType: domain.CardTypeTam, BalanceKurus: 10000}
	t0 := time.Date(2026, 10, 26, 10, 0, 0, 0, IstanbulLocation)

	var history []pricing.TripHistoryItem

	// 1. İlk Biniş (T0) -> 2000 kuruş
	res1 := engine.CalculateFare(card, tariff, "LINE_1", "STAT_1", 1, t0, history)
	if res1.FareChargedKurus != 2000 || res1.IsTransfer {
		t.Fatalf("[FAIL T1-03] 1. biniş: beklenen 2000, alınan %d", res1.FareChargedKurus)
	}
	history = append(history, pricing.TripHistoryItem{
		LineID: "LINE_1", StationID: "STAT_1", Direction: 1, TappedAt: t0, IsTransfer: false, TransferLevel: 0,
	})

	// 2. 1. Aktarma (T0 + 15m) -> 1000 kuruş (%50 indirim)
	t1 := t0.Add(15 * time.Minute)
	res2 := engine.CalculateFare(card, tariff, "LINE_2", "STAT_2", 1, t1, history)
	if res2.FareChargedKurus != 1000 || !res2.IsTransfer || res2.TransferLevel != 1 {
		t.Fatalf("[FAIL T1-03] 2. biniş (1. Aktarma): beklenen 1000, alınan %d (level=%d)", res2.FareChargedKurus, res2.TransferLevel)
	}
	history = append(history, pricing.TripHistoryItem{
		LineID: "LINE_2", StationID: "STAT_2", Direction: 1, TappedAt: t1, IsTransfer: true, TransferLevel: 1,
	})

	// 3. 2. Aktarma (T0 + 30m) -> 500 kuruş (%75 indirim)
	t2 := t0.Add(30 * time.Minute)
	res3 := engine.CalculateFare(card, tariff, "LINE_3", "STAT_3", 1, t2, history)
	if res3.FareChargedKurus != 500 || !res3.IsTransfer || res3.TransferLevel != 2 {
		t.Fatalf("[FAIL T1-03] 3. biniş (2. Aktarma): beklenen 500, alınan %d (level=%d)", res3.FareChargedKurus, res3.TransferLevel)
	}
	history = append(history, pricing.TripHistoryItem{
		LineID: "LINE_3", StationID: "STAT_3", Direction: 1, TappedAt: t2, IsTransfer: true, TransferLevel: 2,
	})

	// 4. 3. Aktarma (T0 + 45m) -> 2000 kuruş (İNDİRİMSİZ, maksimum 2 aktarma aşıldı)
	t3 := t0.Add(45 * time.Minute)
	res4 := engine.CalculateFare(card, tariff, "LINE_4", "STAT_4", 1, t3, history)
	if res4.FareChargedKurus != 2000 {
		t.Errorf("[FAIL T1-03] 4. biniş (3. Aktarma): beklenen 2000 kuruş (indirimsiz), alınan %d", res4.FareChargedKurus)
	}
	history = append(history, pricing.TripHistoryItem{
		LineID: "LINE_4", StationID: "STAT_4", Direction: 1, TappedAt: t3, IsTransfer: res4.IsTransfer, TransferLevel: res4.TransferLevel,
	})

	// 5. 4. Aktarma (T0 + 60m) -> 2000 kuruş (İNDİRİMSİZ, hala ilk 90 dk penceresinde)
	t4 := t0.Add(60 * time.Minute)
	res5 := engine.CalculateFare(card, tariff, "LINE_5", "STAT_5", 1, t4, history)
	if res5.FareChargedKurus != 2000 {
		t.Errorf("[FAIL T1-03 - CRITICAL] 5. biniş (4. Aktarma): beklenen 2000 kuruş! Alınan: %d. (3. biniş IsTransfer=false olduğu için yeni indirim zinciri başlatmış olabilir!)",
			res5.FareChargedKurus)
	}
}

// ============================================================================
// T1-04: Aynı Hatta Ters Yön ve Aynı Hat Kuralı
// ============================================================================
func TestT1_04_SameLineReverseDirection(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff()
	card := domain.Card{ID: "CARD_01", CardType: domain.CardTypeTam, BalanceKurus: 10000}
	t0 := time.Date(2026, 10, 26, 11, 0, 0, 0, IstanbulLocation)

	t.Run("ForwardThenReverseOnSameLine_MustChargeFullFare", func(t *testing.T) {
		history := []pricing.TripHistoryItem{
			{LineID: "LINE_500T", StationID: "STAT_KADIKOY", Direction: 1, TappedAt: t0, IsTransfer: false},
		}

		// 20 dk sonra aynı hatta TERS yön (Direction 2)
		t1 := t0.Add(20 * time.Minute)
		res := engine.CalculateFare(card, tariff, "LINE_500T", "STAT_CEVIZLIBAG", 2, t1, history)

		if res.IsTransfer || res.FareChargedKurus != 2000 {
			t.Errorf("[FAIL T1-04] Aynı hatta ters yöne geri dönüşe indirim uygulandı! Beklenen 2000, alınan: %d", res.FareChargedKurus)
		}
	})

	t.Run("ForwardThenForwardOnSameLine_MustNotBeTransfer", func(t *testing.T) {
		history := []pricing.TripHistoryItem{
			{LineID: "LINE_500T", StationID: "STAT_A", Direction: 1, TappedAt: t0, IsTransfer: false},
		}

		t1 := t0.Add(20 * time.Minute)
		res := engine.CalculateFare(card, tariff, "LINE_500T", "STAT_B", 1, t1, history)

		if res.IsTransfer || res.FareChargedKurus != 2000 {
			t.Errorf("[FAIL T1-04] Aynı hatta aynı yönde tekrar binişe aktarma indirimi verildi! Beklenen 2000, alınan: %d", res.FareChargedKurus)
		}
	})

	t.Run("A_Fwd_Then_B_Then_A_Rev_MustNotDiscountA_Rev", func(t *testing.T) {
		history := []pricing.TripHistoryItem{
			{LineID: "LINE_A", StationID: "STAT_1", Direction: 1, TappedAt: t0, IsTransfer: false},
			{LineID: "LINE_B", StationID: "STAT_2", Direction: 1, TappedAt: t0.Add(15 * time.Minute), IsTransfer: true, TransferLevel: 1},
		}

		// T0 + 35m: Line A'ya bu kez ters yönde (Direction 2) biniş
		t2 := t0.Add(35 * time.Minute)
		res := engine.CalculateFare(card, tariff, "LINE_A", "STAT_3", 2, t2, history)

		if res.IsTransfer || res.FareChargedKurus != 2000 {
			t.Errorf("[FAIL T1-04] A(ileri) -> B -> A(ters) rotasında A(ters) aktarma sayıldı! Beklenen 2000, alınan: %d", res.FareChargedKurus)
		}
	})
}

// ============================================================================
// T1-05: Gece Tarifesi ve Timezone Tutarlılığı (Europe/Istanbul vs UTC)
// ============================================================================
func TestT1_05_NightFareAndTimezone(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff()
	card := domain.Card{ID: "CARD_01", CardType: domain.CardTypeTam, BalanceKurus: 10000}

	// 1. Standart Saat Sınırları (00:00 - 05:59:59 Gece, 06:00 Gündüz)
	times := []struct {
		name         string
		hour         int
		minute       int
		second       int
		expectNight  bool
		expectedFare domain.Kurus
	}{
		{"23:59:59 (Gündüz/Normal)", 23, 59, 59, false, 2000},
		{"00:00:00 (Tam Gece Başlangıcı)", 0, 0, 0, true, 3000},
		{"00:00:01 (Gece)", 0, 0, 1, true, 3000},
		{"03:30:00 (Gece Yarısı)", 3, 30, 0, true, 3000},
		{"05:59:59 (Gece Sonu)", 5, 59, 59, true, 3000},
		{"06:00:00 (Gündüz Başlangıcı)", 6, 0, 0, false, 2000},
	}

	for _, tc := range times {
		t.Run(tc.name, func(t *testing.T) {
			tLocal := time.Date(2026, 10, 26, tc.hour, tc.minute, tc.second, 0, IstanbulLocation)
			res := engine.CalculateFare(card, tariff, "LINE_1", "STAT_1", 1, tLocal, nil)

			if res.IsNightFare != tc.expectNight || res.FareChargedKurus != tc.expectedFare {
				t.Errorf("[FAIL T1-05] %s: Beklenen (Night=%v, Fare=%d), Production (Night=%v, Fare=%d)",
					tc.name, tc.expectNight, tc.expectedFare, res.IsNightFare, res.FareChargedKurus)
			}
		})
	}

	// 2. KRİTİK: UTC vs Europe/Istanbul Zaman Tutarlılığı Testi!
	// İstanbul saatine göre 01:30 gece (Night fare olmalı: 3000 kuruş).
	// UTC cinsinden bu zaman: bir önceki gün 22:30 UTC'dir! (UTC+3)
	t.Run("TimezoneConsistency_UTC_vs_Istanbul", func(t *testing.T) {
		istanbulTime := time.Date(2026, 10, 26, 1, 30, 0, 0, IstanbulLocation) // 01:30 TSİ
		utcTime := istanbulTime.UTC()                                          // 22:30 UTC (aynı fiziksel an)

		resIstanbul := engine.CalculateFare(card, tariff, "LINE_1", "STAT_1", 1, istanbulTime, nil)
		resUTC := engine.CalculateFare(card, tariff, "LINE_1", "STAT_1", 1, utcTime, nil)

		if resIstanbul.FareChargedKurus != resUTC.FareChargedKurus {
			t.Errorf("[FAIL T1-05 - CRITICAL TIMEZONE BUG] Aynı fiziksel biniş zamanı timezone'a göre farklı ücretlendirildi! Istanbul: %d kuruş (Night=%v), UTC: %d kuruş (Night=%v)",
				resIstanbul.FareChargedKurus, resIstanbul.IsNightFare, resUTC.FareChargedKurus, resUTC.IsNightFare)
		}
	})
}

// ============================================================================
// T1-06: Öğrenci Günlük Limiti
// ============================================================================
func TestT1_06_StudentDailyLimit(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff() // StudentDailyLimit = 10
	now := time.Date(2026, 10, 26, 14, 0, 0, 0, IstanbulLocation)

	// 10. biniş öncesi ve 10. biniş: 1000 kuruş
	cardBeforeLimit := domain.Card{
		ID: "CARD_STU", CardType: domain.CardTypeOgrenci, DailyTapsCount: 9, BalanceKurus: 20000,
	}
	resBefore := engine.CalculateFare(cardBeforeLimit, tariff, "LINE_1", "STAT_1", 1, now, nil)
	if resBefore.FareChargedKurus != 1000 {
		t.Errorf("[FAIL T1-06] 10. biniş (DailyTapsCount=9): beklenen 1000 kuruş, alınan %d", resBefore.FareChargedKurus)
	}

	// 11. biniş: Limit doldu (DailyTapsCount >= 10) -> Tam tarife (2000 kuruş)
	cardAtLimit := domain.Card{
		ID: "CARD_STU", CardType: domain.CardTypeOgrenci, DailyTapsCount: 10, BalanceKurus: 20000,
	}
	resAtLimit := engine.CalculateFare(cardAtLimit, tariff, "LINE_1", "STAT_1", 1, now, nil)
	if resAtLimit.FareChargedKurus != 2000 {
		t.Errorf("[FAIL T1-06] 11. biniş (DailyTapsCount=10): beklenen 2000 kuruş (tam), alınan %d", resAtLimit.FareChargedKurus)
	}

	// 12. biniş: 2000 kuruş
	cardPastLimit := domain.Card{
		ID: "CARD_STU", CardType: domain.CardTypeOgrenci, DailyTapsCount: 15, BalanceKurus: 20000,
	}
	resPastLimit := engine.CalculateFare(cardPastLimit, tariff, "LINE_1", "STAT_1", 1, now, nil)
	if resPastLimit.FareChargedKurus != 2000 {
		t.Errorf("[FAIL T1-06] 16. biniş (DailyTapsCount=15): beklenen 2000 kuruş (tam), alınan %d", resPastLimit.FareChargedKurus)
	}
}

// ============================================================================
// T1-07: Aylık Abonman Tavanı (Fare Capping)
// ============================================================================
func TestT1_07_MonthlyFareCapping(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff() // MonthlyCappingLimitKurus = 150000 (1500 TL)
	now := time.Date(2026, 10, 26, 14, 0, 0, 0, IstanbulLocation)

	// 1. Tavana 500 kuruş kala tam biniş (2000 kuruşluk biniş 500 kuruşa düşmeli)
	cardNearCap := domain.Card{
		ID: "CARD_01", CardType: domain.CardTypeTam,
		MonthlyCappingSpent: 149500, // 150000 - 500
		BalanceKurus:        10000,
	}
	resNear := engine.CalculateFare(cardNearCap, tariff, "LINE_1", "STAT_1", 1, now, nil)
	if resNear.FareChargedKurus != 500 {
		t.Errorf("[FAIL T1-07] Tavan tamamlama: beklenen 500 kuruş, alınan %d", resNear.FareChargedKurus)
	}

	// 2. Tavan tam dolduğunda (Spent >= 150000) -> 0 kuruş (Ücretsiz)
	cardAtCap := domain.Card{
		ID: "CARD_01", CardType: domain.CardTypeTam,
		MonthlyCappingSpent: 150000,
		BalanceKurus:        10000,
	}
	resAtCap := engine.CalculateFare(cardAtCap, tariff, "LINE_1", "STAT_1", 1, now, nil)
	if resAtCap.FareChargedKurus != 0 || !resAtCap.IsCapped {
		t.Errorf("[FAIL T1-07] Tavan sonrası biniş: beklenen 0 kuruş (IsCapped=true), alınan %d (IsCapped=%v)",
			resAtCap.FareChargedKurus, resAtCap.IsCapped)
	}
}

// ============================================================================
// T1-08: Sırasız Olay Deterministikliği (Pure Engine Level)
// ============================================================================
func TestT1_08_OutOfOrder_PureEngineSort(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff()
	card := domain.Card{ID: "CARD_01", CardType: domain.CardTypeTam, BalanceKurus: 10000}

	t0 := time.Date(2026, 10, 26, 10, 0, 0, 0, IstanbulLocation)
	t1 := t0.Add(20 * time.Minute)
	t2 := t0.Add(40 * time.Minute)

	// Geçmiş listesi ters sırada sunuluyor (T1 önce, T0 sonra)
	unorderedHistory := []pricing.TripHistoryItem{
		{LineID: "LINE_B", StationID: "STAT_B", Direction: 1, TappedAt: t1, IsTransfer: true, TransferLevel: 1},
		{LineID: "LINE_A", StationID: "STAT_A", Direction: 1, TappedAt: t0, IsTransfer: false, TransferLevel: 0},
	}

	// T2 anındaki 2. aktarma hesaplanıyor (Hat C)
	res := engine.CalculateFare(card, tariff, "LINE_C", "STAT_C", 1, t2, unorderedHistory)

	// Sırasız geçmişe rağmen motor tarihe göre sıralayıp 2. aktarma (%75 indirim = 500 kuruş) hesaplamalı
	if res.FareChargedKurus != 500 || !res.IsTransfer || res.TransferLevel != 2 {
		t.Errorf("[FAIL T1-08] Sırasız geçmişte 2. aktarma hatalı! Beklenen 500 kuruş (level=2), alınan: %d (level=%d, transfer=%v)",
			res.FareChargedKurus, res.TransferLevel, res.IsTransfer)
	}
}

// ============================================================================
// T1-09: Çoklu Kural Etkileşimleri
// ============================================================================
func TestT1_09_MultiRuleInteractions(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff()

	// 1. Gece + Aktarma (Tam Kart):
	// Gece tarifesi baz ücreti: 2000 + 1000 = 3000 kuruş.
	// 1. Aktarma indirimi: %50 -> 3000'in %50'si = 1500 kuruş.
	t.Run("Night_Plus_Transfer", func(t *testing.T) {
		card := domain.Card{ID: "CARD_01", CardType: domain.CardTypeTam, BalanceKurus: 10000}
		tNight0 := time.Date(2026, 10, 26, 2, 0, 0, 0, IstanbulLocation)
		history := []pricing.TripHistoryItem{
			{LineID: "LINE_A", StationID: "STAT_A", Direction: 1, TappedAt: tNight0, IsTransfer: false},
		}

		tNight1 := tNight0.Add(20 * time.Minute)
		res := engine.CalculateFare(card, tariff, "LINE_B", "STAT_B", 1, tNight1, history)

		// Production engine: calculates (3000 * 50) / 100 = 1500 kuruş
		if res.FareChargedKurus != 1500 || !res.IsTransfer || !res.IsNightFare {
			t.Errorf("[FAIL T1-09] Gece + 1. Aktarma: beklenen 1500 kuruş (IsTransfer=true, IsNight=true), alınan %d (transfer=%v, night=%v)",
				res.FareChargedKurus, res.IsTransfer, res.IsNightFare)
		}
	})

	// 2. Öğrenci + Gece Tarifesi:
	// Öğrenci baz: 1000 kuruş. Gece sürşarjı %50: 1000 + 500 = 1500 kuruş.
	t.Run("Student_Plus_Night", func(t *testing.T) {
		card := domain.Card{ID: "CARD_STU", CardType: domain.CardTypeOgrenci, BalanceKurus: 10000}
		tNight := time.Date(2026, 10, 26, 3, 0, 0, 0, IstanbulLocation)
		res := engine.CalculateFare(card, tariff, "LINE_A", "STAT_A", 1, tNight, nil)

		if res.FareChargedKurus != 1500 || !res.IsNightFare {
			t.Errorf("[FAIL T1-09] Öğrenci + Gece: beklenen 1500 kuruş, alınan %d", res.FareChargedKurus)
		}
	})

	// 3. Öğrenci Günlük Limit Aşımı + Gece:
	// Günlük limit doldu (DailyTapsCount >= 10) -> Tam tarife (2000 kuruş) + Gece (%50) = 3000 kuruş!
	t.Run("Student_DailyLimit_Plus_Night", func(t *testing.T) {
		card := domain.Card{ID: "CARD_STU", CardType: domain.CardTypeOgrenci, DailyTapsCount: 10, BalanceKurus: 10000}
		tNight := time.Date(2026, 10, 26, 3, 0, 0, 0, IstanbulLocation)
		res := engine.CalculateFare(card, tariff, "LINE_A", "STAT_A", 1, tNight, nil)

		if res.FareChargedKurus != 3000 || !res.IsNightFare {
			t.Errorf("[FAIL T1-09] Öğrenci limit aşımı + Gece: beklenen 3000 kuruş, alınan %d", res.FareChargedKurus)
		}
	})
}

// ============================================================================
// T1-10: Integer Money & Kesin Kuruş Hassasiyeti
// ============================================================================
func TestT1_10_IntegerMoneyPrecision(t *testing.T) {
	engine := pricing.NewPricingEngine()
	card := domain.Card{ID: "CARD_01", CardType: domain.CardTypeTam, BalanceKurus: 10000}
	now := time.Date(2026, 10, 26, 12, 0, 0, 0, IstanbulLocation)

	// Tek kuruşluk ve küsuratlı indirim testleri (Float yuvarlama hatası olmamalı)
	oddTariff := domain.Tariff{
		ID:                   "ODD_TARIFF",
		BaseFareKurus:        1733, // 17.33 TL
		Transfer1DiscountPct: 33,   // %33
	}

	history := []pricing.TripHistoryItem{
		{LineID: "LINE_A", StationID: "STAT_1", Direction: 1, TappedAt: now.Add(-10 * time.Minute), IsTransfer: false},
	}

	res := engine.CalculateFare(card, oddTariff, "LINE_B", "STAT_2", 1, now, history)

	// 1733 - (1733 * 33)/100 = 1733 - 571 = 1162 kuruş
	expected := domain.Kurus(1733 - (1733*33)/100)
	if res.FareChargedKurus != expected {
		t.Errorf("[FAIL T1-10] Tamsayı kuruş yuvarlama sapması! Beklenen: %d, Alınan: %d", expected, res.FareChargedKurus)
	}

	// Normal binişte asla negatif ücret çıkmamalı
	if res.FareChargedKurus < 0 {
		t.Errorf("[FAIL T1-10] Biniş ücreti negatif çıktı: %d", res.FareChargedKurus)
	}
}
