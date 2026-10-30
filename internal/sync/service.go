package sync

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
)

type SyncDBPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

type SyncService struct {
	mu            sync.Mutex
	pricingEngine *pricing.PricingEngine
	ledgerService ledger.LedgerService
	negativeLimit domain.Kurus
	pool          SyncDBPool
	processedKeys map[string]*domain.ValidatorEvent
	cardsStore    map[string]*domain.Card
	tripHistories map[string][]pricing.TripHistoryItem
}

func NewSyncService(
	pricingEngine *pricing.PricingEngine,
	ledgerService ledger.LedgerService,
	negativeToleranceLimit domain.Kurus,
) *SyncService {
	return &SyncService{
		pricingEngine: pricingEngine,
		ledgerService: ledgerService,
		negativeLimit: negativeToleranceLimit,
		processedKeys: make(map[string]*domain.ValidatorEvent),
		cardsStore:    make(map[string]*domain.Card),
		tripHistories: make(map[string][]pricing.TripHistoryItem),
	}
}

func (s *SyncService) SetPool(pool *pgxpool.Pool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if pool == nil {
		s.pool = nil
	} else {
		s.pool = pool
	}
}

func (s *SyncService) SetDBPool(pool SyncDBPool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.pool = pool
}

func (s *SyncService) RegisterCard(card domain.Card) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := card
	s.cardsStore[card.ID] = &c
}

type IngestEventRequest struct {
	IdempotencyKey string        `json:"idempotency_key"`
	ValidatorID    string        `json:"validator_id"`
	SequenceNo     int64         `json:"sequence_no"`
	CardID         string        `json:"card_id"`
	LineID         string        `json:"line_id"`
	StationID      string        `json:"station_id"`
	Direction      int           `json:"direction"`
	TappedAt       time.Time     `json:"tapped_at"`
	Tariff         domain.Tariff `json:"tariff,omitempty"`
}

// IngestOfflineEvent processes a single offline validator event idempotently with persistence support
func (s *SyncService) IngestOfflineEvent(ctx context.Context, req IngestEventRequest) (*domain.ValidatorEvent, error) {
	s.mu.Lock()
	pool := s.pool
	s.mu.Unlock()

	if pool != nil {
		return s.ingestPersistent(ctx, pool, req)
	}
	return s.ingestMemory(ctx, req)
}

