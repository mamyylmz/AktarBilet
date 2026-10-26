package pricing

import (
	"sort"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/domain"
)

var locTRT = time.FixedZone("UTC+3", 3*3600)

// PricingEngine calculates fares deterministically according to Görev 1
type PricingEngine struct{}

func NewPricingEngine() *PricingEngine {
	return &PricingEngine{}
}

type TripHistoryItem struct {
	LineID        string
	StationID     string
	Direction     int
	TappedAt      time.Time
	IsTransfer    bool
	TransferLevel int
}

type CalculationResult struct {
	FareChargedKurus domain.Kurus
	IsTransfer       bool
	TransferLevel    int
	IsNightFare      bool
	IsCapped         bool
	Reason           string
}

// CalculateFare computes deterministic fare based on transfer window, reverse route, night fare, daily & monthly caps
func (e *PricingEngine) CalculateFare(
	card domain.Card,
	tariff domain.Tariff,
	currentLineID string,
	currentStationID string,
	currentDirection int,
	tappedAt time.Time,
	history []TripHistoryItem,
) CalculationResult {
	// 1. Ücretsiz Kart Kontrolü
	if card.CardType == domain.CardTypeUcretsiz {
		return CalculationResult{
			FareChargedKurus: 0,
			IsTransfer:       false,
			TransferLevel:    0,
			Reason:           "Ücretsiz kart tipi",
		}
	}

	// 2. Aylık Tavan (Fare Capping) Kontrolü
	if tariff.MonthlyCappingLimitKurus > 0 && card.MonthlyCappingSpent >= tariff.MonthlyCappingLimitKurus {
		return CalculationResult{
			FareChargedKurus: 0,
			IsTransfer:       false,
			TransferLevel:    0,
			IsCapped:         true,
			Reason:           "Aylık abonman tavanına ulaşıldı (Fare Capped)",
		}
	}

	// 3. Baz Ücret Tespiti (Öğrenci günlük limiti dahil)
	var baseFare domain.Kurus
	switch card.CardType {
	case domain.CardTypeOgrenci:
		if tariff.StudentDailyLimit > 0 && card.DailyTapsCount >= tariff.StudentDailyLimit {
			baseFare = tariff.BaseFareKurus
		} else {
			baseFare = tariff.StudentFareKurus
		}
	case domain.CardTypeYasli:
		baseFare = tariff.ElderlyFareKurus
	case domain.CardTypeTam:
		fallthrough
	default:
		baseFare = tariff.BaseFareKurus
	}

	// 4. Gece Tarifesi Kontrolü (00:00 - 05:59 Europe/Istanbul Saati - TRT UTC+3)
	localTime := tappedAt.In(locTRT)
	hour := localTime.Hour()
	isNightFare := hour >= 0 && hour < 6
	calculatedFare := baseFare
	if isNightFare && tariff.NightSurchargePct > 0 {
		surcharge := (baseFare * domain.Kurus(tariff.NightSurchargePct)) / 100
		calculatedFare += surcharge
	}

	// 5. Deterministik Sıralama ve 90 Dakikalık Aktarma Penceresi Analizi
	sortedHistory := make([]TripHistoryItem, len(history))
	copy(sortedHistory, history)
	sort.Slice(sortedHistory, func(i, j int) bool {
		return sortedHistory[i].TappedAt.Before(sortedHistory[j].TappedAt)
	})

	transferWindow := 90 * time.Minute
	var windowStart time.Time
	var chainTrips []TripHistoryItem

	// Geçmiş binişleri kronolojik olarak pencere bazında grupla
	for _, h := range sortedHistory {
		if h.TappedAt.After(tappedAt) {
			continue
		}
		if windowStart.IsZero() || h.TappedAt.Sub(windowStart) > transferWindow {
			windowStart = h.TappedAt
			chainTrips = []TripHistoryItem{h}
		} else {
			chainTrips = append(chainTrips, h)
		}
	}

	isTransfer := false
	transferLevel := 0
	reason := "Standart biniş"

	// Eğer tappedAt geçerli bir 90 dakikalık pencere içindeyse
	if !windowStart.IsZero() && tappedAt.Sub(windowStart) <= transferWindow {
		// Kural: Aynı Hatta Biniş / Ters Yön Kontrolü
		sameLineFound := false
		for _, trip := range chainTrips {
			if trip.LineID == currentLineID {
				sameLineFound = true
				break
			}
		}

		if sameLineFound {
			isTransfer = false
			transferLevel = 0
			reason = "Aynı hatta geri dönüş/biniş nedeniyle aktarma sayılmadı"
		} else {
			// Bu pencere içinde kaç adet indirimli aktarma yapılmış?
			discountedTransfersInChain := 0
			if len(chainTrips) > 1 {
				for _, trip := range chainTrips[1:] {
					if trip.IsTransfer {
						discountedTransfersInChain++
					}
				}
			}

			if discountedTransfersInChain < 2 {
				isTransfer = true
				transferLevel = discountedTransfersInChain + 1

				var discountPct int
				if transferLevel == 1 {
					discountPct = tariff.Transfer1DiscountPct
				} else {
					discountPct = tariff.Transfer2DiscountPct
				}

				discountAmount := (calculatedFare * domain.Kurus(discountPct)) / 100
				calculatedFare -= discountAmount
				reason = "İndirimli aktarma uygulandı"
			} else {
				reason = "Maksimum 2 aktarma limitine ulaşıldı"
			}
		}
	}

	// 6. Aylık Capping Sınırına Kalan Tutar Kontrolü
	if tariff.MonthlyCappingLimitKurus > 0 {
		remainingUntilCap := tariff.MonthlyCappingLimitKurus - card.MonthlyCappingSpent
		if remainingUntilCap > 0 && calculatedFare > remainingUntilCap {
			calculatedFare = remainingUntilCap
			reason += " (Aylık tavan tamamlama ücreti)"
		}
	}

	return CalculationResult{
		FareChargedKurus: calculatedFare,
		IsTransfer:       isTransfer,
		TransferLevel:    transferLevel,
		IsNightFare:      isNightFare,
		IsCapped:         false,
		Reason:           reason,
	}
}
