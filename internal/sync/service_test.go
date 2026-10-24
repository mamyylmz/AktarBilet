package sync

import (
	"context"
	"testing"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
)

func TestSyncService_Idempotency(t *testing.T) {
	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	led := ledger.NewMemoryLedger()

	cardID := "CARD_SYNC_IDEMPOTENT"
	initialBalance := domain.Kurus(10000) // 100.00 TL
	led.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: initialBalance,
	})

	tariff := domain.Tariff{
		ID:            "tariff_test",
		BaseFareKurus: 2000, // 20.00 TL
	}

	syncSvc := NewSyncService(pEngine, led, 5000)
	syncSvc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: initialBalance,
	})

	now := time.Now()
	req := IngestEventRequest{
		IdempotencyKey: "IDEMP_KEY_001",
		ValidatorID:    "VAL_BUS_101",
		SequenceNo:     1,
		CardID:         cardID,
		LineID:         "LINE_10",
		StationID:      "STAT_1",
		Direction:      1,
		TappedAt:       now,
		Tariff:         tariff,
	}

	// 1. İlk gönderim
	ev1, err := syncSvc.IngestOfflineEvent(ctx, req)
	if err != nil {
		t.Fatalf("İlk biniş başarısız: %v", err)
	}
	if ev1.FareChargedKurus != 2000 {
		t.Fatalf("Beklenen ücret 2000 kuruş, alınan: %d", ev1.FareChargedKurus)
	}

	// 2. Mükerrer gönderim (Aynı IdempotencyKey)
	ev2, err := syncSvc.IngestOfflineEvent(ctx, req)
	if err != nil {
		t.Fatalf("Mükerrer gönderim hata verdi: %v", err)
	}
	if ev2.ID != ev1.ID {
		t.Fatalf("İdempotent yanıt aynı olay ID'sine sahip olmalı! ev1: %s, ev2: %s", ev1.ID, ev2.ID)
	}

	// Bakiye kontrolü: 100 TL - 20 TL = 80 TL (8000 kuruş). İkinci biniş tekrar düşmemeli!
	bal, _ := led.GetCardBalance(ctx, cardID)
	if bal != 8000 {
		t.Fatalf("İdempotansi bozuldu! Bakiye 8000 kuruş olmalıydı, alınan: %d", bal)
	}
}

func TestSyncService_NegativeTolerancePolicy(t *testing.T) {
	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	led := ledger.NewMemoryLedger()

	cardID := "CARD_NEG_TOLERANCE"
	initialBalance := domain.Kurus(1000) // 10.00 TL
	led.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: initialBalance,
	})

	tariff := domain.Tariff{
		ID:            "tariff_test",
		BaseFareKurus: 2000, // 20.00 TL
	}

	// 50 TL (-5000 kuruş) tolerans limiti
	syncSvc := NewSyncService(pEngine, led, 5000)
	syncSvc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: initialBalance,
	})

	req := IngestEventRequest{
		IdempotencyKey: "IDEMP_NEG_1",
		ValidatorID:    "VAL_BUS_202",
		SequenceNo:     1,
		CardID:         cardID,
		LineID:         "LINE_20",
		StationID:      "STAT_5",
		Direction:      1,
		TappedAt:       time.Now(),
		Tariff:         tariff,
	}

	// Biniş 1: 10 TL - 20 TL = -10 TL. Tolerans dahilinde geçişe izin verilmeli (NEGATIVE_TOLERANCE)
	ev, err := syncSvc.IngestOfflineEvent(ctx, req)
	if err != nil {
		t.Fatalf("Toleranslı biniş reddedilmemeliydi: %v", err)
	}
	if ev.Status != domain.EventStatusNegativeTolerance {
		t.Fatalf("Durum NEGATIVE_TOLERANCE olmalıydı, alınan: %s", ev.Status)
	}

	bal, _ := led.GetCardBalance(ctx, cardID)
	if bal != -1000 {
		t.Fatalf("Bakiye -1000 kuruş olmalı, alınan: %d", bal)
	}
}

func TestSyncService_BlacklistedCardRejection(t *testing.T) {
	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	led := ledger.NewMemoryLedger()

	cardID := "CARD_BLACKLISTED"
	led.RegisterCard(domain.Card{
		ID:            cardID,
		CardType:      domain.CardTypeTam,
		BalanceKurus:  10000,
		IsBlacklisted: true,
	})

	tariff := domain.Tariff{BaseFareKurus: 2000}
	syncSvc := NewSyncService(pEngine, led, 5000)
	syncSvc.RegisterCard(domain.Card{
		ID:            cardID,
		CardType:      domain.CardTypeTam,
		BalanceKurus:  10000,
		IsBlacklisted: true,
	})

	req := IngestEventRequest{
		IdempotencyKey: "IDEMP_BLK_1",
		ValidatorID:    "VAL_BUS_303",
		SequenceNo:     1,
		CardID:         cardID,
		LineID:         "LINE_30",
		StationID:      "STAT_9",
		Direction:      1,
		TappedAt:       time.Now(),
		Tariff:         tariff,
	}

	ev, err := syncSvc.IngestOfflineEvent(ctx, req)
	if err != nil {
		t.Fatalf("Kara liste senkronizasyonunda hata dönmemeli, REJECTED durum dönmeli: %v", err)
	}
	if ev.Status != domain.EventStatusRejected {
		t.Fatalf("Durum REJECTED olmalıydı, alınan: %s", ev.Status)
	}
}
