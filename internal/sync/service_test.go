package sync

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

	// 3. Aynı anahtar, farklı kart/parametre -> ErrIdempotencyConflict dönmeli
	conflictingReq := req
	conflictingReq.CardID = "CARD_DIFFERENT"
	_, err = syncSvc.IngestOfflineEvent(ctx, conflictingReq)
	if !errors.Is(err, ErrIdempotencyConflict) {
		t.Fatalf("Beklenen ErrIdempotencyConflict hatası alınamadı, alınan: %v", err)
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

func TestSyncService_IngestBatch_ChronologicalSort(t *testing.T) {
	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	led := ledger.NewMemoryLedger()

	cardID := "CARD_BATCH_SORT"
	led.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	})

	tariff := domain.Tariff{
		ID:                   "tariff_istanbul",
		BaseFareKurus:        2000,
		Transfer1DiscountPct: 50,
	}

	syncSvc := NewSyncService(pEngine, led, 5000)
	syncSvc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
	})

	t0 := time.Date(2026, 10, 24, 14, 0, 0, 0, time.UTC)
	// İki olay: ev1 saat 14:00 (ilk biniş 20 TL), ev2 saat 14:15 (aktarma 10 TL)
	// Batch içine KASTEN ters sırada [ev2, ev1] koyuyoruz:
	batch := []IngestEventRequest{
		{
			IdempotencyKey: "BATCH_EV_2",
			ValidatorID:    "VAL_BUS_2",
			SequenceNo:     2,
			CardID:         cardID,
			LineID:         "LINE_14ES",
			StationID:      "STAT_KADIKOY",
			Direction:      1,
			TappedAt:       t0.Add(15 * time.Minute),
			Tariff:         tariff,
		},
		{
			IdempotencyKey: "BATCH_EV_1",
			ValidatorID:    "VAL_BUS_1",
			SequenceNo:     1,
			CardID:         cardID,
			LineID:         "LINE_500T",
			StationID:      "STAT_AYRILIK",
			Direction:      1,
			TappedAt:       t0,
			Tariff:         tariff,
		},
	}

	results, err := syncSvc.IngestBatch(ctx, batch)
	if err != nil {
		t.Fatalf("IngestBatch başarısız: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("2 olay bekleniyordu, alınan: %d", len(results))
	}

	// Sıralama sayesinde ilk işlenen ev1 (tam biniş 2000 kuruş) olmalı
	if results[0].FareChargedKurus != 2000 || results[0].IsTransfer {
		t.Fatalf("İlk olay tam biniş olmalıydı: %+v", results[0])
	}
	// İkinci işlenen ev2 (aktarma 1000 kuruş) olmalı
	if results[1].FareChargedKurus != 1000 || !results[1].IsTransfer {
		t.Fatalf("İkinci olay indirimli aktarma olmalıydı: %+v", results[1])
	}
}

func TestSyncService_ExceedNegativeLimit_Blacklist(t *testing.T) {
	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	led := ledger.NewMemoryLedger()

	cardID := "CARD_EXCEED_LIMIT"
	// Bakiye -4000 kuruş. Negatif limit 5000 kuruş (50 TL).
	led.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: -4000,
	})

	tariff := domain.Tariff{BaseFareKurus: 2000}
	syncSvc := NewSyncService(pEngine, led, 5000)
	syncSvc.RegisterCard(domain.Card{
		ID:           cardID,
		CardType:     domain.CardTypeTam,
		BalanceKurus: -4000,
	})

	req := IngestEventRequest{
		IdempotencyKey: "IDEMP_EXCEED_1",
		ValidatorID:    "VAL_BUS_999",
		SequenceNo:     1,
		CardID:         cardID,
		LineID:         "LINE_500T",
		StationID:      "STAT_1",
		Direction:      1,
		TappedAt:       time.Now(),
		Tariff:         tariff,
	}

	ev, err := syncSvc.IngestOfflineEvent(ctx, req)
	if err != nil {
		t.Fatalf("Hata dönmemeli, REJECTED dönmeli: %v", err)
	}
	if ev.Status != domain.EventStatusRejected {
		t.Fatalf("Limit aşımında REJECTED bekleniyordu, alınan: %s", ev.Status)
	}
}