func (s *SyncService) ingestPersistent(ctx context.Context, pool SyncDBPool, req IngestEventRequest) (*domain.ValidatorEvent, error) {
	if req.IdempotencyKey == "" {
		req.IdempotencyKey = fmt.Sprintf("SYNC:%s:%d", req.ValidatorID, req.SequenceNo)
	}

	// 1. İdempotansi Kontrolü: jw_validator_events tablosunu sorgula
	var existing domain.ValidatorEvent
	var rejectionReasonSql *string
	err := pool.QueryRow(ctx, `
		SELECT id, idempotency_key, validator_id, sequence_no, card_id, line_id, station_id, direction,
		       tapped_at, received_at, fare_charged_kurus, is_transfer, transfer_level, status, rejection_reason
		FROM jw_validator_events
		WHERE idempotency_key = $1 OR (validator_id = $2 AND sequence_no = $3)
	`, req.IdempotencyKey, req.ValidatorID, req.SequenceNo).Scan(
		&existing.ID, &existing.IdempotencyKey, &existing.ValidatorID, &existing.SequenceNo,
		&existing.CardID, &existing.LineID, &existing.StationID, &existing.Direction,
		&existing.TappedAt, &existing.ReceivedAt, &existing.FareChargedKurus,
		&existing.IsTransfer, &existing.TransferLevel, &existing.Status, &rejectionReasonSql,
	)
	if err == nil {
		if rejectionReasonSql != nil {
			existing.RejectionReason = *rejectionReasonSql
		}
		if existing.IdempotencyKey == req.IdempotencyKey && (existing.CardID != req.CardID || existing.ValidatorID != req.ValidatorID) {
			return nil, ErrIdempotencyConflict
		}
		return &existing, nil // İdempotent: zaten işlenmiş
	}

	// 2. Kartın varlığını ve durumunu doğrula
	var card domain.Card
	err = pool.QueryRow(ctx, `
		SELECT id, card_type, balance_kurus, is_blacklisted, daily_taps_count, monthly_capping_spent_kurus
		FROM jw_cards WHERE id = $1
	`, req.CardID).Scan(&card.ID, &card.CardType, &card.BalanceKurus, &card.IsBlacklisted, &card.DailyTapsCount, &card.MonthlyCappingSpent)
	if err != nil {
		return nil, fmt.Errorf("kart bulunamadı: %s", req.CardID)
	}

	// Kara Liste Kontrolü
	if card.IsBlacklisted {
		evID := uuid.New()
		event := &domain.ValidatorEvent{
			ID:               evID,
			IdempotencyKey:   req.IdempotencyKey,
			ValidatorID:      req.ValidatorID,
			SequenceNo:       req.SequenceNo,
			CardID:           req.CardID,
			LineID:           req.LineID,
			StationID:        req.StationID,
			Direction:        req.Direction,
			TappedAt:         req.TappedAt,
			ReceivedAt:       time.Now(),
			FareChargedKurus: 0,
			Status:           domain.EventStatusRejected,
			RejectionReason:  "Kart kara listede",
		}
		_, _ = pool.Exec(ctx, `
			INSERT INTO jw_validator_events (
				id, idempotency_key, validator_id, sequence_no, card_id, line_id, station_id,
				direction, tapped_at, received_at, fare_charged_kurus, is_transfer, transfer_level, status, rejection_reason
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), 0, false, 0, $10, $11)
			ON CONFLICT (idempotency_key) DO NOTHING
		`, event.ID, event.IdempotencyKey, event.ValidatorID, event.SequenceNo, event.CardID, event.LineID, event.StationID, event.Direction, event.TappedAt, event.Status, event.RejectionReason)
		return event, nil
	}

	// 3. Tarife Kontrolü: İstekte tarife yoksa veritabanından güncel tarifeyi al
	tariff := req.Tariff
	if tariff.BaseFareKurus == 0 {
		_ = pool.QueryRow(ctx, `
			SELECT id, name, mode, base_fare_kurus, student_fare_kurus, elderly_fare_kurus,
			       transfer1_discount_pct, transfer2_discount_pct, night_surcharge_pct,
			       student_daily_limit, monthly_capping_limit_kurus, is_active
			FROM jw_tariffs WHERE is_active = true LIMIT 1
		`).Scan(&tariff.ID, &tariff.Name, &tariff.Mode, &tariff.BaseFareKurus,
			&tariff.StudentFareKurus, &tariff.ElderlyFareKurus, &tariff.Transfer1DiscountPct,
			&tariff.Transfer2DiscountPct, &tariff.NightSurchargePct, &tariff.StudentDailyLimit,
			&tariff.MonthlyCappingLimitKurus, &tariff.IsActive)
		if tariff.BaseFareKurus == 0 {
			tariff.BaseFareKurus = 2000
			tariff.Transfer1DiscountPct = 50
			tariff.Transfer2DiscountPct = 75
		}
	}

	// 4. Son 90 dakikalık biniş geçmişini çek (Aktarma analizi için)
	rows, err := pool.Query(ctx, `
		SELECT line_id, station_id, direction, tapped_at, is_transfer, transfer_level
		FROM jw_validator_events
		WHERE card_id = $1 AND tapped_at >= ($2::timestamptz - INTERVAL '90 minutes') AND tapped_at < $2::timestamptz AND status = 'PROCESSED'
		ORDER BY tapped_at ASC
	`, req.CardID, req.TappedAt)
	var history []pricing.TripHistoryItem
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var it pricing.TripHistoryItem
			if err := rows.Scan(&it.LineID, &it.StationID, &it.Direction, &it.TappedAt, &it.IsTransfer, &it.TransferLevel); err == nil {
				history = append(history, it)
			}
		}
	}

	// 5. Ücret Motoru Hesaplaması (T-1)
	calcRes := s.pricingEngine.CalculateFare(
		card,
		tariff,
		req.LineID,
		req.StationID,
		req.Direction,
		req.TappedAt,
		history,
	)

	// 5.1 Sırasız Olay Uzlaşması (Out-of-Order Delivery Reconciliation / CORE-04)
	if pool != nil {
		rowsFuture, errFuture := pool.Query(ctx, `
			SELECT id, fare_charged_kurus, line_id, station_id, direction, tapped_at
			FROM jw_validator_events
			WHERE card_id = $1
			  AND tapped_at > $2::timestamptz
			  AND tapped_at <= ($2::timestamptz + INTERVAL '90 minutes')
			  AND status = 'PROCESSED'
			ORDER BY tapped_at ASC
		`, req.CardID, req.TappedAt)
		if errFuture == nil {
			type futureEventItem struct {
				id        uuid.UUID
				fare      domain.Kurus
				lineID    string
				stationID string
				direction int
				tappedAt  time.Time
			}
			var futureEvents []futureEventItem
			for rowsFuture.Next() {
				var fe futureEventItem
				if err := rowsFuture.Scan(&fe.id, &fe.fare, &fe.lineID, &fe.stationID, &fe.direction, &fe.tappedAt); err == nil {
					futureEvents = append(futureEvents, fe)
				}
			}
			rowsFuture.Close()

			if len(futureEvents) > 0 {
				chronologicalHistory := append([]pricing.TripHistoryItem{}, history...)
				chronologicalHistory = append(chronologicalHistory, pricing.TripHistoryItem{
					LineID:        req.LineID,
					StationID:     req.StationID,
					Direction:     req.Direction,
					TappedAt:      req.TappedAt,
					IsTransfer:    calcRes.IsTransfer,
					TransferLevel: calcRes.TransferLevel,
				})

				runningSpent := card.MonthlyCappingSpent + calcRes.FareChargedKurus

				for _, fe := range futureEvents {
					evalCard := card
					evalCard.MonthlyCappingSpent = runningSpent
					if evalCard.MonthlyCappingSpent < 0 {
						evalCard.MonthlyCappingSpent = 0
					}

					futureCalc := s.pricingEngine.CalculateFare(
						evalCard, tariff, fe.lineID, fe.stationID, fe.direction, fe.tappedAt, chronologicalHistory,
					)
					correction := fe.fare - futureCalc.FareChargedKurus
					if correction != 0 {
						ref := fmt.Sprintf("OOO-REPRICE:%s", fe.id)
						if _, err := s.ledgerService.StornoCorrection(ctx, req.CardID, correction, ref, "Sırasız olay kronolojik ücret uzlaşması"); err != nil && !errors.Is(err, ledger.ErrIdempotencyConflict) {
							return nil, fmt.Errorf("sırasız olay defter uzlaşması başarısız: %w", err)
						}
						_, _ = pool.Exec(ctx, `
							UPDATE jw_cards
							SET monthly_capping_spent_kurus = GREATEST(0, monthly_capping_spent_kurus - $1 + $2)
							WHERE id = $3
						`, fe.fare, futureCalc.FareChargedKurus, req.CardID)
					}
					if _, err := pool.Exec(ctx, `
						UPDATE jw_validator_events
						SET is_transfer = $1, transfer_level = $2, fare_charged_kurus = $3
						WHERE id = $4
					`, futureCalc.IsTransfer, futureCalc.TransferLevel, futureCalc.FareChargedKurus, fe.id); err != nil {
						return nil, fmt.Errorf("sırasız olay kaydı uzlaştırılamadı: %w", err)
					}

					runningSpent = runningSpent - fe.fare + futureCalc.FareChargedKurus
					if runningSpent < 0 {
						runningSpent = 0
					}

					chronologicalHistory = append(chronologicalHistory, pricing.TripHistoryItem{
						LineID:        fe.lineID,
						StationID:     fe.stationID,
						Direction:     fe.direction,
						TappedAt:      fe.tappedAt,
						IsTransfer:    futureCalc.IsTransfer,
						TransferLevel: futureCalc.TransferLevel,
					})
				}
			}
		}
	}

	// 6. Deftere İşleme ve Satır Kilitleme (T-3)
	refID := fmt.Sprintf("SYNC:%s:%d", req.ValidatorID, req.SequenceNo)
	desc := fmt.Sprintf("%s binişi (Seq: %d)", req.LineID, req.SequenceNo)

	_, err = s.ledgerService.ProcessTap(ctx, req.CardID, calcRes.FareChargedKurus, refID, desc, s.negativeLimit)

	var eventStatus domain.ValidatorEventStatus = domain.EventStatusProcessed
	var rejectionReason string

	if err != nil {
		if errors.Is(err, ledger.ErrInsufficientBalance) {
			eventStatus = domain.EventStatusRejected
			rejectionReason = "Yetersiz bakiye (Negatif tolerans limiti aşıldı)"
			_, _ = pool.Exec(ctx, "UPDATE jw_cards SET is_blacklisted = true, blacklist_reason = 'Aşırı negatif bakiye' WHERE id = $1", req.CardID)
		} else if errors.Is(err, ledger.ErrCardBlacklisted) {
			eventStatus = domain.EventStatusRejected
			rejectionReason = "Kart kara listede"
		} else {
			eventStatus = domain.EventStatusRejected
			rejectionReason = err.Error()
		}
	} else {
		currentBal, _ := s.ledgerService.GetCardBalance(ctx, req.CardID)
		if currentBal < 0 {
			eventStatus = domain.EventStatusNegativeTolerance
		}

		// Kart sayaçlarını güncelle
		_, _ = pool.Exec(ctx, `
			UPDATE jw_cards 
			SET daily_taps_count = daily_taps_count + 1,
			    monthly_capping_spent_kurus = monthly_capping_spent_kurus + $1,
			    last_tap_at = $2,
			    last_station_id = $3
			WHERE id = $4
		`, calcRes.FareChargedKurus, req.TappedAt, req.StationID, req.CardID)
	}

	// 7. Olayı jw_validator_events tablosuna kalıcı olarak yaz
	evID := uuid.New()
	event := &domain.ValidatorEvent{
		ID:               evID,
		IdempotencyKey:   req.IdempotencyKey,
		ValidatorID:      req.ValidatorID,
		SequenceNo:       req.SequenceNo,
		CardID:           req.CardID,
		LineID:           req.LineID,
		StationID:        req.StationID,
		Direction:        req.Direction,
		TappedAt:         req.TappedAt,
		ReceivedAt:       time.Now(),
		FareChargedKurus: calcRes.FareChargedKurus,
		IsTransfer:       calcRes.IsTransfer,
		TransferLevel:    calcRes.TransferLevel,
		Status:           eventStatus,
		RejectionReason:  rejectionReason,
	}

	_, err = pool.Exec(ctx, `
		INSERT INTO jw_validator_events (
			id, idempotency_key, validator_id, sequence_no, card_id, line_id, station_id,
			direction, tapped_at, received_at, fare_charged_kurus, is_transfer, transfer_level, status, rejection_reason
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, NOW(), $10, $11, $12, $13, $14)
		ON CONFLICT (idempotency_key) DO UPDATE 
		SET status = EXCLUDED.status, fare_charged_kurus = EXCLUDED.fare_charged_kurus
	`, event.ID, event.IdempotencyKey, event.ValidatorID, event.SequenceNo, event.CardID, event.LineID, event.StationID,
		event.Direction, event.TappedAt, event.FareChargedKurus, event.IsTransfer, event.TransferLevel,
		event.Status, event.RejectionReason)
	if err != nil {
		return nil, fmt.Errorf("olay veritabanına yazılamadı: %w", err)
	}

	return event, nil
}

