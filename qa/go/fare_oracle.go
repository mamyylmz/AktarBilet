package goqa

import (
	"sort"
	"time"
)

// ReferenceCardType matches domain card types
type ReferenceCardType string

const (
	RefCardTam      ReferenceCardType = "tam"
	RefCardOgrenci  ReferenceCardType = "ogrenci"
	RefCardYasli    ReferenceCardType = "yasli"
	RefCardUcretsiz ReferenceCardType = "ucretsiz"
	RefCardAbonman  ReferenceCardType = "abonman"
)

// ReferenceTariff defines complete fare engine parameters in int64 kuruş
type ReferenceTariff struct {
	ID                       string
	BaseFareKurus            int64
	StudentFareKurus         int64
	ElderlyFareKurus         int64
	Transfer1DiscountPct     int
	Transfer2DiscountPct     int
	NightSurchargePct        int
	StudentDailyLimit        int
	MonthlyCappingLimitKurus int64
}

// ReferenceCard holds card balance and state in int64 kuruş
type ReferenceCard struct {
	ID                  string
	CardType            ReferenceCardType
	BalanceKurus        int64
	IsBlacklisted       bool
	DailyTapsCount      int
	MonthlyCappingSpent int64
}

// ReferenceTrip represents a single tap event
type ReferenceTrip struct {
	LineID    string
	StationID string
	Direction int // 1: forward, 2: reverse
	TappedAt  time.Time
}

// ReferenceResult is the expected outcome computed by the Independent Oracle
type ReferenceResult struct {
	FareChargedKurus int64
	IsTransfer       bool
	TransferLevel    int
	IsNightFare      bool
	IsCapped         bool
	Reason           string
}

// IstanbulLocation defines the official Turkish timezone for transit operations
var IstanbulLocation = time.FixedZone("UTC+3", 3*3600)

// ReferenceFareOracle is a zero-trust independent reference calculation engine.
// It does NOT import or call any production code from internal/pricing.
type ReferenceFareOracle struct{}

func NewReferenceFareOracle() *ReferenceFareOracle {
	return &ReferenceFareOracle{}
}

