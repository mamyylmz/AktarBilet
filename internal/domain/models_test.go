package domain

import (
	"testing"
)

func TestDomain_CurrencyConversions(t *testing.T) {
	kurus := Kurus(2550)
	if kurus != 2550 {
		t.Fatalf("Kuruş değeri hatalı: beklenen 2550, bulunan %d", kurus)
	}

	formatted := kurus.FormatTL()
	if formatted != "25.50" {
		t.Fatalf("Kuruştan TL'ye biçimlendirme hatalı: beklenen '25.50', bulunan '%s'", formatted)
	}

	negativeKurus := Kurus(-500)
	if negativeKurus.FormatTL() != "-5.00" {
		t.Fatalf("Negatif kuruş biçimlendirme hatalı: beklenen '-5.00', bulunan '%s'", negativeKurus.FormatTL())
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
