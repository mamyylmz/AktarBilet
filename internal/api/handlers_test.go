package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/settlement"
	"github.com/mamyylmz/AktarBilet/internal/sync"
)

type testRig struct {
	router     http.Handler
	authSvc    *auth.AuthService
	ledgerSvc  *ledger.MemoryLedger
	syncSvc    *sync.SyncService
	settleSvc  *settlement.SettlementService
	anomalyDet *anomaly.AnomalyDetector
	yolcuUser  *domain.User
	adminUser  *domain.User
	kontrolor  *domain.User
	muhasebe   *domain.User
	operasyon  *domain.User
	yolcuCard  domain.Card
	otherCard  domain.Card
}

func setupTestRig() *testRig {
	authSvc := auth.NewAuthService("test-secret-12345678901234567890")
	pricingEngine := pricing.NewPricingEngine()
	ledSvc := ledger.NewMemoryLedger()
	syncSvc := sync.NewSyncService(pricingEngine, ledSvc, 5000)
	settleSvc := settlement.NewSettlementService()
	anomalyDet := anomaly.NewAnomalyDetector()

	yolcuUID := uuid.New()
	yolcuUser := &domain.User{ID: yolcuUID, Email: "yolcu@aktarbilet.gov.tr", FullName: "Yolcu Test", Role: domain.RoleYolcu}
	adminUser := &domain.User{ID: uuid.New(), Email: "seed.sedir@ddo-case.invalid", FullName: "Sistem Yöneticisi", Role: domain.RoleAdmin}
	kontrolor := &domain.User{ID: uuid.New(), Email: "kontrolor@aktarbilet.gov.tr", FullName: "Kontrolör Test", Role: domain.RoleKontrolor}
	muhasebe := &domain.User{ID: uuid.New(), Email: "muhasebe@aktarbilet.gov.tr", FullName: "Muhasebe Test", Role: domain.RoleMuhasebe}
	operasyon := &domain.User{ID: uuid.New(), Email: "operasyon@aktarbilet.gov.tr", FullName: "Operasyon Test", Role: domain.RoleOperasyon}

	yolcuCard := domain.Card{
		ID:           "CARD_YOLCU_001",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 10000,
		UserID:       &yolcuUID,
	}
	otherCard := domain.Card{
		ID:           "CARD_OTHER_999",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 5000,
		UserID:       nil,
	}

	ledSvc.RegisterCard(yolcuCard)
	ledSvc.RegisterCard(otherCard)
	syncSvc.RegisterCard(yolcuCard)
	syncSvc.RegisterCard(otherCard)

	h := NewAPIHandler(nil, authSvc, pricingEngine, ledSvc, syncSvc, settleSvc, anomalyDet)
	router := SetupRouter(h, "f85f26")

	return &testRig{
		router:     router,
		authSvc:    authSvc,
		ledgerSvc:  ledSvc,
		syncSvc:    syncSvc,
		settleSvc:  settleSvc,
		anomalyDet: anomalyDet,
		yolcuUser:  yolcuUser,
		adminUser:  adminUser,
		kontrolor:  kontrolor,
		muhasebe:   muhasebe,
		operasyon:  operasyon,
		yolcuCard:  yolcuCard,
		otherCard:  otherCard,
	}
}

func (r *testRig) token(u *domain.User) string {
	tok, _, _ := r.authSvc.GenerateTokens(u)
	return tok
}

func TestAPI_LoginHandler(t *testing.T) {
	rig := setupTestRig()

	// 1. Seed kullanıcı login başarılı
	body, _ := json.Marshal(map[string]string{
		"email":    "seed.sedir@ddo-case.invalid",
		"password": "SedirPassword2026!",
	})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Seed kullanıcı login 200 dönmeli, alınan: %d", w.Code)
	}

	// 2. Hatalı şifre
	badBody, _ := json.Marshal(map[string]string{
		"email":    "seed.sedir@ddo-case.invalid",
		"password": "WrongPassword",
	})
	reqBad := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(badBody))
	wBad := httptest.NewRecorder()
	rig.router.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusUnauthorized {
		t.Fatalf("Hatalı şifre 401 dönmeli, alınan: %d", wBad.Code)
	}

	// 3. Geçersiz JSON
	reqInvalid := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte("{invalid-json")))
	wInvalid := httptest.NewRecorder()
	rig.router.ServeHTTP(wInvalid, reqInvalid)
	if wInvalid.Code != http.StatusBadRequest {
		t.Fatalf("Geçersiz JSON 400 dönmeli, alınan: %d", wInvalid.Code)
	}
}

