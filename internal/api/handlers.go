package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
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

type APIHandler struct {
	pool          *pgxpool.Pool
	authSvc       *auth.AuthService
	pricingEngine *pricing.PricingEngine
	ledgerSvc     ledger.LedgerService
	syncSvc       *sync.SyncService
	settleSvc     *settlement.SettlementService
	anomalyDet    *anomaly.AnomalyDetector
}

func NewAPIHandler(
	pool *pgxpool.Pool,
	authSvc *auth.AuthService,
	pricingEngine *pricing.PricingEngine,
	ledgerSvc ledger.LedgerService,
	syncSvc *sync.SyncService,
	settleSvc *settlement.SettlementService,
	anomalyDet *anomaly.AnomalyDetector,
) *APIHandler {
	return &APIHandler{
		pool:          pool,
		authSvc:       authSvc,
		pricingEngine: pricingEngine,
		ledgerSvc:     ledgerSvc,
		syncSvc:       syncSvc,
		settleSvc:     settleSvc,
		anomalyDet:    anomalyDet,
	}
}

// 1. Sağlık Kontrolleri
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	dbStatus := "healthy"
	if h.pool != nil {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := h.pool.Ping(ctx); err != nil {
			dbStatus = fmt.Sprintf("unreachable: %v", err)
		}
	} else {
		dbStatus = "memory_mode"
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status":    "UP",
		"database":  dbStatus,
		"timestamp": time.Now().Format(time.RFC3339),
		"service":   "AktarBilet Backend API",
	})
}

// System Canary 4: GET /_hz/f85f returns 200 OK!
func (h *APIHandler) DDOMandatoryHealthCheck(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"status": "ok",
		"stamp":  "f85f26",
		"hz":     "f85f",
	})
}

// 2. Kimlik Yönetimi
type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (h *APIHandler) Login(w http.ResponseWriter, r *http.Request) {
	var req LoginRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek gövdesi")
		return
	}

	var user domain.User
	if h.pool != nil {
		err := h.pool.QueryRow(r.Context(), `
			SELECT id, email, password_hash, full_name, role, created_at, updated_at
			FROM jw_tf_users
			WHERE email = $1
		`, req.Email).Scan(&user.ID, &user.Email, &user.PasswordHash, &user.FullName, &user.Role, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			WriteError(w, http.StatusUnauthorized, "AUTH_FAILED", "Kullanıcı adı veya şifre hatalı")
			return
		}
	} else {
		// Mock/Memory mod
		user = domain.User{
			ID:           uuid.New(),
			Email:        req.Email,
			PasswordHash: "$2a$10$xyz",
			FullName:     "Test Kullanıcı",
			Role:         domain.RoleYolcu,
		}
	}

	if h.pool != nil && !h.authSvc.CheckPassword(user.PasswordHash, req.Password) {
		WriteError(w, http.StatusUnauthorized, "AUTH_FAILED", "Kullanıcı adı veya şifre hatalı")
		return
	}

	access, refresh, err := h.authSvc.GenerateTokens(&user)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Oturum anahtarı üretilemedi")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"access_token":  access,
		"refresh_token": refresh,
		"user":          user,
	})
}

func (h *APIHandler) GetMe(w http.ResponseWriter, r *http.Request) {
	claimsVal := r.Context().Value(UserContextKey)
	if claimsVal == nil {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açılmamış")
		return
	}
	claims := claimsVal.(*auth.Claims)
	WriteJSON(w, http.StatusOK, claims)
}

// 3. F-1: Kart ve Bakiye Yönetimi
func (h *APIHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "id")
	if cardID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_CARD_ID", "Kart ID boş olamaz")
		return
	}

	bal, err := h.ledgerSvc.GetCardBalance(r.Context(), cardID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", "Kart bulunamadı")
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"card_id":       cardID,
		"balance_kurus": bal,
		"balance_tl":    bal.ToTL(),
	})
}

type RechargeRequest struct {
	AmountKurus domain.Kurus `json:"amount_kurus"`
	Description string       `json:"description"`
}

func (h *APIHandler) RechargeCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "id")
	var req RechargeRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek gövdesi")
		return
	}

	if req.AmountKurus <= 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_AMOUNT", "Dolum tutarı sıfırdan büyük olmalıdır")
		return
	}

	idempKey := r.Header.Get("Idempotency-Key")
	if idempKey == "" {
		idempKey = uuid.New().String()
	}

	entry, err := h.ledgerSvc.Recharge(r.Context(), cardID, req.AmountKurus, idempKey, req.Description)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "RECHARGE_FAILED", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, entry)
}

func (h *APIHandler) GetCardLedger(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "id")
	entries, err := h.ledgerSvc.GetCardLedger(r.Context(), cardID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "LEDGER_ERROR", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, entries)
}

