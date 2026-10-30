package ledger

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func TestLedger_BalanceEqualsSumOfEntries(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_LEDGER_TEST_1"
	svc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 0,
	})

	// 1. Dolum: +100 TL (10000 kuruş)
	_, err := svc.Recharge(ctx, cardID, 10000, "REF_DOLUM_1", "Kiosk dolumu")
	if err != nil {
		t.Fatalf("Dolum başarısız: %v", err)
	}

	// 2. Biniş: -20 TL (2000 kuruş)
	_, err = svc.ProcessTap(ctx, cardID, 2000, "VAL_TAP_1", "Otobüs binişi", 0)
	if err != nil {
		t.Fatalf("Biniş başarısız: %v", err)
	}

	// 3. Aktarma Binişi: -10 TL (1000 kuruş)
	_, err = svc.ProcessTap(ctx, cardID, 1000, "VAL_TAP_2", "Tramvay aktarması", 0)
	if err != nil {
		t.Fatalf("Aktarma binişi başarısız: %v", err)
	}

	// 4. İade: +5 TL (500 kuruş)
	_, err = svc.Refund(ctx, cardID, 500, "REF_IADE_1", "Gecikme iadesi")
	if err != nil {
		t.Fatalf("İade başarısız: %v", err)
	}

	// 5. Storno Düzeltme: +10 TL (1000 kuruş)
	_, err = svc.StornoCorrection(ctx, cardID, 1000, "REF_STORNO_1", "Hatalı çekim düzeltmesi")
	if err != nil {
		t.Fatalf("Storno başarısız: %v", err)
	}

	// Bakiye doğrulaması: 10000 - 2000 - 1000 + 500 + 1000 = 8500 kuruş (85.00 TL)
	currentBalance, err := svc.GetCardBalance(ctx, cardID)
	if err != nil {
		t.Fatalf("Bakiye okuma hatası: %v", err)
	}
	if currentBalance != 8500 {
		t.Fatalf("Beklenen güncel bakiye 8500 kuruş, bulunan: %d", currentBalance)
	}

	// Tüm defter hareketlerini topla: SUM(amount_kurus) == currentBalance olmalı!
	entries, err := svc.GetCardLedger(ctx, cardID)
	if err != nil {
		t.Fatalf("Defter okuma hatası: %v", err)
	}

	var sumKurus domain.Kurus
	for _, e := range entries {
		sumKurus += e.AmountKurus
	}

	if sumKurus != currentBalance {
		t.Fatalf("Defter hareketleri toplamı (%d kuruş) ile kart bakiyesi (%d kuruş) uyuşmuyor!",
			sumKurus, currentBalance)
	}
}

func TestLedger_ConcurrentTapsRaceCondition(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_CONCURRENT_RACE"
	initialBalance := domain.Kurus(250000) // 2500.00 TL

	svc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: initialBalance,
	})

	numGoroutines := 100
	farePerTap := domain.Kurus(2000) // Her okumada 20.00 TL (Toplam 2000 TL harcanacak)

	var wg sync.WaitGroup
	wg.Add(numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(idx int) {
			defer wg.Done()
			refID := fmt.Sprintf("CONCURRENT_VAL_%d", idx)
			_, err := svc.ProcessTap(ctx, cardID, farePerTap, refID, "Eşzamanlı okuma", 0)
			if err != nil {
				t.Errorf("Eşzamanlı binişte hata: %v", err)
			}
		}(i)
	}

	wg.Wait()

	// 1000 TL - (100 * 20 TL) = -1000 TL = 0 TL (100000 - 200000)
	expectedFinalBalance := initialBalance - (domain.Kurus(numGoroutines) * farePerTap)
	finalBalance, err := svc.GetCardBalance(ctx, cardID)
	if err != nil {
		t.Fatalf("Bakiye okuma hatası: %v", err)
	}

	if finalBalance != expectedFinalBalance {
		t.Fatalf("Eşzamanlılık yarış durumu (Race Condition) tespit edildi! Beklenen: %d kuruş, Bulunan: %d kuruş",
			expectedFinalBalance, finalBalance)
	}

	// Defter toplamını da kontrol et
	entries, err := svc.GetCardLedger(ctx, cardID)
	if err != nil {
		t.Fatalf("Defter okuma hatası: %v", err)
	}
	if len(entries) != numGoroutines {
		t.Fatalf("Defterdeki hareket sayısı eksik! Beklenen: %d, Bulunan: %d", numGoroutines, len(entries))
	}
}