func (s *SyncService) ingestMemory(ctx context.Context, req IngestEventRequest) (*domain.ValidatorEvent, error) {
	s.mu.Lock()

	if req.IdempotencyKey == "" {
		req.IdempotencyKey = fmt.Sprintf("SYNC:%s:%d", req.ValidatorID, req.SequenceNo)
	}

	seqKey := fmt.Sprintf("%s:%d", req.ValidatorID, req.SequenceNo)
	if existing, exists := s.processedKeys[req.IdempotencyKey]; exists {
		s.mu.Unlock()
		if existing.CardID != req.CardID || existing.ValidatorID != req.ValidatorID {
			return nil, ErrIdempotencyConflict
		}
		return existing, nil
	}
	if existing, exists := s.processedKeys[seqKey]; exists {
		s.mu.Unlock()
		if existing.CardID != req.CardID || existing.ValidatorID != req.ValidatorID {
			return nil, ErrIdempotencyConflict
		}
		return existing, nil
	}

	card, exists := s.cardsStore[req.CardID]
	if !exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("kart bulunamadı: %s", req.CardID)
	}

	if card.IsBlacklisted {
		event := &domain.ValidatorEvent{
			ID:               uuid.New(),
			IdempotencyKey:   req.IdempotencyKey,
			ValidatorID:      req.ValidatorID,
			SequenceNo:       req.SequenceNo,
			CardID:           req.CardID,
			LineID:           req.LineID,
			StationID:        req.StationID,
			Direction:        req.Direction,
			TappedAt:         req.TappedAt,
			ReceivedAt:       time.Now(),
			FareChargedKurus: 0,
			Status:           domain.EventStatusRejected,
			RejectionReason:  "Kart kara listede",
		}
		s.processedKeys[req.IdempotencyKey] = event
		s.processedKeys[seqKey] = event
		s.mu.Unlock()
		return event, nil
	}

	history := s.tripHistories[req.CardID]
	calcRes := s.pricingEngine.CalculateFare(
		*card,
		req.Tariff,
		req.LineID,
		req.StationID,
		req.Direction,
		req.TappedAt,
		history,
	)

	s.mu.Unlock()

	refID := fmt.Sprintf("SYNC:%s:%d", req.ValidatorID, req.SequenceNo)
	desc := fmt.Sprintf("%s binişi (Seq: %d)", req.LineID, req.SequenceNo)

	_, err := s.ledgerService.ProcessTap(ctx, req.CardID, calcRes.FareChargedKurus, refID, desc, s.negativeLimit)

	s.mu.Lock()
	defer s.mu.Unlock()

	var eventStatus domain.ValidatorEventStatus = domain.EventStatusProcessed
	var rejectionReason string

	if err != nil {
		if errors.Is(err, ledger.ErrInsufficientBalance) {
			eventStatus = domain.EventStatusRejected
			rejectionReason = "Yetersiz bakiye (Negatif tolerans limiti aşıldı)"
			card.IsBlacklisted = true
			card.BlacklistReason = "Aşırı negatif bakiye"
		} else if errors.Is(err, ledger.ErrCardBlacklisted) {
			eventStatus = domain.EventStatusRejected
			rejectionReason = "Kart kara listede"
		} else {
			eventStatus = domain.EventStatusRejected
			rejectionReason = err.Error()
		}
	} else {
		currentBal, _ := s.ledgerService.GetCardBalance(ctx, req.CardID)
		if currentBal < 0 {
			eventStatus = domain.EventStatusNegativeTolerance
		}

		s.tripHistories[req.CardID] = append(s.tripHistories[req.CardID], pricing.TripHistoryItem{
			LineID:        req.LineID,
			StationID:     req.StationID,
			Direction:     req.Direction,
			TappedAt:      req.TappedAt,
			IsTransfer:    calcRes.IsTransfer,
			TransferLevel: calcRes.TransferLevel,
		})

		card.DailyTapsCount++
		card.MonthlyCappingSpent += calcRes.FareChargedKurus
	}

	event := &domain.ValidatorEvent{
		ID:               uuid.New(),
		IdempotencyKey:   req.IdempotencyKey,
		ValidatorID:      req.ValidatorID,
		SequenceNo:       req.SequenceNo,
		CardID:           req.CardID,
		LineID:           req.LineID,
		StationID:        req.StationID,
		Direction:        req.Direction,
		TappedAt:         req.TappedAt,
		ReceivedAt:       time.Now(),
		FareChargedKurus: calcRes.FareChargedKurus,
		IsTransfer:       calcRes.IsTransfer,
		TransferLevel:    calcRes.TransferLevel,
		Status:           eventStatus,
		RejectionReason:  rejectionReason,
	}

	s.processedKeys[req.IdempotencyKey] = event
	s.processedKeys[seqKey] = event
	return event, nil
}

// IngestBatch processes a batch of offline events deterministically, sorted chronologically by TappedAt (ADR-004)
func (s *SyncService) IngestBatch(ctx context.Context, batch []IngestEventRequest) ([]*domain.ValidatorEvent, error) {
	// Donanımsal tapped_at zaman damgasına göre deterministik sırala
	sort.Slice(batch, func(i, j int) bool {
		return batch[i].TappedAt.Before(batch[j].TappedAt)
	})

	results := make([]*domain.ValidatorEvent, 0, len(batch))
	for _, req := range batch {
		ev, err := s.IngestOfflineEvent(ctx, req)
		if err != nil {
			return nil, err
		}
		results = append(results, ev)
	}
	return results, nil
}
