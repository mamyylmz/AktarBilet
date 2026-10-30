package sync

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
)

func getTestPool(t *testing.T) *pgxpool.Pool {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@127.0.0.1:5432/aktarbilet?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil || pool.Ping(ctx) != nil {
		t.Skip("PostgreSQL test ortamına erişilemedi, Persistent test atlanıyor")
		return nil
	}
	return pool
}

func TestSyncService_LivePersistentIngest(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	ledSvc := ledger.NewPostgresLedger(pool)
	syncSvc := NewSyncService(pEngine, ledSvc, 5000)
	syncSvc.SetPool(pool)

	nowNano := time.Now().UnixNano()
	cardID := "CARD_SYNC_PG_" + uuid.NewString()
	_, err := pool.Exec(ctx, `
		INSERT INTO jw_cards (id, card_type, balance_kurus, is_blacklisted, created_at, updated_at)
		VALUES ($1, 'tam', 0, false, NOW(), NOW())
	`, cardID)
	if err != nil {
		t.Fatalf("Persistent sync test kartı oluşturulamadı: %v", err)
	}
	if _, err = ledSvc.Recharge(ctx, cardID, 20000, "SYNC_OPEN_"+uuid.NewString(), "Persistent sync test açılış bakiyesi"); err != nil {
		t.Fatalf("Persistent sync test açılış bakiyesi ledger'a yazılamadı: %v", err)
	}

	t0 := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	key1 := fmt.Sprintf("SYNC_KEY_%d", nowNano)
	seq1 := nowNano % 100000000
	req1 := IngestEventRequest{
		IdempotencyKey: key1,
		ValidatorID:    "VAL_500T_01",
		SequenceNo:     seq1,
		CardID:         cardID,
		LineID:         "LINE_5",
		StationID:      "STAT_ESHOT_10177",
		Direction:      1,
		TappedAt:       t0,
	}

	// 1. İlk Persistent Tap
	ev1, err := syncSvc.IngestOfflineEvent(ctx, req1)
	if err != nil {
		t.Fatalf("IngestOfflineEvent başarısız: %v", err)
	}
	if ev1.Status != domain.EventStatusProcessed {
		t.Fatalf("Olay durumu PROCESSED olmalıydı, alınan: %s", ev1.Status)
	}
	if ev1.FareChargedKurus != 2000 {
		t.Fatalf("Tam ücret 2000 kuruş olmalıydı, alınan: %d", ev1.FareChargedKurus)
	}

	// 2. İdempotansi Kontrolü
	ev1Repeat, err := syncSvc.IngestOfflineEvent(ctx, req1)
	if err != nil || ev1Repeat.ID != ev1.ID {
		t.Fatalf("İdempotent olay aynı ID'ye sahip olmalı: %+v", ev1Repeat)
	}

	// 3. İkinci Biniş (Aktarma): 15 dk sonra farklı hat
	key2 := fmt.Sprintf("SYNC_KEY_2_%d", nowNano)
	req2 := IngestEventRequest{
		IdempotencyKey: key2,
		ValidatorID:    "VAL_14ES_01",
		SequenceNo:     seq1 + 1,
		CardID:         cardID,
		LineID:         "LINE_10",
		StationID:      "STAT_ESHOT_10145",
		Direction:      1,
		TappedAt:       t0.Add(15 * time.Minute),
	}
	ev2, err := syncSvc.IngestOfflineEvent(ctx, req2)
	if err != nil {
		t.Fatalf("İkinci biniş başarısız: %v", err)
	}
	if !ev2.IsTransfer || ev2.FareChargedKurus != 1000 {
		t.Fatalf("İkinci biniş indirimli aktarma olmalıydı (%d kuruş, transfer: %v)", ev2.FareChargedKurus, ev2.IsTransfer)
	}
}
