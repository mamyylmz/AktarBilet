package main

import (
	"context"
	"errors"
	"flag"
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

	resetFlag := flag.Bool("reset", false, "Veritabanını sıfırla ve başlangıç seed durumuna getir")
	flag.Parse()

	// 3. Veritabanı Bağlantı Havuzu (pgxpool)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	var pool *pgxpool.Pool
	var err error
	pool, err = pgxpool.New(ctx, cfg.DatabaseURL)
	if err != nil {
		slog.Warn("Veritabanı havuzu oluşturulamadı, in-memory modda devam ediliyor", "error", err)
		pool = nil
	} else if err := pool.Ping(ctx); err != nil {
		slog.Warn("Veritabanına bağlanılamadı, in-memory modda devam ediliyor", "error", err)
		pool = nil
	} else {
		slog.Info("PostgreSQL veritabanı bağlantısı başarılı.")
		if *resetFlag {
			slog.Info("Veritabanı tabloları sıfırlanıyor (reset-demo)...")
			_, _ = pool.Exec(ctx, `
				TRUNCATE TABLE jw_ledger, jw_validator_events, jw_penalties, jw_anomalies, jw_settlement_ledger, jw_audit_logs CASCADE;
				UPDATE jw_cards SET balance_kurus = 0, daily_taps_count = 0, monthly_capping_spent_kurus = 0, last_tap_at = NULL, last_station_id = NULL;
			`)
			if err := runMigrationsAndSeed(pool, cfg.JWTSecret); err != nil {
				slog.Error("Yeniden seed hatası", "error", err)
				os.Exit(1)
			}
			slog.Info("Veritabanı ve demo ortamı başarıyla sıfırlandı.")
			return
		}

		// Otomatik Migrasyon ve Seed
		if err := runMigrationsAndSeed(pool, cfg.JWTSecret); err != nil {
			slog.Error("Migrasyon veya seed hatası", "error", err)
		}
	}

	// 4. Servislerin Başlatılması
	authSvc := auth.NewAuthService(cfg.JWTSecret)
	pricingEngine := pricing.NewPricingEngine()

	var ledgerSvc ledger.LedgerService
	if pool != nil {
		ledgerSvc = ledger.NewPostgresLedger(pool)
	} else {
		ledgerSvc = ledger.NewMemoryLedger()
	}

	syncSvc := sync.NewSyncService(pricingEngine, ledgerSvc, 5000) // 50 TL negatif tolerans
	if pool != nil {
		syncSvc.SetPool(pool)
	}
	settleSvc := settlement.NewSettlementService()
	anomalyDet := anomaly.NewAnomalyDetector()

	// Durakları anomaly detector'e tanıt
	anomalyDet.RegisterStation(anomaly.StationCoordinate{ID: "STAT_KADIKOY", Name: "Kadıköy İskele", Latitude: 40.9904, Longitude: 29.0254})
	anomalyDet.RegisterStation(anomaly.StationCoordinate{ID: "STAT_EMINONU", Name: "Eminönü", Latitude: 41.0175, Longitude: 28.9734})
	anomalyDet.RegisterStation(anomaly.StationCoordinate{ID: "STAT_KABATAS", Name: "Kabataş", Latitude: 41.0355, Longitude: 28.9922})
	anomalyDet.RegisterStation(anomaly.StationCoordinate{ID: "STAT_SARIYER", Name: "Sarıyer", Latitude: 41.1685, Longitude: 29.0554})

	// 5. API Yönlendirici ve HTTP Sunucu
	handler := api.NewAPIHandler(pool, authSvc, pricingEngine, ledgerSvc, syncSvc, settleSvc, anomalyDet)
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

func runMigrationsAndSeed(pool *pgxpool.Pool, jwtSecret string) error {
	ctx := context.Background()

	// 001_initial_schema.sql oku ve uygula
	migration1, err := os.ReadFile("migrations/001_initial_schema.sql")
	if err == nil {
		if _, err := pool.Exec(ctx, string(migration1)); err != nil {
			slog.Warn("Migrasyon 1 çalıştırma uyarısı", "error", err)
		} else {
			slog.Info("001_initial_schema.sql başarıyla uygulandı.")
		}
	}

	// 002_views_and_indexes.sql oku ve uygula
	migration2, err := os.ReadFile("migrations/002_views_and_indexes.sql")
	if err == nil {
		if _, err := pool.Exec(ctx, string(migration2)); err != nil {
			slog.Warn("Migrasyon 2 çalıştırma uyarısı", "error", err)
		} else {
			slog.Info("002_views_and_indexes.sql başarıyla uygulandı.")
		}
	}

	// 003_constraints_and_triggers.sql oku ve uygula
	migration3, err := os.ReadFile("migrations/003_constraints_and_triggers.sql")
	if err == nil {
		if _, err := pool.Exec(ctx, string(migration3)); err != nil {
			slog.Warn("Migrasyon 3 çalıştırma uyarısı", "error", err)
		} else {
			slog.Info("003_constraints_and_triggers.sql başarıyla uygulandı.")
		}
	}

	// Seed verilerini çalıştır
	authSvc := auth.NewAuthService(jwtSecret)
	return seed.SeedDatabase(ctx, pool, authSvc)
}