func TestAPI_GetMe(t *testing.T) {
	rig := setupTestRig()

	// 1. Yetkisiz GET /api/v1/auth/me
	reqUnauth := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	wUnauth := httptest.NewRecorder()
	rig.router.ServeHTTP(wUnauth, reqUnauth)
	if wUnauth.Code != http.StatusUnauthorized {
		t.Fatalf("Yetkisiz /auth/me 401 dönmeli, alınan: %d", wUnauth.Code)
	}

	// 2. Yetkili GET /api/v1/auth/me
	reqAuth := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	reqAuth.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	wAuth := httptest.NewRecorder()
	rig.router.ServeHTTP(wAuth, reqAuth)
	if wAuth.Code != http.StatusOK {
		t.Fatalf("Yetkili /auth/me 200 dönmeli, alınan: %d", wAuth.Code)
	}
}

func TestAPI_GetCard_And_IDOR(t *testing.T) {
	rig := setupTestRig()

	// 1. Yolcu kendi kartını sorgulayabilir
	reqOwn := httptest.NewRequest("GET", "/api/v1/cards/CARD_YOLCU_001", nil)
	reqOwn.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	wOwn := httptest.NewRecorder()
	rig.router.ServeHTTP(wOwn, reqOwn)
	if wOwn.Code != http.StatusOK {
		t.Fatalf("Yolcu kendi kartını 200 ile görebilmeli, alınan: %d", wOwn.Code)
	}

	// 2. Yolcu başkasına ait kartı sorgulayamaz (IDOR koruması -> 403)
	otherUID := uuid.New()
	privateCard := domain.Card{
		ID:           "CARD_PRIVATE_123",
		CardType:     domain.CardTypeTam,
		BalanceKurus: 15000,
		UserID:       &otherUID,
	}
	rig.ledgerSvc.RegisterCard(privateCard)

	reqIDOR := httptest.NewRequest("GET", "/api/v1/cards/CARD_PRIVATE_123", nil)
	reqIDOR.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	wIDOR := httptest.NewRecorder()
	rig.router.ServeHTTP(wIDOR, reqIDOR)
	if wIDOR.Code != http.StatusForbidden {
		t.Fatalf("Yolcu başkasının kartını sorguladığında 403 Forbidden dönmeli, alınan: %d", wIDOR.Code)
	}

	// 3. Olmayan kart -> 404 Not Found
	reqNotFound := httptest.NewRequest("GET", "/api/v1/cards/CARD_NOT_FOUND", nil)
	reqNotFound.Header.Set("Authorization", "Bearer "+rig.token(rig.adminUser))
	wNotFound := httptest.NewRecorder()
	rig.router.ServeHTTP(wNotFound, reqNotFound)
	if wNotFound.Code != http.StatusNotFound {
		t.Fatalf("Olmayan kart 404 dönmeli, alınan: %d", wNotFound.Code)
	}
}

func TestAPI_RechargeCard(t *testing.T) {
	rig := setupTestRig()

	// 1. Başarılı Dolum
	body, _ := json.Marshal(map[string]interface{}{
		"amount_kurus": 5000,
		"description":  "50 TL Dolum",
	})
	req := httptest.NewRequest("POST", "/api/v1/cards/CARD_YOLCU_001/recharge", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	req.Header.Set("Idempotency-Key", "REC_TEST_001")
	w := httptest.NewRecorder()
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Dolum 200 dönmeli, alınan: %d", w.Code)
	}

	// 2. İdempotansi Testi: Aynı anahtarla tekrar istek atıldığında bakiye tekrar artmamalı
	balBefore, _ := rig.ledgerSvc.GetCardBalance(nil, "CARD_YOLCU_001")
	reqRepeat := httptest.NewRequest("POST", "/api/v1/cards/CARD_YOLCU_001/recharge", bytes.NewReader(body))
	reqRepeat.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	reqRepeat.Header.Set("Idempotency-Key", "REC_TEST_001")
	wRepeat := httptest.NewRecorder()
	rig.router.ServeHTTP(wRepeat, reqRepeat)
	if wRepeat.Code != http.StatusOK {
		t.Fatalf("Tekrarlı dolum 200 dönmeli, alınan: %d", wRepeat.Code)
	}
	balAfter, _ := rig.ledgerSvc.GetCardBalance(nil, "CARD_YOLCU_001")
	if balBefore != balAfter {
		t.Fatalf("İdempotent dolumda bakiye değişmemeliydi! Önce: %d, Sonra: %d", balBefore, balAfter)
	}

	// 3. Negatif tutar dolumu reddedilmeli -> 400 Bad Request
	badAmountBody, _ := json.Marshal(map[string]interface{}{"amount_kurus": -100})
	reqBad := httptest.NewRequest("POST", "/api/v1/cards/CARD_YOLCU_001/recharge", bytes.NewReader(badAmountBody))
	reqBad.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	wBad := httptest.NewRecorder()
	rig.router.ServeHTTP(wBad, reqBad)
	if wBad.Code != http.StatusBadRequest {
		t.Fatalf("Negatif tutar 400 dönmeli, alınan: %d", wBad.Code)
	}
}

