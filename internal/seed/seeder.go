package seed

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

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
			FullName: "Mehmet Demir (Örnek Yolcu)",
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
	return []domain.Line{
		{ID: "LINE_500T", Name: "500T Tuzla - Cevizlibağ", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_14ES", Name: "14ES Esenkent - Kadıköy", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_T1", Name: "T1 Kabataş - Bağcılar Tramvay", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_VAPUR_KE", Name: "Kadıköy - Eminönü Vapuru", Mode: domain.ModeFerry, Operator: "belediye", IsActive: true},
		{ID: "LINE_M1A", Name: "M1A Yenikapı - Atatürk Havalimanı Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M2", Name: "M2 Yenikapı - Hacıosman Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M3", Name: "M3 Kirazlı - Kayaşehir Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M4", Name: "M4 Kadıköy - Sabiha Gökçen Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M5", Name: "M5 Üsküdar - Çekmeköy Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M7", Name: "M7 Yıldız - Mahmutbey Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M8", Name: "M8 Bostancı - Parseller Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M9", Name: "M9 Ataköy - Olimpiyat Metro", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_M11", Name: "M11 Gayrettepe - İstanbul Havalimanı", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_T4", Name: "T4 Topkapı - Mescid-i Selam Tramvay", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_T5", Name: "T5 Eminönü - Alibeyköy Tramvay", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
		{ID: "LINE_VAPUR_BU", Name: "Beşiktaş - Üsküdar Vapuru", Mode: domain.ModeFerry, Operator: "belediye", IsActive: true},
		{ID: "LINE_VAPUR_KK", Name: "Kadıköy - Karaköy Vapuru", Mode: domain.ModeFerry, Operator: "belediye", IsActive: true},
		{ID: "LINE_VAPUR_BO", Name: "Boğaz Hattı Vapuru", Mode: domain.ModeFerry, Operator: "belediye", IsActive: true},
		{ID: "LINE_VAPUR_ADALAR", Name: "Bostancı - Adalar Vapuru", Mode: domain.ModeFerry, Operator: "belediye", IsActive: true},
		{ID: "LINE_34AS", Name: "34AS Avcılar - Söğütlüçeşme Metrobüs", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_34BZ", Name: "34BZ Beylikdüzü - Zincirlikuyu Metrobüs", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_34G", Name: "34G Beylikdüzü - Söğütlüçeşme Metrobüs", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_34Z", Name: "34Z Zincirlikuyu - Söğütlüçeşme Metrobüs", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_15F", Name: "15F Beykoz - Kadıköy", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_16D", Name: "16D Altkaynarca - Pendik - Kadıköy", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_19F", Name: "19F Fındıklı Mahallesi - Kadıköy", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_25T", Name: "25T Sarıyer - Taksim", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_40T", Name: "40T İstinye Dereiçi - Taksim", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_70FE", Name: "70FE Feriköy - Eminönü", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_89C", Name: "89C Başakşehir - Taksim", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_97GE", Name: "97GE Güneşli - Eminönü", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_145T", Name: "145T Beylikdüzü - Taksim", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_150", Name: "150 Hacıosman Metro - Sarıyer", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_151", Name: "151 Kilyos - Hacıosman Metro", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_152", Name: "152 Zekeriyaköy - Hacıosman Metro", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_153", Name: "153 Bahçeköy - Sarıyer", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_154", Name: "154 Hacıosman Metro - Koç Üniversitesi", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_250", Name: "250 Sabiha Gökçen - İzmit Otogar", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
		{ID: "LINE_500ES", Name: "500ES Tuzla - Esenler Otogar", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
		{ID: "LINE_E10", Name: "E-10 Sabiha Gökçen Havalimanı - Kadıköy", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
	}
}

func SeedDatabase(ctx context.Context, pool *pgxpool.Pool, authSvc *auth.AuthService) error {
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

	// 3. Duraklar
	stations := []domain.Station{
		{ID: "STAT_KADIKOY", Name: "Kadıköy İskele", Mode: domain.ModeFerry, Latitude: 40.9904, Longitude: 29.0254, ZoneID: 1},
		{ID: "STAT_EMINONU", Name: "Eminönü İskele & Meydan", Mode: domain.ModeFerry, Latitude: 41.0175, Longitude: 28.9734, ZoneID: 1},
		{ID: "STAT_KABATAS", Name: "Kabataş Tramvay", Mode: domain.ModeTram, Latitude: 41.0355, Longitude: 28.9922, ZoneID: 1},
		{ID: "STAT_BAGCIAR", Name: "Bağcılar Meydan", Mode: domain.ModeTram, Latitude: 41.0341, Longitude: 28.8572, ZoneID: 2},
		{ID: "STAT_AYRILIK", Name: "Ayrılık Çeşmesi", Mode: domain.ModeBus, Latitude: 40.9982, Longitude: 29.0305, ZoneID: 1},
		{ID: "STAT_SARIYER", Name: "Sarıyer Merkez", Mode: domain.ModeBus, Latitude: 41.1685, Longitude: 29.0554, ZoneID: 3},
	}

	for _, s := range stations {
		_, err := pool.Exec(ctx, `
			INSERT INTO jw_stations (id, name, mode, latitude, longitude, zone_id, created_at)
			VALUES ($1, $2, $3, $4, $5, $6, NOW())
			ON CONFLICT (id) DO UPDATE 
			SET name = EXCLUDED.name, latitude = EXCLUDED.latitude, longitude = EXCLUDED.longitude
		`, s.ID, s.Name, s.Mode, s.Latitude, s.Longitude, s.ZoneID)
		if err != nil {
			return fmt.Errorf("durak seed hatası (%s): %w", s.ID, err)
		}
	}

	// 4. Standart Tarife
	tariff := domain.Tariff{
		ID:                       "TARIFF_ISTANBUL_STD",
		Name:                     "İstanbul Şehir İçi Entegre Tarife",
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

	_, err := pool.Exec(ctx, `
		INSERT INTO jw_tariffs (
			id, name, mode, base_fare_kurus, student_fare_kurus, elderly_fare_kurus,
			transfer1_discount_pct, transfer2_discount_pct, night_surcharge_pct,
			student_daily_limit, monthly_capping_limit_kurus, is_active, created_at
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, NOW())
		ON CONFLICT (id) DO UPDATE 
		SET base_fare_kurus = EXCLUDED.base_fare_kurus, is_active = EXCLUDED.is_active
	`, tariff.ID, tariff.Name, tariff.Mode, tariff.BaseFareKurus, tariff.StudentFareKurus, tariff.ElderlyFareKurus,
		tariff.Transfer1DiscountPct, tariff.Transfer2DiscountPct, tariff.NightSurchargePct,
		tariff.StudentDailyLimit, tariff.MonthlyCappingLimitKurus, tariff.IsActive)
	if err != nil {
		return fmt.Errorf("tarife seed hatası: %w", err)
	}

	// 5. Validatörler
	validators := []struct {
		ID        string
		LineID    string
		StationID string
		Plate     string
	}{
		{ID: "VAL_500T_01", LineID: "LINE_500T", StationID: "STAT_AYRILIK", Plate: "34 HO 1001"},
		{ID: "VAL_14ES_01", LineID: "LINE_14ES", StationID: "STAT_KADIKOY", Plate: "34 HO 2002"},
		{ID: "VAL_T1_01", LineID: "LINE_T1", StationID: "STAT_KABATAS", Plate: "T1-TR-01"},
		{ID: "VAL_VAPUR_01", LineID: "LINE_VAPUR_KE", StationID: "STAT_KADIKOY", Plate: "SH-VAPUR-01"},
	}

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
	yolcuUID, hasYolcu := userIDs["yolcu@aktarbilet.gov.tr"]
	var yolcuUIDPtr *uuid.UUID
	if hasYolcu {
		yolcuUIDPtr = &yolcuUID
	}

	demoCards := []domain.Card{
		{
			ID:           "CARD_TAM_001",
			CardType:     domain.CardTypeTam,
			BalanceKurus: 50000, // 500.00 TL (Not assigned to yolcu user, owned independently)
			UserID:       nil,
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
		{
			ID:           "CARD_YOLCU_001",
			CardType:     domain.CardTypeTam,
			BalanceKurus: 50000, // 500.00 TL (Owned by yolcu@aktarbilet.gov.tr)
			UserID:       yolcuUIDPtr,
		},
	}

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

			// Başlangıç bakiye defter hareketi ekle (idempotent reference_id)
			if c.BalanceKurus > 0 {
				refID := fmt.Sprintf("SEED_INIT_%s", c.ID)
				_, err = pool.Exec(ctx, `
					INSERT INTO jw_ledger (id, card_id, entry_type, amount_kurus, balance_after_kurus, reference_id, description, created_at)
					VALUES ($1, $2, 'dolum', $3, $3, $4, 'Başlangıç bakiye yüklemesi', NOW())
					ON CONFLICT (reference_id) DO NOTHING
				`, uuid.New(), c.ID, c.BalanceKurus, refID)
				if err != nil {
					return fmt.Errorf("ledger seed hatası (%s): %w", c.ID, err)
				}
			}
		} else {
			// Kart zaten var; balance_kurus DEĞİŞTİRİLMEZ! Yalnızca kara liste durumu güncellenir.
			_, _ = pool.Exec(ctx, `
				UPDATE jw_cards 
				SET is_blacklisted = $1, blacklist_reason = $2, user_id = COALESCE(user_id, $3)
				WHERE id = $4
			`, c.IsBlacklisted, c.BlacklistReason, c.UserID, c.ID)
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

	slog.Info("Otomatik seed başarıyla tamamlandı.")
	return nil
}
