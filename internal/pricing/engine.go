package pricing

import (
	"sort"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/domain"
)

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
			// Günlük limit aşıldı, tam tarife uygulanır
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

	// 4. Gece Tarifesi Kontrolü (00:00 - 05:59)
	hour := tappedAt.Hour()
	isNightFare := hour >= 0 && hour < 6
	calculatedFare := baseFare
	if isNightFare && tariff.NightSurchargePct > 0 {
		surcharge := (baseFare * domain.Kurus(tariff.NightSurchargePct)) / 100
		calculatedFare += surcharge
	}

	// 5. Deterministik Sıralama ve 90 Dakikalık Aktarma Penceresi Analizi
	// Tarihe göre artan sırada sırala (sırasız gelen paketler için determinizm garantisi)
	sortedHistory := make([]TripHistoryItem, len(history))
	copy(sortedHistory, history)
	sort.Slice(sortedHistory, func(i, j int) bool {
		return sortedHistory[i].TappedAt.Before(sortedHistory[j].TappedAt)
	})

	// Aktarma durumu değişkenleri
	transferWindow := 90 * time.Minute
	var initialTripTime time.Time
	var recentTransfersInChain int
	linesInCurrentChain := make(map[string]bool)

	// Aktarma zincirini geriye dönük inşa et
	// En son binişten geriye doğru aktarma zincirini tespit et
	for i := len(sortedHistory) - 1; i >= 0; i-- {
		item := sortedHistory[i]
		diff := tappedAt.Sub(item.TappedAt)
		if diff < 0 || diff > transferWindow {
			break
		}

		linesInCurrentChain[item.LineID] = true

		if !item.IsTransfer {
			// Bu biniş zincirin başlangıcı
			initialTripTime = item.TappedAt
			break
		}
		recentTransfersInChain++
	}

	isTransfer := false
	transferLevel := 0
	reason := "Standart biniş"

	// Eğer 90 dakika içinde geçerli bir başlangıç binişi varsa
	if !initialTripTime.IsZero() && tappedAt.Sub(initialTripTime) <= transferWindow {
		// Kural: Ters Yön ve Aynı Hat Kuralı ("aynı hatta geri dönüş aktarma sayılmaz")
		if linesInCurrentChain[currentLineID] {
			// Aynı hatta biniş: Aktarma sayılmaz, yeni tam yolculuk başlar!
			isTransfer = false
			transferLevel = 0
			reason = "Aynı hatta geri dönüş/biniş nedeniyle aktarma sayılmadı"
		} else if recentTransfersInChain < 2 {
			// En fazla 2 aktarma indirimli
			isTransfer = true
			transferLevel = recentTransfersInChain + 1

			var discountPct int
			if transferLevel == 1 {
				discountPct = tariff.Transfer1DiscountPct
			} else {
				discountPct = tariff.Transfer2DiscountPct
			}

			// İndirimi uygula (kuruş tam sayı aritmetiği)
			discountAmount := (calculatedFare * domain.Kurus(discountPct)) / 100
			calculatedFare -= discountAmount
			reason = "İndirimli aktarma uygulandı"
		} else {
			reason = "Maksimum 2 aktarma limitine ulaşıldı"
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