// CalculateExpectedFare evaluates a trip against full spec rules deterministically
func (o *ReferenceFareOracle) CalculateExpectedFare(
	card ReferenceCard,
	tariff ReferenceTariff,
	currentTrip ReferenceTrip,
	history []ReferenceTrip,
) ReferenceResult {
	// 1. Ücretsiz Kart Kuralı
	if card.CardType == RefCardUcretsiz {
		return ReferenceResult{
			FareChargedKurus: 0,
			IsTransfer:       false,
			TransferLevel:    0,
			Reason:           "Ücretsiz kart",
		}
	}

	// 2. Aylık Tavan (Fare Capping) Kontrolü
	if tariff.MonthlyCappingLimitKurus > 0 && card.MonthlyCappingSpent >= tariff.MonthlyCappingLimitKurus {
		return ReferenceResult{
			FareChargedKurus: 0,
			IsTransfer:       false,
			TransferLevel:    0,
			IsCapped:         true,
			Reason:           "Aylık tavan aşıldı (Ücretsiz geçiş)",
		}
	}

	// 3. Baz Ücret Tespiti (Kart türü & Öğrenci Günlük Limit Kontrolü)
	var baseFare int64
	switch card.CardType {
	case RefCardOgrenci:
		if tariff.StudentDailyLimit > 0 && card.DailyTapsCount >= tariff.StudentDailyLimit {
			// Günlük limit aşıldı -> tam yetişkin ücreti
			baseFare = tariff.BaseFareKurus
		} else {
			baseFare = tariff.StudentFareKurus
		}
	case RefCardYasli:
		baseFare = tariff.ElderlyFareKurus
	case RefCardTam, RefCardAbonman:
		fallthrough
	default:
		baseFare = tariff.BaseFareKurus
	}

	// 4. Gece Tarifesi (00:00 - 05:59:59 Europe/Istanbul Saati)
	localTime := currentTrip.TappedAt.In(IstanbulLocation)
	hour := localTime.Hour()
	isNightFare := hour >= 0 && hour < 6

	calculatedFare := baseFare
	if isNightFare && tariff.NightSurchargePct > 0 {
		surcharge := (baseFare * int64(tariff.NightSurchargePct)) / 100
		calculatedFare += surcharge
	}

	// 5. Deterministik 90 Dakikalık Aktarma Penceresi Analizi
	// Tarihe göre kronolojik sırala
	sortedHistory := make([]ReferenceTrip, len(history))
	copy(sortedHistory, history)
	sort.Slice(sortedHistory, func(i, j int) bool {
		return sortedHistory[i].TappedAt.Before(sortedHistory[j].TappedAt)
	})

	transferWindow := 90 * time.Minute
	var chainInitialTrip *ReferenceTrip
	var discountedTransfersCount int
	var sameLineSameDirectionFound bool
	var sameLineReverseDirectionFound bool

	// Aktarma zincirinin başlangıcını tespit et
	// Zincir: Son 90 dakika içindeki ilk biniş
	for i := 0; i < len(sortedHistory); i++ {
		hTrip := sortedHistory[i]
		diff := currentTrip.TappedAt.Sub(hTrip.TappedAt)
		if diff >= 0 && diff <= transferWindow {
			// Aktarma penceresi içindeki ilk biniş zincirin başıdır
			if chainInitialTrip == nil {
				chainInitialTrip = &sortedHistory[i]
			}

			// Aynı hat ve yön kontrolleri
			if hTrip.LineID == currentTrip.LineID {
				if hTrip.Direction == currentTrip.Direction {
					sameLineSameDirectionFound = true
				} else {
					sameLineReverseDirectionFound = true
				}
			}

			// Bu biniş indirimli aktarma mıydı?
			// 1. biniş = initial (indirimli değil)
			// 2. biniş = 1. aktarma (indirimli)
			// 3. biniş = 2. aktarma (indirimli)
			// 4. biniş ve sonrası = indirimsiz
			if i > 0 && i <= 2 {
				discountedTransfersCount++
			}
		}
	}

	isTransfer := false
	transferLevel := 0
	reason := "Standart biniş"

	// Eğer 90 dakika içinde geçerli bir başlangıç binişi varsa
	if chainInitialTrip != nil {
		diffFromStart := currentTrip.TappedAt.Sub(chainInitialTrip.TappedAt)
		if diffFromStart <= transferWindow {
			// Ters Yön Kuralı: Aynı hatta geri dönüş aktarma sayılmaz!
			if sameLineReverseDirectionFound || sameLineSameDirectionFound {
				isTransfer = false
				transferLevel = 0
				reason = "Aynı hatta geri dönüş/biniş nedeniyle aktarma uygulanmadı"
			} else if discountedTransfersCount < 2 {
				// En fazla 2 indirimli aktarma
				isTransfer = true
				transferLevel = discountedTransfersCount + 1

				var discountPct int
				if transferLevel == 1 {
					discountPct = tariff.Transfer1DiscountPct
				} else {
					discountPct = tariff.Transfer2DiscountPct
				}

				discountAmount := (calculatedFare * int64(discountPct)) / 100
				calculatedFare -= discountAmount
				reason = "İndirimli aktarma"
			} else {
				reason = "Maksimum 2 indirimli aktarma limitine ulaşıldı"
			}
		}
	}

	// 6. Aylık Tavan Tamamlama Kontrolü
	isCapped := false
	if tariff.MonthlyCappingLimitKurus > 0 {
		remainingUntilCap := tariff.MonthlyCappingLimitKurus - card.MonthlyCappingSpent
		if remainingUntilCap <= 0 {
			calculatedFare = 0
			isCapped = true
			reason = "Aylık tavan tamamlandı"
		} else if calculatedFare > remainingUntilCap {
			calculatedFare = remainingUntilCap
			isCapped = true
			reason += " (Aylık tavan tamamlama)"
		}
	}

	return ReferenceResult{
		FareChargedKurus: calculatedFare,
		IsTransfer:       isTransfer,
		TransferLevel:    transferLevel,
		IsNightFare:      isNightFare,
		IsCapped:         isCapped,
		Reason:           reason,
	}
}