func TestAPI_GetCardLedger(t *testing.T) {
	rig := setupTestRig()

	// Defter sorgulama
	req := httptest.NewRequest("GET", "/api/v1/cards/CARD_YOLCU_001/ledger", nil)
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	w := httptest.NewRecorder()
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Defter sorgulama 200 dönmeli, alınan: %d", w.Code)
	}
}

func TestAPI_ProcessTap_And_Batch(t *testing.T) {
	rig := setupTestRig()

	// 1. Tekil validatör tap işlemi
	tapBody, _ := json.Marshal(map[string]interface{}{
		"validator_id": "VAL_500T_01",
		"sequence_no":  1001,
		"card_id":      "CARD_YOLCU_001",
		"line_id":      "LINE_500T",
		"station_id":   "STAT_KADIKOY",
		"direction":    1,
		"tapped_at":    time.Now().Format(time.RFC3339),
	})
	reqTap := httptest.NewRequest("POST", "/api/v1/validators/tap", bytes.NewReader(tapBody))
	wTap := httptest.NewRecorder()
	rig.router.ServeHTTP(wTap, reqTap)
	if wTap.Code != http.StatusOK {
		t.Fatalf("Validatör biniş 200 dönmeli, alınan: %d (%s)", wTap.Code, wTap.Body.String())
	}

	// 2. Anonim eksik istek -> 401 Unauthorized
	reqAnon := httptest.NewRequest("POST", "/api/v1/validators/tap", bytes.NewReader([]byte("{}")))
	wAnon := httptest.NewRecorder()
	rig.router.ServeHTTP(wAnon, reqAnon)
	if wAnon.Code != http.StatusUnauthorized {
		t.Fatalf("Cihaz kimliksiz anonim tap 401 dönmeli, alınan: %d", wAnon.Code)
	}

	// 3. Toplu Batch Sync
	batchBody, _ := json.Marshal([]map[string]interface{}{
		{
			"validator_id": "VAL_500T_02",
			"sequence_no":  2001,
			"card_id":      "CARD_YOLCU_001",
			"line_id":      "LINE_500T",
			"station_id":   "STAT_AYRILIK",
			"direction":    1,
			"tapped_at":    time.Now().Format(time.RFC3339),
		},
	})
	reqBatch := httptest.NewRequest("POST", "/api/v1/validators/batch-sync", bytes.NewReader(batchBody))
	wBatch := httptest.NewRecorder()
	rig.router.ServeHTTP(wBatch, reqBatch)
	if wBatch.Code != http.StatusOK {
		t.Fatalf("Batch sync 200 dönmeli, alınan: %d (%s)", wBatch.Code, wBatch.Body.String())
	}
}

func TestAPI_StaticCatalogEndpoints(t *testing.T) {
	rig := setupTestRig()

	// GET /lines
	reqLines := httptest.NewRequest("GET", "/api/v1/lines", nil)
	wLines := httptest.NewRecorder()
	rig.router.ServeHTTP(wLines, reqLines)
	if wLines.Code != http.StatusOK {
		t.Fatalf("GET /lines 200 dönmeli, alınan: %d", wLines.Code)
	}

	// GET /stations
	reqStations := httptest.NewRequest("GET", "/api/v1/stations", nil)
	wStations := httptest.NewRecorder()
	rig.router.ServeHTTP(wStations, reqStations)
	if wStations.Code != http.StatusOK {
		t.Fatalf("GET /stations 200 dönmeli, alınan: %d", wStations.Code)
	}

	// GET /tariffs
	reqTariffs := httptest.NewRequest("GET", "/api/v1/tariffs", nil)
	wTariffs := httptest.NewRecorder()
	rig.router.ServeHTTP(wTariffs, reqTariffs)
	if wTariffs.Code != http.StatusOK {
		t.Fatalf("GET /tariffs 200 dönmeli, alınan: %d", wTariffs.Code)
	}
}

