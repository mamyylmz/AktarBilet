package domain

import (
	"testing"
)

func TestDomain_CurrencyConversions(t *testing.T) {
	tl := 25.50
	kurus := TLToKurus(tl)
	if kurus != 2550 {
		t.Fatalf("TL'den kuruşa dönüşüm hatalı: beklenen 2550, bulunan %d", kurus)
	}

	convertedTL := kurus.ToTL()
	if convertedTL != 25.50 {
		t.Fatalf("Kuruştan TL'ye dönüşüm hatalı: beklenen 25.50, bulunan %.2f", convertedTL)
	}
}

func TestDomain_RolesAndStatuses(t *testing.T) {
	if RoleYolcu != "yolcu" || RoleKontrolor != "kontrolor" || RoleAdmin != "admin" {
		t.Fatalf("Rol sabitleri hatalı")
	}
	if EventStatusProcessed != "PROCESSED" || EventStatusNegativeTolerance != "NEGATIVE_TOLERANCE" {
		t.Fatalf("Olay durum sabitleri hatalı")
	}
}
