package seed

import (
	"testing"

	"github.com/mamyylmz/AktarBilet/internal/domain"
)

func TestSeed_UserCanaryIntegrity(t *testing.T) {
	// Sistem Canary Kuralı 5: Seed verisinde ilk kullanıcının e-postası seed.sedir@ddo-case.invalid olmalı
	users := DefaultSeedUsers()
	if len(users) == 0 {
		t.Fatal("Seed kullanıcı listesi boş olamaz")
	}

	canaryEmail := "seed.sedir@ddo-case.invalid"
	firstUser := users[0]
	if firstUser.Email != canaryEmail {
		t.Fatalf("İlk seed kullanıcısının e-postası %s olmalı, bulunan: %s", canaryEmail, firstUser.Email)
	}
	if firstUser.Role != domain.RoleAdmin {
		t.Fatalf("İlk seed kullanıcısının rolü admin olmalı, bulunan: %s", firstUser.Role)
	}

	// 6 demo hesabı bulunmalı
	if len(users) < 6 {
		t.Fatalf("En az 6 seed kullanıcısı bekleniyordu, bulunan: %d", len(users))
	}
}

func TestSeed_LinesCardinality(t *testing.T) {
	lines := DefaultSeedLines()
	if len(lines) < 40 {
		t.Fatalf("En az 40 hat tanımlı olmalı, bulunan: %d", len(lines))
	}

	// Hat modları ve id uniqueness kontrolü
	seen := make(map[string]bool)
	for _, l := range lines {
		if seen[l.ID] {
			t.Fatalf("Mükerrer hat ID'si tespit edildi: %s", l.ID)
		}
		seen[l.ID] = true

		if l.Mode != domain.ModeBus && l.Mode != domain.ModeTram && l.Mode != domain.ModeFerry {
			t.Fatalf("Geçersiz hat modu: %s", l.Mode)
		}
	}
}
