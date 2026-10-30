package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/api"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/config"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/seed"
	"github.com/mamyylmz/AktarBilet/internal/settlement"
	"github.com/mamyylmz/AktarBilet/internal/sync"
)

func main() {
	// 1. Yapılandırılmış JSON Loglama (slog)
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))
	slog.SetDefault(logger)

	slog.Info("AktarBilet Backend API (ddo-pelikan-api) başlatılıyor...")

	// 2. Konfigürasyonu yükle
	cfg := config.Load()

	if cfg.Environment == "production" && (cfg.JWTSecret == "super-secret-jwt-key-ddo-aktarbilet-2026" || len(cfg.JWTSecret) < 32) {
		slog.Error("FATAL GÜVENLİK HATASI: Üretim ortamında varsayılan veya zayıf (<32 karakter) JWT_SECRET kabul edilemez! Sunucu fail-closed modunda durduruluyor.")
		os.Exit(1)
	}

	deviceCredentials, err := sync.LoadDeviceCredentials(cfg.ValidatorCredentialsFile)
	if err != nil {
		slog.Error("FATAL: validator device credentials could not be loaded", "error", err)
		os.Exit(1)
	}

	resetFlag := flag.Bool("reset", false, "Veritabanını sıfırla ve başlangıç seed durumuna getir")
	flag.Parse()

	// 3. Veritabanı Bağlantı Havuzu (pgxpool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pool *pgxpool.Pool
	pool, err = pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Warn("PostgreSQL veritabanı havuzu oluşturulamadı, servis fail-closed (503) modunda başlatılıyor", "error", err)
		pool = nil
	} else if err := pool.Ping(ctx); err != nil {
		slog.Warn("PostgreSQL veritabanına ping atılamadı, servis fail-closed (503) modunda başlatılıyor", "error", err)
		pool.Close()
		pool = nil
	} else {
		slog.Info("PostgreSQL veritabanı bağlantısı başarılı.")
		if *resetFlag {
			slog.Info("Veritabanı tabloları sıfırlanıyor (reset-demo)...")
			_, _ = pool.Exec(ctx, `
				TRUNCATE TABLE jw_ledger, jw_validator_events, jw_penalties, jw_anomalies, jw_settlement_ledger, jw_audit_logs CASCADE;
				UPDATE jw_cards SET balance_kurus = 0, daily_taps_count = 0, monthly_capping_spent_kurus = 0, last_tap_at = NULL, last_station_id = NULL;
			`)
			if err := runMigrationsAndSeed(pool, cfg.JWTSecret, deviceCredentials); err != nil {
				slog.Error("Yeniden seed hatası", "error", err)
				os.Exit(1)
			}
			slog.Info("Veritabanı ve demo ortamı başarıyla sıfırlandı.")
			return
		}

		// Otomatik Migrasyon ve Seed
		if err := runMigrationsAndSeed(pool, cfg.JWTSecret, deviceCredentials); err != nil {
			slog.Error("Migrasyon veya seed hatası, sunucu durduruluyor", "error", err)
			os.Exit(1)
		}
	}

	// 4. Servislerin Başlatılması
	authSvc := auth.NewAuthService(cfg.JWTSecret)
	pricingEngine := pricing.NewPricingEngine()

	var ledgerSvc ledger.LedgerService
	if pool != nil {
		ledgerSvc = ledger.NewPostgresLedger(pool)
	} else {
		mem := ledger.NewMemoryLedger()
		for _, card := range seed.DefaultDemoCards(nil) {
			mem.RegisterCard(card)
		}
		ledgerSvc = mem
	}

	syncSvc := sync.NewSyncService(pricingEngine, ledgerSvc, 5000) // 50 TL negatif tolerans
	if pool != nil {
		syncSvc.SetPool(pool)
	}
	settleSvc := settlement.NewSettlementService()
	anomalyDet := anomaly.NewAnomalyDetector()

	// Seed validators use official ESHOT stop coordinates for anomaly distance checks.
	validatorStops := map[string]bool{"STAT_ESHOT_10177": true, "STAT_ESHOT_10145": true, "STAT_ESHOT_10008": true, "STAT_ESHOT_10255": true}
	for _, station := range seed.DefaultSeedStations() {
		if validatorStops[station.ID] {
			anomalyDet.RegisterStation(anomaly.StationCoordinate{
				ID: station.ID, Name: station.Name, Latitude: station.Latitude, Longitude: station.Longitude,
			})
		}
	}

	// 5. API Yönlendirici ve HTTP Sunucu
	deviceAuth := sync.NewDeviceAuthManager(deviceCredentials, pool)
	handler := api.NewAPIHandler(pool, authSvc, pricingEngine, ledgerSvc, syncSvc, settleSvc, anomalyDet)
	handler.SetDeviceAuth(deviceAuth)
	router := api.SetupRouter(handler, cfg.DDOStampHeader)

	srv := &http.Server{
		Addr:         ":" + cfg.Port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// 6. Graceful Shutdown Yönetimi
	serverErrors := make(chan error, 1)
	go func() {
		slog.Info("HTTP Sunucu dinlemede", "port", cfg.Port, "stamp", cfg.DDOStampHeader)
		serverErrors <- srv.ListenAndServe()
	}()

	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error("Sunucu çalışma hatası", "error", err)
		}
	case sig := <-shutdown:
		slog.Info("Düzgün kapanma (Graceful shutdown) sinyali alındı", "signal", sig.String())
		ctxShutdown, cancelShutdown := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancelShutdown()

		if err := srv.Shutdown(ctxShutdown); err != nil {
			slog.Error("Sunucu düzgün kapatılamadı, zorla sonlandırılıyor", "error", err)
			_ = srv.Close()
		}
		if pool != nil {
			pool.Close()
		}
		slog.Info("Sunucu ve veritabanı bağlantıları başarıyla kapatıldı.")
	}
}