func TestSyncService_Errors(t *testing.T) {
	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	led := ledger.NewMemoryLedger()
	syncSvc := NewSyncService(pEngine, led, 5000)
	syncSvc.SetPool(nil)

	// Olmayan kart
	_, err := syncSvc.IngestOfflineEvent(ctx, IngestEventRequest{
		CardID: "CARD_NOT_FOUND",
	})
	if err == nil {
		t.Fatalf("Olmayan kartta hata bekleniyordu")
	}
}

type mockSyncRow struct {
	scanFunc func(dest ...any) error
}

func (m *mockSyncRow) Scan(dest ...any) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	return nil
}

type mockSyncRows struct {
	items []pricing.TripHistoryItem
	idx   int
}

func (m *mockSyncRows) Close()                                       {}
func (m *mockSyncRows) Err() error                                   { return nil }
func (m *mockSyncRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockSyncRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockSyncRows) Next() bool {
	if m.idx < len(m.items) {
		m.idx++
		return true
	}
	return false
}
func (m *mockSyncRows) Scan(dest ...any) error {
	if m.idx == 0 || m.idx > len(m.items) {
		return errors.New("no row")
	}
	it := m.items[m.idx-1]
	if len(dest) >= 6 {
		*dest[0].(*string) = it.LineID
		*dest[1].(*string) = it.StationID
		*dest[2].(*int) = it.Direction
		*dest[3].(*time.Time) = it.TappedAt
		*dest[4].(*bool) = it.IsTransfer
		*dest[5].(*int) = it.TransferLevel
	}
	return nil
}
func (m *mockSyncRows) Values() ([]any, error) { return nil, nil }
func (m *mockSyncRows) RawValues() [][]byte    { return nil }
func (m *mockSyncRows) Conn() *pgx.Conn        { return nil }

type mockSyncPool struct {
	queryRowFunc func(sql string, args ...any) pgx.Row
	queryFunc    func(sql string, args ...any) (pgx.Rows, error)
	execFunc     func(sql string, args ...any) (pgconn.CommandTag, error)
}

func (m *mockSyncPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(sql, arguments...)
	}
	return pgconn.NewCommandTag("INSERT 1"), nil
}

func (m *mockSyncPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(sql, args...)
	}
	return &mockSyncRows{}, nil
}

func (m *mockSyncPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(sql, args...)
	}
	return &mockSyncRow{}
}