func TestLedger_NegativeBalanceTolerance(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_NEGATIVE_TOLERANCE"
	svc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 1000, // 10 TL bakiye var
	})

	// 20 TL biniş yapılmak isteniyor. Tolerans limiti: 50 TL (5000 kuruş)
	// Bakiye 10 TL - 20 TL = -10 TL olacak. -10 TL >= -50 TL olduğu için geçiş onaylanmalı!
	_, err := svc.ProcessTap(ctx, cardID, 2000, "VAL_NEG_1", "Negatif toleranslı biniş", 5000)
	if err != nil {
		t.Fatalf("Negatif tolerans dahilinde biniş onaylanmalıydı: %v", err)
	}

	bal, _ := svc.GetCardBalance(ctx, cardID)
	if bal != -1000 {
		t.Fatalf("Beklenen negatif bakiye -1000 kuruş, bulunan: %d", bal)
	}

	// Şimdi bakiye -10 TL. Bir 50 TL biniş daha istenirse: -10 - 50 = -60 TL (< -50 TL limit). Reddedilmeli!
	_, err = svc.ProcessTap(ctx, cardID, 5000, "VAL_NEG_2", "Limit aşımı biniş", 5000)
	if err != ErrInsufficientBalance {
		t.Fatalf("Limit aşımında yetersiz bakiye hatası bekleniyordu, alınan: %v", err)
	}
}

func TestLedger_EdgeCasesAndValidation(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_EDGE_1"
	svc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	})

	// 1. Negatif / sıfır dolum tutarı
	_, err := svc.Recharge(ctx, cardID, -500, "REF_BAD_1", "Hatalı dolum")
	if err != ErrInvalidAmount {
		t.Fatalf("Negatif dolumda ErrInvalidAmount bekleniyordu, alınan: %v", err)
	}

	// 2. Negatif iade tutarı
	_, err = svc.Refund(ctx, cardID, 0, "REF_BAD_2", "Sıfır iade")
	if err != ErrInvalidAmount {
		t.Fatalf("Sıfır iadede ErrInvalidAmount bekleniyordu, alınan: %v", err)
	}

	// 3. Sıfır storno tutarı
	_, err = svc.StornoCorrection(ctx, cardID, 0, "REF_BAD_3", "Sıfır storno")
	if err != ErrInvalidAmount {
		t.Fatalf("Sıfır stornoda ErrInvalidAmount bekleniyordu, alınan: %v", err)
	}

	// Ücretsiz kart ve aylık tavanı dolmuş kart sıfır ücretle binebilir.
	entry, err := svc.ProcessTap(ctx, cardID, 0, "REF_FREE_1", "Ücretsiz kart binişi", 0)
	if err != nil || entry == nil || entry.AmountKurus != 0 || entry.BalanceAfterKurus != 10000 {
		t.Fatalf("Sıfır ücretli biniş bakiyeyi koruyarak kaydedilmeliydi: entry=%+v err=%v", entry, err)
	}
	_, err = svc.ProcessTap(ctx, cardID, -1, "REF_BAD_4", "Negatif ücret", 0)
	if err != ErrInvalidAmount {
		t.Fatalf("Negatif binişte ErrInvalidAmount bekleniyordu, alınan: %v", err)
	}

	// 5. Olmayan kart
	_, err = svc.GetCardBalance(ctx, "NON_EXISTENT_CARD")
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("Olmayan kart bakiyesinde ErrCardNotFound bekleniyordu, alınan: %v", err)
	}

	_, err = svc.GetCardLedger(ctx, "NON_EXISTENT_CARD")
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("Olmayan kart defterinde ErrCardNotFound bekleniyordu, alınan: %v", err)
	}

	_, err = svc.Recharge(ctx, "NON_EXISTENT_CARD", 1000, "REF_NO", "Dolum")
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("Olmayan kart dolumunda ErrCardNotFound bekleniyordu, alınan: %v", err)
	}

	_, err = svc.Refund(ctx, "NON_EXISTENT_CARD", 1000, "REF_NO", "İade")
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("Olmayan kart iadesinde ErrCardNotFound bekleniyordu, alınan: %v", err)
	}

	_, err = svc.StornoCorrection(ctx, "NON_EXISTENT_CARD", 1000, "REF_NO", "Storno")
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("Olmayan kart stornosunda ErrCardNotFound bekleniyordu, alınan: %v", err)
	}

	_, err = svc.ProcessTap(ctx, "NON_EXISTENT_CARD", 1000, "REF_NO", "Biniş", 0)
	if !errors.Is(err, ErrCardNotFound) {
		t.Fatalf("Olmayan kart binişinde ErrCardNotFound bekleniyordu, alınan: %v", err)
	}
}

