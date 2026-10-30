package seed

import (
	"context"
	"embed"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

// ESHOT route and stop records are embedded snapshots of the official İzmir
// Metropolitan Municipality open-data resources (CC BY 4.0).
//
//go:embed data/eshot-otobus-hatlari.csv data/eshot-otobus-duraklari.csv
var officialTransitData embed.FS

type DBExecutor interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	QueryRow(ctx context.Context, sql string, arguments ...any) pgx.Row
}

type SeedUser struct {
	Email    string
	Password string
	FullName string
	Role     domain.UserRole
}

func DefaultSeedUsers() []SeedUser {
	return []SeedUser{
		{
			Email:    "seed.sedir@ddo-case.invalid", // ZORUNLU SİSTEM CANARY KULLANICISI
			Password: "SedirPassword2026!",
			FullName: "Sedir Sistem Yöneticisi",
			Role:     domain.RoleAdmin,
		},
		{
			Email:    "yolcu@aktarbilet.gov.tr",
			Password: "YolcuPassword123!",
			FullName: "Mert Yılmaz (Demo Tam Kart)",
			Role:     domain.RoleYolcu,
		},
		{
			Email:    "ogrenci.demo@aktarbilet.gov.tr",
			Password: "OgrenciDemo2026!",
			FullName: "Ece Kaya (Demo Öğrenci Kartı)",
			Role:     domain.RoleYolcu,
		},
		{
			Email:    "yasli.demo@aktarbilet.gov.tr",
			Password: "YasliDemo2026!",
			FullName: "Ayşe Yıldız (Demo Yaşlı Kartı)",
			Role:     domain.RoleYolcu,
		},
		{
			Email:    "ucretsiz.demo@aktarbilet.gov.tr",
			Password: "UcretsizDemo2026!",
			FullName: "Deniz Aras (Demo Ücretsiz Kart)",
			Role:     domain.RoleYolcu,
		},
		{
			Email:    "kontrolor@aktarbilet.gov.tr",
			Password: "KontrolorPassword123!",
			FullName: "Ali Saha (Saha Kontrolörü)",
			Role:     domain.RoleKontrolor,
		},
		{
			Email:    "hatyoneticisi@aktarbilet.gov.tr",
			Password: "HatYoneticisi123!",
			FullName: "Zeynep Hat (Hat ve Tarife Amiri)",
			Role:     domain.RoleHatYoneticisi,
		},
		{
			Email:    "muhasebe@aktarbilet.gov.tr",
			Password: "MuhasebePassword123!",
			FullName: "Canan Finans (Gelir Paylaşım Müdürü)",
			Role:     domain.RoleMuhasebe,
		},
		{
			Email:    "operasyon@aktarbilet.gov.tr",
			Password: "OperasyonPassword123!",
			FullName: "Burak Sistem (Operasyon ve İzleme)",
			Role:     domain.RoleOperasyon,
		},
	}
}

