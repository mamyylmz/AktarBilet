package ledger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

var (
	ErrInsufficientBalance = errors.New("yetersiz bakiye")
	ErrCardBlacklisted     = errors.New("kart kara listede")
	ErrInvalidAmount       = errors.New("geçersiz tutar")
	ErrCardNotFound        = errors.New("kart bulunamadı")
	ErrIdempotencyConflict = errors.New("idempotency anahtarı çakışması: farklı parametrelerle daha önce kullanılmış")
)

type LedgerService interface {
	ProcessTap(ctx context.Context, cardID string, fare domain.Kurus, refID, desc string, allowNegativeLimit domain.Kurus) (*domain.LedgerEntry, error)
	Recharge(ctx context.Context, cardID string, amount domain.Kurus, refID, desc string) (*domain.LedgerEntry, error)
	Refund(ctx context.Context, cardID string, amount domain.Kurus, refID, desc string) (*domain.LedgerEntry, error)
	StornoCorrection(ctx context.Context, cardID string, amount domain.Kurus, refID, desc string) (*domain.LedgerEntry, error)
	GetCardBalance(ctx context.Context, cardID string) (domain.Kurus, error)
	GetCardLedger(ctx context.Context, cardID string) ([]domain.LedgerEntry, error)
}

// MemoryLedger provides thread-safe, card-level serialized ledger operations for unit/concurrency tests
type MemoryLedger struct {
	mu        sync.Mutex
	cardLocks map[string]*sync.Mutex
	cards     map[string]*domain.Card
	entries   map[string][]domain.LedgerEntry
}

func NewMemoryLedger() *MemoryLedger {
	return &MemoryLedger{
		cardLocks: make(map[string]*sync.Mutex),
		cards:     make(map[string]*domain.Card),
		entries:   make(map[string][]domain.LedgerEntry),
	}
}

func (m *MemoryLedger) getCardLock(cardID string) *sync.Mutex {
	m.mu.Lock()
	defer m.mu.Unlock()
	lock, exists := m.cardLocks[cardID]
	if !exists {
		lock = &sync.Mutex{}
		m.cardLocks[cardID] = lock
	}
	return lock
}

func (m *MemoryLedger) RegisterCard(card domain.Card) {
	m.mu.Lock()
	defer m.mu.Unlock()
	c := card
	m.cards[card.ID] = &c
}

func (m *MemoryLedger) ProcessTap(
	ctx context.Context,
	cardID string,
	fare domain.Kurus,
	refID, desc string,
	allowNegativeLimit domain.Kurus,
) (*domain.LedgerEntry, error) {
	if fare < 0 {
		return nil, ErrInvalidAmount
	}

	// Kart bazlı serileştirme (SELECT ... FOR UPDATE eşdeğeri kilit)
	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	// İdempotansi Kontrolü: Aynı refID ile biniş daha önce işlenmişse mükerrer düşme!
	if refID != "" {
		for _, e := range m.entries[cardID] {
			if e.ReferenceID == refID {
				return &e, nil
			}
		}
	}

	card, ok := m.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCardNotFound, cardID)
	}

	if card.IsBlacklisted {
		return nil, ErrCardBlacklisted
	}

	newBalance := card.BalanceKurus - fare
	// Negatif bakiye toleransı kontrolü (Örn: -50 TL / -5000 kuruşa kadar izin verilebilir)
	if newBalance < -allowNegativeLimit {
		return nil, ErrInsufficientBalance
	}

	card.BalanceKurus = newBalance
	now := time.Now()
	card.LastTapAt = &now

	entry := domain.LedgerEntry{
		ID:                uuid.New(),
		CardID:            cardID,
		EntryType:         domain.LedgerBinis,
		AmountKurus:       -fare,
		BalanceAfterKurus: newBalance,
		ReferenceID:       refID,
		Description:       desc,
		CreatedAt:         now,
	}

	m.entries[cardID] = append(m.entries[cardID], entry)
	return &entry, nil
}