func TestAPI_SettlementEndpoints(t *testing.T) {
	rig := setupTestRig()

	// 1. Muhasebe dönemi kapatır
	p := rig.settleSvc.CreatePeriod("Test Period", time.Now().Add(-24*time.Hour), time.Now())
	closeBody, _ := json.Marshal(map[string]interface{}{
		"period_id": p.ID.String(),
	})

	reqClose := httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(closeBody))
	reqClose.Header.Set("Authorization", "Bearer "+rig.token(rig.muhasebe))
	wClose := httptest.NewRecorder()
	rig.router.ServeHTTP(wClose, reqClose)
	if wClose.Code != http.StatusOK {
		t.Fatalf("Dönem kapatma 200 dönmeli, alınan: %d (%s)", wClose.Code, wClose.Body.String())
	}

	// 2. Kapanış sonrası storno kaydı
	stornoBody, _ := json.Marshal(map[string]interface{}{
		"period_id":     p.ID.String(),
		"operator":      "ozel_halk",
		"storno_amount": -500,
		"reason":        "Ceza düzeltmesi",
	})
	reqStorno := httptest.NewRequest("POST", "/api/v1/settlements/storno", bytes.NewReader(stornoBody))
	reqStorno.Header.Set("Authorization", "Bearer "+rig.token(rig.muhasebe))
	wStorno := httptest.NewRecorder()
	rig.router.ServeHTTP(wStorno, reqStorno)
	if wStorno.Code != http.StatusOK {
		t.Fatalf("Storno kaydı 200 dönmeli, alınan: %d (%s)", wStorno.Code, wStorno.Body.String())
	}
}

func TestAPI_PenaltiesAndAnomalies(t *testing.T) {
	rig := setupTestRig()

	// 1. Ceza listeleme
	reqListPen := httptest.NewRequest("GET", "/api/v1/penalties", nil)
	reqListPen.Header.Set("Authorization", "Bearer "+rig.token(rig.kontrolor))
	wListPen := httptest.NewRecorder()
	rig.router.ServeHTTP(wListPen, reqListPen)
	if wListPen.Code != http.StatusOK {
		t.Fatalf("Ceza listeleme 200 dönmeli, alınan: %d", wListPen.Code)
	}

	// 2. Anomali listeleme (Operasyon yetkili)
	reqAnom := httptest.NewRequest("GET", "/api/v1/anomalies", nil)
	reqAnom.Header.Set("Authorization", "Bearer "+rig.token(rig.operasyon))
	wAnom := httptest.NewRecorder()
	rig.router.ServeHTTP(wAnom, reqAnom)
	if wAnom.Code != http.StatusOK {
		t.Fatalf("Anomali listeleme 200 dönmeli, alınan: %d", wAnom.Code)
	}

	// 3. Isı haritası
	reqHeat := httptest.NewRequest("GET", "/api/v1/reports/heatmap", nil)
	wHeat := httptest.NewRecorder()
	rig.router.ServeHTTP(wHeat, reqHeat)
	if wHeat.Code != http.StatusOK {
		t.Fatalf("Isı haritası 200 dönmeli, alınan: %d", wHeat.Code)
	}

	// 4. Tarife Etki Simülasyonu
	simBody, _ := json.Marshal(map[string]interface{}{
		"base_fare_change_pct":    10,
		"student_fare_change_pct": 20,
	})
	reqSim := httptest.NewRequest("POST", "/api/v1/simulation/fare-impact", bytes.NewReader(simBody))
	reqSim.Header.Set("Authorization", "Bearer "+rig.token(rig.adminUser))
	wSim := httptest.NewRecorder()
	rig.router.ServeHTTP(wSim, reqSim)
	if wSim.Code != http.StatusOK {
		t.Fatalf("Simülasyon 200 dönmeli, alınan: %d", wSim.Code)
	}
}