func DefaultSeedLines() []domain.Line {
	records := readOfficialCSV("data/eshot-otobus-hatlari.csv", 6)
	lines := make([]domain.Line, 0, len(records)+4)
	seen := make(map[string]struct{}, len(records)+4)
	legacyIDs := make(map[string]struct{})
	for _, id := range LegacyIstanbulSeedLineIDs() {
		legacyIDs[id] = struct{}{}
	}
	for _, row := range records[1:] {
		number := strings.TrimSpace(row[0])
		name := strings.TrimSpace(row[1])
		if number == "" || name == "" {
			continue
		}
		id := "LINE_" + number
		if _, conflictsWithHistoricalIstanbulLine := legacyIDs[id]; conflictsWithHistoricalIstanbulLine {
			id = "LINE_ESHOT_" + number
		}
		if _, exists := seen[id]; exists {
			panic("duplicate ESHOT line number: " + number)
		}
		seen[id] = struct{}{}
		lines = append(lines, domain.Line{ID: id, Name: number + " " + name, Mode: domain.ModeBus, Operator: "belediye", IsActive: true})
	}
	// İzmir Büyükşehir Belediyesi confirms these operating tram corridors.
	lines = append(lines,
		domain.Line{ID: "LINE_TRAM_KONAK", Name: "Konak Tramvayı — Halkapınar - Fahrettin Altay", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		domain.Line{ID: "LINE_TRAM_KARSIYAKA", Name: "Karşıyaka Tramvayı — Alaybey - Flamingo", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		domain.Line{ID: "LINE_TRAM_CIGLI", Name: "Çiğli Tramvayı — Flamingo - Ataşehir", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		domain.Line{ID: "LINE_IZDENIZ_BOSTANLI_KONAK", Name: "İZDENİZ Bostanlı - Karşıyaka - Konak", Mode: domain.ModeFerry, Operator: "belediye", IsActive: true},
	)
	return lines
}

func LegacyIstanbulSeedLineIDs() []string {
	return []string{
		"LINE_500T", "LINE_14ES", "LINE_T1", "LINE_VAPUR_KE", "LINE_M1A", "LINE_M2", "LINE_M3", "LINE_M4",
		"LINE_M5", "LINE_M7", "LINE_M8", "LINE_M9", "LINE_M11", "LINE_T4", "LINE_T5", "LINE_VAPUR_BU",
		"LINE_VAPUR_KK", "LINE_VAPUR_BO", "LINE_VAPUR_ADALAR", "LINE_34AS", "LINE_34BZ", "LINE_34G", "LINE_34Z",
		"LINE_15F", "LINE_16D", "LINE_19F", "LINE_25T", "LINE_40T", "LINE_70FE", "LINE_89C", "LINE_97GE",
		"LINE_145T", "LINE_150", "LINE_151", "LINE_152", "LINE_153", "LINE_154", "LINE_250", "LINE_500ES", "LINE_E10",
	}
}

func DefaultSeedStations() []domain.Station {
	records := readOfficialCSV("data/eshot-otobus-duraklari.csv", 5)
	stations := make([]domain.Station, 0, len(records)-1)
	for _, row := range records[1:] {
		id, errID := strconv.Atoi(strings.TrimSpace(row[0]))
		lat, errLat := strconv.ParseFloat(strings.TrimSpace(row[2]), 64)
		lon, errLon := strconv.ParseFloat(strings.TrimSpace(row[3]), 64)
		name := strings.TrimSpace(row[1])
		if errID != nil || errLat != nil || errLon != nil || name == "" || lat < 37 || lat > 39 || lon < 26 || lon > 28 {
			continue
		}
		stations = append(stations, domain.Station{ID: fmt.Sprintf("STAT_ESHOT_%d", id), Name: name, Mode: domain.ModeBus, Latitude: lat, Longitude: lon, ZoneID: 1})
	}
	return stations
}

func readOfficialCSV(path string, expectedColumns int) [][]string {
	file, err := officialTransitData.Open(path)
	if err != nil {
		panic("official transit data unavailable: " + err.Error())
	}
	defer file.Close()
	reader := csv.NewReader(file)
	reader.Comma = ';'
	reader.FieldsPerRecord = expectedColumns
	reader.LazyQuotes = false
	var records [][]string
	for {
		row, err := reader.Read()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			panic("invalid official transit CSV " + path + ": " + err.Error())
		}
		records = append(records, row)
	}
	return records
}

func DefaultSeedTariff() domain.Tariff {
	return domain.Tariff{
		ID:                       "TARIFF_IZMIR_STD",
		Name:                     "AktarBilet Simülasyon Tarifesi",
		Mode:                     domain.ModeBus,
		BaseFareKurus:            2000,   // 20.00 TL
		StudentFareKurus:         1000,   // 10.00 TL
		ElderlyFareKurus:         1500,   // 15.00 TL
		Transfer1DiscountPct:     50,     // %50 (10.00 TL)
		Transfer2DiscountPct:     75,     // %75 (5.00 TL)
		NightSurchargePct:        50,     // %50 (30.00 TL)
		StudentDailyLimit:        10,     // 10 indirimli biniş
		MonthlyCappingLimitKurus: 150000, // 1500.00 TL tavan
		IsActive:                 true,
	}
}

type SeedValidator struct {
	ID        string
	LineID    string
	StationID string
	Plate     string
}

func DefaultSeedValidators() []SeedValidator {
	return []SeedValidator{
		{ID: "VAL_500T_01", LineID: "LINE_5", StationID: "STAT_ESHOT_10177", Plate: "IZM-DEMO-01"},
		{ID: "VAL_14ES_01", LineID: "LINE_10", StationID: "STAT_ESHOT_10145", Plate: "IZM-DEMO-02"},
		{ID: "VAL_T1_01", LineID: "LINE_15", StationID: "STAT_ESHOT_10008", Plate: "IZM-DEMO-03"},
		{ID: "VAL_VAPUR_01", LineID: "LINE_16", StationID: "STAT_ESHOT_10255", Plate: "IZM-DEMO-04"},
	}
}

func DefaultDemoCards(yolcuUID *uuid.UUID) []domain.Card {
	userIDs := make(map[string]uuid.UUID)
	if yolcuUID != nil {
		userIDs["yolcu@aktarbilet.gov.tr"] = *yolcuUID
	}
	return DefaultDemoCardsForUsers(userIDs)
}

func DefaultDemoCardsForUsers(userIDs map[string]uuid.UUID) []domain.Card {
	userID := func(email string) *uuid.UUID {
		id, ok := userIDs[email]
		if !ok {
			return nil
		}
		return &id
	}
	return []domain.Card{
		// Unassigned compatibility fixture retained for existing inspection and API scenarios.
		{
			ID:           "CARD_TAM_001",
			CardType:     domain.CardTypeTam,
			BalanceKurus: 50000, // 500.00 TL (Not assigned to yolcu user, owned independently)
			UserID:       nil,
		},
		{
			ID:           "CARD_YOLCU_001",
			CardType:     domain.CardTypeTam,
			BalanceKurus: 50000,
			UserID:       userID("yolcu@aktarbilet.gov.tr"),
		},
		{
			ID:           "CARD_DEMO_OGRENCI_001",
			CardType:     domain.CardTypeOgrenci,
			BalanceKurus: 25000,
			UserID:       userID("ogrenci.demo@aktarbilet.gov.tr"),
		},
		{
			ID:           "CARD_DEMO_YASLI_001",
			CardType:     domain.CardTypeYasli,
			BalanceKurus: 15000,
			UserID:       userID("yasli.demo@aktarbilet.gov.tr"),
		},
		{
			ID:           "CARD_DEMO_UCRETSIZ_001",
			CardType:     domain.CardTypeUcretsiz,
			BalanceKurus: 0,
			UserID:       userID("ucretsiz.demo@aktarbilet.gov.tr"),
		},
		{
			ID:           "CARD_OGRENCI_001",
			CardType:     domain.CardTypeOgrenci,
			BalanceKurus: 25000, // 250.00 TL
			UserID:       nil,
		},
		{
			ID:           "CARD_YASLI_001",
			CardType:     domain.CardTypeYasli,
			BalanceKurus: 15000, // 150.00 TL
			UserID:       nil,
		},
		{
			ID:              "CARD_KARA_LISTE_001",
			CardType:        domain.CardTypeTam,
			BalanceKurus:    0,
			IsBlacklisted:   true,
			BlacklistReason: "Sahte kimlik tespiti (Kara liste demo)",
			UserID:          nil,
		},
	}
}

func SeedDatabase(ctx context.Context, pool *pgxpool.Pool, authSvc *auth.AuthService) error {
	if pool == nil {
		return nil
	}
	return SeedDatabaseWithExecutor(ctx, pool, authSvc)
}

func SeedDatabaseWithExecutor(ctx context.Context, pool DBExecutor, authSvc *auth.AuthService) error {
	slog.Info("Otomatik seed işlemi başlatılıyor...")

	// 1. Kullanıcılar (System Canary: seed.sedir@ddo-case.invalid İLK sırada olmalıdır!)
	users := DefaultSeedUsers()

	userIDs := make(map[string]uuid.UUID)
	for _, u := range users {
		hash, err := authSvc.HashPassword(u.Password)
		if err != nil {
			return fmt.Errorf("şifre özetlenemedi (%s): %w", u.Email, err)
		}

		var uid uuid.UUID
		err = pool.QueryRow(ctx, `
			INSERT INTO jw_tf_users (id, email, password_hash, full_name, role, created_at, updated_at)
			VALUES (gen_random_uuid(), $1, $2, $3, $4, NOW(), NOW())
			ON CONFLICT (email) DO UPDATE 
			SET password_hash = EXCLUDED.password_hash, full_name = EXCLUDED.full_name, role = EXCLUDED.role
			RETURNING id
		`, u.Email, hash, u.FullName, u.Role).Scan(&uid)
		if err != nil {
			return fmt.Errorf("kullanıcı seed hatası (%s): %w", u.Email, err)
		}
		userIDs[u.Email] = uid

		// jw_user_roles tablosuna rol eşleştirmesi
		_, _ = pool.Exec(ctx, `
			INSERT INTO jw_user_roles (user_id, role)
			VALUES ($1, $2)
			ON CONFLICT (user_id, role) DO NOTHING
		`, uid, u.Role)

		if u.Email == "seed.sedir@ddo-case.invalid" {
			for _, r := range []string{"admin", "yolcu", "kontrolor", "hat_yoneticisi", "muhasebe", "operasyon"} {
				_, _ = pool.Exec(ctx, `
					INSERT INTO jw_user_roles (user_id, role)
					VALUES ($1, $2)
					ON CONFLICT (user_id, role) DO NOTHING
				`, uid, r)
			}
		}
	}

	// 2. 40 Hat (Otobüs, Tramvay, Vapur)
	lines := DefaultSeedLines()

	for _, l := range lines {
		_, err := pool.Exec(ctx, `
			INSERT INTO jw_lines (id, name, mode, operator, is_active, created_at)
			VALUES ($1, $2, $3, $4, $5, NOW())
			ON CONFLICT (id) DO UPDATE 
			SET name = EXCLUDED.name, operator = EXCLUDED.operator, is_active = EXCLUDED.is_active
		`, l.ID, l.Name, l.Mode, l.Operator, l.IsActive)
		if err != nil {
			return fmt.Errorf("hat seed hatası (%s): %w", l.ID, err)
		}
	}
	if _, err := pool.Exec(ctx, `
		UPDATE jw_lines
		SET is_active = false,
		    name = CASE id
		        WHEN 'LINE_152' THEN '152 Zekeriyaköy - Hacıosman Metro'
		        WHEN 'LINE_153' THEN '153 Bahçeköy - Sarıyer'
		        WHEN 'LINE_154' THEN '154 Hacıosman Metro - Koç Üniversitesi'
		        ELSE name
		    END
		WHERE id = ANY($1::varchar[])
	`, LegacyIstanbulSeedLineIDs()); err != nil {
		return fmt.Errorf("eski İstanbul hatları pasife alınamadı: %w", err)
	}

	// 3. Duraklar
	stations := DefaultSeedStations()

	for offset := 0; offset < len(stations); offset += 500 {
		end := offset + 500
		if end > len(stations) {
			end = len(stations)
		}
		batch := stations[offset:end]
		ids, names, latitudes, longitudes := make([]string, len(batch)), make([]string, len(batch)), make([]string, len(batch)), make([]string, len(batch))
		for i, station := range batch {
			ids[i], names[i] = station.ID, station.Name
			latitudes[i] = strconv.FormatFloat(station.Latitude, 'f', -1, 64)
			longitudes[i] = strconv.FormatFloat(station.Longitude, 'f', -1, 64)
		}
		_, err := pool.Exec(ctx, `
			INSERT INTO jw_stations (id, name, mode, latitude, longitude, zone_id, is_active, created_at)
			SELECT id, name, 'bus', latitude::numeric, longitude::numeric, 1, true, NOW()
			FROM unnest($1::text[], $2::text[], $3::text[], $4::text[]) AS s(id, name, latitude, longitude)
			ON CONFLICT (id) DO UPDATE SET name = EXCLUDED.name, mode = EXCLUDED.mode,
				latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude, is_active = true
		`, ids, names, latitudes, longitudes)
		if err != nil {
			return fmt.Errorf("ESHOT durakları seed edilemedi (batch %d): %w", offset/500+1, err)
		}
	}
	if _, err := pool.Exec(ctx, `UPDATE jw_stations SET is_active = false WHERE id IN ('STAT_KADIKOY','STAT_EMINONU','STAT_KABATAS','STAT_BAGCIAR','STAT_AYRILIK','STAT_SARIYER')`); err != nil {
		return fmt.Errorf("eski İstanbul durakları pasife alınamadı: %w", err)
	}

	// 4. Standart Tarife
	tariff := DefaultSeedTariff()

	_, err := pool.Exec(ctx, `
		INSERT INTO jw_tariffs (
			id, name, mode, base_fare_kurus, student_fare_kurus, elderly_fare_kurus,
			transfer1_discount_pct, transfer2_discount_pct, night_surcharge_pct,
			student_daily_limit, monthly_capping_limit_kurus, is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (id) DO UPDATE 
		SET name = EXCLUDED.name, base_fare_kurus = EXCLUDED.base_fare_kurus, is_active = EXCLUDED.is_active
	`, tariff.ID, tariff.Name, tariff.Mode, tariff.BaseFareKurus, tariff.StudentFareKurus, tariff.ElderlyFareKurus,
		tariff.Transfer1DiscountPct, tariff.Transfer2DiscountPct, tariff.NightSurchargePct,
		tariff.StudentDailyLimit, tariff.MonthlyCappingLimitKurus, tariff.IsActive)
	if err != nil {
		return fmt.Errorf("tarife seed hatası: %w", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE jw_tariffs SET is_active = false WHERE id = 'TARIFF_ISTANBUL_STD'`); err != nil {
		return fmt.Errorf("eski İstanbul tarifesi pasife alınamadı: %w", err)
	}

	// 5. Validatörler
	validators := DefaultSeedValidators()

	for _, v := range validators {
		_, err := pool.Exec(ctx, `
			INSERT INTO jw_validators (id, line_id, station_id, vehicle_plate, is_active, created_at)
			VALUES ($1, $2, $3, $4, true, NOW())
			ON CONFLICT (id) DO UPDATE 
			SET line_id = EXCLUDED.line_id, station_id = EXCLUDED.station_id, vehicle_plate = EXCLUDED.vehicle_plate
		`, v.ID, v.LineID, v.StationID, v.Plate)
		if err != nil {
			return fmt.Errorf("validatör seed hatası (%s): %w", v.ID, err)
		}
	}

	// 6. Demo Kartlar
	demoCards := DefaultDemoCardsForUsers(userIDs)

	for _, c := range demoCards {
		// Kart daha önce eklenmiş mi kontrol et
		var existingCount int
		err := pool.QueryRow(ctx, "SELECT COUNT(*) FROM jw_cards WHERE id = $1", c.ID).Scan(&existingCount)
		if err == nil && existingCount == 0 {
			// Yeni kart oluştur
			_, err = pool.Exec(ctx, `
				INSERT INTO jw_cards (id, user_id, card_type, balance_kurus, is_blacklisted, blacklist_reason, created_at, updated_at)
				VALUES ($1, $2, $3, $4, $5, $6, NOW(), NOW())
			`, c.ID, c.UserID, c.CardType, c.BalanceKurus, c.IsBlacklisted, c.BlacklistReason)
			if err != nil {
				return fmt.Errorf("kart seed hatası (%s): %w", c.ID, err)
			}
		} else {
			// Kart zaten var; balance_kurus DEĞİŞTİRİLMEZ! Yalnızca kara liste durumu güncellenir.
			_, _ = pool.Exec(ctx, `
				UPDATE jw_cards 
				SET is_blacklisted = $1, blacklist_reason = $2, user_id = COALESCE(user_id, $3)
				WHERE id = $4
			`, c.IsBlacklisted, c.BlacklistReason, c.UserID, c.ID)
		}

		// Başlangıç bakiye defter hareketi ekle (idempotent reference_id)
		if c.BalanceKurus > 0 {
			refID := fmt.Sprintf("SEED_INIT_%s", c.ID)
			_, err = pool.Exec(ctx, `
				INSERT INTO jw_ledger (id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at)
				VALUES ($1, $2, 'dolum', $3, $3, $4, 'Başlangıç bakiye yüklemesi', NOW())
				ON CONFLICT (reference_id) DO NOTHING
			`, uuid.New(), c.ID, c.BalanceKurus, refID)
			if err != nil {
				slog.Warn("Başlangıç ledger seed uyarısı", "card_id", c.ID, "error", err)
			}
			// Kartın bakiyesini defter toplamına eşitle (kart_bakiyesi == SUM(jw_ledger.amount) değişmezi)
			_, _ = pool.Exec(ctx, `
				UPDATE jw_cards 
				SET balance_kurus = (SELECT COALESCE(SUM(amount_kurus), 0) FROM jw_ledger WHERE card_id = $1)
				WHERE id = $1
			`, c.ID)
		}
	}

	// 7. 200.000 Kart (Kapasite gereksinimi: balance_kurus=0, dolayısıyla ledger sum=0 değişmezi %100 korunur)
	var totalCards int
	_ = pool.QueryRow(ctx, "SELECT COUNT(*) FROM jw_cards").Scan(&totalCards)
	if totalCards < 200000 {
		slog.Info("200.000 kart bulk üretim başlatılıyor...")
		_, err = pool.Exec(ctx, `
			INSERT INTO jw_cards (id, card_type, balance_kurus, is_blacklisted, created_at, updated_at)
			SELECT 
				'CARD_SIM_' || LPAD(s::text, 6, '0'),
				'tam',
				0,
				false,
				NOW(),
				NOW()
			FROM generate_series(1, 200000) AS s
			ON CONFLICT (id) DO NOTHING;
		`)
		if err != nil {
			slog.Warn("200.000 kart bulk seed uyarısı", "error", err)
		} else {
			slog.Info("200.000 kart başarıyla oluşturuldu.")
		}
	}

	// 7. Gelir Paylaşımı Dönemleri:
	// - 2026-EYLUL: Geçmiş kapalı dönem (T-4 Storno ve denetim testleri için hazır)
	// - 2026-EKIM: Aktif açık dönem (Kapanış -> Reopen yaşam döngüsü için hazır)
	_, err = pool.Exec(ctx, `
		INSERT INTO jw_settlement_periods (
			id, period_name, start_date, end_date, is_closed, closed_at,
			total_revenue_kurus, belediye_share_kurus, ozel_halk_share_kurus, created_at
		) VALUES 
		(
			'00000000-0000-0000-0000-000000000000', '2026-EYLUL',
			DATE_TRUNC('month', NOW()) - INTERVAL '1 month', DATE_TRUNC('month', NOW()), true, DATE_TRUNC('month', NOW()),
			146860000, 85160000, 36900000, NOW() - INTERVAL '30 days'
		),
		(
			'00000000-0000-0000-0000-000000000001', '2026-EKIM',
			DATE_TRUNC('month', NOW()), DATE_TRUNC('month', NOW()) + INTERVAL '1 month', false, NULL,
			146860000, 85160000, 36900000, NOW()
		) ON CONFLICT (id) DO UPDATE SET
			period_name = EXCLUDED.period_name,
			start_date = EXCLUDED.start_date,
			end_date = EXCLUDED.end_date;
	`)
	if err != nil {
		slog.Warn("Settlement periods seed uyarısı", "error", err)
	}

	slog.Info("Otomatik seed başarıyla tamamlandı.")
	return nil
}
