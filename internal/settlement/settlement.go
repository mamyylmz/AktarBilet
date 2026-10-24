package settlement

import (
	"errors"
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

var (
	ErrPeriodAlreadyClosed = errors.New("dönem zaten kapatılmış")
	ErrPeriodNotClosed     = errors.New("dönem henüz kapatılmamış")
	ErrPeriodNotFound      = errors.New("dönem bulunamadı")
	ErrInvalidShareTotal   = errors.New("kuruş dağıtımı toplam geliri tutmuyor")
)

type SettlementLedgerEntry struct {
	ID           uuid.UUID
	PeriodID     uuid.UUID
	EventID      *uuid.UUID
	Operator     string
	AmountKurus  domain.Kurus
	IsStorno     bool
	StornoReason string
	CreatedAt    time.Time
}

type SettlementPeriod struct {
	ID                uuid.UUID
	Name              string
	StartDate         time.Time
	EndDate           time.Time
	IsClosed          bool
	ClosedAt          *time.Time
	TotalRevenueKurus domain.Kurus
	OperatorShares    map[string]domain.Kurus
	Entries           []SettlementLedgerEntry
}

type OperatorLeg struct {
	Operator      string
	DistanceMeter int
	TripsCount    int
}

type SettlementService struct {
	mu      sync.Mutex
	periods map[uuid.UUID]*SettlementPeriod
}

func NewSettlementService() *SettlementService {
	return &SettlementService{
		periods: make(map[uuid.UUID]*SettlementPeriod),
	}
}

// CalculateRevenueShare allocates total revenue among operators using the Largest Remainder Method
// ensuring that the sum of kuruş shares strictly equals total revenue without losing a single kuruş!
func CalculateRevenueShare(totalRevenue domain.Kurus, legs []OperatorLeg) (map[string]domain.Kurus, error) {
	if totalRevenue == 0 || len(legs) == 0 {
		return map[string]domain.Kurus{}, nil
	}

	// Toplam ağırlığı hesapla (Mesafe bazlı ağırlık; 0 ise biniş adedi)
	totalWeight := 0
	for _, leg := range legs {
		w := leg.DistanceMeter
		if w <= 0 {
			w = leg.TripsCount * 1000 // Varsayılan biniş ağırlığı
		}
		totalWeight += w
	}

	if totalWeight == 0 {
		totalWeight = len(legs)
	}

	type shareCalc struct {
		operator  string
		baseKurus domain.Kurus
		remainder int64 // Çarpım kalıntısı
	}

	calculations := make([]shareCalc, len(legs))
	var allocatedBase domain.Kurus

	for i, leg := range legs {
		w := leg.DistanceMeter
		if w <= 0 {
			w = leg.TripsCount * 1000
		}

		numerator := int64(totalRevenue) * int64(w)
		base := domain.Kurus(numerator / int64(totalWeight))
		rem := numerator % int64(totalWeight)

		calculations[i] = shareCalc{
			operator:  leg.Operator,
			baseKurus: base,
			remainder: rem,
		}
		allocatedBase += base
	}

	// Kuruş farkını hesapla
	diffKurus := totalRevenue - allocatedBase

	// Kalıntılara göre azalan sırada sırala (Largest Remainder Method)
	sort.Slice(calculations, func(i, j int) bool {
		return calculations[i].remainder > calculations[j].remainder
	})

	// Kuruş farklarını en büyük kalıntılara 1'er kuruş olarak dağıt
	for i := 0; i < int(diffKurus); i++ {
		idx := i % len(calculations)
		calculations[idx].baseKurus++
	}

	result := make(map[string]domain.Kurus)
	var finalSum domain.Kurus
	for _, c := range calculations {
		result[c.operator] += c.baseKurus
		finalSum += c.baseKurus
	}

	if finalSum != totalRevenue {
		return nil, fmt.Errorf("%w: beklenen %d, hesaplanan %d", ErrInvalidShareTotal, totalRevenue, finalSum)
	}

	return result, nil
}

func (s *SettlementService) CreatePeriod(name string, start, end time.Time) *SettlementPeriod {
	s.mu.Lock()
	defer s.mu.Unlock()

	p := &SettlementPeriod{
		ID:             uuid.New(),
		Name:           name,
		StartDate:      start,
		EndDate:        end,
		IsClosed:       false,
		OperatorShares: make(map[string]domain.Kurus),
		Entries:        make([]SettlementLedgerEntry, 0),
	}
	s.periods[p.ID] = p
	return p
}

func (s *SettlementService) RecordTripSettlement(
	periodID uuid.UUID,
	eventID *uuid.UUID,
	totalRevenue domain.Kurus,
	legs []OperatorLeg,
) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	period, ok := s.periods[periodID]
	if !ok {
		return ErrPeriodNotFound
	}
	if period.IsClosed {
		return ErrPeriodAlreadyClosed
	}

	shares, err := CalculateRevenueShare(totalRevenue, legs)
	if err != nil {
		return err
	}

	period.TotalRevenueKurus += totalRevenue
	now := time.Now()

	for op, amount := range shares {
		period.OperatorShares[op] += amount
		period.Entries = append(period.Entries, SettlementLedgerEntry{
			ID:          uuid.New(),
			PeriodID:    periodID,
			EventID:     eventID,
			Operator:    op,
			AmountKurus: amount,
			IsStorno:    false,
			CreatedAt:   now,
		})
	}
	return nil
}

func (s *SettlementService) ClosePeriod(periodID uuid.UUID) (*SettlementPeriod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	period, ok := s.periods[periodID]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	if period.IsClosed {
		return nil, ErrPeriodAlreadyClosed
	}

	now := time.Now()
	period.IsClosed = true
	period.ClosedAt = &now
	return period, nil
}

// RecordStornoAfterClose adds a storno/correction entry after a period has already closed
func (s *SettlementService) RecordStornoAfterClose(
	periodID uuid.UUID,
	eventID *uuid.UUID,
	operator string,
	stornoAmount domain.Kurus,
	reason string,
) (*SettlementLedgerEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	period, ok := s.periods[periodID]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	if !period.IsClosed {
		return nil, ErrPeriodNotClosed
	}

	entry := SettlementLedgerEntry{
		ID:           uuid.New(),
		PeriodID:     periodID,
		EventID:      eventID,
		Operator:     operator,
		AmountKurus:  stornoAmount,
		IsStorno:     true,
		StornoReason: reason,
		CreatedAt:    time.Now(),
	}

	period.Entries = append(period.Entries, entry)
	period.OperatorShares[operator] += stornoAmount
	period.TotalRevenueKurus += stornoAmount

	return &entry, nil
}

func (s *SettlementService) GetPeriod(periodID uuid.UUID) (*SettlementPeriod, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	period, ok := s.periods[periodID]
	if !ok {
		return nil, ErrPeriodNotFound
	}
	return period, nil
}
