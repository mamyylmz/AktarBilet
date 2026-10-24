package domain

import (
	"time"

	"github.com/google/uuid"
)

// Currency is in Kuruş (1 TL = 100 kuruş). No float!
type Kurus int64

func TLToKurus(tl float64) Kurus {
	return Kurus(tl * 100)
}

func (k Kurus) ToTL() float64 {
	return float64(k) / 100.0
}

type UserRole string

const (
	RoleYolcu         UserRole = "yolcu"
	RoleKontrolor     UserRole = "kontrolor"
	RoleHatYoneticisi UserRole = "hat_yoneticisi"
	RoleMuhasebe      UserRole = "muhasebe"
	RoleOperasyon     UserRole = "operasyon"
	RoleAdmin         UserRole = "admin"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	FullName     string    `json:"full_name"`
	Role         UserRole  `json:"role"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CardType string

const (
	CardTypeTam      CardType = "tam"
	CardTypeOgrenci  CardType = "ogrenci"
	CardTypeYasli    CardType = "yasli"
	CardTypeUcretsiz CardType = "ucretsiz"
)

type Card struct {
	ID                   string     `json:"id"`
	UserID               *uuid.UUID `json:"user_id,omitempty"`
	CardType             CardType   `json:"card_type"`
	BalanceKurus         Kurus      `json:"balance_kurus"`
	IsBlacklisted        bool       `json:"is_blacklisted"`
	BlacklistReason      string     `json:"blacklist_reason,omitempty"`
	DailyTapsCount       int        `json:"daily_taps_count"`
	LastDailyResetDate   time.Time  `json:"last_daily_reset_date"`
	MonthlyCappingSpent  Kurus      `json:"monthly_capping_spent_kurus"`
	LastMonthlyResetDate time.Time  `json:"last_monthly_reset_date"`
	LastTapAt            *time.Time `json:"last_tap_at,omitempty"`
	LastStationID        *string    `json:"last_station_id,omitempty"`
	CreatedAt            time.Time  `json:"created_at"`
	UpdatedAt            time.Time  `json:"updated_at"`
}

type LedgerEntryType string

const (
	LedgerDolum          LedgerEntryType = "dolum"
	LedgerBinis          LedgerEntryType = "binis"
	LedgerIade           LedgerEntryType = "iade"
	LedgerCeza           LedgerEntryType = "ceza"
	LedgerStornoDuzeltme LedgerEntryType = "storno_duzeltme"
)

type LedgerEntry struct {
	ID                uuid.UUID       `json:"id"`
	CardID            string          `json:"card_id"`
	EntryType         LedgerEntryType `json:"entry_type"`
	AmountKurus       Kurus           `json:"amount_kurus"` // Positive for credit, Negative for debit
	BalanceAfterKurus Kurus           `json:"balance_after_kurus"`
	ReferenceID       string          `json:"reference_id,omitempty"`
	Description       string          `json:"description,omitempty"`
	CreatedAt         time.Time       `json:"created_at"`
}

type LineMode string

const (
	ModeBus   LineMode = "bus"
	ModeTram  LineMode = "tram"
	ModeFerry LineMode = "ferry"
)

type Line struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mode      LineMode  `json:"mode"`
	Operator  string    `json:"operator"` // "belediye" or "ozel_halk"
	IsActive  bool      `json:"is_active"`
	CreatedAt time.Time `json:"created_at"`
}

type Station struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Mode      LineMode  `json:"mode"`
	Latitude  float64   `json:"latitude"`
	Longitude float64   `json:"longitude"`
	ZoneID    int       `json:"zone_id"`
	CreatedAt time.Time `json:"created_at"`
}

type Tariff struct {
	ID                       string    `json:"id"`
	Name                     string    `json:"name"`
	Mode                     LineMode  `json:"mode"`
	BaseFareKurus            Kurus     `json:"base_fare_kurus"`
	StudentFareKurus         Kurus     `json:"student_fare_kurus"`
	ElderlyFareKurus         Kurus     `json:"elderly_fare_kurus"`
	Transfer1DiscountPct     int       `json:"transfer1_discount_pct"`
	Transfer2DiscountPct     int       `json:"transfer2_discount_pct"`
	NightSurchargePct        int       `json:"night_surcharge_pct"`
	StudentDailyLimit        int       `json:"student_daily_limit"`
	MonthlyCappingLimitKurus Kurus     `json:"monthly_capping_limit_kurus"`
	IsActive                 bool      `json:"is_active"`
	CreatedAt                time.Time `json:"created_at"`
}

type ValidatorEventStatus string

const (
	EventStatusProcessed         ValidatorEventStatus = "PROCESSED"
	EventStatusRejected          ValidatorEventStatus = "REJECTED"
	EventStatusNegativeTolerance ValidatorEventStatus = "NEGATIVE_TOLERANCE"
	EventStatusAnomalyFlagged    ValidatorEventStatus = "ANOMALY_FLAGGED"
)

type ValidatorEvent struct {
	ID               uuid.UUID            `json:"id"`
	IdempotencyKey   string               `json:"idempotency_key"`
	ValidatorID      string               `json:"validator_id"`
	SequenceNo       int64                `json:"sequence_no"`
	CardID           string               `json:"card_id"`
	LineID           string               `json:"line_id"`
	StationID        string               `json:"station_id"`
	Direction        int                  `json:"direction"` // 1: gidiş, 2: dönüş
	TappedAt         time.Time            `json:"tapped_at"`
	ReceivedAt       time.Time            `json:"received_at"`
	FareChargedKurus Kurus                `json:"fare_charged_kurus"`
	IsTransfer       bool                 `json:"is_transfer"`
	TransferLevel    int                  `json:"transfer_level"`
	Status           ValidatorEventStatus `json:"status"`
	RejectionReason  string               `json:"rejection_reason,omitempty"`
}

type PenaltyReason string

const (
	PenaltyKartsizBinis PenaltyReason = "kartsiz_binis"
	PenaltyUsulsuzKart  PenaltyReason = "usulsuz_kart_kullanimi"
	PenaltySahteKart    PenaltyReason = "sahte_kart"
	PenaltyDiger        PenaltyReason = "diger"
)

type Penalty struct {
	ID                 uuid.UUID     `json:"id"`
	InspectorID        uuid.UUID     `json:"inspector_id"`
	CardID             string        `json:"card_id"`
	LineID             string        `json:"line_id"`
	StationID          string        `json:"station_id"`
	PenaltyAmountKurus Kurus         `json:"penalty_amount_kurus"`
	Reason             PenaltyReason `json:"reason"`
	IssuedAt           time.Time     `json:"issued_at"`
	Notes              string        `json:"notes,omitempty"`
}

type AnomalyType string

const (
	AnomalyImpossibleSpeed AnomalyType = "IMPOSSIBLE_TRAVEL_SPEED"
	AnomalyCloneCard       AnomalyType = "CLONE_CARD_DETECTED"
	AnomalyConcurrentTap   AnomalyType = "CONCURRENT_TAP"
)

type Anomaly struct {
	ID                 uuid.UUID   `json:"id"`
	CardID             string      `json:"card_id"`
	EventID            *uuid.UUID  `json:"event_id,omitempty"`
	AnomalyType        AnomalyType `json:"anomaly_type"`
	Description        string      `json:"description"`
	CalculatedSpeedKmh float64     `json:"calculated_speed_kmh"`
	TimeDeltaSeconds   int         `json:"time_delta_seconds"`
	DistanceMeters     int         `json:"distance_meters"`
	DetectedAt         time.Time   `json:"detected_at"`
}
