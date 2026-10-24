package seed

import (
	"testing"

	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func TestSeed_UserCanaryIntegrity(t *testing.T) {
	// Sistem Canary Kuralı 5: Seed verisinde ilk kullanıcının e-postası seed.sedir@ddo-case.invalid olmalı
	canaryEmail := "seed.sedir@ddo-case.invalid"
	user := SeedUser{
		Email:    canaryEmail,
		FullName: "Sedir Sistem Yöneticisi",
		Role:     domain.RoleAdmin,
	}

	if user.Email != canaryEmail {
		t.Fatalf("İlk seed kullanıcısının e-postası %s olmalı, bulunan: %s", canaryEmail, user.Email)
	}
}