func (m *MemoryLedger) Recharge(
	ctx context.Context,
	cardID string,
	amount domain.Kurus,
	refID, desc string,
) (*domain.LedgerEntry, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	// İdempotansi kontrolü
	if refID != "" {
		for _, e := range m.entries[cardID] {
			if e.ReferenceID == refID {
				if e.AmountKurus != amount || e.CardID != cardID {
					return nil, ErrIdempotencyConflict
				}
				entryCopy := e
				return &entryCopy, nil
			}
		}
	}

	card, ok := m.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCardNotFound, cardID)
	}

	newBalance := card.BalanceKurus + amount
	card.BalanceKurus = newBalance

	entry := domain.LedgerEntry{
		ID:                uuid.New(),
		CardID:            cardID,
		EntryType:         domain.LedgerDolum,
		AmountKurus:       amount,
		BalanceAfterKurus: newBalance,
		ReferenceID:       refID,
		Description:       desc,
		CreatedAt:         time.Now(),
	}

	m.entries[cardID] = append(m.entries[cardID], entry)
	return &entry, nil
}

func (m *MemoryLedger) Refund(
	ctx context.Context,
	cardID string,
	amount domain.Kurus,
	refID, desc string,
) (*domain.LedgerEntry, error) {
	if amount <= 0 {
		return nil, ErrInvalidAmount
	}

	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	card, ok := m.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCardNotFound, cardID)
	}

	newBalance := card.BalanceKurus + amount
	card.BalanceKurus = newBalance

	entry := domain.LedgerEntry{
		ID:                uuid.New(),
		CardID:            cardID,
		EntryType:         domain.LedgerIade,
		AmountKurus:       amount,
		BalanceAfterKurus: newBalance,
		ReferenceID:       refID,
		Description:       desc,
		CreatedAt:         time.Now(),
	}

	m.entries[cardID] = append(m.entries[cardID], entry)
	return &entry, nil
}

func (m *MemoryLedger) StornoCorrection(
	ctx context.Context,
	cardID string,
	amount domain.Kurus,
	refID, desc string,
) (*domain.LedgerEntry, error) {
	if amount == 0 {
		return nil, ErrInvalidAmount
	}

	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	card, ok := m.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("%w: %s", ErrCardNotFound, cardID)
	}

	newBalance := card.BalanceKurus + amount
	card.BalanceKurus = newBalance

	entry := domain.LedgerEntry{
		ID:                uuid.New(),
		CardID:            cardID,
		EntryType:         domain.LedgerStornoDuzeltme,
		AmountKurus:       amount,
		BalanceAfterKurus: newBalance,
		ReferenceID:       refID,
		Description:       desc,
		CreatedAt:         time.Now(),
	}

	m.entries[cardID] = append(m.entries[cardID], entry)
	return &entry, nil
}

func (m *MemoryLedger) GetCardBalance(ctx context.Context, cardID string) (domain.Kurus, error) {
	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	card, ok := m.cards[cardID]
	if !ok {
		return 0, fmt.Errorf("%w: %s", ErrCardNotFound, cardID)
	}
	return card.BalanceKurus, nil
}

func (m *MemoryLedger) GetCardLedger(ctx context.Context, cardID string) ([]domain.LedgerEntry, error) {
	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	if _, ok := m.cards[cardID]; !ok {
		return nil, fmt.Errorf("%w: %s", ErrCardNotFound, cardID)
	}

	history, ok := m.entries[cardID]
	if !ok {
		return []domain.LedgerEntry{}, nil
	}
	result := make([]domain.LedgerEntry, len(history))
	copy(result, history)
	return result, nil
}

