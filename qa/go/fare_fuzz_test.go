package goqa

import (
	"encoding/json"
	"math/rand"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
)

type FuzzFailureRecord struct {
	Iteration    int                       `json:"iteration"`
	Property     string                    `json:"property"`
	Description  string                    `json:"description"`
	CardType     domain.CardType           `json:"card_type"`
	TappedAt     string                    `json:"tapped_at"`
	ExpectedFare int64                     `json:"expected_fare"`
	ActualFare   int64                     `json:"actual_fare"`
	History      []pricing.TripHistoryItem `json:"history"`
}

func TestT1_PropertyAndFuzz_1000Cases(t *testing.T) {
	engine := pricing.NewPricingEngine()
	tariff := defaultTestTariff()

	// Deterministik tohum (Seed = 42)
	rng := rand.New(rand.NewSource(42))

	lines := []string{"LINE_5", "LINE_10", "LINE_15", "LINE_VAPUR", "LINE_M4", "LINE_35", "LINE_800"}
	stations := []string{"STAT_A", "STAT_B", "STAT_C", "STAT_D", "STAT_E", "STAT_F"}
	cardTypes := []domain.CardType{domain.CardTypeTam, domain.CardTypeOgrenci, domain.CardTypeYasli, domain.CardTypeUcretsiz}

	baseTime := time.Date(2026, 10, 26, 0, 0, 0, 0, IstanbulLocation)

	var failures []FuzzFailureRecord

	for iter := 1; iter <= 1000; iter++ {
		cType := cardTypes[rng.Intn(len(cardTypes))]
		card := domain.Card{
			ID:                  "FUZZ_CARD",
			CardType:            cType,
			BalanceKurus:        100000,
			DailyTapsCount:      rng.Intn(15),
			MonthlyCappingSpent: domain.Kurus(rng.Int63n(160000)),
		}

		// Random trip time in a 48-hour span
		tripOffset := time.Duration(rng.Int63n(48*3600)) * time.Second
		tapTimeIstanbul := baseTime.Add(tripOffset)

		// Generate 0 to 6 past history trips within the preceding 120 minutes
		numHistory := rng.Intn(7)
		var history []pricing.TripHistoryItem
		hTime := tapTimeIstanbul

		for h := 0; h < numHistory; h++ {
			stepBack := time.Duration(rng.Intn(30)+5) * time.Minute
			hTime = hTime.Add(-stepBack)
			hLine := lines[rng.Intn(len(lines))]
			hDir := rng.Intn(2) + 1
			isTrans := (h > 0)
			level := 0
			if isTrans {
				level = (h % 2) + 1
			}
			history = append([]pricing.TripHistoryItem{
				{
					LineID:        hLine,
					StationID:     stations[rng.Intn(len(stations))],
					Direction:     hDir,
					TappedAt:      hTime,
					IsTransfer:    isTrans,
					TransferLevel: level,
				},
			}, history...)
		}

		currLine := lines[rng.Intn(len(lines))]
		currDir := rng.Intn(2) + 1

		// 1. Invariant: Non-negative fare
		res := engine.CalculateFare(card, tariff, currLine, stations[0], currDir, tapTimeIstanbul, history)
		if res.FareChargedKurus < 0 {
			failures = append(failures, FuzzFailureRecord{
				Iteration: iter, Property: "Non-Negative Fare",
				Description: "Ücret negatif olamaz",
				CardType:    cType, TappedAt: tapTimeIstanbul.Format(time.RFC3339),
				ActualFare: int64(res.FareChargedKurus), History: history,
			})
		}

		// 2. Invariant: Max fare upper bound (Cannot exceed 3000 kuruş for normal rides)
		if res.FareChargedKurus > 3000 {
			failures = append(failures, FuzzFailureRecord{
				Iteration: iter, Property: "Max Fare Bound",
				Description: "Ücret tavanı aşılamaz",
				CardType:    cType, TappedAt: tapTimeIstanbul.Format(time.RFC3339),
				ActualFare: int64(res.FareChargedKurus), History: history,
			})
		}

		// 3. Invariant: Timezone Consistency
		// The exact same physical instant sent as UTC must evaluate to identical fare!
		resUTC := engine.CalculateFare(card, tariff, currLine, stations[0], currDir, tapTimeIstanbul.UTC(), history)
		if res.FareChargedKurus != resUTC.FareChargedKurus {
			failures = append(failures, FuzzFailureRecord{
				Iteration: iter, Property: "Timezone Invariance",
				Description: "Aynı fiziksel an UTC ve Istanbul formatında farklı ücretlendirildi",
				CardType:    cType, TappedAt: tapTimeIstanbul.Format(time.RFC3339),
				ExpectedFare: int64(res.FareChargedKurus),
				ActualFare:   int64(resUTC.FareChargedKurus),
				History:      history,
			})
		}

		// 4. Invariant: Monthly Capping
		if card.MonthlyCappingSpent >= tariff.MonthlyCappingLimitKurus && res.FareChargedKurus != 0 {
			failures = append(failures, FuzzFailureRecord{
				Iteration: iter, Property: "Fare Capping Limit",
				Description: "Aylık tavan aşıldığında ücret 0 olmalıdır",
				CardType:    cType, TappedAt: tapTimeIstanbul.Format(time.RFC3339),
				ExpectedFare: 0,
				ActualFare:   int64(res.FareChargedKurus),
				History:      history,
			})
		}
	}

	// Artifacts output
	outPath := filepath.Join("..", "artifacts", "t1-fare-engine", "fuzz_failures.json")
	_ = os.MkdirAll(filepath.Dir(outPath), 0755)
	raw, _ := json.MarshalIndent(failures, "", "  ")
	_ = os.WriteFile(outPath, raw, 0644)

	t.Logf("1000 vakalık property ve fuzz testi tamamlandı. Tespit edilen ihlal sayısı: %d", len(failures))

	if len(failures) > 0 {
		t.Errorf("Property-based testte %d ihlal tespit edildi! (İlk hata: %s - %s)",
			len(failures), failures[0].Property, failures[0].Description)
	}
}
