package settlement

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func TestSettlement_LargestRemainderKurusIntegrity(t *testing.T) {
	// Senaryo: Toplam 17.50 TL (1750 kuruş) aktarmalı gelir
	// 3 Operatör:
	// - Belediye Otobüsü (Mesafe: 5000m)
	// - Özel Halk Otobüsü (Mesafe: 7000m)
	// - Şehir Hatları Vapuru (Mesafe: 3000m)
	// Toplam Mesafe: 15000m
	totalRevenue := domain.Kurus(1750)
	legs := []OperatorLeg{
		{Operator: "belediye", DistanceMeter: 5000},
		{Operator: "ozel_halk", DistanceMeter: 7000},
		{Operator: "sehir_hatlari", DistanceMeter: 3000},
	}

	shares, err := CalculateRevenueShare(totalRevenue, legs)
	if err != nil {
		t.Fatalf("Gelir paylaşımı hesaplama hatası: %v", err)
	}

	var sumKurus domain.Kurus
	for op, share := range shares {
		t.Logf("Operatör: %s -> Pay: %d kuruş (%s TL)", op, share, share.FormatTL())
		sumKurus += share
	}

	// Kuruş dağıtımı toplamı asla bozmamalı!
	if sumKurus != totalRevenue {
		t.Fatalf("Kuruş yuvarlama hatası! Toplam beklenen: %d kuruş, Dağıtılan toplam: %d kuruş", totalRevenue, sumKurus)
	}
}

func TestSettlement_PeriodClosingAndStorno(t *testing.T) {
	svc := NewSettlementService()
	start := time.Date(2026, 10, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(2026, 10, 31, 23, 59, 59, 0, time.UTC)

	period := svc.CreatePeriod("Ekim 2026 Dönemi", start, end)

	// Yolculuk 1 geliri: 20 TL (2000 kuruş), Belediye (%60) & Özel Halk (%40)
	eventID := uuid.New()
	legs := []OperatorLeg{
		{Operator: "belediye", DistanceMeter: 6000},
		{Operator: "ozel_halk", DistanceMeter: 4000},
	}
	err := svc.RecordTripSettlement(period.ID, &eventID, 2000, legs)
	if err != nil {
		t.Fatalf("Gelir kaydı başarısız: %v", err)
	}

	// 1. Dönem Kapatma
	closedPeriod, err := svc.ClosePeriod(period.ID)
	if err != nil {
		t.Fatalf("Dönem kapatma başarısız: %v", err)
	}
	if !closedPeriod.IsClosed || closedPeriod.ClosedAt == nil {
		t.Fatalf("Dönem kapalı olarak işaretlenmedi!")
	}

	// 2. Kapatılan döneme yeni biniş ekleme denemesi -> REDDEDİLMELİ
	err = svc.RecordTripSettlement(period.ID, &eventID, 2000, legs)
	if err != ErrPeriodAlreadyClosed {
		t.Fatalf("Kapatılan döneme yeni biniş eklenememeliydi, alınan hata: %v", err)
	}

	// 3. Kapanıştan sonra düzeltme (Ters Kayıt / Storno)
	// Örneğin: Özel Halk Otobüsü için 500 kuruşluk ceza veya iade stornosu
	stornoEntry, err := svc.RecordStornoAfterClose(period.ID, &eventID, "ozel_halk", -500, "Geçersiz okuma cezası storno kaydı")
	if err != nil {
		t.Fatalf("Storno kaydı başarısız: %v", err)
	}
	if !stornoEntry.IsStorno || stornoEntry.AmountKurus != -500 {
		t.Fatalf("Storno kaydı hatalı: %+v", stornoEntry)
	}

	// Dönem toplamı ve operatör payı storno ile güncellenmiş olmalı
	updatedPeriod, _ := svc.GetPeriod(period.ID)
	expectedTotal := domain.Kurus(1500) // 2000 - 500
	if updatedPeriod.TotalRevenueKurus != expectedTotal {
		t.Fatalf("Storno sonrası toplam ciro hatalı! Beklenen: %d, Alınan: %d", expectedTotal, updatedPeriod.TotalRevenueKurus)
	}
}
