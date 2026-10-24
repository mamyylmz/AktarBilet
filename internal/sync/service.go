package sync

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
)

var (
	ErrDuplicateEvent = errors.New("mükerrer validatör olayı (idempotent)")
)

type SyncService struct {
	mu            sync.Mutex
	pricingEngine *pricing.PricingEngine
	ledgerService ledger.LedgerService
	processedKeys map[string]*domain.ValidatorEvent
	validatorSeqs map[string]int64
	negativeLimit domain.Kurus
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
		processedKeys: make(map[string]*domain.ValidatorEvent),
		validatorSeqs: make(map[string]int64),
		negativeLimit: negativeToleranceLimit,
		cardsStore:    make(map[string]*domain.Card),
		tripHistories: make(map[string][]pricing.TripHistoryItem),
	}
}

func (s *SyncService) RegisterCard(card domain.Card) {
	s.mu.Lock()
	defer s.mu.Unlock()
	c := card
	s.cardsStore[card.ID] = &c
}

type IngestEventRequest struct {
	IdempotencyKey string
	ValidatorID    string
	SequenceNo     int64
	CardID         string
	LineID         string
	StationID      string
	Direction      int
	TappedAt       time.Time
	Tariff         domain.Tariff
}

// IngestOfflineEvent processes a single offline validator event idempotently
func (s *SyncService) IngestOfflineEvent(ctx context.Context, req IngestEventRequest) (*domain.ValidatorEvent, error) {
	s.mu.Lock()

	// 1. İdempotansi Kontrolü: IdempotencyKey veya (ValidatorID, SequenceNo)
	seqKey := fmt.Sprintf("%s:%d", req.ValidatorID, req.SequenceNo)
	if existing, exists := s.processedKeys[req.IdempotencyKey]; exists {
		s.mu.Unlock()
		return existing, nil // İdempotent: Daha önce işlenmiş olayın sonucunu dön
	}
	if existing, exists := s.processedKeys[seqKey]; exists {
		s.mu.Unlock()
		return existing, nil
	}

	card, exists := s.cardsStore[req.CardID]
	if !exists {
		s.mu.Unlock()
		return nil, fmt.Errorf("kart bulunamadı: %s", req.CardID)
	}

	// 2. Kara Liste Kontrolü (Offline dönemde sonradan kara listeye girmişse)
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

	// 3. Ücret Motoru Hesaplaması (T-1)
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

	// 4. Deftere İşleme ve Negatif Bakiye Toleransı (T-3)
	refID := fmt.Sprintf("SYNC:%s:%d", req.ValidatorID, req.SequenceNo)
	desc := fmt.Sprintf("%s binişi (Seq: %d)", req.LineID, req.SequenceNo)

	_, err := s.ledgerService.ProcessTap(ctx, req.CardID, calcRes.FareChargedKurus, refID, desc, s.negativeLimit)

	s.mu.Lock()
	defer s.mu.Unlock()

	var eventStatus domain.ValidatorEventStatus = domain.EventStatusProcessed
	var rejectionReason string

	if err != nil {
		if errors.Is(err, ledger.ErrInsufficientBalance) {
			// Negatif limit de aşıldı
			eventStatus = domain.EventStatusRejected
			rejectionReason = "Yetersiz bakiye (Negatif tolerans limiti aşıldı)"
			// Kartı kara listeye al
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
		// Başarılı biniş: Eğer bakiye eksiye düştüyse NEGATIVE_TOLERANCE durumuna işaretle
		currentBal, _ := s.ledgerService.GetCardBalance(ctx, req.CardID)
		if currentBal < 0 {
			eventStatus = domain.EventStatusNegativeTolerance
		}

		// Geçmişe ekle
		s.tripHistories[req.CardID] = append(s.tripHistories[req.CardID], pricing.TripHistoryItem{
			LineID:        req.LineID,
			StationID:     req.StationID,
			Direction:     req.Direction,
			TappedAt:      req.TappedAt,
			IsTransfer:    calcRes.IsTransfer,
			TransferLevel: calcRes.TransferLevel,
		})

		// Kart sayaçlarını güncelle
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

// IngestBatch processes a batch of offline events deterministically
func (s *SyncService) IngestBatch(ctx context.Context, batch []IngestEventRequest) ([]*domain.ValidatorEvent, error) {
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
