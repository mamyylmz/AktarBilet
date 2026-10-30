package ledger

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
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
		t.Skip("PostgreSQL test ortamına erişilemedi, PostgresLedger testi atlanıyor")
		return nil
	}
	return pool
}

func TestPostgresLedger_LiveDBOperations(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	pl := NewPostgresLedger(pool)

	cardID := "CARD_PG_TEST_" + uuid.NewString()
	// Test kartını ve başlangıç bakiyesini ledger üzerinden oluştur.
	_, err := pool.Exec(ctx, `
		INSERT INTO jw_cards (id, card_type, balance_kurus, is_blacklisted, created_at, updated_at)
		VALUES ($1, 'tam', 0, false, NOW(), NOW())
	`, cardID)
	if err != nil {
		t.Fatalf("Test kartı oluşturulamadı: %v", err)
	}
	if _, err = pl.Recharge(ctx, cardID, 10000, "PG_OPEN_"+uuid.NewString(), "PG test açılış bakiyesi"); err != nil {
		t.Fatalf("Test açılış bakiyesi ledger'a yazılamadı: %v", err)
	}

	// 1. GetCardBalance
	bal, err := pl.GetCardBalance(ctx, cardID)
	if err != nil {
		t.Fatalf("GetCardBalance başarısız: %v", err)
	}
	if bal != 10000 {
		t.Fatalf("Başlangıç bakiyesi 10000 kuruş olmalı, alınan: %d", bal)
	}

	// 2. Recharge
	refDolum := fmt.Sprintf("PG_DOLUM_%d", time.Now().UnixNano())
	entry, err := pl.Recharge(ctx, cardID, 5000, refDolum, "PG Test Dolumu")
	if err != nil {
		t.Fatalf("Recharge başarısız: %v", err)
	}
	if entry.AmountKurus != 5000 || entry.BalanceAfterKurus != 15000 {
		t.Fatalf("Recharge bakiye hatası: %+v", entry)
	}

	// Recharge İdempotansi
	entry2, err := pl.Recharge(ctx, cardID, 5000, refDolum, "Tekrarlı Dolum")
	if err != nil || entry2.ID != entry.ID {
		t.Fatalf("İdempotent recharge aynı kaydı dönmeli")
	}

	// 3. ProcessTap
	refTap := fmt.Sprintf("PG_TAP_%d", time.Now().UnixNano())
	tapEntry, err := pl.ProcessTap(ctx, cardID, 2000, refTap, "PG Test Biniş", 5000)
	if err != nil {
		t.Fatalf("ProcessTap başarısız: %v", err)
	}
	if tapEntry.AmountKurus != -2000 || tapEntry.BalanceAfterKurus != 13000 {
		t.Fatalf("ProcessTap bakiye hatası: %+v", tapEntry)
	}

	// ProcessTap İdempotansi
	tapEntry2, err := pl.ProcessTap(ctx, cardID, 2000, refTap, "Tekrarlı Biniş", 5000)
	if err != nil || tapEntry2.ID != tapEntry.ID {
		t.Fatalf("İdempotent tap aynı kaydı dönmeli")
	}

	// 4. Refund
	refRefund := fmt.Sprintf("PG_REFUND_%d", time.Now().UnixNano())
	refundEntry, err := pl.Refund(ctx, cardID, 1000, refRefund, "İade Testi")
	if err != nil {
		t.Fatalf("Refund başarısız: %v", err)
	}
	if refundEntry.AmountKurus != 1000 || refundEntry.BalanceAfterKurus != 14000 {
		t.Fatalf("Refund bakiye hatası: %+v", refundEntry)
	}

	// 5. StornoCorrection
	refStorno := fmt.Sprintf("PG_STORNO_%d", time.Now().UnixNano())
	stornoEntry, err := pl.StornoCorrection(ctx, cardID, -500, refStorno, "Storno Testi")
	if err != nil {
		t.Fatalf("StornoCorrection başarısız: %v", err)
	}
	if stornoEntry.AmountKurus != -500 || stornoEntry.BalanceAfterKurus != 13500 {
		t.Fatalf("Storno bakiye hatası: %+v", stornoEntry)
	}

	// 6. GetCardLedger
	entries, err := pl.GetCardLedger(ctx, cardID)
	if err != nil {
		t.Fatalf("GetCardLedger başarısız: %v", err)
	}
	if len(entries) < 4 {
		t.Fatalf("En az 4 hareket listelenmeli, bulunan: %d", len(entries))
	}
}

func TestPostgresLedger_InsufficientBalanceAndBlacklist(t *testing.T) {
	pool := getTestPool(t)
	defer pool.Close()

	ctx := context.Background()
	pl := NewPostgresLedger(pool)

	cardID := "CARD_PG_LIMIT_" + uuid.NewString()
	_, err := pool.Exec(ctx, `
		INSERT INTO jw_cards (id, card_type, balance_kurus, is_blacklisted, created_at, updated_at)
		VALUES ($1, 'tam', 0, false, NOW(), NOW())
	`, cardID)
	if err != nil {
		t.Fatalf("Limit test kartı oluşturulamadı: %v", err)
	}

	// Bakiye 0. 50 TL tolerans limiti var. 60 TL biniş reddedilmeli
	refBad := fmt.Sprintf("PG_BAD_%d", time.Now().UnixNano())
	_, err = pl.ProcessTap(ctx, cardID, 6000, refBad, "Aşırı biniş", 5000)
	if err != ErrInsufficientBalance {
		t.Fatalf("Limit aşımında ErrInsufficientBalance bekleniyordu, alınan: %v", err)
	}

	// Kara listeye al
	_, _ = pool.Exec(ctx, "UPDATE jw_cards SET is_blacklisted = true WHERE id = $1", cardID)
	refBlk := fmt.Sprintf("PG_BLK_%d", time.Now().UnixNano())
	_, err = pl.ProcessTap(ctx, cardID, 1000, refBlk, "Kara liste biniş", 5000)
	if err != ErrCardBlacklisted {
		t.Fatalf("Kara listede ErrCardBlacklisted bekleniyordu, alınan: %v", err)
	}
}