func TestAPI_ErrorHandlingBranches(t *testing.T) {
	rig := setupTestRig()

	// 1. Recharge Bad JSON
	reqBadRecharge := httptest.NewRequest("POST", "/api/v1/cards/CARD_YOLCU_001/recharge", bytes.NewReader([]byte("{bad-json")))
	reqBadRecharge.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	wBadRecharge := httptest.NewRecorder()
	rig.router.ServeHTTP(wBadRecharge, reqBadRecharge)
	if wBadRecharge.Code != http.StatusBadRequest {
		t.Fatalf("Bozuk JSON 400 dönmeli, alınan: %d", wBadRecharge.Code)
	}

	// 2. Recharge IDOR Probe Blocked
	bodyIDOR, _ := json.Marshal(map[string]interface{}{"amount_kurus": 5000, "description": "IDOR probe"})
	reqIDORProbe := httptest.NewRequest("POST", "/api/v1/cards/CARD_OTHER_999/recharge", bytes.NewReader(bodyIDOR))
	reqIDORProbe.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	wIDORProbe := httptest.NewRecorder()
	rig.router.ServeHTTP(wIDORProbe, reqIDORProbe)
	if wIDORProbe.Code != http.StatusForbidden {
		t.Fatalf("IDOR probe 403 dönmeli, alınan: %d", wIDORProbe.Code)
	}

	// 3. Tap Bad JSON
	reqBadTap := httptest.NewRequest("POST", "/api/v1/validators/tap", bytes.NewReader([]byte("{bad-json")))
	wBadTap := httptest.NewRecorder()
	rig.router.ServeHTTP(wBadTap, reqBadTap)
	if wBadTap.Code != http.StatusBadRequest {
		t.Fatalf("Bozuk tap JSON 400 dönmeli, alınan: %d", wBadTap.Code)
	}

	// 4. Batch Sync Bad JSON (Authenticated)
	reqBadBatch := httptest.NewRequest("POST", "/api/v1/validators/batch-sync", bytes.NewReader([]byte("{bad-json")))
	reqBadBatch.Header.Set("Authorization", "Bearer "+rig.token(rig.adminUser))
	wBadBatch := httptest.NewRecorder()
	rig.router.ServeHTTP(wBadBatch, reqBadBatch)
	if wBadBatch.Code != http.StatusBadRequest {
		t.Fatalf("Bozuk batch JSON 400 dönmeli, alınan: %d", wBadBatch.Code)
	}

	// 5. Settlement Close Bad JSON
	reqBadClose := httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader([]byte("{bad-json")))
	reqBadClose.Header.Set("Authorization", "Bearer "+rig.token(rig.muhasebe))
	wBadClose := httptest.NewRecorder()
	rig.router.ServeHTTP(wBadClose, reqBadClose)
	if wBadClose.Code != http.StatusBadRequest {
		t.Fatalf("Bozuk close JSON 400 dönmeli, alınan: %d", wBadClose.Code)
	}

	// 6. Settlement Storno Bad JSON
	reqBadStorno := httptest.NewRequest("POST", "/api/v1/settlements/storno", bytes.NewReader([]byte("{bad-json")))
	reqBadStorno.Header.Set("Authorization", "Bearer "+rig.token(rig.muhasebe))
	wBadStorno := httptest.NewRecorder()
	rig.router.ServeHTTP(wBadStorno, reqBadStorno)
	if wBadStorno.Code != http.StatusBadRequest {
		t.Fatalf("Bozuk storno JSON 400 dönmeli, alınan: %d", wBadStorno.Code)
	}

	// 7. Simulation Bad JSON
	reqBadSim := httptest.NewRequest("POST", "/api/v1/simulation/fare-impact", bytes.NewReader([]byte("{bad-json")))
	reqBadSim.Header.Set("Authorization", "Bearer "+rig.token(rig.adminUser))
	wBadSim := httptest.NewRecorder()
	rig.router.ServeHTTP(wBadSim, reqBadSim)
	if wBadSim.Code != http.StatusBadRequest {
		t.Fatalf("Bozuk simulation JSON 400 dönmeli, alınan: %d", wBadSim.Code)
	}

	// 8. IDOR on GetCardLedger: Yolcu başkasının defterini sorgulayamaz
	reqIDORLedger := httptest.NewRequest("GET", "/api/v1/cards/CARD_OTHER_999/ledger", nil)
	reqIDORLedger.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	wIDORLedger := httptest.NewRecorder()
	rig.router.ServeHTTP(wIDORLedger, reqIDORLedger)
	if wIDORLedger.Code != http.StatusForbidden {
		t.Fatalf("Başkasının defteri 403 Forbidden dönmeli, alınan: %d", wIDORLedger.Code)
	}
}

