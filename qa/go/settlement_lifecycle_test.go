package goqa

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/api"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/settlement"
	"github.com/mamyylmz/AktarBilet/internal/sync"
)

type SettlementTestRig struct {
	router     http.Handler
	authSvc    *auth.AuthService
	settleSvc  *settlement.SettlementService
	adminUser  *domain.User
	muhasebe   *domain.User
	yolcuUser  *domain.User
	adminToken string
	muhToken   string
	yolcuToken string
}

func setupSettlementRig() *SettlementTestRig {
	authSvc := auth.NewAuthService("settlement-test-secret-key-32chars!!")
	pricingEngine := pricing.NewPricingEngine()
	ledSvc := ledger.NewMemoryLedger()
	syncSvc := sync.NewSyncService(pricingEngine, ledSvc, 5000)
	settleSvc := settlement.NewSettlementService()
	anomalyDet := anomaly.NewAnomalyDetector()

	adminUser := &domain.User{
		ID:       uuid.New(),
		Email:    "seed.sedir@ddo-case.invalid",
		FullName: "Sistem Yöneticisi",
		Role:     domain.RoleAdmin,
	}
	muhasebe := &domain.User{
		ID:       uuid.New(),
		Email:    "muhasebe@aktarbilet.gov.tr",
		FullName: "Canan Finans",
		Role:     domain.RoleMuhasebe,
	}
	yolcuUser := &domain.User{
		ID:       uuid.New(),
		Email:    "yolcu@aktarbilet.gov.tr",
		FullName: "Mehmet Demir",
		Role:     domain.RoleYolcu,
	}

	adminTok, _, _ := authSvc.GenerateTokens(adminUser)
	muhTok, _, _ := authSvc.GenerateTokens(muhasebe)
	yolcuTok, _, _ := authSvc.GenerateTokens(yolcuUser)

	h := api.NewAPIHandler(nil, authSvc, pricingEngine, ledSvc, syncSvc, settleSvc, anomalyDet)
	router := api.SetupRouter(h, "f85f26")

	return &SettlementTestRig{
		router:     router,
		authSvc:    authSvc,
		settleSvc:  settleSvc,
		adminUser:  adminUser,
		muhasebe:   muhasebe,
		yolcuUser:  yolcuUser,
		adminToken: adminTok,
		muhToken:   muhTok,
		yolcuToken: yolcuTok,
	}
}

func TestSettlement_FullLifecycle_Audit(t *testing.T) {
	rig := setupSettlementRig()
	periodID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// 1. Unauthenticated GET /settlements must be 401
	{
		req := httptest.NewRequest("GET", "/api/v1/settlements", nil)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("[1. Unauth GET] 401 bekleniyordu, alınan: %d", w.Code)
		}
	}

	// 2. Yolcu GET /settlements must be 403 Forbidden
	{
		req := httptest.NewRequest("GET", "/api/v1/settlements", nil)
		req.Header.Set("Authorization", "Bearer "+rig.yolcuToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("[2. Yolcu GET] 403 bekleniyordu, alınan: %d", w.Code)
		}
	}

	// 3. Muhasebe GET /settlements must be 200 OK
	{
		req := httptest.NewRequest("GET", "/api/v1/settlements", nil)
		req.Header.Set("Authorization", "Bearer "+rig.muhToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("[3. Muhasebe GET] 200 bekleniyordu, alınan: %d", w.Code)
		}
	}

	// 4. Initial Close Period with Muhasebe Token
	{
		closePayload := map[string]interface{}{
			"period_id":   periodID.String(),
			"period_name": "2026-EKIM",
		}
		body, _ := json.Marshal(closePayload)
		req := httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rig.muhToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("[4. Close Period 1] 200 bekleniyordu, alınan: %d, body: %s", w.Code, w.Body.String())
		}
	}

	// 5. Duplicate Close on already closed period:
	// Verify behavior: does it return 409 ALREADY_CLOSED?
	{
		closePayload := map[string]interface{}{
			"period_id":   periodID.String(),
			"period_name": "2026-EKIM",
		}
		body, _ := json.Marshal(closePayload)
		req := httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rig.muhToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		t.Logf("[5. Duplicate Close] status code: %d, body: %s", w.Code, w.Body.String())
		if w.Code != http.StatusConflict {
			t.Errorf("[5. Duplicate Close] 409 Conflict bekleniyordu, alınan: %d", w.Code)
		}
	}

	// 6. Close period WITHOUT period_id (only period_name)
	{
		closePayload := map[string]interface{}{
			"period_name": "2026-EKIM",
		}
		body, _ := json.Marshal(closePayload)
		req := httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rig.muhToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		t.Logf("[6. Close without period_id] status code: %d, body: %s", w.Code, w.Body.String())
		// Current bug: fails with 400 INVALID_UUID because period_id is required
	}

	// 7. Reopen Period with valid period_name and reason
	{
		reopenPayload := map[string]interface{}{
			"period_name": "2026-EKIM",
			"reason":      "Yıllık finansal denetim ve mutabakat incelemesi",
		}
		body, _ := json.Marshal(reopenPayload)
		req := httptest.NewRequest("POST", "/api/v1/settlements/reopen", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rig.muhToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		if w.Code != http.StatusOK {
			t.Errorf("[7. Reopen Period] 200 bekleniyordu, alınan: %d, body: %s", w.Code, w.Body.String())
		}
	}

	// 8. Reopen without Auth
	{
		reopenPayload := map[string]interface{}{
			"period_name": "2026-EKIM",
			"reason":      "Yetkisiz deneme",
		}
		body, _ := json.Marshal(reopenPayload)
		req := httptest.NewRequest("POST", "/api/v1/settlements/reopen", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		if w.Code != http.StatusUnauthorized {
			t.Errorf("[8. Reopen Unauth] 401 bekleniyordu, alınan: %d", w.Code)
		}
	}

	// 9. Reopen with Yolcu Token (Unauthorized role)
	{
		reopenPayload := map[string]interface{}{
			"period_name": "2026-EKIM",
			"reason":      "Yolcu rolüyle yetkisiz deneme",
		}
		body, _ := json.Marshal(reopenPayload)
		req := httptest.NewRequest("POST", "/api/v1/settlements/reopen", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rig.yolcuToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		if w.Code != http.StatusForbidden {
			t.Errorf("[9. Reopen Yolcu] 403 bekleniyordu, alınan: %d", w.Code)
		}
	}

	// 10. Close again after Reopen
	{
		closePayload := map[string]interface{}{
			"period_id":   periodID.String(),
			"period_name": "2026-EKIM",
		}
		body, _ := json.Marshal(closePayload)
		req := httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer "+rig.adminToken)
		w := httptest.NewRecorder()
		rig.router.ServeHTTP(w, req)
		t.Logf("[10. Close after Reopen in memory] status code: %d, body: %s", w.Code, w.Body.String())
	}
}