func TestLedger_BlacklistAndIdempotency(t *testing.T) {
	ctx := context.Background()
	svc := NewMemoryLedger()

	cardID := "CARD_BLACK_IDEMP"
	svc.RegisterCard(domain.Card{
		ID:            cardID,
		CardType:      domain.CardTypeTam,
		BalanceKurus:  10000,
		IsBlacklisted: true,
	})

	// Kara listedeki kart biniş yapamaz
	_, err := svc.ProcessTap(ctx, cardID, 2000, "VAL_BLK_1", "Kara liste biniş", 0)
	if err != ErrCardBlacklisted {
		t.Fatalf("Kara liste kartta ErrCardBlacklisted bekleniyordu, alınan: %v", err)
	}

	// Normal kart ile Recharge & Tap idempotancy
	normCard := "CARD_IDEMP_TEST"
	svc.RegisterCard(domain.Card{
		ID:           normCard,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 5000,
	})

	// Recharge idempotency
	e1, err := svc.Recharge(ctx, normCard, 1000, "REC_SAME_KEY", "İlk dolum")
	if err != nil {
		t.Fatalf("İlk dolum başarısız: %v", err)
	}
	e2, err := svc.Recharge(ctx, normCard, 1000, "REC_SAME_KEY", "Mükerrer dolum")
	if err != nil {
		t.Fatalf("Mükerrer dolum hata vermemeli: %v", err)
	}
	if e1.ID != e2.ID {
		t.Fatalf("İdempotent dolum aynı girdiyi dönmeli")
	}

	// ProcessTap idempotency
	t1, err := svc.ProcessTap(ctx, normCard, 2000, "TAP_SAME_KEY", "İlk biniş", 0)
	if err != nil {
		t.Fatalf("İlk biniş başarısız: %v", err)
	}
	t2, err := svc.ProcessTap(ctx, normCard, 2000, "TAP_SAME_KEY", "Mükerrer biniş", 0)
	if err != nil {
		t.Fatalf("Mükerrer biniş hata vermemeli: %v", err)
	}
	if t1.ID != t2.ID {
		t.Fatalf("İdempotent biniş aynı girdiyi dönmeli")
	}
}

type mockLedgerRow struct {
	scanFunc func(dest ...any) error
}

func (r *mockLedgerRow) Scan(dest ...any) error {
	if r.scanFunc != nil {
		return r.scanFunc(dest...)
	}
	return nil
}

type mockLedgerTx struct {
	queryRowFunc func(sql string, args ...any) pgx.Row
	execFunc     func(sql string, args ...any) (pgconn.CommandTag, error)
}

func (m *mockLedgerTx) Begin(ctx context.Context) (pgx.Tx, error) { return m, nil }
func (m *mockLedgerTx) Commit(ctx context.Context) error          { return nil }
func (m *mockLedgerTx) Rollback(ctx context.Context) error        { return nil }
func (m *mockLedgerTx) CopyFrom(ctx context.Context, tableName pgx.Identifier, columnNames []string, rowSrc pgx.CopyFromSource) (int64, error) {
	return 0, nil
}
func (m *mockLedgerTx) SendBatch(ctx context.Context, b *pgx.Batch) pgx.BatchResults { return nil }
func (m *mockLedgerTx) LargeObjects() pgx.LargeObjects                               { return pgx.LargeObjects{} }
func (m *mockLedgerTx) Prepare(ctx context.Context, name, sql string) (*pgconn.StatementDescription, error) {
	return nil, nil
}
func (m *mockLedgerTx) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(sql, arguments...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}
func (m *mockLedgerTx) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (m *mockLedgerTx) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(sql, args...)
	}
	return &mockLedgerRow{}
}
func (m *mockLedgerTx) Conn() *pgx.Conn { return nil }

type mockStarter struct {
	tx *mockLedgerTx
}

func (s *mockStarter) Begin(ctx context.Context) (pgx.Tx, error) { return s.tx, nil }
func (s *mockStarter) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	return nil, nil
}
func (s *mockStarter) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	return s.tx.QueryRow(ctx, sql, args...)
}