func TestAPI_ComprehensiveBranches(t *testing.T) {
	rig := setupTestRig()

	// 1. Login with unknown email and wrong password
	w := httptest.NewRecorder()
	body, _ := json.Marshal(map[string]string{"email": "nonexistent@test.com", "password": "wrong"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Bilinmeyen kullanıcı 401 dönmeli, alınan: %d", w.Code)
	}

	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]string{"email": "yolcu@aktarbilet.gov.tr", "password": "WrongPassword!"})
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Yanlış şifre 401 dönmeli, alınan: %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte("{invalid-json")))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Bozuk JSON login 400 dönmeli, alınan: %d", w.Code)
	}

	// 2. HealthCheck when pool is nil
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/health", nil)
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Health check 200 dönmeli, alınan: %d", w.Code)
	}

	// 3. Recharge amount <= 0
	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"amount_kurus": -500})
	req = httptest.NewRequest("POST", "/api/v1/cards/CARD_YOLCU_001/recharge", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Negatif recharge 400 dönmeli, alınan: %d", w.Code)
	}

	// 4. CreatePenalty with amount <= 0 and empty card
	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"card_id": "CARD_YOLCU_001", "penalty_amount_kurus": -100})
	req = httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.kontrolor))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Negatif ceza 400 dönmeli, alınan: %d", w.Code)
	}

	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"card_id": "", "penalty_amount_kurus": 5000})
	req = httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.kontrolor))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Boş kart ID ceza 400 dönmeli, alınan: %d", w.Code)
	}

	// 5. CreatePenalty success in-memory
	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"card_id": "CARD_YOLCU_001", "penalty_amount_kurus": 5000, "reason": "kartsiz_binis"})
	req = httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.kontrolor))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("Başarılı ceza 201 dönmeli, alınan: %d", w.Code)
	}

	// 6. Storno with invalid UUID and invalid amount
	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"period_id": "invalid-uuid", "amount_kurus": 100})
	req = httptest.NewRequest("POST", "/api/v1/settlements/storno", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.muhasebe))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Geçersiz UUID storno 400 dönmeli, alınan: %d", w.Code)
	}

	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"period_id": uuid.New().String(), "amount_kurus": 0})
	req = httptest.NewRequest("POST", "/api/v1/settlements/storno", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.muhasebe))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Sıfır storno 400 dönmeli, alınan: %d", w.Code)
	}

	// 7. Settlement Close with invalid UUID
	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"period_id": "not-a-uuid"})
	req = httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.muhasebe))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("Geçersiz UUID close 400 dönmeli, alınan: %d", w.Code)
	}

	// 8. AuthMiddleware with bad token / invalid scheme
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Basic invalid-scheme")
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Basic auth 401 dönmeli, alınan: %d", w.Code)
	}

	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	req.Header.Set("Authorization", "Bearer invalid-jwt-token")
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("Geçersiz JWT 401 dönmeli, alınan: %d", w.Code)
	}

	// 9. BatchSyncValidators with anonymous hardware batch
	w = httptest.NewRecorder()
	batchEvents := []map[string]interface{}{
		{
			"validator_id": "VAL_500T_01",
			"sequence_no":  123,
			"card_id":      "CARD_YOLCU_001",
			"line_id":      "LINE_500T",
			"station_id":   "STAT_KADIKOY",
			"direction":    1,
			"tapped_at":    time.Now().Format(time.RFC3339),
		},
	}
	body, _ = json.Marshal(batchEvents)
	req = httptest.NewRequest("POST", "/api/v1/validators/batch-sync", bytes.NewReader(body))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("Donanım validatör batch sync 200 dönmeli, alınan: %d", w.Code)
	}

	// 10. RoleAuthorization failure: Yolcu trying to access muhasebe close
	w = httptest.NewRecorder()
	body, _ = json.Marshal(map[string]interface{}{"period_id": uuid.New().String()})
	req = httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+rig.token(rig.yolcuUser))
	rig.router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("Yolcu muhasebe işlemine 403 dönmeli, alınan: %d", w.Code)
	}
}

