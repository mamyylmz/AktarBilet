package anomaly

import (
	"context"
	"math"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

type StationCoordinate struct {
	ID        string
	Name      string
	Latitude  float64
	Longitude float64
}

type CardLastTap struct {
	StationID string
	TappedAt  time.Time
}

type AnomalyDetector struct {
	mu          sync.Mutex
	stations    map[string]StationCoordinate
	lastTaps    map[string]CardLastTap
	anomalies   []domain.Anomaly
	maxSpeedKmh float64 // Eşik hız: Varsayılan 120 km/s (Toplu taşımada fiziksel imkansız sınır)
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		stations:    make(map[string]StationCoordinate),
		lastTaps:    make(map[string]CardLastTap),
		anomalies:   make([]domain.Anomaly, 0),
		maxSpeedKmh: 120.0,
	}
}

func (d *AnomalyDetector) RegisterStation(station StationCoordinate) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.stations[station.ID] = station
}

// HaversineDistance calculates approximate distance in meters between two lat/long points
func HaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadiusMeters = 6371000.0

	dLat := (lat2 - lat1) * (math.Pi / 180.0)
	dLon := (lon2 - lon1) * (math.Pi / 180.0)

	rLat1 := lat1 * (math.Pi / 180.0)
	rLat2 := lat2 * (math.Pi / 180.0)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(rLat1)*math.Cos(rLat2)*math.Sin(dLon/2)*math.Sin(dLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadiusMeters * c
}

type CheckResult struct {
	IsAnomaly          bool
	AnomalyType        domain.AnomalyType
	CalculatedSpeedKmh float64
	DistanceMeters     int
	TimeDeltaSeconds   int
	Description        string
}

// CheckTap inspects a tap against previous taps on the same card for impossible travel speed and clone cards
func (d *AnomalyDetector) CheckTap(
	ctx context.Context,
	cardID string,
	currentStationID string,
	tappedAt time.Time,
	eventID *uuid.UUID,
) CheckResult {
	d.mu.Lock()
	defer d.mu.Unlock()

	lastTap, exists := d.lastTaps[cardID]
	// Kartın son okuma bilgisini güncelle
	d.lastTaps[cardID] = CardLastTap{
		StationID: currentStationID,
		TappedAt:  tappedAt,
	}

	if !exists {
		return CheckResult{IsAnomaly: false}
	}

	// Eğer aynı durakta ise hız kontrolüne gerek yok
	if lastTap.StationID == currentStationID {
		return CheckResult{IsAnomaly: false}
	}

	timeDelta := tappedAt.Sub(lastTap.TappedAt).Seconds()
	if timeDelta <= 0 {
		// Eşzamanlı veya sırasız aynı anlık biniş -> Olası klon kart
		anom := domain.Anomaly{
			ID:                 uuid.New(),
			CardID:             cardID,
			EventID:            eventID,
			AnomalyType:        domain.AnomalyCloneCard,
			Description:        "Aynı kart ile eşzamanlı farklı duraklarda biniş (Klon Kart Şüphesi)",
			CalculatedSpeedKmh: 999.0,
			TimeDeltaSeconds:   int(timeDelta),
			DistanceMeters:     0,
			DetectedAt:         time.Now(),
		}
		d.anomalies = append(d.anomalies, anom)
		return CheckResult{
			IsAnomaly:   true,
			AnomalyType: domain.AnomalyCloneCard,
			Description: anom.Description,
		}
	}

	st1, hasSt1 := d.stations[lastTap.StationID]
	st2, hasSt2 := d.stations[currentStationID]
	if !hasSt1 || !hasSt2 {
		return CheckResult{IsAnomaly: false}
	}

	distanceMeters := HaversineDistance(st1.Latitude, st1.Longitude, st2.Latitude, st2.Longitude)
	speedMps := distanceMeters / timeDelta
	speedKmh := speedMps * 3.6

	// Görev 5 Kuralı: 5 dakika (300 sn) içinde uzak duraklar ve hız > 120 km/s ise imkansız seyahat
	if speedKmh > d.maxSpeedKmh && timeDelta <= 600 {
		desc := "Fiziksel olarak imkansız seyahat hızı tespit edildi (Kopya/Klon Kart Şüphesi)"
		anom := domain.Anomaly{
			ID:                 uuid.New(),
			CardID:             cardID,
			EventID:            eventID,
			AnomalyType:        domain.AnomalyImpossibleSpeed,
			Description:        desc,
			CalculatedSpeedKmh: speedKmh,
			TimeDeltaSeconds:   int(timeDelta),
			DistanceMeters:     int(distanceMeters),
			DetectedAt:         time.Now(),
		}
		d.anomalies = append(d.anomalies, anom)
		return CheckResult{
			IsAnomaly:          true,
			AnomalyType:        domain.AnomalyImpossibleSpeed,
			CalculatedSpeedKmh: speedKmh,
			DistanceMeters:     int(distanceMeters),
			TimeDeltaSeconds:   int(timeDelta),
			Description:        desc,
		}
	}

	return CheckResult{IsAnomaly: false}
}

func (d *AnomalyDetector) GetAnomalies() []domain.Anomaly {
	d.mu.Lock()
	defer d.mu.Unlock()
	res := make([]domain.Anomaly, len(d.anomalies))
	copy(res, d.anomalies)
	return res
}