func TestPostgresLedger_MockOperations(t *testing.T) {
	ctx := context.Background()
	noLedgerEntry := func() pgx.Row {
		return &mockLedgerRow{scanFunc: func(dest ...any) error { return pgx.ErrNoRows }}
	}
	cardLockRow := func(blacklisted bool) pgx.Row {
		return &mockLedgerRow{scanFunc: func(dest ...any) error {
			*(dest[0].(*bool)) = blacklisted
			return nil
		}}
	}
	ledgerBalanceRow := func(balance domain.Kurus) pgx.Row {
		return &mockLedgerRow{scanFunc: func(dest ...any) error {
			*(dest[0].(*domain.Kurus)) = balance
			return nil
		}}
	}

	// 1. Constructor test
	pNil := NewPostgresLedger(nil)
	if pNil == nil {
		t.Fatal("NewPostgresLedger nil dönmemeli")
	}

	tx := &mockLedgerTx{}
	starter := &mockStarter{tx: tx}
	p := NewPostgresLedgerWithStarter(starter)

	// 2. ProcessTap - Normal flow
	tx.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "WHERE reference_id") {
			return noLedgerEntry()
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return cardLockRow(false)
		}
		if strings.Contains(sql, "SUM(amount_kurus)") {
			return ledgerBalanceRow(10000)
		}
		if strings.Contains(sql, "INSERT INTO jw_ledger") {
			return &mockLedgerRow{
				scanFunc: func(dest ...any) error {
					*(dest[0].(*time.Time)) = time.Now()
					return nil
				},
			}
		}
		return noLedgerEntry()
	}

	entry, err := p.ProcessTap(ctx, "CARD_MOCK_1", 2000, "REF_TAP_MOCK", "Mock tap", 0)
	if err != nil {
		t.Fatalf("ProcessTap beklenmeyen hata: %v", err)
	}
	if entry.AmountKurus != -2000 || entry.BalanceAfterKurus != 8000 {
		t.Fatalf("Yanlış bakiye/tutar: %+v", entry)
	}

	// 3. ProcessTap - Blacklist error
	tx.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "WHERE reference_id") {
			return noLedgerEntry()
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return cardLockRow(true)
		}
		if strings.Contains(sql, "SUM(amount_kurus)") {
			return ledgerBalanceRow(10000)
		}
		return noLedgerEntry()
	}
	_, err = p.ProcessTap(ctx, "CARD_MOCK_1", 2000, "REF_TAP_MOCK_2", "Mock tap", 0)
	if err != ErrCardBlacklisted {
		t.Fatalf("ErrCardBlacklisted bekleniyordu, alınan: %v", err)
	}

	// 4. ProcessTap - Insufficient balance error
	tx.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "WHERE reference_id") {
			return noLedgerEntry()
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return cardLockRow(false)
		}
		if strings.Contains(sql, "SUM(amount_kurus)") {
			return ledgerBalanceRow(1000)
		}
		return noLedgerEntry()
	}
	_, err = p.ProcessTap(ctx, "CARD_MOCK_1", 2000, "REF_TAP_MOCK_3", "Mock tap", 0)
	if err != ErrInsufficientBalance {
		t.Fatalf("ErrInsufficientBalance bekleniyordu, alınan: %v", err)
	}

	// 5. Recharge - Normal flow
	tx.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "WHERE reference_id") {
			return noLedgerEntry()
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return cardLockRow(false)
		}
		if strings.Contains(sql, "SUM(amount_kurus)") {
			return ledgerBalanceRow(5000)
		}
		if strings.Contains(sql, "INSERT INTO jw_ledger") {
			return &mockLedgerRow{
				scanFunc: func(dest ...any) error {
					*(dest[0].(*time.Time)) = time.Now()
					return nil
				},
			}
		}
		return noLedgerEntry()
	}
	recEntry, err := p.Recharge(ctx, "CARD_MOCK_1", 5000, "REF_REC_MOCK", "Mock recharge")
	if err != nil {
		t.Fatalf("Recharge beklenmeyen hata: %v", err)
	}
	if recEntry.BalanceAfterKurus != 10000 {
		t.Fatalf("Yanlış recharge bakiye: %d", recEntry.BalanceAfterKurus)
	}

	// 6. Refund
	refEntry, err := p.Refund(ctx, "CARD_MOCK_1", 5000, "REF_REFUND_MOCK", "Mock refund")
	if err != nil {
		t.Fatalf("Refund beklenmeyen hata: %v", err)
	}
	if refEntry == nil {
		t.Fatal("Refund entry nil olmamalı")
	}

	// 7. StornoCorrection
	stornoEntry, err := p.StornoCorrection(ctx, "CARD_MOCK_1", 2000, "REF_STORNO_MOCK", "Mock storno")
	if err != nil {
		t.Fatalf("StornoCorrection beklenmeyen hata: %v", err)
	}
	if stornoEntry == nil {
		t.Fatal("Storno entry nil olmamalı")
	}

	// 8. GetCardBalance
	tx.queryRowFunc = func(sql string, args ...any) pgx.Row {
		return &mockLedgerRow{
			scanFunc: func(dest ...any) error {
				*(dest[0].(*domain.Kurus)) = 15000
				return nil
			},
		}
	}
	bal, err := p.GetCardBalance(ctx, "CARD_MOCK_1")
	if err != nil {
		t.Fatalf("GetCardBalance beklenmeyen hata: %v", err)
	}
	if bal != 15000 {
		t.Fatalf("GetCardBalance yanlış değer: %d", bal)
	}
}