type mockAPIRow struct {
	scanFunc func(dest ...any) error
}

func (m *mockAPIRow) Scan(dest ...any) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	return nil
}

type mockAPIRows struct {
	scanFunc func(dest ...any) error
	count    int
	idx      int
}

func (m *mockAPIRows) Close()                                       {}
func (m *mockAPIRows) Err() error                                   { return nil }
func (m *mockAPIRows) CommandTag() pgconn.CommandTag                { return pgconn.CommandTag{} }
func (m *mockAPIRows) FieldDescriptions() []pgconn.FieldDescription { return nil }
func (m *mockAPIRows) Next() bool {
	if m.idx < m.count {
		m.idx++
		return true
	}
	return false
}
func (m *mockAPIRows) Scan(dest ...any) error {
	if m.scanFunc != nil {
		return m.scanFunc(dest...)
	}
	return nil
}
func (m *mockAPIRows) Values() ([]any, error) { return nil, nil }
func (m *mockAPIRows) RawValues() [][]byte    { return nil }
func (m *mockAPIRows) Conn() *pgx.Conn        { return nil }

type mockAPIPool struct {
	execFunc     func(sql string, args ...any) (pgconn.CommandTag, error)
	queryFunc    func(sql string, args ...any) (pgx.Rows, error)
	queryRowFunc func(sql string, args ...any) pgx.Row
}

func (m *mockAPIPool) Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error) {
	if m.execFunc != nil {
		return m.execFunc(sql, arguments...)
	}
	return pgconn.NewCommandTag("UPDATE 1"), nil
}

func (m *mockAPIPool) Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error) {
	if m.queryFunc != nil {
		return m.queryFunc(sql, args...)
	}
	return &mockAPIRows{}, nil
}

func (m *mockAPIPool) QueryRow(ctx context.Context, sql string, args ...any) pgx.Row {
	if m.queryRowFunc != nil {
		return m.queryRowFunc(sql, args...)
	}
	return &mockAPIRow{}
}

