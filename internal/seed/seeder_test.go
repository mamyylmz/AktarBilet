package seed

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/auth"
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

func TestSeed_StationsAndTariff(t *testing.T) {
	stations := DefaultSeedStations()
	if len(stations) < 5 {
		t.Fatalf("En az 5 durak olmalı, bulunan: %d", len(stations))
	}
	for _, s := range stations {
		if s.Latitude == 0 || s.Longitude == 0 {
			t.Fatalf("Durak koordinatları boş olamaz: %s", s.ID)
		}
	}

	tariff := DefaultSeedTariff()
	if tariff.BaseFareKurus != 2000 {
		t.Fatalf("Standart tam bilet 20.00 TL (2000 kuruş) olmalı, bulunan: %d", tariff.BaseFareKurus)
	}
	if tariff.StudentFareKurus != 1000 {
		t.Fatalf("Öğrenci bileti 10.00 TL (1000 kuruş) olmalı, bulunan: %d", tariff.StudentFareKurus)
	}
}

func TestSeed_ValidatorsAndCards(t *testing.T) {
	validators := DefaultSeedValidators()
	if len(validators) < 4 {
		t.Fatalf("En az 4 validatör tanımlı olmalı, bulunan: %d", len(validators))
	}

	cards := DefaultDemoCards(nil)
	if len(cards) < 4 {
		t.Fatalf("En az 4 demo kart olmalı, bulunan: %d", len(cards))
	}

	foundTam := false
	for _, c := range cards {
		if c.ID == "CARD_TAM_001" {
			foundTam = true
			if c.BalanceKurus != 50000 {
				t.Fatalf("CARD_TAM_001 bakiyesi 500.00 TL (50000 kuruş) olmalı, bulunan: %d", c.BalanceKurus)
			}
		}
	}
	if !foundTam {
		t.Fatal("CARD_TAM_001 demo kartı bulunamadı")
	}
}

func TestSeed_NilPool(t *testing.T) {
	err := SeedDatabase(nil, nil, nil)
	if err != nil {
		t.Fatalf("Nil pool ile SeedDatabase hata vermemeli, alınan: %v", err)
	}
}

func TestSeed_LiveDatabase(t *testing.T) {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@127.0.0.1:5432/aktarbilet?sslmode=disable"
	}
	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()

	pool, err := pgxpool.New(pingCtx, dbURL)
	if err != nil || pool.Ping(pingCtx) != nil {
		t.Skip("PostgreSQL test ortamına erişilemedi, SeedDatabase testi atlanıyor")
		return
	}
	defer pool.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	authSvc := auth.NewAuthService("test-secret-12345678901234567890")
	err = SeedDatabase(ctx, pool, authSvc)
	if err != nil {
		t.Fatalf("SeedDatabase başarısız: %v", err)
	}
}

type mockSeedRow struct {
	val any
}

func (m *mockSeedRow) Scan(dest ...any) error {
	if len(dest) > 0 && m.val != nil {
		switch d := dest[0].(type) {
		case *uuid.UUID:
			*d = uuid.New()
		case *int:
			*d = 0 // triggers bulk card generation (< 200000)
		}
	}
	return nil
}

type mockSeedExecutor struct{}

func (m *mockSeedExecutor) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	return pgconn.NewCommandTag("INSERT 0 1"), nil
}

func (m *mockSeedExecutor) QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row {
	return &mockSeedRow{val: 0}
}

func TestSeed_FullDatabaseExecution(t *testing.T) {
	ctx := context.Background()
	authSvc := auth.NewAuthService("test-secret-12345678901234567890")
	exec := &mockSeedExecutor{}

	err := SeedDatabaseWithExecutor(ctx, exec, authSvc)
	if err != nil {
		t.Fatalf("SeedDatabaseWithExecutor beklenmeyen hata: %v", err)
	}
}