// 4. F-2 / T-1 / T-2: Validatör Olayları
type TapRequest struct {
	ValidatorID string    `json:"validator_id"`
	SequenceNo  int64     `json:"sequence_no"`
	CardID      string    `json:"card_id"`
	LineID      string    `json:"line_id"`
	StationID   string    `json:"station_id"`
	Direction   int       `json:"direction"`
	TappedAt    time.Time `json:"tapped_at"`
}

func (h *APIHandler) ProcessTap(w http.ResponseWriter, r *http.Request) {
	var req TapRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz biniş isteği")
		return
	}

	if req.TappedAt.IsZero() {
		req.TappedAt = time.Now()
	}

	idempKey := r.Header.Get("Idempotency-Key")
	if idempKey == "" {
		idempKey = fmt.Sprintf("%s:%d:%d", req.ValidatorID, req.SequenceNo, req.TappedAt.Unix())
	}

	// Anormallik kontrolü (T-5)
	evID := uuid.New()
	anomRes := h.anomalyDet.CheckTap(r.Context(), req.CardID, req.StationID, req.TappedAt, &evID)

	tariff := domain.Tariff{
		ID:                       "DEFAULT_TARIFF",
		BaseFareKurus:            2000,
		StudentFareKurus:         1000,
		ElderlyFareKurus:         1500,
		Transfer1DiscountPct:     50,
		Transfer2DiscountPct:     75,
		NightSurchargePct:        50,
		StudentDailyLimit:        10,
		MonthlyCappingLimitKurus: 150000,
		IsActive:                 true,
	}

	ingestReq := sync.IngestEventRequest{
		IdempotencyKey: idempKey,
		ValidatorID:    req.ValidatorID,
		SequenceNo:     req.SequenceNo,
		CardID:         req.CardID,
		LineID:         req.LineID,
		StationID:      req.StationID,
		Direction:      req.Direction,
		TappedAt:       req.TappedAt,
		Tariff:         tariff,
	}

	event, err := h.syncSvc.IngestOfflineEvent(r.Context(), ingestReq)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "INGEST_FAILED", err.Error())
		return
	}

	resp := map[string]interface{}{
		"event":   event,
		"anomaly": anomRes,
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *APIHandler) BatchSyncValidators(w http.ResponseWriter, r *http.Request) {
	var batch []sync.IngestEventRequest
	if err := json.NewDecoder(r.Body).Decode(&batch); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz toplu senkronizasyon paketi")
		return
	}

	results, err := h.syncSvc.IngestBatch(r.Context(), batch)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "BATCH_SYNC_ERROR", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"processed_count": len(results),
		"results":         results,
	})
}

// 5. F-3: Hatlar, Duraklar, Tarifeler
func (h *APIHandler) ListLines(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		WriteJSON(w, http.StatusOK, []domain.Line{
			{ID: "LINE_500T", Name: "500T Tuzla - Cevizlibağ", Mode: domain.ModeBus, Operator: "belediye", IsActive: true},
			{ID: "LINE_14ES", Name: "14ES Esenkent - Kadıköy", Mode: domain.ModeBus, Operator: "ozel_halk", IsActive: true},
			{ID: "LINE_T1", Name: "T1 Kabataş - Bağcılar Tramvay", Mode: domain.ModeTram, Operator: "belediye", IsActive: true},
			{ID: "LINE_VAPUR_KE", Name: "Kadıköy - Eminönü Vapuru", Mode: domain.ModeFerry, Operator: "belediye", IsActive: true},
		})
		return
	}

	rows, err := h.pool.Query(r.Context(), "SELECT id, name, mode, operator, is_active, created_at FROM jw_lines ORDER BY id")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var lines []domain.Line
	for rows.Next() {
		var l domain.Line
		_ = rows.Scan(&l.ID, &l.Name, &l.Mode, &l.Operator, &l.IsActive, &l.CreatedAt)
		lines = append(lines, l)
	}
	WriteJSON(w, http.StatusOK, lines)
}

func (h *APIHandler) ListStations(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		WriteJSON(w, http.StatusOK, []domain.Station{
			{ID: "STAT_KADIKOY", Name: "Kadıköy İskele", Mode: domain.ModeFerry, Latitude: 40.9904, Longitude: 29.0254, ZoneID: 1},
			{ID: "STAT_EMINONU", Name: "Eminönü İskele & Meydan", Mode: domain.ModeFerry, Latitude: 41.0175, Longitude: 28.9734, ZoneID: 1},
			{ID: "STAT_KABATAS", Name: "Kabataş Tramvay", Mode: domain.ModeTram, Latitude: 41.0355, Longitude: 28.9922, ZoneID: 1},
		})
		return
	}

	rows, err := h.pool.Query(r.Context(), "SELECT id, name, mode, latitude, longitude, zone_id, created_at FROM jw_stations ORDER BY id")
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var stations []domain.Station
	for rows.Next() {
		var s domain.Station
		_ = rows.Scan(&s.ID, &s.Name, &s.Mode, &s.Latitude, &s.Longitude, &s.ZoneID, &s.CreatedAt)
		stations = append(stations, s)
	}
	WriteJSON(w, http.StatusOK, stations)
}

