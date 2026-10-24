package anomaly

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func setupTestStations(detector *AnomalyDetector) {
	// Kadıköy İskelesi (40.9904, 29.0254)
	detector.RegisterStation(StationCoordinate{
		ID:        "STAT_KADIKOY",
		Name:      "Kadıköy İskele",
		Latitude:  40.9904,
		Longitude: 29.0254,
	})

	// Ayrılık Çeşmesi (40.9982, 29.0305) ~1.0 km
	detector.RegisterStation(StationCoordinate{
		ID:        "STAT_AYRILIK",
		Name:      "Ayrılık Çeşmesi",
		Latitude:  40.9982,
		Longitude: 29.0305,
	})

	// Sarıyer Merkez (41.1685, 29.0554) ~20.0 km
	detector.RegisterStation(StationCoordinate{
		ID:        "STAT_SARIYER",
		Name:      "Sarıyer Merkez",
		Latitude:  41.1685,
		Longitude: 29.0554,
	})
}

func TestAnomaly_ImpossibleSpeed(t *testing.T) {
	ctx := context.Background()
	detector := NewAnomalyDetector()
	setupTestStations(detector)

	cardID := "CARD_CLONE_TEST"
	t1 := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	evID1 := uuid.New()

	// 1. Kadıköy'de biniş (10:00:00)
	res1 := detector.CheckTap(ctx, cardID, "STAT_KADIKOY", t1, &evID1)
	if res1.IsAnomaly {
		t.Fatalf("İlk biniş anomali olmamalı")
	}

	// 2. 4 dakika sonra Sarıyer'de biniş (10:04:00) - Mesafe: ~20 km, Hız: ~300 km/s!
	t2 := t1.Add(4 * time.Minute)
	evID2 := uuid.New()
	res2 := detector.CheckTap(ctx, cardID, "STAT_SARIYER", t2, &evID2)

	if !res2.IsAnomaly {
		t.Fatalf("4 dakika içinde Kadıköy'den Sarıyer'e biniş anomali olarak işaretlenmeliydi!")
	}
	if res2.AnomalyType != domain.AnomalyImpossibleSpeed {
		t.Fatalf("Anomali türü IMPOSSIBLE_TRAVEL_SPEED olmalıydı, alınan: %s", res2.AnomalyType)
	}
	if res2.CalculatedSpeedKmh < 200.0 {
		t.Fatalf("Hesaplanan hız > 200 km/s olmalı, bulunan: %.2f km/s", res2.CalculatedSpeedKmh)
	}
}

func TestAnomaly_NormalTravel(t *testing.T) {
	ctx := context.Background()
	detector := NewAnomalyDetector()
	setupTestStations(detector)

	cardID := "CARD_NORMAL_TEST"
	t1 := time.Date(2026, 10, 24, 10, 0, 0, 0, time.UTC)
	evID1 := uuid.New()

	// Kadıköy (10:00:00)
	detector.CheckTap(ctx, cardID, "STAT_KADIKOY", t1, &evID1)

	// Ayrılık Çeşmesi 8 dakika sonra (10:08:00) - Mesafe: ~1 km, Hız: ~7.5 km/s -> Normal
	t2 := t1.Add(8 * time.Minute)
	evID2 := uuid.New()
	res2 := detector.CheckTap(ctx, cardID, "STAT_AYRILIK", t2, &evID2)

	if res2.IsAnomaly {
		t.Fatalf("Normal seyahat hızı anomali olarak işaretlenmemeli: %+v", res2)
	}
}

func BenchmarkAnomaly_HighThroughput1MPerHour(b *testing.B) {
	ctx := context.Background()
	detector := NewAnomalyDetector()
	setupTestStations(detector)

	t0 := time.Now()
	evID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		detector.CheckTap(ctx, "BENCH_CARD", "STAT_AYRILIK", t0.Add(time.Duration(i)*time.Second), &evID)
	}
}
