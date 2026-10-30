package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/settlement"
	"github.com/mamyylmz/AktarBilet/internal/sync"
)

func getLiveAPITestPool(t *testing.T) *pgxpool.Pool {
	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@127.0.0.1:5432/aktarbilet?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil || pool.Ping(ctx) != nil {
		t.Skip("PostgreSQL test ortamına erişilemedi, Live API testleri atlanıyor")
		return nil
	}
	return pool
}

func TestAPI_LiveDatabaseEndpoints(t *testing.T) {
	pool := getLiveAPITestPool(t)
	defer pool.Close()

	ctx := context.Background()
	authSvc := auth.NewAuthService("test-secret-12345678901234567890")
	pricingEngine := pricing.NewPricingEngine()
	ledSvc := ledger.NewPostgresLedger(pool)
	syncSvc := sync.NewSyncService(pricingEngine, ledSvc, 5000)
	syncSvc.SetPool(pool)
	settleSvc := settlement.NewSettlementService()
	anomalyDet := anomaly.NewAnomalyDetector()

	h := NewAPIHandler(pool, authSvc, pricingEngine, ledSvc, syncSvc, settleSvc, anomalyDet)
	router := SetupRouter(h, "f85f26")

	// Get seeded user IDs for foreign keys
	var kontrolorID, adminID, muhasebeID string
	_ = pool.QueryRow(ctx, "SELECT id FROM jw_tf_users WHERE role = 'kontrolor' LIMIT 1").Scan(&kontrolorID)
	_ = pool.QueryRow(ctx, "SELECT id FROM jw_tf_users WHERE role = 'admin' LIMIT 1").Scan(&adminID)
	_ = pool.QueryRow(ctx, "SELECT id FROM jw_tf_users WHERE role = 'muhasebe' LIMIT 1").Scan(&muhasebeID)

	kontrolorUID, _ := uuid.Parse(kontrolorID)
	adminUID, _ := uuid.Parse(adminID)
	muhasebeUID, _ := uuid.Parse(muhasebeID)

	// Admin and Kontrolor users
	adminToken, _, _ := authSvc.GenerateTokens(&domain.User{ID: adminUID, Role: domain.RoleAdmin})
	kontrolorToken, _, _ := authSvc.GenerateTokens(&domain.User{ID: kontrolorUID, Role: domain.RoleKontrolor})
	muhasebeToken, _, _ := authSvc.GenerateTokens(&domain.User{ID: muhasebeUID, Role: domain.RoleMuhasebe})

	// 1. Health check with healthy DB
	reqHealth := httptest.NewRequest("GET", "/health", nil)
	wHealth := httptest.NewRecorder()
	router.ServeHTTP(wHealth, reqHealth)
	if wHealth.Code != http.StatusOK {
		t.Fatalf("Health check 200 dönmeli, alınan: %d", wHealth.Code)
	}

	// 2. ListLines from DB
	reqLines := httptest.NewRequest("GET", "/api/v1/lines", nil)
	wLines := httptest.NewRecorder()
	router.ServeHTTP(wLines, reqLines)
	if wLines.Code != http.StatusOK {
		t.Fatalf("ListLines 200 dönmeli, alınan: %d", wLines.Code)
	}

	// 3. ListStations from DB
	reqStations := httptest.NewRequest("GET", "/api/v1/stations", nil)
	wStations := httptest.NewRecorder()
	router.ServeHTTP(wStations, reqStations)
	if wStations.Code != http.StatusOK {
		t.Fatalf("ListStations 200 dönmeli, alınan: %d", wStations.Code)
	}

	// 4. ListTariffs from DB
	reqTariffs := httptest.NewRequest("GET", "/api/v1/tariffs", nil)
	wTariffs := httptest.NewRecorder()
	router.ServeHTTP(wTariffs, reqTariffs)
	if wTariffs.Code != http.StatusOK {
		t.Fatalf("ListTariffs 200 dönmeli, alınan: %d", wTariffs.Code)
	}

	// 5. CreatePenalty with real card in DB
	penBody, _ := json.Marshal(map[string]interface{}{
		"card_id":              "CARD_TAM_001",
		"penalty_amount_kurus": 5000,
		"reason":               "kartsiz_binis",
		"validator_id":         "VAL_500T_01",
	})
	reqPenalty := httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(penBody))
	reqPenalty.Header.Set("Authorization", "Bearer "+kontrolorToken)
	wPenalty := httptest.NewRecorder()
	router.ServeHTTP(wPenalty, reqPenalty)
	if wPenalty.Code != http.StatusCreated {
		t.Fatalf("Ceza yazma 201 Created dönmeli, alınan: %d (%s)", wPenalty.Code, wPenalty.Body.String())
	}

	// 6. ListPenalties from DB
	reqListPen := httptest.NewRequest("GET", "/api/v1/penalties", nil)
	reqListPen.Header.Set("Authorization", "Bearer "+adminToken)
	wListPen := httptest.NewRecorder()
	router.ServeHTTP(wListPen, reqListPen)
	if wListPen.Code != http.StatusOK {
		t.Fatalf("ListPenalties 200 dönmeli, alınan: %d", wListPen.Code)
	}

	// 7. GetHeatmap from DB
	reqHeatmap := httptest.NewRequest("GET", "/api/v1/reports/heatmap", nil)
	wHeatmap := httptest.NewRecorder()
	router.ServeHTTP(wHeatmap, reqHeatmap)
	if wHeatmap.Code != http.StatusOK {
		t.Fatalf("GetHeatmap 200 dönmeli, alınan: %d", wHeatmap.Code)
	}

	// 8. CloseSettlementPeriod and Storno in DB
	var periodID string
	err := pool.QueryRow(ctx, `
		INSERT INTO jw_settlement_periods (period_name, start_date, end_date)
		VALUES ($1, NOW() - INTERVAL '1 day', NOW())
		ON CONFLICT (period_name) DO UPDATE SET is_closed = false, closed_at = NULL
		RETURNING id
	`, fmt.Sprintf("LIVE_PERIOD_%d", time.Now().UnixNano())).Scan(&periodID)
	if err == nil {
		closeBody, _ := json.Marshal(map[string]string{"period_id": periodID})
		reqClose := httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(closeBody))
		reqClose.Header.Set("Authorization", "Bearer "+muhasebeToken)
		wClose := httptest.NewRecorder()
		router.ServeHTTP(wClose, reqClose)
		if wClose.Code != http.StatusOK {
			t.Fatalf("Dönem kapatma 200 dönmeli, alınan: %d (%s)", wClose.Code, wClose.Body.String())
		}

		stornoBody, _ := json.Marshal(map[string]interface{}{
			"period_id":     periodID,
			"operator":      "belediye",
			"storno_amount": -500,
			"reason":        "Ceza düzeltmesi stornosu",
		})
		reqStorno := httptest.NewRequest("POST", "/api/v1/settlements/storno", bytes.NewReader(stornoBody))
		reqStorno.Header.Set("Authorization", "Bearer "+muhasebeToken)
		wStorno := httptest.NewRecorder()
		router.ServeHTTP(wStorno, reqStorno)
		if wStorno.Code != http.StatusOK {
			t.Fatalf("Storno 200 dönmeli, alınan: %d (%s)", wStorno.Code, wStorno.Body.String())
		}

		// Anahtar verilmeyen bağımsız storno istekleri aynı boş anahtarı paylaşmamalı.
		reqStornoAgain := httptest.NewRequest("POST", "/api/v1/settlements/storno", bytes.NewReader(stornoBody))
		reqStornoAgain.Header.Set("Authorization", "Bearer "+muhasebeToken)
		wStornoAgain := httptest.NewRecorder()
		router.ServeHTTP(wStornoAgain, reqStornoAgain)
		if wStornoAgain.Code != http.StatusOK {
			t.Fatalf("Anahtarsız bağımsız storno 200 dönmeli, alınan: %d (%s)", wStornoAgain.Code, wStornoAgain.Body.String())
		}
		var unkeyedStornoCount int
		if err := pool.QueryRow(ctx, `
			SELECT count(*) FROM jw_settlement_ledger
			WHERE period_id = $1 AND is_storno = true AND idempotency_key IS NULL
		`, periodID).Scan(&unkeyedStornoCount); err != nil {
			t.Fatalf("Anahtarsız storno kayıtları sorgulanamadı: %v", err)
		}
		if unkeyedStornoCount != 2 {
			t.Fatalf("İki ayrı anahtarsız storno ayrı kaydedilmeli, bulunan: %d", unkeyedStornoCount)
		}
	}
}