type DBTxStarter interface {
	Begin(ctx context.Context) (pgx.Tx, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// PostgresLedger implements PostgreSQL with row-level locking (SELECT ... FOR UPDATE)
type PostgresLedger struct {
	pool DBTxStarter
}

// lockCardAndReadLedgerBalance locks the card row before summing the ledger.
// Keeping the SUM in a separate statement is important at READ COMMITTED:
// a statement that waits on FOR UPDATE may retain its pre-wait snapshot for
// subqueries and otherwise calculate a stale balance after concurrent writes.
func lockCardAndReadLedgerBalance(ctx context.Context, tx pgx.Tx, cardID string) (domain.Kurus, bool, error) {
	var isBlacklisted bool
	if err := tx.QueryRow(ctx, `SELECT is_blacklisted FROM jw_cards WHERE id = $1 FOR UPDATE`, cardID).Scan(&isBlacklisted); err != nil {
		return 0, false, fmt.Errorf("kart kilitlenemedi: %w", err)
	}
	var balance domain.Kurus
	if err := tx.QueryRow(ctx, `SELECT COALESCE(SUM(amount_kurus), 0) FROM jw_ledger WHERE card_id = $1`, cardID).Scan(&balance); err != nil {
		return 0, false, fmt.Errorf("defter bakiyesi okunamadı: %w", err)
	}
	return balance, isBlacklisted, nil
}

func findLedgerEntryByReference(ctx context.Context, tx pgx.Tx, refID string) (*domain.LedgerEntry, error) {
	var entry domain.LedgerEntry
	var reference, description *string
	err := tx.QueryRow(ctx, `
		SELECT id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at
		FROM jw_ledger WHERE reference_id = $1
	`, refID).Scan(&entry.ID, &entry.CardID, &entry.EntryType, &entry.AmountKurus, &entry.BalanceAfterKurus, &reference, &description, &entry.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	if reference != nil {
		entry.ReferenceID = *reference
	}
	if description != nil {
		entry.Description = *description
	}
	return &entry, nil
}

func NewPostgresLedger(pool *pgxpool.Pool) *PostgresLedger {
	return &PostgresLedger{pool: pool}
}

func NewPostgresLedgerWithStarter(pool DBTxStarter) *PostgresLedger {
	return &PostgresLedger{pool: pool}
}

func (p *PostgresLedger) ProcessTap(
	ctx context.Context,
	cardID string,
	fare domain.Kurus,
	refID, desc string,
	allowNegativeLimit domain.Kurus,
) (*domain.LedgerEntry, error) {
	if fare < 0 {
		return nil, ErrInvalidAmount
	}
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx begin hatası: %w", err)
	}
	defer tx.Rollback(ctx)

	// Idempotency: refID daha önce işlenmişse aynısını dön
	if refID != "" {
		existing, err := findLedgerEntryByReference(ctx, tx, refID)
		if err != nil {
			return nil, fmt.Errorf("idempotency kaydı okunamadı: %w", err)
		}
		if existing != nil {
			if existing.AmountKurus != -fare || existing.CardID != cardID {
				return nil, ErrIdempotencyConflict
			}
			return existing, nil
		}
	}

	// Lock first, then issue a fresh READ COMMITTED statement to sum committed entries.
	currentBalance, isBlacklisted, err := lockCardAndReadLedgerBalance(ctx, tx, cardID)
	if err != nil {
		return nil, err
	}
	// A same-key request may have committed while this transaction waited for the card lock.
	if refID != "" {
		existing, err := findLedgerEntryByReference(ctx, tx, refID)
		if err != nil {
			return nil, fmt.Errorf("idempotency kaydı okunamadı: %w", err)
		}
		if existing != nil {
			if existing.AmountKurus != -fare || existing.CardID != cardID {
				return nil, ErrIdempotencyConflict
			}
			return existing, nil
		}
	}

	if isBlacklisted {
		return nil, ErrCardBlacklisted
	}
	newBalance := currentBalance - fare
	if newBalance < -allowNegativeLimit {
		return nil, ErrInsufficientBalance
	}

	// Kart bakiyesini güncelle
	_, err = tx.Exec(ctx, `
		UPDATE jw_cards 
		SET balance_kurus = $1, last_tap_at = NOW(), updated_at = NOW() 
		WHERE id = $2
	`, newBalance, cardID)
	if err != nil {
		return nil, fmt.Errorf("kart bakiye güncellenemedi: %w", err)
	}

	// Deftere append et
	entryID := uuid.New()
	entry := domain.LedgerEntry{
		ID:                entryID,
		CardID:            cardID,
		EntryType:         domain.LedgerBinis,
		AmountKurus:       -fare,
		BalanceAfterKurus: newBalance,
		ReferenceID:       refID,
		Description:       desc,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO jw_ledger (id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING created_at
	`, entry.ID, entry.CardID, entry.EntryType, entry.AmountKurus, entry.BalanceAfterKurus, entry.ReferenceID, entry.Description).Scan(&entry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("defter kaydı eklenemedi: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit hatası: %w", err)
	}

	return &entry, nil
}

func (p *PostgresLedger) Recharge(
	ctx context.Context,
	cardID string,
	amount domain.Kurus,
	refID, desc string,
) (*domain.LedgerEntry, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx begin hatası: %w", err)
	}
	defer tx.Rollback(ctx)

	// Idempotency kontrolü: Aynı reference_id daha önce işlenmişse aynısını dön
	if refID != "" {
		existing, err := findLedgerEntryByReference(ctx, tx, refID)
		if err != nil {
			return nil, fmt.Errorf("idempotency kaydı okunamadı: %w", err)
		}
		if existing != nil {
			if existing.AmountKurus != amount || existing.CardID != cardID {
				return nil, ErrIdempotencyConflict
			}
			return existing, nil
		}
	}

	currentBalance, _, err := lockCardAndReadLedgerBalance(ctx, tx, cardID)
	if err != nil {
		return nil, err
	}
	if refID != "" {
		existing, err := findLedgerEntryByReference(ctx, tx, refID)
		if err != nil {
			return nil, fmt.Errorf("idempotency kaydı okunamadı: %w", err)
		}
		if existing != nil {
			if existing.AmountKurus != amount || existing.CardID != cardID {
				return nil, ErrIdempotencyConflict
			}
			return existing, nil
		}
	}
	newBalance := currentBalance + amount
	_, err = tx.Exec(ctx, `
		UPDATE jw_cards 
		SET balance_kurus = $1, updated_at = NOW() 
		WHERE id = $2
	`, newBalance, cardID)
	if err != nil {
		return nil, fmt.Errorf("kart bakiye güncellenemedi: %w", err)
	}

	entryID := uuid.New()
	entry := domain.LedgerEntry{
		ID:                entryID,
		CardID:            cardID,
		EntryType:         domain.LedgerDolum,
		AmountKurus:       amount,
		BalanceAfterKurus: newBalance,
		ReferenceID:       refID,
		Description:       desc,
	}

	err = tx.QueryRow(ctx, `
		INSERT INTO jw_ledger (id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW())
		RETURNING created_at
	`, entry.ID, entry.CardID, entry.EntryType, entry.AmountKurus, entry.BalanceAfterKurus, entry.ReferenceID, entry.Description).Scan(&entry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("defter kaydı eklenemedi: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit hatası: %w", err)
	}

	return &entry, nil
}

func (p *PostgresLedger) Refund(
	ctx context.Context,
	cardID string,
	amount domain.Kurus,
	refID, desc string,
) (*domain.LedgerEntry, error) {
	return p.Recharge(ctx, cardID, amount, refID, desc)
}

func (p *PostgresLedger) StornoCorrection(
	ctx context.Context,
	cardID string,
	amount domain.Kurus,
	refID, desc string,
) (*domain.LedgerEntry, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx begin hatası: %w", err)
	}
	defer tx.Rollback(ctx)

	currentBalance, _, err := lockCardAndReadLedgerBalance(ctx, tx, cardID)
	if err != nil {
		return nil, err
	}
	newBalance := currentBalance + amount
	_, err = tx.Exec(ctx, `
		UPDATE jw_cards 
		SET balance_kurus = $1, updated_at = NOW() 
		WHERE id = $2
	`, newBalance, cardID)
	if err != nil {
		return nil, fmt.Errorf("kart güncellenemedi: %w", err)
	}

	entryID := uuid.New()
	entry := domain.LedgerEntry{
		ID:                entryID,
		CardID:            cardID,
		EntryType:         domain.LedgerStornoDuzeltme,
		AmountKurus:       amount,
		BalanceAfterKurus: newBalance,
		ReferenceID:       refID,
		Description:       desc,
		CreatedAt:         time.Now(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO jw_ledger (id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.ID, entry.CardID, entry.EntryType, entry.AmountKurus, entry.BalanceAfterKurus, entry.ReferenceID, entry.Description, entry.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("storno kaydı eklenemedi: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, fmt.Errorf("tx commit: %w", err)
	}

	return &entry, nil
}

func (p *PostgresLedger) GetCardBalance(ctx context.Context, cardID string) (domain.Kurus, error) {
	var balance domain.Kurus
	err := p.pool.QueryRow(ctx, `
		SELECT COALESCE((SELECT SUM(l.amount_kurus) FROM jw_ledger l WHERE l.card_id = c.id), 0)
		FROM jw_cards c WHERE c.id = $1
	`, cardID).Scan(&balance)
	return balance, err
}

func (p *PostgresLedger) GetCardLedger(ctx context.Context, cardID string) ([]domain.LedgerEntry, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at
		FROM jw_ledger
		WHERE card_id = $1
		ORDER BY created_at DESC, id DESC
	`, cardID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var entries []domain.LedgerEntry
	for rows.Next() {
		var e domain.LedgerEntry
		var ref, desc *string
		if err := rows.Scan(&e.ID, &e.CardID, &e.EntryType, &e.AmountKurus, &e.BalanceAfterKurus, &ref, &desc, &e.CreatedAt); err != nil {
			return nil, err
		}
		if ref != nil {
			e.ReferenceID = *ref
		}
		if desc != nil {
			e.Description = *desc
		}
		entries = append(entries, e)
	}
	return entries, nil
}
