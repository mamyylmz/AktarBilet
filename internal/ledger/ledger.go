package ledger

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

var (
	ErrInsufficientBalance = errors.New("yetersiz bakiye")
	ErrCardBlacklisted     = errors.New("kart kara listede")
	ErrInvalidAmount       = errors.New("geçersiz tutar")
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

	card, ok := m.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("kart bulunamadı: %s", cardID)
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

	card, ok := m.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("kart bulunamadı: %s", cardID)
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
		return nil, fmt.Errorf("kart bulunamadı: %s", cardID)
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
	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	card, ok := m.cards[cardID]
	if !ok {
		return nil, fmt.Errorf("kart bulunamadı: %s", cardID)
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
		return 0, fmt.Errorf("kart bulunamadı: %s", cardID)
	}
	return card.BalanceKurus, nil
}

func (m *MemoryLedger) GetCardLedger(ctx context.Context, cardID string) ([]domain.LedgerEntry, error) {
	lock := m.getCardLock(cardID)
	lock.Lock()
	defer lock.Unlock()

	history, ok := m.entries[cardID]
	if !ok {
		return []domain.LedgerEntry{}, nil
	}
	result := make([]domain.LedgerEntry, len(history))
	copy(result, history)
	return result, nil
}

// PostgresLedger implements PostgreSQL with row-level locking (SELECT ... FOR UPDATE)
type PostgresLedger struct {
	pool *pgxpool.Pool
}

func NewPostgresLedger(pool *pgxpool.Pool) *PostgresLedger {
	return &PostgresLedger{pool: pool}
}

func (p *PostgresLedger) ProcessTap(
	ctx context.Context,
	cardID string,
	fare domain.Kurus,
	refID, desc string,
	allowNegativeLimit domain.Kurus,
) (*domain.LedgerEntry, error) {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return nil, fmt.Errorf("tx begin hatası: %w", err)
	}
	defer tx.Rollback(ctx)

	// SELECT ... FOR UPDATE ile kart bazlı serileştirme (ADR-002)
	var currentBalance domain.Kurus
	var isBlacklisted bool
	err = tx.QueryRow(ctx, `
		SELECT balance_kurus, is_blacklisted 
		FROM jw_cards 
		WHERE id = $1 
		FOR UPDATE
	`, cardID).Scan(&currentBalance, &isBlacklisted)
	if err != nil {
		return nil, fmt.Errorf("kart kilitlenemedi: %w", err)
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
		CreatedAt:         time.Now(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO jw_ledger (id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.ID, entry.CardID, entry.EntryType, entry.AmountKurus, entry.BalanceAfterKurus, entry.ReferenceID, entry.Description, entry.CreatedAt)
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

	var currentBalance domain.Kurus
	err = tx.QueryRow(ctx, `
		SELECT balance_kurus 
		FROM jw_cards 
		WHERE id = $1 
		FOR UPDATE
	`, cardID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("kart kilitlenemedi: %w", err)
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
		CreatedAt:         time.Now(),
	}

	_, err = tx.Exec(ctx, `
		INSERT INTO jw_ledger (id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`, entry.ID, entry.CardID, entry.EntryType, entry.AmountKurus, entry.BalanceAfterKurus, entry.ReferenceID, entry.Description, entry.CreatedAt)
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

	var currentBalance domain.Kurus
	err = tx.QueryRow(ctx, `
		SELECT balance_kurus 
		FROM jw_cards 
		WHERE id = $1 
		FOR UPDATE
	`, cardID).Scan(&currentBalance)
	if err != nil {
		return nil, fmt.Errorf("kart kilitlenemedi: %w", err)
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
		SELECT balance_kurus FROM jw_cards WHERE id = $1
	`, cardID).Scan(&balance)
	return balance, err
}

func (p *PostgresLedger) GetCardLedger(ctx context.Context, cardID string) ([]domain.LedgerEntry, error) {
	rows, err := p.pool.Query(ctx, `
		SELECT id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at
		FROM jw_ledger
		WHERE card_id = $1
		ORDER BY created_at DESC
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