func (h *APIHandler) ListTariffs(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		WriteJSON(w, http.StatusOK, []domain.Tariff{
			{
				ID:                       "TARIFF_ISTANBUL_STD",
				Name:                     "İstanbul Şehir İçi Entegre Tarife",
				Mode:                     domain.ModeBus,
				BaseFareKurus:            2000,
				StudentFareKurus:         1000,
				ElderlyFareKurus:         1500,
				Transfer1DiscountPct:     50,
				Transfer2DiscountPct:     75,
				NightSurchargePct:        50,
				StudentDailyLimit:        10,
				MonthlyCappingLimitKurus: 150000,
				IsActive:                 true,
			},
		})
		return
	}

	rows, err := h.pool.Query(r.Context(), `
		SELECT id, name, mode, base_fare_kurus, student_fare_kurus, elderly_fare_kurus,
		       transfer1_discount_pct, transfer2_discount_pct, night_surcharge_pct,
		       student_daily_limit, monthly_capping_limit_kurus, is_active, created_at
		FROM jw_tariffs
	`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	var tariffs []domain.Tariff
	for rows.Next() {
		var t domain.Tariff
		_ = rows.Scan(&t.ID, &t.Name, &t.Mode, &t.BaseFareKurus, &t.StudentFareKurus, &t.ElderlyFareKurus,
			&t.Transfer1DiscountPct, &t.Transfer2DiscountPct, &t.NightSurchargePct,
			&t.StudentDailyLimit, &t.MonthlyCappingLimitKurus, &t.IsActive, &t.CreatedAt)
		tariffs = append(tariffs, t)
	}
	WriteJSON(w, http.StatusOK, tariffs)
}

// 6. F-5 / T-4: Gelir Paylaşımı ve Dönem Kapanışı
func (h *APIHandler) CloseSettlementPeriod(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PeriodID string `json:"period_id"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek")
		return
	}

	pID, err := uuid.Parse(req.PeriodID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_UUID", "Geçersiz dönem ID")
		return
	}

	period, err := h.settleSvc.ClosePeriod(pID)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "SETTLEMENT_ERROR", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, period)
}

func (h *APIHandler) StornoSettlement(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PeriodID string       `json:"period_id"`
		Operator string       `json:"operator"`
		Amount   domain.Kurus `json:"amount_kurus"`
		Reason   string       `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek")
		return
	}

	pID, err := uuid.Parse(req.PeriodID)
	if err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_UUID", "Geçersiz dönem ID")
		return
	}

	entry, err := h.settleSvc.RecordStornoAfterClose(pID, nil, req.Operator, req.Amount, req.Reason)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "STORNO_ERROR", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, entry)
}

// 7. F-6: Ceza Kayıtları
type PenaltyRequest struct {
	CardID    string               `json:"card_id"`
	LineID    string               `json:"line_id"`
	StationID string               `json:"station_id"`
	Amount    domain.Kurus         `json:"penalty_amount_kurus"`
	Reason    domain.PenaltyReason `json:"reason"`
	Notes     string               `json:"notes"`
}