func TestAPI_DatabaseQueryMocks(t *testing.T) {
	rig := setupTestRig()
	pool := &mockAPIPool{}
	h := NewAPIHandler(nil, rig.authSvc, pricing.NewPricingEngine(), rig.ledgerSvc, rig.syncSvc, rig.settleSvc, rig.anomalyDet)
	h.SetDBPool(pool)
	router := SetupRouter(h, "f85f26")

	adminToken := rig.token(rig.adminUser)
	kontrolorToken := rig.token(rig.kontrolor)
	muhasebeToken := rig.token(rig.muhasebe)

	currentRole := domain.RoleAdmin
	pool.queryRowFunc = func(sql string, args ...any) pgx.Row {
		if strings.Contains(sql, "FROM jw_tf_users") {
			return &mockAPIRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*domain.UserRole) = currentRole
					return nil
				},
			}
		}
		if strings.Contains(sql, "EXISTS") {
			return &mockAPIRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*bool) = true
					return nil
				},
			}
		}
		if strings.Contains(sql, "FROM jw_settlement_periods") {
			return &mockAPIRow{
				scanFunc: func(dest ...any) error {
					*dest[0].(*bool) = true
					return nil
				},
			}
		}
		return &mockAPIRow{}
	}

	// 1. ListLines from DB
	pool.queryFunc = func(sql string, args ...any) (pgx.Rows, error) {
		return &mockAPIRows{
			count: 1,
			scanFunc: func(dest ...any) error {
				*dest[0].(*string) = "LINE_500T"
				*dest[1].(*string) = "500T Tuzla"
				*dest[2].(*domain.LineMode) = domain.ModeBus
				*dest[3].(*string) = "belediye"
				*dest[4].(*bool) = true
				*dest[5].(*time.Time) = time.Now()
				return nil
			},
		}, nil
	}
	w := httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/api/v1/lines", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListLines from DB 200 dönmeli, alınan: %d", w.Code)
	}

	// 2. ListStations from DB
	pool.queryFunc = func(sql string, args ...any) (pgx.Rows, error) {
		return &mockAPIRows{
			count: 1,
			scanFunc: func(dest ...any) error {
				*dest[0].(*string) = "STAT_KADIKOY"
				*dest[1].(*string) = "Kadıköy İskele"
				*dest[2].(*domain.LineMode) = domain.ModeFerry
				*dest[3].(*float64) = 40.99
				*dest[4].(*float64) = 29.02
				*dest[5].(*int) = 1
				*dest[6].(*time.Time) = time.Now()
				return nil
			},
		}, nil
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/stations", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListStations from DB 200 dönmeli, alınan: %d", w.Code)
	}

	// 3. ListTariffs from DB
	pool.queryFunc = func(sql string, args ...any) (pgx.Rows, error) {
		return &mockAPIRows{
			count: 1,
			scanFunc: func(dest ...any) error {
				*dest[0].(*string) = "TAR_1"
				*dest[1].(*string) = "Tarife 1"
				*dest[2].(*domain.LineMode) = domain.ModeBus
				*dest[3].(*domain.Kurus) = 2000
				*dest[4].(*domain.Kurus) = 1000
				*dest[5].(*domain.Kurus) = 1500
				*dest[6].(*int) = 50
				*dest[7].(*int) = 75
				*dest[8].(*int) = 50
				*dest[9].(*int) = 10
				*dest[10].(*domain.Kurus) = 150000
				*dest[11].(*bool) = true
				*dest[12].(*time.Time) = time.Now()
				return nil
			},
		}, nil
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/tariffs", nil)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListTariffs from DB 200 dönmeli, alınan: %d", w.Code)
	}

	// 4. CreatePenalty with DB checks
	currentRole = domain.RoleKontrolor
	body, _ := json.Marshal(map[string]interface{}{
		"card_id":              "CARD_YOLCU_001",
		"penalty_amount_kurus": 5000,
		"reason":               "kartsiz_binis",
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/penalties", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+kontrolorToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("CreatePenalty with DB 201 dönmeli, alınan: %d", w.Code)
	}

	// 5. ListPenalties from DB
	currentRole = domain.RoleAdmin
	pool.queryFunc = func(sql string, args ...any) (pgx.Rows, error) {
		return &mockAPIRows{
			count: 1,
			scanFunc: func(dest ...any) error {
				*dest[0].(*uuid.UUID) = uuid.New()
				*dest[1].(*uuid.UUID) = uuid.New()
				*dest[2].(*string) = "CARD_1"
				*dest[3].(*string) = "LINE_1"
				*dest[4].(*string) = "STAT_1"
				*dest[5].(*domain.Kurus) = 5000
				*dest[6].(*domain.PenaltyReason) = domain.PenaltyKartsizBinis
				*dest[7].(*time.Time) = time.Now()
				*dest[8].(*string) = "Notes"
				return nil
			},
		}, nil
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/penalties", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("ListPenalties from DB 200 dönmeli, alınan: %d", w.Code)
	}

	// 6. GetHeatmap from DB
	pool.queryFunc = func(sql string, args ...any) (pgx.Rows, error) {
		return &mockAPIRows{
			count: 1,
			scanFunc: func(dest ...any) error {
				*dest[0].(*string) = "LINE_500T"
				*dest[1].(*string) = "500T Tuzla"
				*dest[2].(*string) = "bus"
				*dest[3].(*string) = "belediye"
				*dest[4].(*int) = 250
				*dest[5].(*float64) = 0.25
				*dest[6].(*string) = "MEDIUM"
				return nil
			},
		}, nil
	}
	w = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/api/v1/reports/heatmap", nil)
	req.Header.Set("Authorization", "Bearer "+adminToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("GetHeatmap from DB 200 dönmeli, alınan: %d", w.Code)
	}

	// 7. CloseSettlementPeriod with DB
	currentRole = domain.RoleMuhasebe
	pID := uuid.New()
	pool.execFunc = func(sql string, args ...any) (pgconn.CommandTag, error) {
		return pgconn.NewCommandTag("UPDATE 1"), nil
	}
	body, _ = json.Marshal(map[string]interface{}{"period_id": pID.String()})
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/settlements/close", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+muhasebeToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("CloseSettlementPeriod with DB 200 dönmeli, alınan: %d", w.Code)
	}

	// 8. StornoSettlement with DB
	body, _ = json.Marshal(map[string]interface{}{
		"period_id":    pID.String(),
		"operator":     "belediye",
		"amount_kurus": 500,
		"reason":       "Mock storno",
	})
	w = httptest.NewRecorder()
	req = httptest.NewRequest("POST", "/api/v1/settlements/storno", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+muhasebeToken)
	router.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("StornoSettlement with DB 200 dönmeli, alınan: %d", w.Code)
	}
}
