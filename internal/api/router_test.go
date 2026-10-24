package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/settlement"
	"github.com/mamyylmz/AktarBilet/internal/sync"
)

func setupTestRouter() (http.Handler, *auth.AuthService) {
	authSvc := auth.NewAuthService("test-secret-12345678901234567890")
	pricingEngine := pricing.NewPricingEngine()
	ledSvc := ledger.NewMemoryLedger()
	syncSvc := sync.NewSyncService(pricingEngine, ledSvc, 5000)
	settleSvc := settlement.NewSettlementService()
	anomalyDet := anomaly.NewAnomalyDetector()

	h := NewAPIHandler(nil, authSvc, pricingEngine, ledSvc, syncSvc, settleSvc, anomalyDet)
	router := SetupRouter(h, "f85f26")
	return router, authSvc
}

func TestAPI_DDOStampHeader(t *testing.T) {
	router, _ := setupTestRouter()

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Health check 200 dönmeli, alınan: %d", w.Code)
	}

	// Zorunlu Sistem Damgası Testi: X-DDO-Stamp: f85f26
	stamp := w.Header().Get("X-DDO-Stamp")
	if stamp != "f85f26" {
		t.Fatalf("X-DDO-Stamp başlığı f85f26 olmalı, alınan: %s", stamp)
	}
}

func TestAPI_DDOMandatoryHealthEndpoint(t *testing.T) {
	router, _ := setupTestRouter()

	// Zorunlu Sistem Ucu: GET /_hz/f85f
	req := httptest.NewRequest("GET", "/_hz/f85f", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("GET /_hz/f85f 200 OK dönmeli, alınan: %d", w.Code)
	}
	if !strings.Contains(w.Body.String(), "f85f26") {
		t.Fatalf("GET /_hz/f85f yanıt gövdesinde f85f26 damgası bulunmalı: %s", w.Body.String())
	}
}

func TestAPI_RoleAuthorization(t *testing.T) {
	router, authSvc := setupTestRouter()

	// 1. Yetkisiz erişim testi (Token yok) -> 401 Unauthorized
	reqUnauthorized := httptest.NewRequest("POST", "/api/v1/penalties", strings.NewReader(`{}`))
	wUnauth := httptest.NewRecorder()
	router.ServeHTTP(wUnauth, reqUnauthorized)

	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("Tokensiz ceza yazma 401 dönmeli, alınan: %d", wUnauth.Code)
	}

	// 2. Yetkisiz Rol Testi: Yolcu rolüyle ceza yazma denemesi -> 403 Forbidden
	yolcuUser := &domain.User{
		ID:       uuid.New(),
		Email:    "yolcu@aktarbilet.gov.tr",
		FullName: "Yolcu Test",
		Role:     domain.RoleYolcu,
	}
	yolcuToken, _, _ := authSvc.GenerateTokens(yolcuUser)

	reqForbidden := httptest.NewRequest("POST", "/api/v1/penalties", strings.NewReader(`{}`))
	reqForbidden.Header.Set("Authorization", "Bearer "+yolcuToken)
	wForbidden := httptest.NewRecorder()
	router.ServeHTTP(wForbidden, reqForbidden)

	if wForbidden.Code != http.StatusForbidden {
		t.Fatalf("Yolcu rolüyle ceza yazma 403 Forbidden dönmeli, alınan: %d", wForbidden.Code)
	}

	// 3. Yetkili Rol Testi: Kontrolör rolüyle ceza yazma -> 201 Created (veya 400 bad request) ama 401/403 DEĞİL!
	kontrolorUser := &domain.User{
		ID:       uuid.New(),
		Email:    "kontrolor@aktarbilet.gov.tr",
		FullName: "Kontrolör Test",
		Role:     domain.RoleKontrolor,
	}
	kontrolorToken, _, _ := authSvc.GenerateTokens(kontrolorUser)

	reqAllowed := httptest.NewRequest("POST", "/api/v1/penalties", strings.NewReader(`{"card_id":"C1","amount_kurus":5000,"reason":"kartsiz_binis"}`))
	reqAllowed.Header.Set("Authorization", "Bearer "+kontrolorToken)
	wAllowed := httptest.NewRecorder()
	router.ServeHTTP(wAllowed, reqAllowed)

	if wAllowed.Code != http.StatusCreated {
		t.Fatalf("Kontrolör rolüyle ceza oluşturma 201 dönmeli, alınan: %d, body: %s", wAllowed.Code, wAllowed.Body.String())
	}
}