func TestSyncService_PersistentPipeline(t *testing.T) {
	ctx := context.Background()
	pEngine := pricing.NewPricingEngine()
	led := ledger.NewMemoryLedger()
	syncSvc := NewSyncService(pEngine, led, 5000)

	pool := &mockSyncPool{}
	syncSvc.SetDBPool(pool)

	// Case 1: Idempotent event found
	pool.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "FROM jw_validator_events") {
			return &mockSyncRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*uuid.UUID) = uuid.New()
					*dest[1].(*string) = "IDEM_1"
					*dest[2].(*string) = "VAL_1"
					*dest[3].(*int64) = 1
					*dest[4].(*string) = "CARD_1"
					*dest[5].(*string) = "LINE_1"
					*dest[6].(*string) = "STAT_1"
					*dest[7].(*int) = 1
					*dest[8].(*time.Time) = time.Now()
					*dest[9].(*time.Time) = time.Now()
					*dest[10].(*domain.Kurus) = 2000
					*dest[11].(*bool) = false
					*dest[12].(*int) = 0
					*dest[13].(*domain.ValidatorEventStatus) = domain.EventStatusProcessed
					str := "Rejection reason"
					*dest[14].(**string) = &str
					return nil
				},
			}
		}
		return &mockSyncRow{scanFunc: func(dest ...any) error { return errors.New("not found") }}
	}

	ev, err := syncSvc.IngestOfflineEvent(ctx, IngestEventRequest{
		IdempotencyKey: "IDEM_1",
		ValidatorID:    "VAL_1",
		SequenceNo:     1,
		CardID:         "CARD_1",
	})
	if err != nil || ev.IdempotencyKey != "IDEM_1" {
		t.Fatalf("Idempotent event başarısız: %v", err)
	}

	// Case 2: Card Not Found in DB
	pool.queryRowFunc = func(sql string, args ...any) pgx.Row {
		return &mockSyncRow{
			scanFunc: func(dest ...any) error {
				return errors.New("kart bulunamadı")
			},
		}
	}
	_, err = syncSvc.IngestOfflineEvent(ctx, IngestEventRequest{
		IdempotencyKey: "IDEM_NEW",
		CardID:         "CARD_NOT_EXIST",
	})
	if err == nil {
		t.Fatal("Olmayan kartta hata bekleniyordu")
	}

	// Case 3: Card Blacklisted
	pool.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "FROM jw_validator_events") {
			return &mockSyncRow{scanFunc: func(dest ...any) error { return errors.New("no event") }}
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return &mockSyncRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*string) = "CARD_BLK"
					*dest[1].(*domain.CardType) = domain.CardTypeTam
					*dest[2].(*domain.Kurus) = 1000
					*dest[3].(*bool) = true // IsBlacklisted
					*dest[4].(*int) = 0
					*dest[5].(*domain.Kurus) = 0
					return nil
				},
			}
		}
		return &mockSyncRow{}
	}
	ev, err = syncSvc.IngestOfflineEvent(ctx, IngestEventRequest{
		IdempotencyKey: "IDEM_BLK",
		CardID:         "CARD_BLK",
	})
	if err != nil || ev.Status != domain.EventStatusRejected {
		t.Fatalf("Kara listedeki kart reddedilmeli, alınan: %v, status: %v", err, ev)
	}

	// Case 4: Normal Tap with Tariff Lookup and Trip History
	led.RegisterCard(domain.Card{ID: "CARD_OK", BalanceKurus: 10000, CardType: domain.CardTypeTam})
	pool.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "FROM jw_validator_events") {
			return &mockSyncRow{scanFunc: func(dest ...any) error { return errors.New("no event") }}
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return &mockSyncRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*string) = "CARD_OK"
					*dest[1].(*domain.CardType) = domain.CardTypeTam
					*dest[2].(*domain.Kurus) = 10000
					*dest[3].(*bool) = false
					*dest[4].(*int) = 0
					*dest[5].(*domain.Kurus) = 0
					return nil
				},
			}
		}
		if strings.Contains(sql, "FROM jw_tariffs") {
			return &mockSyncRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*string) = "TAR_1"
					*dest[1].(*string) = "Tarife"
					*dest[2].(*domain.LineMode) = domain.ModeBus
					*dest[3].(*domain.Kurus) = 2000
					*dest[4].(*domain.Kurus) = 1000
					*dest[5].(*domain.Kurus) = 0
					*dest[6].(*int) = 50
					*dest[7].(*int) = 75
					*dest[8].(*int) = 50
					*dest[9].(*int) = 10
					*dest[10].(*domain.Kurus) = 150000
					*dest[11].(*bool) = true
					return nil
				},
			}
		}
		return &mockSyncRow{}
	}
	pool.queryFunc = func(sql string, args ...any) (pgx.Rows, error) {
		return &mockSyncRows{
			items: []pricing.TripHistoryItem{
				{
					LineID:        "LINE_OTHER",
					StationID:     "STAT_0",
					Direction:     1,
					TappedAt:      time.Now().Add(-10 * time.Minute),
					IsTransfer:    false,
					TransferLevel: 0,
				},
			},
		}, nil
	}

	ev, err = syncSvc.IngestOfflineEvent(ctx, IngestEventRequest{
		IdempotencyKey: "IDEM_OK",
		ValidatorID:    "VAL_1",
		SequenceNo:     2,
		CardID:         "CARD_OK",
		LineID:         "LINE_NEW",
		StationID:      "STAT_1",
		Direction:      1,
		TappedAt:       time.Now(),
	})
	if err != nil || ev.Status != domain.EventStatusProcessed {
		t.Fatalf("Normal biniş PROCESSED olmalı: %v", err)
	}

	// Case 5: Negative tolerance tap (balance drops below 0 but above -negativeLimit)
	led.RegisterCard(domain.Card{ID: "CARD_NEG", BalanceKurus: 500, CardType: domain.CardTypeTam})
	pool.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "FROM jw_validator_events") {
			return &mockSyncRow{scanFunc: func(dest ...any) error { return errors.New("no event") }}
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return &mockSyncRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*string) = "CARD_NEG"
					*dest[1].(*domain.CardType) = domain.CardTypeTam
					*dest[2].(*domain.Kurus) = 500
					*dest[3].(*bool) = false
					*dest[4].(*int) = 0
					*dest[5].(*domain.Kurus) = 0
					return nil
				},
			}
		}
		return &mockSyncRow{}
	}
	ev, err = syncSvc.IngestOfflineEvent(ctx, IngestEventRequest{
		IdempotencyKey: "IDEM_NEG",
		ValidatorID:    "VAL_1",
		SequenceNo:     3,
		CardID:         "CARD_NEG",
		LineID:         "LINE_1",
		StationID:      "STAT_1",
		Direction:      1,
		TappedAt:       time.Now(),
		Tariff:         domain.Tariff{BaseFareKurus: 2000},
	})
	if err != nil || ev.Status != domain.EventStatusNegativeTolerance {
		t.Fatalf("Negatif tolerans durumu bekleniyordu, alınan: %v", ev)
	}

	// Case 6: Insufficient balance beyond negative limit -> blacklisted
	led.RegisterCard(domain.Card{ID: "CARD_FAIL", BalanceKurus: -5000, CardType: domain.CardTypeTam})
	pool.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "FROM jw_validator_events") {
			return &mockSyncRow{scanFunc: func(dest ...any) error { return errors.New("no event") }}
		}
		if strings.Contains(sql, "FROM jw_cards") {
			return &mockSyncRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*string) = "CARD_FAIL"
					*dest[1].(*domain.CardType) = domain.CardTypeTam
					*dest[2].(*domain.Kurus) = -5000
					*dest[3].(*bool) = false
					*dest[4].(*int) = 0
					*dest[5].(*domain.Kurus) = 0
					return nil
				},
			}
		}
		return &mockSyncRow{}
	}
	ev, err = syncSvc.IngestOfflineEvent(ctx, IngestEventRequest{
		IdempotencyKey: "IDEM_FAIL",
		ValidatorID:    "VAL_1",
		SequenceNo:     4,
		CardID:         "CARD_FAIL",
		LineID:         "LINE_1",
		StationID:      "STAT_1",
		Direction:      1,
		TappedAt:       time.Now(),
		Tariff:         domain.Tariff{BaseFareKurus: 2000},
	})
	if err != nil || ev.Status != domain.EventStatusRejected {
		t.Fatalf("Limit aşımında REJECTED bekleniyordu: %v", err)
	}

	// Case 7: IngestBatch with persistence
	batchRes, err := syncSvc.IngestBatch(ctx, []IngestEventRequest{
		{
			IdempotencyKey: "IDEM_BATCH_1",
			ValidatorID:    "VAL_1",
			SequenceNo:     10,
			CardID:         "CARD_OK",
			TappedAt:       time.Now(),
		},
	})
	if err != nil || len(batchRes) != 1 {
		t.Fatalf("IngestBatch başarısız: %v", err)
	}
}