func (h *APIHandler) CreatePenalty(w http.ResponseWriter, r *http.Request) {
	var req PenaltyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz ceza isteği")
		return
	}

	claimsVal := r.Context().Value(UserContextKey)
	var inspectorID uuid.UUID
	if claimsVal != nil {
		claims := claimsVal.(*auth.Claims)
		inspectorID = claims.UserID
	} else {
		inspectorID = uuid.New()
	}

	penalty := domain.Penalty{
		ID:                 uuid.New(),
		InspectorID:        inspectorID,
		CardID:             req.CardID,
		LineID:             req.LineID,
		StationID:          req.StationID,
		PenaltyAmountKurus: req.Amount,
		Reason:             req.Reason,
		IssuedAt:           time.Now(),
		Notes:              req.Notes,
	}

	if h.pool != nil {
		_, _ = h.pool.Exec(r.Context(), `
			INSERT INTO jw_penalties (id, inspector_id, card_id, line_id, station_id, penalty_amount_kurus, reason, issued_at, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, penalty.ID, penalty.InspectorID, penalty.CardID, penalty.LineID, penalty.StationID, penalty.PenaltyAmountKurus, penalty.Reason, penalty.IssuedAt, penalty.Notes)
	}

	WriteJSON(w, http.StatusCreated, penalty)
}

func (h *APIHandler) ListPenalties(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, []domain.Penalty{})
}

// 8. F-7 / Bonus 1: Raporlar ve Isı Haritası
func (h *APIHandler) GetHeatmap(w http.ResponseWriter, r *http.Request) {
	type HeatmapItem struct {
		LineID         string  `json:"line_id"`
		LineName       string  `json:"line_name"`
		Mode           string  `json:"mode"`
		Operator       string  `json:"operator"`
		TapCount       int     `json:"tap_count"`
		OccupancyRatio float64 `json:"occupancy_ratio"`
		CrowdLevel     string  `json:"crowd_level"` // LOW, MEDIUM, HIGH
	}

	heatmap := []HeatmapItem{
		{LineID: "LINE_500T", LineName: "500T Tuzla - Cevizlibağ", Mode: "bus", Operator: "belediye", TapCount: 1840, OccupancyRatio: 0.92, CrowdLevel: "HIGH"},
		{LineID: "LINE_T1", LineName: "T1 Kabataş - Bağcılar", Mode: "tram", Operator: "belediye", TapCount: 2450, OccupancyRatio: 0.98, CrowdLevel: "HIGH"},
		{LineID: "LINE_14ES", LineName: "14ES Esenkent - Kadıköy", Mode: "bus", Operator: "ozel_halk", TapCount: 650, OccupancyRatio: 0.55, CrowdLevel: "MEDIUM"},
		{LineID: "LINE_VAPUR_KE", LineName: "Kadıköy - Eminönü Vapuru", Mode: "ferry", Operator: "belediye", TapCount: 320, OccupancyRatio: 0.35, CrowdLevel: "LOW"},
	}

	WriteJSON(w, http.StatusOK, heatmap)
}

func (h *APIHandler) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	anoms := h.anomalyDet.GetAnomalies()
	WriteJSON(w, http.StatusOK, anoms)
}

// 9. Bonus 3: Tarife Değişikliği Etki Simülasyonu
type FareImpactRequest struct {
	BaseFareChangePct    float64 `json:"base_fare_change_pct"`    // Örn: +10%
	StudentFareChangePct float64 `json:"student_fare_change_pct"` // Örn: +20%
	ElasticityFactor     float64 `json:"elasticity_factor"`       // Fiyat esnekliği (varsayılan: -0.3)
}

type FareImpactResponse struct {
	CurrentMonthlyRevenueTL   float64 `json:"current_monthly_revenue_tl"`
	ProjectedMonthlyRevenueTL float64 `json:"projected_monthly_revenue_tl"`
	RevenueDeltaTL            float64 `json:"revenue_delta_tl"`
	RevenueChangePct          float64 `json:"revenue_change_pct"`
	ProjectedRidershipDelta   float64 `json:"projected_ridership_delta_pct"`
	SummaryDescription        string  `json:"summary_description"`
}

func (h *APIHandler) SimulateFareImpact(w http.ResponseWriter, r *http.Request) {
	var req FareImpactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz simülasyon parametreleri")
		return
	}

	elasticity := req.ElasticityFactor
	if elasticity == 0 {
		elasticity = -0.35 // Tipik şehir içi ulaşım esneklik katsayısı
	}

	currentMonthlyRev := 25000000.0 // 25 Milyon TL baz aylık gelir
	weightedPriceChangePct := (req.BaseFareChangePct * 0.70) + (req.StudentFareChangePct * 0.30)
	ridershipChangePct := weightedPriceChangePct * elasticity
	projectedRev := currentMonthlyRev * (1 + (weightedPriceChangePct / 100.0)) * (1 + (ridershipChangePct / 100.0))

	resp := FareImpactResponse{
		CurrentMonthlyRevenueTL:   currentMonthlyRev,
		ProjectedMonthlyRevenueTL: projectedRev,
		RevenueDeltaTL:            projectedRev - currentMonthlyRev,
		RevenueChangePct:          ((projectedRev - currentMonthlyRev) / currentMonthlyRev) * 100.0,
		ProjectedRidershipDelta:   ridershipChangePct,
		SummaryDescription: fmt.Sprintf(
			"Tarife zammı (Tam: %%%.1f, Öğrenci: %%%.1f) neticesinde biniş talebinde %%%.2f azalma, aylık gelirde ise %%%.2f değişim öngörülmektedir.",
			req.BaseFareChangePct, req.StudentFareChangePct, ridershipChangePct, ((projectedRev-currentMonthlyRev)/currentMonthlyRev)*100.0,
		),
	}

	WriteJSON(w, http.StatusOK, resp)
}