func runMigrationsAndSeed(pool *pgxpool.Pool, jwtSecret string, deviceCredentials map[string]string) error {
	ctx := context.Background()

	// Migration version tracking table
	_, err := pool.Exec(ctx, `
		CREATE TABLE IF NOT EXISTS jw_schema_migrations (
			version VARCHAR(64) PRIMARY KEY,
			applied_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		);
	`)
	if err != nil {
		return fmt.Errorf("schema migrations tablosu oluşturulamadı: %w", err)
	}

	migrations := []struct {
		version string
		path    string
	}{
		{"001", "migrations/001_initial_schema.sql"},
		{"002", "migrations/002_views_and_indexes.sql"},
		{"003", "migrations/003_constraints_and_triggers.sql"},
		{"004", "migrations/004_security_and_integrity_hardening.sql"},
		{"005", "migrations/005_device_auth_and_optimizations.sql"},
		{"006", "migrations/006_security_and_approval_integrity.sql"},
		{"007", "migrations/007_approval_capacity_constraint.sql"},
		{"008", "migrations/008_multi_role_support.sql"},
		{"009", "migrations/009_fix_batch_device_nonce.sql"},
		{"010", "migrations/010_settlement_ledger_idempotency.sql"},
		{"011", "migrations/011_settlement_adjustments.sql"},
		{"012", "migrations/012_active_station_catalog.sql"},
	}

	for _, m := range migrations {
		var applied bool
		err := pool.QueryRow(ctx, "SELECT true FROM jw_schema_migrations WHERE version = $1", m.version).Scan(&applied)
		if err == nil && applied {
			continue
		}

		content, err := os.ReadFile(m.path)
		if err != nil {
			return fmt.Errorf("migrasyon dosyası okunamadı (%s): %w", m.path, err)
		}

		tx, err := pool.Begin(ctx)
		if err != nil {
			return fmt.Errorf("migrasyon transaction başlatılamadı (%s): %w", m.version, err)
		}

		if _, err := tx.Exec(ctx, string(content)); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrasyon çalıştırma hatası (%s): %w", m.version, err)
		}

		if _, err := tx.Exec(ctx, "INSERT INTO jw_schema_migrations (version) VALUES ($1)", m.version); err != nil {
			_ = tx.Rollback(ctx)
			return fmt.Errorf("migrasyon kaydı eklenemedi (%s): %w", m.version, err)
		}

		if err := tx.Commit(ctx); err != nil {
			return fmt.Errorf("migrasyon commit hatası (%s): %w", m.version, err)
		}

		slog.Info("Migrasyon başarıyla uygulandı", "version", m.version, "path", m.path)
	}

	// Seed verilerini çalıştır
	authSvc := auth.NewAuthService(jwtSecret)
	if err := seed.SeedDatabase(ctx, pool, authSvc); err != nil {
		return fmt.Errorf("veritabanı seed hatası: %w", err)
	}

	// Provision runtime-only, per-device credentials after validator seed data
	// exists. The server never derives credentials from a shared master key.
	if len(deviceCredentials) != len(seed.DefaultSeedValidators()) {
		return fmt.Errorf("validator credential count mismatch: got %d, want %d", len(deviceCredentials), len(seed.DefaultSeedValidators()))
	}
	for _, v := range seed.DefaultSeedValidators() {
		devKey, ok := deviceCredentials[v.ID]
		if !ok {
			return fmt.Errorf("validator credential is missing for %s", v.ID)
		}
		_, err := pool.Exec(ctx, `
				INSERT INTO jw_validator_credentials (validator_id, secret_key, is_revoked, last_sequence_no, created_at, updated_at)
				VALUES ($1, $2, false, 0, NOW(), NOW())
				ON CONFLICT (validator_id) DO UPDATE
				SET secret_key = EXCLUDED.secret_key, updated_at = NOW();
		`, v.ID, devKey)
		if err != nil {
			return fmt.Errorf("validatör kimlik bilgisi seed hatası (%s): %w", v.ID, err)
		}
	}

	return nil
}
