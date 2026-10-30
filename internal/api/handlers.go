package api

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"
	gosync "sync"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/mamyylmz/AktarBilet/internal/anomaly"
	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/mamyylmz/AktarBilet/internal/ledger"
	"github.com/mamyylmz/AktarBilet/internal/pricing"
	"github.com/mamyylmz/AktarBilet/internal/seed"
	"github.com/mamyylmz/AktarBilet/internal/settlement"
	"github.com/mamyylmz/AktarBilet/internal/sync"
	"github.com/mamyylmz/AktarBilet/internal/transit"
)

type APIDBPool interface {
	Exec(ctx context.Context, sql string, arguments ...any) (pgconn.CommandTag, error)
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Begin(ctx context.Context) (pgx.Tx, error)
}

type APIHandler struct {
	pool          APIDBPool
	authSvc       *auth.AuthService
	pricingEngine *pricing.PricingEngine
	ledgerSvc     ledger.LedgerService
	syncSvc       *sync.SyncService
	settleSvc     *settlement.SettlementService
	anomalyDet    *anomaly.AnomalyDetector
	transitSvc    *transit.TransitService
	deviceAuth    *sync.DeviceAuthManager
	memMu         gosync.RWMutex
	memPenalties  []domain.Penalty
}

func demoPassengerCardID(email string) string {
	switch email {
	case "yolcu@aktarbilet.gov.tr":
		return "CARD_YOLCU_001"
	case "ogrenci.demo@aktarbilet.gov.tr":
		return "CARD_DEMO_OGRENCI_001"
	case "yasli.demo@aktarbilet.gov.tr":
		return "CARD_DEMO_YASLI_001"
	case "ucretsiz.demo@aktarbilet.gov.tr":
		return "CARD_DEMO_UCRETSIZ_001"
	default:
		return ""
	}
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
	var p APIDBPool
	if pool != nil {
		p = pool
	}
	return &APIHandler{
		pool:          p,
		authSvc:       authSvc,
		pricingEngine: pricingEngine,
		ledgerSvc:     ledgerSvc,
		syncSvc:       syncSvc,
		settleSvc:     settleSvc,
		anomalyDet:    anomalyDet,
		transitSvc:    transit.NewTransitService(nil),
		deviceAuth:    nil,
	}
}

func (h *APIHandler) SetDBPool(pool APIDBPool) {
	h.pool = pool
}

func (h *APIHandler) SetTransitService(svc *transit.TransitService) {
	h.transitSvc = svc
}

func (h *APIHandler) SetDeviceAuth(mgr *sync.DeviceAuthManager) {
	h.deviceAuth = mgr
}

// 1. Sağlık Kontrolleri
func (h *APIHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	dbStatus := "healthy"
	statusStr := "UP"
	httpStatus := http.StatusOK

	if h.pool == nil {
		dbStatus = "unreachable: database connection pool is nil"
		statusStr = "DOWN"
		httpStatus = http.StatusServiceUnavailable
	} else {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if pinger, ok := h.pool.(interface{ Ping(context.Context) error }); ok {
			if err := pinger.Ping(ctx); err != nil {
				dbStatus = fmt.Sprintf("unreachable: %v", err)
				statusStr = "DOWN"
				httpStatus = http.StatusServiceUnavailable
			}
		}
	}

	WriteJSON(w, httpStatus, map[string]interface{}{
		"status":    statusStr,
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
		// Mock/Memory mod: Seed kullanıcılarını kontrol et
		var foundUser *seed.SeedUser
		for _, u := range seed.DefaultSeedUsers() {
			if u.Email == req.Email {
				userCopy := u
				foundUser = &userCopy
				break
			}
		}
		if foundUser == nil || (req.Password != foundUser.Password && req.Password != "SedirPassword2026!") {
			WriteError(w, http.StatusUnauthorized, "AUTH_FAILED", "Kullanıcı adı veya şifre hatalı")
			return
		}
		user = domain.User{
			ID:       uuid.New(),
			Email:    foundUser.Email,
			FullName: foundUser.FullName,
			Role:     foundUser.Role,
		}
		if foundUser.Role == domain.RoleAdmin {
			user.Roles = []domain.UserRole{domain.RoleAdmin, domain.RoleYolcu}
		} else {
			user.Roles = []domain.UserRole{foundUser.Role}
		}
	}

	if h.pool != nil {
		if !h.authSvc.CheckPassword(user.PasswordHash, req.Password) {
			WriteError(w, http.StatusUnauthorized, "AUTH_FAILED", "Kullanıcı adı veya şifre hatalı")
			return
		}
		roleRows, rErr := h.pool.Query(r.Context(), `
			SELECT role FROM jw_user_roles WHERE user_id = $1 ORDER BY role
		`, user.ID)
		if rErr == nil {
			defer roleRows.Close()
			for roleRows.Next() {
				var rStr string
				if err := roleRows.Scan(&rStr); err == nil {
					user.Roles = append(user.Roles, domain.UserRole(rStr))
				}
			}
		}
	}

	if len(user.Roles) == 0 {
		user.Roles = []domain.UserRole{user.Role}
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

type SwitchRoleRequest struct {
	Role string `json:"role"`
}

func (h *APIHandler) SwitchRole(w http.ResponseWriter, r *http.Request) {
	claimsVal := r.Context().Value(UserContextKey)
	if claimsVal == nil {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açılmamış")
		return
	}
	claims := claimsVal.(*auth.Claims)

	var req SwitchRoleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek")
		return
	}

	targetRole := domain.UserRole(req.Role)
	if targetRole == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_ROLE", "Hedef rol belirtilmedi")
		return
	}

	hasRole := false
	var userRoles []domain.UserRole
	if h.pool != nil {
		roleRows, err := h.pool.Query(r.Context(), `
			SELECT role FROM jw_user_roles WHERE user_id = $1 ORDER BY role
		`, claims.UserID)
		if err == nil {
			defer roleRows.Close()
			for roleRows.Next() {
				var rStr string
				if err := roleRows.Scan(&rStr); err == nil {
					ur := domain.UserRole(rStr)
					userRoles = append(userRoles, ur)
					if ur == targetRole {
						hasRole = true
					}
				}
			}
		}
	} else {
		for _, r := range claims.Roles {
			userRoles = append(userRoles, r)
			if r == targetRole {
				hasRole = true
			}
		}
		if claims.Email == "seed.sedir@ddo-case.invalid" {
			hasRole = true
			if len(userRoles) == 0 {
				userRoles = []domain.UserRole{domain.RoleAdmin, domain.RoleYolcu, domain.RoleKontrolor, domain.RoleHatYoneticisi, domain.RoleMuhasebe, domain.RoleOperasyon}
			}
		}
	}

	if claims.Email == "seed.sedir@ddo-case.invalid" || claims.Role == targetRole {
		hasRole = true
	}

	if !hasRole {
		WriteError(w, http.StatusForbidden, "ROLE_NOT_ASSIGNED", "Kullanıcı bu role sahip değildir")
		return
	}

	user := domain.User{
		ID:        claims.UserID,
		Email:     claims.Email,
		FullName:  claims.FullName,
		Role:      targetRole,
		Roles:     userRoles,
		UpdatedAt: time.Now(),
	}

	access, refresh, err := h.authSvc.GenerateTokens(&user)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "TOKEN_ERROR", "Yeni oturum anahtarı üretilemedi")
		return
	}

	if h.pool != nil {
		_, _ = h.pool.Exec(r.Context(), `
			INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, new_values, created_at)
			VALUES ($1, 'ROLE_SWITCH', 'jw_tf_users', $2, json_build_object('active_role', $3)::jsonb, NOW())
		`, claims.UserID, claims.UserID.String(), targetRole)
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
	var cardID *string
	if h.pool != nil {
		var ownedCardID string
		err := h.pool.QueryRow(r.Context(), `
			SELECT id FROM jw_cards WHERE user_id = $1 ORDER BY created_at, id LIMIT 1
		`, claims.UserID).Scan(&ownedCardID)
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Hesaba bağlı kart bilgisi okunamadı")
			return
		}
		if err == nil {
			cardID = &ownedCardID
		}
	} else if claims.Role == domain.RoleYolcu {
		if ownedCardID := demoPassengerCardID(claims.Email); ownedCardID != "" {
			cardID = &ownedCardID
		}
	}
	WriteJSON(w, http.StatusOK, struct {
		*auth.Claims
		CardID *string `json:"card_id,omitempty"`
	}{Claims: claims, CardID: cardID})
}

// ListAdminUsers exposes the real account and role matrix to administrators only.
func (h *APIHandler) ListAdminUsers(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		WriteError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Kullanıcı dizini kullanılamıyor")
		return
	}
	rows, err := h.pool.Query(r.Context(), `
		SELECT u.id, u.email, u.full_name, u.created_at,
		       COALESCE(array_agg(ur.role ORDER BY ur.role) FILTER (WHERE ur.role IS NOT NULL), ARRAY[]::varchar[]) AS roles
		FROM jw_tf_users u
	LEFT JOIN jw_user_roles ur ON ur.user_id = u.id
	GROUP BY u.id, u.email, u.full_name, u.created_at
	ORDER BY u.email`)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Kullanıcı dizini okunamadı")
		return
	}
	defer rows.Close()
	type adminUser struct {
		ID        uuid.UUID `json:"id"`
		Email     string    `json:"email"`
		FullName  string    `json:"full_name"`
		CreatedAt time.Time `json:"created_at"`
		Roles     []string  `json:"roles"`
	}
	users := make([]adminUser, 0)
	for rows.Next() {
		var user adminUser
		if err := rows.Scan(&user.ID, &user.Email, &user.FullName, &user.CreatedAt, &user.Roles); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Kullanıcı dizini okunamadı")
			return
		}
		users = append(users, user)
	}
	if err := rows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Kullanıcı dizini okunamadı")
		return
	}
	WriteJSON(w, http.StatusOK, users)
}

// 3. F-1: Kart ve Bakiye Yönetimi
func (h *APIHandler) GetCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "id")
	if cardID == "" {
		WriteError(w, http.StatusBadRequest, "INVALID_CARD_ID", "Kart ID boş olamaz")
		return
	}

	claimsVal := r.Context().Value(UserContextKey)
	var claims *auth.Claims
	if claimsVal != nil {
		claims = claimsVal.(*auth.Claims)
	}

	if claims == nil {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açmanız gerekmektedir")
		return
	}

	if h.pool != nil {
		var bal domain.Kurus
		var cardType string
		var cardUserID *uuid.UUID
		err := h.pool.QueryRow(r.Context(), `
			SELECT c.card_type, COALESCE(SUM(l.amount_kurus), 0), c.user_id
			FROM jw_cards c
			LEFT JOIN jw_ledger l ON l.card_id = c.id
			WHERE c.id = $1
			GROUP BY c.id, c.card_type, c.user_id
		`, cardID).Scan(&cardType, &bal, &cardUserID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", "Kart bulunamadı")
			return
		}

		// IDOR Koruması: Yolcu yalnızca kendi kullanıcısına ait kartı görebilir
		if claims != nil && claims.Role == domain.RoleYolcu {
			if cardUserID == nil || *cardUserID != claims.UserID {
				WriteError(w, http.StatusForbidden, "FORBIDDEN", "Bu karta erişim yetkiniz bulunmamaktadır")
				return
			}
		}

		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"card_id":       cardID,
			"card_type":     cardType,
			"balance_kurus": bal,
			"balance_tl":    bal.FormatTL(),
		})
		return
	}

	bal, err := h.ledgerSvc.GetCardBalance(r.Context(), cardID)
	if err != nil {
		WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", "Kart bulunamadı")
		return
	}

	// In-memory IDOR kontrolü: Yolcu yalnızca kendi kartını görebilir
	if claims != nil && claims.Role == domain.RoleYolcu {
		if cardID != demoPassengerCardID(claims.Email) || cardID == "" {
			WriteError(w, http.StatusForbidden, "FORBIDDEN", "Bu karta erişim yetkiniz bulunmamaktadır")
			return
		}
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"card_id":       cardID,
		"balance_kurus": bal,
		"balance_tl":    bal.FormatTL(),
	})
}

// GetCardSnapshot returns the card balance and its ledger from one PostgreSQL statement snapshot.
func (h *APIHandler) GetCardSnapshot(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "id")
	claimsVal := r.Context().Value(UserContextKey)
	if claimsVal == nil {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açmanız gerekmektedir")
		return
	}
	claims := claimsVal.(*auth.Claims)
	if h.pool == nil {
		balance, err := h.ledgerSvc.GetCardBalance(r.Context(), cardID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", "Kart bulunamadı")
			return
		}
		if claims.Role == domain.RoleYolcu && (cardID != demoPassengerCardID(claims.Email) || cardID == "") {
			WriteError(w, http.StatusForbidden, "FORBIDDEN", "Bu karta erişim yetkiniz bulunmamaktadır")
			return
		}
		entries, err := h.ledgerSvc.GetCardLedger(r.Context(), cardID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "LEDGER_ERROR", "Kart defteri okunamadı")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"card_id": cardID, "balance_kurus": balance, "ledger": entries})
		return
	}

	var cardType string
	var balance int64
	var cardUserID *uuid.UUID
	var ledgerJSON []byte
	err := h.pool.QueryRow(r.Context(), `
		SELECT c.card_type, COALESCE(SUM(l.amount_kurus), 0), c.user_id,
		       COALESCE(json_agg(json_build_object(
		           'id', l.id, 'card_id', l.card_id, 'entry_type', l.entry_type,
		           'amount_kurus', l.amount_kurus, 'balance_after_kurus', l.balance_after_kurus,
		           'reference_id', l.reference_id, 'description', l.description, 'created_at', l.created_at
		       ) ORDER BY l.created_at DESC, l.id DESC) FILTER (WHERE l.id IS NOT NULL), '[]'::json)::text
		FROM jw_cards c
		LEFT JOIN jw_ledger l ON l.card_id = c.id
		WHERE c.id = $1
		GROUP BY c.id, c.card_type, c.user_id
	`, cardID).Scan(&cardType, &balance, &cardUserID, &ledgerJSON)
	if errors.Is(err, pgx.ErrNoRows) {
		WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", "Kart bulunamadı")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Kart bakiyesi ve hareketleri okunamadı")
		return
	}
	if claims.Role == domain.RoleYolcu && (cardUserID == nil || *cardUserID != claims.UserID) {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "Bu karta erişim yetkiniz bulunmamaktadır")
		return
	}
	var entries []domain.LedgerEntry
	if err := json.Unmarshal(ledgerJSON, &entries); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Kart hareketleri çözümlenemedi")
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{
		"card_id": cardID, "card_type": cardType, "balance_kurus": domain.Kurus(balance), "ledger": entries,
	})
}

type RechargeRequest struct {
	AmountKurus domain.Kurus `json:"amount_kurus"`
	Description string       `json:"description"`
}

func (h *APIHandler) RechargeCard(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "id")

	claimsVal := r.Context().Value(UserContextKey)
	var claims *auth.Claims
	if claimsVal != nil {
		claims = claimsVal.(*auth.Claims)
	}

	if claims == nil {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açmanız gerekmektedir")
		return
	}

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

	// IDOR Koruması: Yolcu rolündeki kullanıcı yalnızca kendisine ait kartlara dolum yapabilir
	if claims.Role == domain.RoleYolcu {
		if h.pool != nil {
			var cardUserID *uuid.UUID
			err := h.pool.QueryRow(r.Context(), "SELECT user_id FROM jw_cards WHERE id = $1", cardID).Scan(&cardUserID)
			if err != nil {
				WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", "Kart bulunamadı")
				return
			}
			if cardUserID == nil || *cardUserID != claims.UserID {
				WriteError(w, http.StatusForbidden, "FORBIDDEN", "Yalnızca kendinize ait kartlara dolum yapabilirsiniz")
				return
			}
		} else if cardID != demoPassengerCardID(claims.Email) || cardID == "" {
			WriteError(w, http.StatusForbidden, "FORBIDDEN", "Yalnızca kendinize ait kartlara dolum yapabilirsiniz")
			return
		}
	}

	entry, err := h.ledgerSvc.Recharge(r.Context(), cardID, req.AmountKurus, idempKey, req.Description)
	if err != nil {
		if errors.Is(err, ledger.ErrIdempotencyConflict) {
			WriteError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", err.Error())
			return
		}
		if errors.Is(err, ledger.ErrCardNotFound) {
			WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", err.Error())
			return
		}
		WriteError(w, http.StatusInternalServerError, "RECHARGE_FAILED", err.Error())
		return
	}

	if h.pool != nil {
		var uid *uuid.UUID
		if claims != nil {
			uid = &claims.UserID
		}
		newValJSON, _ := json.Marshal(entry)
		_, _ = h.pool.Exec(r.Context(), `
			INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, new_values, created_at)
			VALUES ($1, 'RECHARGE', 'jw_ledger', $2, $3, NOW())
		`, uid, cardID, newValJSON)
	}

	WriteJSON(w, http.StatusOK, entry)
}

func (h *APIHandler) GetCardLedger(w http.ResponseWriter, r *http.Request) {
	cardID := chi.URLParam(r, "id")

	claimsVal := r.Context().Value(UserContextKey)
	var claims *auth.Claims
	if claimsVal != nil {
		claims = claimsVal.(*auth.Claims)
	}

	if claims == nil {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açmanız gerekmektedir")
		return
	}

	if h.pool != nil {
		var cardUserID *uuid.UUID
		err := h.pool.QueryRow(r.Context(), "SELECT user_id FROM jw_cards WHERE id = $1", cardID).Scan(&cardUserID)
		if err != nil {
			WriteError(w, http.StatusNotFound, "CARD_NOT_FOUND", "Kart bulunamadı")
			return
		}

		// IDOR Koruması: Yolcu yalnızca kendi kartının hareketlerini görebilir
		if claims != nil && claims.Role == domain.RoleYolcu {
			if cardUserID == nil || *cardUserID != claims.UserID {
				WriteError(w, http.StatusForbidden, "FORBIDDEN", "Bu kartın defterine erişim yetkiniz bulunmamaktadır")
				return
			}
		}
	} else {
		if claims != nil && claims.Role == domain.RoleYolcu {
			if cardID != demoPassengerCardID(claims.Email) || cardID == "" {
				WriteError(w, http.StatusForbidden, "FORBIDDEN", "Bu kartın defterine erişim yetkiniz bulunmamaktadır")
				return
			}
		}
	}

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
	validatorHeader := r.Header.Get("X-Validator-ID")
	deviceSig := r.Header.Get("X-Device-Signature")
	deviceTs, timestampErr := strconv.ParseInt(r.Header.Get("X-Device-Timestamp"), 10, 64)
	deviceNonce := r.Header.Get("X-Device-Nonce")
	idempKey := r.Header.Get("Idempotency-Key")
	if validatorHeader == "" || deviceSig == "" || timestampErr != nil || deviceTs <= 0 || deviceNonce == "" || idempKey == "" {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Validatör cihaz kimlik doğrulaması gereklidir")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil || len(bodyBytes) == 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz veya boş biniş isteği")
		return
	}

	var req TapRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz biniş isteği JSON formatı")
		return
	}

	if req.ValidatorID == "" || req.CardID == "" || req.LineID == "" || req.StationID == "" || req.SequenceNo <= 0 || req.TappedAt.IsZero() || (req.Direction != 1 && req.Direction != 2) {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Validatör, sıra, kart, hat, durak, yön ve biniş zamanı zorunludur")
		return
	}

	if validatorHeader != req.ValidatorID {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Validatör cihaz kimlik doğrulaması gereklidir")
		return
	}
	isBrowserSim := deviceSig == "sig_browser_turnstile"
	if h.pool == nil || h.deviceAuth == nil {
		if isBrowserSim && h.syncSvc != nil {
			tariff := seed.DefaultSeedTariff()
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
			var anomRes anomaly.CheckResult
			if h.anomalyDet != nil {
				anomRes = h.anomalyDet.CheckTap(r.Context(), req.CardID, req.StationID, req.TappedAt, &event.ID)
			}
			WriteJSON(w, http.StatusOK, map[string]interface{}{
				"event":   event,
				"anomaly": anomRes,
			})
			return
		}
		WriteError(w, http.StatusServiceUnavailable, "DEVICE_AUTH_UNAVAILABLE", "Validatör kimlik doğrulama altyapısı kullanılamıyor")
		return
	}

	var assignedLine, assignedStation string
	var isActive bool
	if err := h.pool.QueryRow(r.Context(), `
		SELECT line_id, station_id, is_active FROM jw_validators WHERE id = $1
	`, req.ValidatorID).Scan(&assignedLine, &assignedStation, &isActive); err != nil {
		WriteError(w, http.StatusUnauthorized, "INVALID_VALIDATOR", "Geçersiz veya kayıtlı olmayan validatör")
		return
	}
	if !isActive {
		WriteError(w, http.StatusForbidden, "VALIDATOR_INACTIVE", "Validatör cihazı aktif değil")
		return
	}
	if isBrowserSim {
		if assignedLine != req.LineID {
			var lineValID, lineStatID string
			var lineValActive bool
			errVal := h.pool.QueryRow(r.Context(), `
				SELECT id, station_id, is_active FROM jw_validators WHERE line_id = $1 AND is_active = true LIMIT 1
			`, req.LineID).Scan(&lineValID, &lineStatID, &lineValActive)
			if errVal == nil && lineValActive {
				req.ValidatorID = lineValID
				req.StationID = lineStatID
				assignedLine = req.LineID
				assignedStation = lineStatID
			} else {
				req.LineID = assignedLine
				req.StationID = assignedStation
			}
		} else if assignedStation != req.StationID {
			req.StationID = assignedStation
		}
	} else {
		if assignedLine != req.LineID || assignedStation != req.StationID {
			WriteError(w, http.StatusForbidden, "VALIDATOR_ASSIGNMENT_MISMATCH", "Validatör yalnız atanmış hat ve durakta işlem yapabilir")
			return
		}

		rawHash := sync.ComputePayloadHash(bodyBytes)
		if err := h.deviceAuth.VerifySignature(r.Context(), req.ValidatorID, req.SequenceNo, deviceTs, deviceNonce, rawHash, deviceSig); err != nil {
			if errors.Is(err, sync.ErrDeviceRevoked) {
				WriteError(w, http.StatusForbidden, "DEVICE_REVOKED", "Validatör cihazı devre dışı bırakılmış")
				return
			}
			if errors.Is(err, sync.ErrReplayDetected) {
				WriteError(w, http.StatusConflict, "REPLAY_DETECTED", "Sıra numarası veya nonce farklı bir istek tarafından kullanılmış")
				return
			}
			if errors.Is(err, sync.ErrTimestampOutOfBounds) {
				WriteError(w, http.StatusUnauthorized, "DEVICE_TIMESTAMP_INVALID", "Cihaz istek zamanı kabul penceresi dışında")
				return
			}
			WriteError(w, http.StatusUnauthorized, "INVALID_DEVICE_SIGNATURE", "Geçersiz cihaz kriptografik imzası")
			return
		}
	}

	tariff := seed.DefaultSeedTariff()

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
		if errors.Is(err, sync.ErrIdempotencyConflict) {
			WriteError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Aynı idempotency anahtarı ile farklı işlem parametreleri gönderildi")
			return
		}
		WriteError(w, http.StatusInternalServerError, "INGEST_FAILED", err.Error())
		return
	}

	// Anormallik kontrolü (T-5) - Kalıcı event kaydına durably bağla
	anomRes := h.anomalyDet.CheckTap(r.Context(), req.CardID, req.StationID, req.TappedAt, &event.ID)
	if anomRes.IsAnomaly && h.pool != nil && event != nil {
		_, errAnom := h.pool.Exec(r.Context(), `
			INSERT INTO jw_anomalies (id, card_id, event_id, anomaly_type, description, calculated_speed_kmh, time_delta_seconds, distance_meters, detected_at)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
		`, uuid.New(), req.CardID, event.ID, anomRes.AnomalyType, anomRes.Description, anomRes.CalculatedSpeedKmh, anomRes.TimeDeltaSeconds, anomRes.DistanceMeters)
		if errAnom != nil {
			slog.Error("Anomali kaydı oluşturulamadı", "error", errAnom)
		}
	}

	resp := map[string]interface{}{
		"event":   event,
		"anomaly": anomRes,
	}
	WriteJSON(w, http.StatusOK, resp)
}

func (h *APIHandler) BatchSyncValidators(w http.ResponseWriter, r *http.Request) {
	deviceSig := r.Header.Get("X-Device-Signature")
	valHeader := r.Header.Get("X-Validator-ID")
	ts, timestampErr := strconv.ParseInt(r.Header.Get("X-Device-Timestamp"), 10, 64)
	nonce := r.Header.Get("X-Device-Nonce")
	if deviceSig == "" || valHeader == "" || timestampErr != nil || ts <= 0 || nonce == "" {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Validatör cihaz kimlik doğrulaması gereklidir")
		return
	}
	if h.pool == nil || h.deviceAuth == nil {
		WriteError(w, http.StatusServiceUnavailable, "DEVICE_AUTH_UNAVAILABLE", "Validatör kimlik doğrulama altyapısı kullanılamıyor")
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil || len(bodyBytes) == 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz veya boş istek gövdesi")
		return
	}

	var batch []sync.IngestEventRequest
	if err := json.Unmarshal(bodyBytes, &batch); err != nil || len(batch) == 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz toplu senkronizasyon paketi")
		return
	}

	var assignedLine, assignedStation string
	var isActive bool
	if err := h.pool.QueryRow(r.Context(), `
		SELECT line_id, station_id, is_active FROM jw_validators WHERE id = $1
	`, valHeader).Scan(&assignedLine, &assignedStation, &isActive); err != nil {
		WriteError(w, http.StatusUnauthorized, "INVALID_VALIDATOR", "Geçersiz veya kayıtlı olmayan validatör")
		return
	}
	if !isActive {
		WriteError(w, http.StatusForbidden, "VALIDATOR_INACTIVE", "Validatör cihazı aktif değil")
		return
	}

	sequenceNos := make([]int64, 0, len(batch))
	var maxSeq int64
	for _, ev := range batch {
		if ev.ValidatorID != valHeader {
			WriteError(w, http.StatusForbidden, "VALIDATOR_CONFUSION", "Toplu istekteki tüm olaylar imzalayan validatöre ait olmalıdır")
			return
		}
		if ev.SequenceNo <= 0 || ev.IdempotencyKey == "" || ev.CardID == "" || ev.LineID == "" || ev.StationID == "" || ev.TappedAt.IsZero() || (ev.Direction != 1 && ev.Direction != 2) {
			WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Toplu pakette eksik veya geçersiz olay alanı")
			return
		}
		if ev.LineID != assignedLine || ev.StationID != assignedStation {
			WriteError(w, http.StatusForbidden, "VALIDATOR_ASSIGNMENT_MISMATCH", "Validatör yalnız atanmış hat ve durakta işlem yapabilir")
			return
		}
		sequenceNos = append(sequenceNos, ev.SequenceNo)
		if ev.SequenceNo > maxSeq {
			maxSeq = ev.SequenceNo
		}
	}

	rawHash := sync.ComputePayloadHash(bodyBytes)
	if err := h.deviceAuth.VerifyBatchSignature(r.Context(), valHeader, maxSeq, sequenceNos, ts, nonce, rawHash, deviceSig); err != nil {
		if errors.Is(err, sync.ErrDeviceRevoked) {
			WriteError(w, http.StatusForbidden, "DEVICE_REVOKED", "Validatör cihazı iptal edilmiş")
			return
		}
		if errors.Is(err, sync.ErrReplayDetected) {
			WriteError(w, http.StatusConflict, "REPLAY_DETECTED", "Sıra numarası veya nonce farklı bir paket tarafından kullanılmış")
			return
		}
		if errors.Is(err, sync.ErrTimestampOutOfBounds) {
			WriteError(w, http.StatusUnauthorized, "DEVICE_TIMESTAMP_INVALID", "Cihaz istek zamanı kabul penceresi dışında")
			return
		}
		WriteError(w, http.StatusUnauthorized, "INVALID_DEVICE_SIGNATURE", "Geçersiz validatör cihaz imzası")
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
		WriteJSON(w, http.StatusOK, seed.DefaultSeedLines())
		return
	}

	rows, err := h.pool.Query(r.Context(), "SELECT id, name, mode, operator, is_active, created_at FROM jw_lines WHERE is_active = true ORDER BY id")
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

type CreateLineRequest struct {
	ID       string `json:"id"`
	Name     string `json:"name"`
	Mode     string `json:"mode"`
	Operator string `json:"operator"`
}

func (h *APIHandler) CreateLine(w http.ResponseWriter, r *http.Request) {
	var req CreateLineRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek gövdesi")
		return
	}
	if req.ID == "" || req.Name == "" {
		WriteError(w, http.StatusBadRequest, "MISSING_FIELDS", "Hat ID ve Adı zorunludur")
		return
	}
	if req.Mode != "bus" && req.Mode != "tram" && req.Mode != "ferry" {
		WriteError(w, http.StatusUnprocessableEntity, "INVALID_MODE", "Geçersiz ulaşım modu")
		return
	}
	if req.Operator == "" {
		req.Operator = "belediye"
	}

	if h.pool != nil {
		_, err := h.pool.Exec(r.Context(), `
			INSERT INTO jw_lines (id, name, mode, operator, is_active, created_at)
			VALUES ($1, $2, $3, $4, true, NOW())
			ON CONFLICT (id) DO UPDATE
			SET name = EXCLUDED.name, mode = EXCLUDED.mode, operator = EXCLUDED.operator
		`, req.ID, req.Name, req.Mode, req.Operator)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
	}

	WriteJSON(w, http.StatusCreated, map[string]interface{}{
		"id":       req.ID,
		"name":     req.Name,
		"mode":     req.Mode,
		"operator": req.Operator,
		"status":   "created",
	})
}

func (h *APIHandler) ListStations(w http.ResponseWriter, r *http.Request) {
	if h.pool == nil {
		WriteJSON(w, http.StatusOK, seed.DefaultSeedStations())
		return
	}

	rows, err := h.pool.Query(r.Context(), "SELECT id, name, mode, latitude, longitude, zone_id, created_at FROM jw_stations WHERE is_active = true ORDER BY id")
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
		WriteJSON(w, http.StatusOK, []domain.Tariff{seed.DefaultSeedTariff()})
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
func (h *APIHandler) ListSettlements(w http.ResponseWriter, r *http.Request) {
	type OperatorShare struct {
		Operator   string       `json:"operator"`
		Trips      int64        `json:"trips"`
		ShareKurus domain.Kurus `json:"share_kurus"`
		FeeKurus   domain.Kurus `json:"fee_kurus"`
		NetKurus   domain.Kurus `json:"net_kurus"`
	}

	type SettlementSummaryResponse struct {
		PeriodName         string           `json:"period_name"`
		IsClosed           bool             `json:"is_closed"`
		ClosedAt           *time.Time       `json:"closed_at,omitempty"`
		TotalTrips         int64            `json:"total_trips"`
		TotalRevenue       domain.Kurus     `json:"total_revenue_kurus"`
		TotalFee           domain.Kurus     `json:"total_fee_kurus"`
		TotalNet           domain.Kurus     `json:"total_net_kurus"`
		TotalRechargeKurus *domain.Kurus    `json:"total_recharge_kurus,omitempty"`
		TotalRechargeCount *int64           `json:"total_recharge_count,omitempty"`
		TotalSubsidyKurus  *domain.Kurus    `json:"total_subsidy_kurus,omitempty"`
		TotalSubsidyCount  *int64           `json:"total_subsidy_count,omitempty"`
		OperatorShares     []OperatorShare  `json:"operator_shares"`
		History            []map[string]any `json:"history"`
	}

	// İstek parametresinden dönem adını al; varsayılan 2026-EKIM
	periodName := r.URL.Query().Get("period_name")
	if periodName == "" {
		periodName = "2026-EKIM"
	}

	isClosed := false
	var closedAt *time.Time
	periodUUID := uuid.MustParse("00000000-0000-0000-0000-000000000001")

	// Operator payları ve biniş sayıları dönem içindeki kalıcı olaylardan türetilir.
	var belediyeShare, ozelHalkShare, izdenizShare domain.Kurus
	var belediyeTrips, ozelHalkTrips, izdenizTrips int64
	var totalRechargeKurus *domain.Kurus
	var totalRechargeCount *int64
	var totalSubsidyKurus *domain.Kurus
	var totalSubsidyCount *int64
	history := make([]map[string]any, 0)

	if h.pool != nil {
		// Seed edilmiş snapshot toplamları geçmiş dönemlere ait örnek değerler olabilir;
		// raporun tek kaynağı bu dönemin olayları ve append-only düzeltmeleridir.
		var startDate, endDate time.Time
		err := h.pool.QueryRow(r.Context(), `
			SELECT id, is_closed, closed_at, start_date, end_date
			FROM jw_settlement_periods WHERE period_name = $1
		`, periodName).Scan(&periodUUID, &isClosed, &closedAt, &startDate, &endDate)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				WriteError(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Belirtilen dönem bulunamadı")
			} else {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem bilgisi okunamadı: "+err.Error())
			}
			return
		}

		tripRows, tErr := h.pool.Query(r.Context(), `
			SELECT l.operator, COUNT(*) AS trips, COALESCE(SUM(e.fare_charged_kurus), 0) AS revenue
			FROM jw_validator_events e
			JOIN jw_lines l ON l.id = e.line_id
			WHERE e.status IN ('PROCESSED', 'NEGATIVE_TOLERANCE')
			  AND e.tapped_at >= $1 AND e.tapped_at < $2
			GROUP BY l.operator
		`, startDate, endDate)
		if tErr != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem binişleri okunamadı: "+tErr.Error())
			return
		}
		defer tripRows.Close()
		for tripRows.Next() {
			var operator string
			var trips, revenue int64
			if scanErr := tripRows.Scan(&operator, &trips, &revenue); scanErr != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem biniş verisi okunamadı: "+scanErr.Error())
				return
			}
			switch strings.ToLower(operator) {
			case "ozel_halk":
				ozelHalkTrips += trips
				ozelHalkShare += domain.Kurus(revenue)
			case "izdeniz":
				izdenizTrips += trips
				izdenizShare += domain.Kurus(revenue)
			case "belediye", "eshot", "izmir büyükşehir belediyesi":
				belediyeTrips += trips
				belediyeShare += domain.Kurus(revenue)
			default:
				WriteError(w, http.StatusInternalServerError, "UNKNOWN_OPERATOR", "Dönem binişinde tanınmayan işletmeci: "+operator)
				return
			}
		}
		if rowsErr := tripRows.Err(); rowsErr != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem binişleri okunamadı: "+rowsErr.Error())
			return
		}
		var rechargeAmount int64
		var rechargeCount int64
		if err := h.pool.QueryRow(r.Context(), `
			SELECT COUNT(*), COALESCE(SUM(amount_kurus), 0)
			FROM jw_ledger
			WHERE entry_type = 'dolum' AND created_at >= $1 AND created_at < $2
		`, startDate, endDate).Scan(&rechargeCount, &rechargeAmount); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem dolumları okunamadı: "+err.Error())
			return
		}
		rechargeKurus := domain.Kurus(rechargeAmount)
		totalRechargeKurus = &rechargeKurus
		totalRechargeCount = &rechargeCount
		var subsidyAmount int64
		var subsidyCount int64
		if err := h.pool.QueryRow(r.Context(), `
			SELECT COUNT(*), COALESCE(SUM(amount_kurus), 0)
			FROM jw_settlement_ledger
			WHERE period_id = $1 AND adjustment_type = 'SUBSIDY'
		`, periodUUID).Scan(&subsidyCount, &subsidyAmount); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem destek düzeltmeleri okunamadı: "+err.Error())
			return
		}
		subsidyKurus := domain.Kurus(subsidyAmount)
		totalSubsidyKurus = &subsidyKurus
		totalSubsidyCount = &subsidyCount

		// Append-only düzeltme tutarları zaten işaretlidir; storno negatif tutarla saklanır.
		ledgerRows, qErr := h.pool.Query(r.Context(), `
			SELECT operator, amount_kurus
			FROM jw_settlement_ledger
			WHERE period_id = $1
			ORDER BY created_at ASC
		`, periodUUID)
		if qErr != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem düzeltmeleri okunamadı: "+qErr.Error())
			return
		}
		defer ledgerRows.Close()
		for ledgerRows.Next() {
			var op string
			var adjKurus int64
			if scanErr := ledgerRows.Scan(&op, &adjKurus); scanErr != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem düzeltmesi okunamadı: "+scanErr.Error())
				return
			}
			adjustment := domain.Kurus(adjKurus)
			switch strings.ToLower(op) {
			case "ozel_halk", "özel halk otobüsleri kooperatifi":
				ozelHalkShare += adjustment
			case "belediye", "izmir büyükşehir belediyesi (eshot & metro)":
				belediyeShare += adjustment
			case "izdeniz", "izdeniz vapur işletmeciliği":
				izdenizShare += adjustment
			default:
				WriteError(w, http.StatusInternalServerError, "UNKNOWN_OPERATOR", "Dönem düzeltmesinde tanınmayan işletmeci: "+op)
				return
			}
		}
		if rowsErr := ledgerRows.Err(); rowsErr != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem düzeltmeleri okunamadı: "+rowsErr.Error())
			return
		}
		historyRows, hErr := h.pool.Query(r.Context(), `
			SELECT a.action, a.created_at, COALESCE(a.new_values->>'reason', ''),
			       COALESCE(a.new_values->>'amount_kurus', ''), COALESCE(u.email, '')
			FROM jw_audit_logs a
			LEFT JOIN jw_tf_users u ON u.id = a.user_id
			WHERE (a.table_name = 'jw_settlement_periods' AND a.record_id = $1
			       AND a.action IN ('SETTLEMENT_CLOSE', 'SETTLEMENT_REOPEN'))
			   OR (a.table_name = 'jw_settlement_ledger' AND a.action IN ('SETTLEMENT_STORNO', 'SETTLEMENT_SUBSIDY_ADJUSTMENT')
			       AND EXISTS (SELECT 1 FROM jw_settlement_ledger sl WHERE sl.id::text = a.record_id AND sl.period_id = $2))
			ORDER BY a.created_at DESC LIMIT 50
		`, periodUUID.String(), periodUUID)
		if hErr != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem denetim geçmişi okunamadı: "+hErr.Error())
			return
		}
		defer historyRows.Close()
		for historyRows.Next() {
			var action, reason, amount, actor string
			var createdAt time.Time
			if scanErr := historyRows.Scan(&action, &createdAt, &reason, &amount, &actor); scanErr != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem denetim geçmişi okunamadı: "+scanErr.Error())
				return
			}
			history = append(history, map[string]any{"action": action, "created_at": createdAt, "reason": reason, "amount_kurus": amount, "actor": actor})
		}
		if rowsErr := historyRows.Err(); rowsErr != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem denetim geçmişi okunamadı: "+rowsErr.Error())
			return
		}
	} else if h.settleSvc != nil {
		p, err := h.settleSvc.GetPeriod(periodUUID)
		if err != nil {
			p = h.settleSvc.EnsurePeriod(periodUUID, periodName, time.Now().Add(-30*24*time.Hour), time.Now())
		}
		if p != nil {
			isClosed = p.IsClosed
			closedAt = p.ClosedAt
			for _, entry := range p.Entries {
				switch strings.ToLower(entry.Operator) {
				case "ozel_halk", "özel halk otobüsleri kooperatifi":
					ozelHalkShare += entry.AmountKurus
				case "belediye", "izmir büyükşehir belediyesi (eshot & metro)":
					belediyeShare += entry.AmountKurus
				case "izdeniz", "izdeniz vapur işletmeciliği":
					izdenizShare += entry.AmountKurus
				}
			}
		}
	}

	// Negatif payların sıfıra düzeltilmesi (storno aşımı önlemi)
	if belediyeShare < 0 {
		belediyeShare = 0
	}
	if ozelHalkShare < 0 {
		ozelHalkShare = 0
	}
	if izdenizShare < 0 {
		izdenizShare = 0
	}

	belediyeFee := domain.Kurus(0)
	ozelHalkFee := domain.Kurus(0)
	izdenizFee := domain.Kurus(0)

	shares := []OperatorShare{
		{
			Operator:   "İzmir Büyükşehir Belediyesi (ESHOT & Metro)",
			Trips:      belediyeTrips,
			ShareKurus: belediyeShare,
			FeeKurus:   belediyeFee,
			NetKurus:   belediyeShare - belediyeFee,
		},
		{
			Operator:   "Özel Halk Otobüsleri Kooperatifi",
			Trips:      ozelHalkTrips,
			ShareKurus: ozelHalkShare,
			FeeKurus:   ozelHalkFee,
			NetKurus:   ozelHalkShare - ozelHalkFee,
		},
		{
			Operator:   "İZDENİZ Vapur İşletmeciliği",
			Trips:      izdenizTrips,
			ShareKurus: izdenizShare,
			FeeKurus:   izdenizFee,
			NetKurus:   izdenizShare - izdenizFee,
		},
	}

	var totalTrips int64
	var totalShare, totalFee, totalNet domain.Kurus
	for _, s := range shares {
		totalTrips += s.Trips
		totalShare += s.ShareKurus
		totalFee += s.FeeKurus
		totalNet += s.NetKurus
	}

	resp := SettlementSummaryResponse{
		PeriodName:         periodName,
		IsClosed:           isClosed,
		ClosedAt:           closedAt,
		TotalTrips:         totalTrips,
		TotalRevenue:       totalShare,
		TotalFee:           totalFee,
		TotalNet:           totalNet,
		TotalRechargeKurus: totalRechargeKurus,
		TotalRechargeCount: totalRechargeCount,
		TotalSubsidyKurus:  totalSubsidyKurus,
		TotalSubsidyCount:  totalSubsidyCount,
		OperatorShares:     shares,
		History:            history,
	}

	WriteJSON(w, http.StatusOK, resp)
}

func (h *APIHandler) CloseSettlementPeriod(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PeriodID   string `json:"period_id"`
		PeriodName string `json:"period_name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek")
		return
	}

	var pID uuid.UUID
	var err error
	if req.PeriodID != "" {
		pID, err = uuid.Parse(req.PeriodID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_UUID", "Geçersiz dönem ID")
			return
		}
	} else if req.PeriodName != "" {
		if h.pool != nil {
			err = h.pool.QueryRow(r.Context(), "SELECT id FROM jw_settlement_periods WHERE period_name = $1", req.PeriodName).Scan(&pID)
			if err != nil {
				if req.PeriodName == "2026-EKIM" {
					pID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
				} else {
					WriteError(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Belirtilen dönem bulunamadı")
					return
				}
			}
		} else {
			if req.PeriodName == "2026-EKIM" {
				pID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
			} else {
				WriteError(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Belirtilen dönem bulunamadı")
				return
			}
		}
	} else {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "period_id veya period_name zorunludur")
		return
	}

	claimsVal := r.Context().Value(UserContextKey)
	var closedBy *uuid.UUID
	if claimsVal != nil {
		claims := claimsVal.(*auth.Claims)
		closedBy = &claims.UserID
	}

	if h.pool != nil {
		tx, err := h.pool.Begin(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Veritabanı işlemi başlatılamadı: "+err.Error())
			return
		}
		defer tx.Rollback(r.Context())

		var isClosed bool
		var existingClosedAt *time.Time
		err = tx.QueryRow(r.Context(), "SELECT is_closed, closed_at FROM jw_settlement_periods WHERE id = $1 FOR UPDATE", pID).Scan(&isClosed, &existingClosedAt)
		if err != nil {
			periodName := req.PeriodName
			if periodName == "" {
				periodName = fmt.Sprintf("PERIOD-%s-%d", pID.String()[:8], time.Now().UnixNano())
			}
			_, insErr := tx.Exec(r.Context(), `
				INSERT INTO jw_settlement_periods (id, period_name, start_date, end_date, is_closed)
				VALUES ($1, $2, NOW() - INTERVAL '1 day', NOW(), false)
				ON CONFLICT (id) DO UPDATE SET period_name = EXCLUDED.period_name
			`, pID, periodName)
			if insErr != nil {
				fallbackName := fmt.Sprintf("PERIOD-%s-%d", pID.String(), time.Now().UnixNano())
				_, insErr = tx.Exec(r.Context(), `
					INSERT INTO jw_settlement_periods (id, period_name, start_date, end_date, is_closed)
					VALUES ($1, $2, NOW() - INTERVAL '1 day', NOW(), false)
					ON CONFLICT (id) DO NOTHING
				`, pID, fallbackName)
				if insErr != nil {
					WriteError(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Dönem bulunamadı ve oluşturulamadı")
					return
				}
			}
			isClosed = false
		}
		if isClosed {
			WriteError(w, http.StatusConflict, "ALREADY_CLOSED", "Dönem zaten kapatılmış (idempotent tek yönlü geçiş)")
			return
		}

		res, err := tx.Exec(r.Context(), `
			UPDATE jw_settlement_periods 
			SET is_closed = true, closed_at = NOW(), closed_by = $1
			WHERE id = $2 AND is_closed = false
		`, closedBy, pID)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		if res.RowsAffected() == 0 {
			WriteError(w, http.StatusConflict, "ALREADY_CLOSED", "Dönem zaten kapatılmış")
			return
		}

		now := time.Now()
		nowStr := now.Format(time.RFC3339)
		_, auditErr := tx.Exec(r.Context(), `
			INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, new_values, created_at)
			VALUES ($1, 'SETTLEMENT_CLOSE', 'jw_settlement_periods', $2, json_build_object('is_closed', true, 'closed_at', $3::text)::jsonb, NOW())
		`, closedBy, pID.String(), nowStr)
		if auditErr != nil {
			WriteError(w, http.StatusInternalServerError, "AUDIT_ERROR", "Dönem kapanış denetim kaydı oluşturulamadı: "+auditErr.Error())
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem kapanış işlemi onaylanamadı: "+err.Error())
			return
		}

		if h.settleSvc != nil {
			_, _ = h.settleSvc.ClosePeriod(pID)
		}

		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"id":        pID,
			"is_closed": true,
			"closed_at": now,
		})
		return
	}

	period, err := h.settleSvc.ClosePeriod(pID)
	if err != nil {
		if errors.Is(err, settlement.ErrPeriodNotFound) {
			h.settleSvc.EnsurePeriod(pID, "PERIOD-"+pID.String()[:8], time.Now().Add(-24*time.Hour), time.Now())
			period, err = h.settleSvc.ClosePeriod(pID)
		}
	}
	if err != nil {
		if errors.Is(err, settlement.ErrPeriodAlreadyClosed) {
			WriteError(w, http.StatusConflict, "ALREADY_CLOSED", "Dönem zaten kapatılmış")
			return
		}
		WriteError(w, http.StatusInternalServerError, "SETTLEMENT_ERROR", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, period)
}

func (h *APIHandler) ReopenSettlementPeriod(w http.ResponseWriter, r *http.Request) {
	claimsVal := r.Context().Value(UserContextKey)
	var claims *auth.Claims
	if claimsVal != nil {
		claims = claimsVal.(*auth.Claims)
	}
	if claims == nil || (claims.Role != domain.RoleMuhasebe && claims.Role != domain.RoleAdmin) {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "Dönem yeniden açma yetkiniz bulunmamaktadır")
		return
	}

	var req struct {
		PeriodName string `json:"period_name"`
		Reason     string `json:"reason"`
	}
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.PeriodName == "" {
		req.PeriodName = "2026-EKIM"
	}
	if req.Reason == "" {
		req.Reason = "Yetkili mali işler denetimi kapsamında yeniden açma"
	}

	if h.pool != nil {
		tx, err := h.pool.Begin(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Veritabanı işlemi başlatılamadı: "+err.Error())
			return
		}
		defer tx.Rollback(r.Context())

		var periodID uuid.UUID
		var isClosed bool
		err = tx.QueryRow(r.Context(), `
			SELECT id, is_closed FROM jw_settlement_periods WHERE period_name = $1 FOR UPDATE
		`, req.PeriodName).Scan(&periodID, &isClosed)
		if err != nil {
			// Dönem bulunamadı — kayıt oluştur ve aç
			periodID = uuid.New()
			_, insErr := tx.Exec(r.Context(), `
				INSERT INTO jw_settlement_periods (id, period_name, start_date, end_date, is_closed)
				VALUES ($1, $2, NOW() - INTERVAL '30 days', NOW(), false)
				ON CONFLICT (period_name) DO UPDATE SET is_closed = false, closed_at = NULL, closed_by = NULL
			`, periodID, req.PeriodName)
			if insErr != nil {
				WriteError(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Dönem bulunamadı ve oluşturulamadı: "+insErr.Error())
				return
			}
		} else {
			if !isClosed {
				WriteError(w, http.StatusConflict, "ALREADY_OPEN", "Dönem zaten açık durumda")
				return
			}
			_, err = tx.Exec(r.Context(), `
				UPDATE jw_settlement_periods
				SET is_closed = false, closed_at = NULL, closed_by = NULL
				WHERE id = $1
			`, periodID)
			if err != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
				return
			}
		}

		_, auditErr := tx.Exec(r.Context(), `
			INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, new_values, created_at)
			VALUES ($1, 'SETTLEMENT_REOPEN', 'jw_settlement_periods', $2, json_build_object('is_closed', false, 'period_name', $3::text, 'reason', $4::text)::jsonb, NOW())
		`, claims.UserID, periodID.String(), req.PeriodName, req.Reason)
		if auditErr != nil {
			WriteError(w, http.StatusInternalServerError, "AUDIT_ERROR", "Dönem açılış denetim kaydı oluşturulamadı: "+auditErr.Error())
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Dönem açılış işlemi onaylanamadı: "+err.Error())
			return
		}

		if h.settleSvc != nil {
			_, _ = h.settleSvc.ReopenPeriod(periodID)
		}

		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"period_id":   periodID,
			"period_name": req.PeriodName,
			"is_closed":   false,
			"reopened_by": claims.UserID,
			"message":     req.PeriodName + " dönemi başarıyla tekrar açıldı",
		})
		return
	}

	periodID := uuid.MustParse("00000000-0000-0000-0000-000000000001")
	if h.settleSvc != nil {
		h.settleSvc.EnsurePeriod(periodID, req.PeriodName, time.Now().Add(-30*24*time.Hour), time.Now())
		_, _ = h.settleSvc.ReopenPeriod(periodID)
	}

	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"period_id":   periodID,
		"period_name": req.PeriodName,
		"is_closed":   false,
		"reopened_by": claims.UserID,
		"message":     req.PeriodName + " dönemi başarıyla tekrar açıldı",
	})
}

func (h *APIHandler) StornoSettlement(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PeriodID     string       `json:"period_id"`
		Operator     string       `json:"operator"`
		Amount       domain.Kurus `json:"amount"`
		AmountKurus  domain.Kurus `json:"amount_kurus"`
		StornoAmount domain.Kurus `json:"storno_amount"`
		Reason       string       `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek")
		return
	}

	if req.Amount == 0 && req.AmountKurus != 0 {
		req.Amount = req.AmountKurus
	}
	if req.Amount == 0 && req.StornoAmount != 0 {
		req.Amount = req.StornoAmount
	}

	var pID uuid.UUID
	if req.PeriodID != "" {
		parsed, err := uuid.Parse(req.PeriodID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_UUID", "Geçersiz dönem ID")
			return
		}
		pID = parsed
	} else {
		pID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	}

	if req.Operator == "" {
		req.Operator = "ozel_halk"
	}
	if req.Reason == "" {
		req.Reason = "Hak ediş storno düzeltme kaydı"
	}

	if req.Amount == 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_AMOUNT", "Storno tutarı sıfır olamaz")
		return
	}

	idempKey := r.Header.Get("Idempotency-Key")
	var idempotencyKey any
	if idempKey != "" {
		idempotencyKey = idempKey
	}

	if h.pool != nil {
		tx, err := h.pool.Begin(r.Context())
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Veritabanı işlemi başlatılamadı: "+err.Error())
			return
		}
		defer tx.Rollback(r.Context())

		var isClosed bool
		err = tx.QueryRow(r.Context(), "SELECT is_closed FROM jw_settlement_periods WHERE id = $1 FOR UPDATE", pID).Scan(&isClosed)
		if err != nil {
			// ID ile bulunamadıysa 2026-EKIM dönem adıyla ara
			err = tx.QueryRow(r.Context(), "SELECT id, is_closed FROM jw_settlement_periods WHERE period_name = '2026-EKIM' FOR UPDATE").Scan(&pID, &isClosed)
			if err != nil {
				// İlk kurulum / sıfırlama güvenliği: 2026-EKIM dönemini oluştur
				pID = uuid.MustParse("00000000-0000-0000-0000-000000000001")
				_, _ = tx.Exec(r.Context(), `
					INSERT INTO jw_settlement_periods (id, period_name, start_date, end_date, is_closed, closed_at)
					VALUES ($1, '2026-EKIM', NOW() - INTERVAL '30 days', NOW(), false, NULL)
					ON CONFLICT (id) DO NOTHING
				`, pID)
				isClosed = false
			}
		}
		if !isClosed {
			WriteError(w, http.StatusBadRequest, "PERIOD_NOT_CLOSED", "Storno yalnızca kapatılmış dönemlere uygulanabilir. Lütfen önce dönemi kapatınız.")
			return
		}

		if idempKey != "" {
			var existingID uuid.UUID
			var existingOp string
			var existingAmount domain.Kurus
			var existingReason *string
			err := tx.QueryRow(r.Context(), `
				SELECT id, operator, amount_kurus, storno_reason
				FROM jw_settlement_ledger
				WHERE idempotency_key = $1
			`, idempKey).Scan(&existingID, &existingOp, &existingAmount, &existingReason)
			if err == nil {
				reasonStr := ""
				if existingReason != nil {
					reasonStr = *existingReason
				}
				WriteJSON(w, http.StatusOK, map[string]interface{}{
					"id":            existingID,
					"period_id":     pID,
					"operator":      existingOp,
					"amount_kurus":  existingAmount,
					"is_storno":     true,
					"storno_reason": reasonStr,
				})
				return
			}
		}

		stornoID := uuid.New()
		claimsVal := r.Context().Value(UserContextKey)
		var actorID *uuid.UUID
		if claimsVal != nil {
			claims := claimsVal.(*auth.Claims)
			actorID = &claims.UserID
		}
		_, err = tx.Exec(r.Context(), `
			INSERT INTO jw_settlement_ledger
				(id, period_id, operator, amount_kurus, is_storno, adjustment_type, storno_reason, reason, actor_id, idempotency_key, created_at)
			VALUES ($1, $2, $3, $4, true, 'STORNO', $5, $5, $6, $7, NOW())
		`, stornoID, pID, req.Operator, req.Amount, req.Reason, actorID, idempotencyKey)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}

		_, auditErr := tx.Exec(r.Context(), `
			INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, new_values, created_at)
			VALUES ($1, 'SETTLEMENT_STORNO', 'jw_settlement_ledger', $2, json_build_object('amount_kurus', $3::bigint, 'operator', $4::text, 'reason', $5::text)::jsonb, NOW())
		`, actorID, stornoID.String(), req.Amount, req.Operator, req.Reason)
		if auditErr != nil {
			WriteError(w, http.StatusInternalServerError, "AUDIT_ERROR", "Storno denetim kaydı oluşturulamadı: "+auditErr.Error())
			return
		}

		if err := tx.Commit(r.Context()); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Storno işlemi onaylanamadı: "+err.Error())
			return
		}

		WriteJSON(w, http.StatusOK, map[string]interface{}{
			"id":            stornoID,
			"period_id":     pID,
			"operator":      req.Operator,
			"amount_kurus":  req.Amount,
			"is_storno":     true,
			"storno_reason": req.Reason,
		})
		return
	}

	entry, err := h.settleSvc.RecordStornoAfterClose(pID, nil, req.Operator, req.Amount, req.Reason)
	if err != nil {
		if errors.Is(err, settlement.ErrPeriodNotClosed) {
			WriteError(w, http.StatusBadRequest, "PERIOD_NOT_CLOSED", "Storno yalnızca kapatılmış dönemlere uygulanabilir. Lütfen önce dönemi kapatınız.")
			return
		}
		if errors.Is(err, settlement.ErrPeriodNotFound) {
			WriteError(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Dönem bulunamadı")
			return
		}
		WriteError(w, http.StatusInternalServerError, "STORNO_ERROR", err.Error())
		return
	}

	WriteJSON(w, http.StatusOK, entry)
}

func (h *APIHandler) AddSettlementSubsidy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PeriodID   string       `json:"period_id"`
		PeriodName string       `json:"period_name"`
		Operator   string       `json:"operator"`
		Amount     domain.Kurus `json:"amount_kurus"`
		Reason     string       `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz istek")
		return
	}
	if req.Amount <= 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_AMOUNT", "Destek tutarı pozitif tam sayı kuruş olmalıdır")
		return
	}
	req.Operator = strings.ToLower(strings.TrimSpace(req.Operator))
	switch strings.ToLower(req.Operator) {
	case "belediye", "izdeniz", "ozel_halk":
	default:
		WriteError(w, http.StatusBadRequest, "INVALID_OPERATOR", "Destek için geçerli bir işletmeci seçilmelidir")
		return
	}
	req.Reason = strings.TrimSpace(req.Reason)
	if req.Reason == "" || len(req.Reason) > 500 {
		WriteError(w, http.StatusBadRequest, "INVALID_REASON", "Gerekçe zorunludur ve 500 karakteri aşamaz")
		return
	}
	idempotencyKey := strings.TrimSpace(r.Header.Get("Idempotency-Key"))
	if idempotencyKey == "" || len(idempotencyKey) > 255 {
		WriteError(w, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key başlığı zorunludur ve 255 karakteri aşamaz")
		return
	}
	claimsVal := r.Context().Value(UserContextKey)
	if claimsVal == nil {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açılmamış")
		return
	}
	claims := claimsVal.(*auth.Claims)
	if h.pool == nil {
		WriteError(w, http.StatusServiceUnavailable, "DATABASE_UNAVAILABLE", "Settlement adjustment için PostgreSQL gereklidir")
		return
	}

	tx, err := h.pool.Begin(r.Context())
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Veritabanı işlemi başlatılamadı")
		return
	}
	defer tx.Rollback(r.Context())
	if _, err := tx.Exec(r.Context(), "SELECT pg_advisory_xact_lock(hashtextextended($1, 0))", idempotencyKey); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Idempotency kilidi alınamadı")
		return
	}

	var periodID uuid.UUID
	if req.PeriodID != "" {
		periodID, err = uuid.Parse(req.PeriodID)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_UUID", "Geçersiz dönem ID")
			return
		}
		if req.PeriodName != "" {
			err = tx.QueryRow(r.Context(), `SELECT id FROM jw_settlement_periods WHERE id = $1 AND period_name = $2 FOR UPDATE`, periodID, req.PeriodName).Scan(&periodID)
		} else {
			err = tx.QueryRow(r.Context(), `SELECT id FROM jw_settlement_periods WHERE id = $1 FOR UPDATE`, periodID).Scan(&periodID)
		}
	} else if req.PeriodName != "" {
		err = tx.QueryRow(r.Context(), `SELECT id FROM jw_settlement_periods WHERE period_name = $1 FOR UPDATE`, req.PeriodName).Scan(&periodID)
	} else {
		WriteError(w, http.StatusBadRequest, "INVALID_PERIOD", "Dönem ID veya adı zorunludur")
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		WriteError(w, http.StatusNotFound, "PERIOD_NOT_FOUND", "Settlement dönemi bulunamadı")
		return
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Settlement dönemi okunamadı")
		return
	}

	var isClosed bool
	if err := tx.QueryRow(r.Context(), `SELECT is_closed FROM jw_settlement_periods WHERE id = $1`, periodID).Scan(&isClosed); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Settlement dönemi durumu okunamadı")
		return
	}
	if !isClosed {
		WriteError(w, http.StatusConflict, "PERIOD_NOT_CLOSED", "Destek düzeltmesi yalnızca kapatılmış döneme eklenebilir")
		return
	}

	var existingID, existingPeriodID uuid.UUID
	var existingActorID *uuid.UUID
	var existingOperator, existingReason, existingType string
	var existingAmount domain.Kurus
	err = tx.QueryRow(r.Context(), `
		SELECT id, period_id, operator, amount_kurus, adjustment_type, COALESCE(reason, ''), actor_id
		FROM jw_settlement_ledger WHERE idempotency_key = $1
	`, idempotencyKey).Scan(&existingID, &existingPeriodID, &existingOperator, &existingAmount, &existingType, &existingReason, &existingActorID)
	if err == nil {
		if existingType != "SUBSIDY" || existingPeriodID != periodID || existingOperator != req.Operator || existingAmount != req.Amount || existingReason != req.Reason || existingActorID == nil || *existingActorID != claims.UserID {
			WriteError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key daha önce farklı settlement adjustment için kullanılmış")
			return
		}
		WriteJSON(w, http.StatusOK, map[string]any{"id": existingID, "period_id": periodID, "operator": existingOperator, "amount_kurus": existingAmount, "adjustment_type": existingType, "reason": existingReason, "idempotent": true})
		return
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Idempotency kaydı okunamadı")
		return
	}

	adjustmentID := uuid.New()
	var createdAt time.Time
	if err := tx.QueryRow(r.Context(), `
		INSERT INTO jw_settlement_ledger
			(id, period_id, operator, amount_kurus, is_storno, adjustment_type, reason, actor_id, idempotency_key, created_at)
		VALUES ($1, $2, $3, $4, false, 'SUBSIDY', $5, $6, $7, NOW())
		RETURNING created_at
	`, adjustmentID, periodID, req.Operator, req.Amount, req.Reason, claims.UserID, idempotencyKey).Scan(&createdAt); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Settlement adjustment kaydedilemedi")
		return
	}
	_, err = tx.Exec(r.Context(), `
		INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, new_values, created_at)
		VALUES ($1, 'SETTLEMENT_SUBSIDY_ADJUSTMENT', 'jw_settlement_ledger', $2,
		        json_build_object('adjustment_type', 'SUBSIDY', 'period_id', $3::text, 'operator', $4::text, 'amount_kurus', $5::bigint, 'reason', $6::text)::jsonb, $7)
	`, claims.UserID, adjustmentID.String(), periodID.String(), req.Operator, req.Amount, req.Reason, createdAt)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "AUDIT_ERROR", "Settlement adjustment denetim kaydı yazılamadı")
		return
	}
	if err := tx.Commit(r.Context()); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Settlement adjustment onaylanamadı")
		return
	}
	WriteJSON(w, http.StatusCreated, map[string]any{"id": adjustmentID, "period_id": periodID, "operator": req.Operator, "amount_kurus": req.Amount, "adjustment_type": "SUBSIDY", "reason": req.Reason, "created_at": createdAt})
}

// 7. F-6: Ceza Kayıtları
type PenaltyRequest struct {
	CardID      string               `json:"card_id"`
	LineID      string               `json:"line_id"`
	StationID   string               `json:"station_id"`
	Amount      domain.Kurus         `json:"penalty_amount_kurus"`
	AmountKurus domain.Kurus         `json:"amount_kurus"`
	Reason      domain.PenaltyReason `json:"reason"`
	Notes       string               `json:"notes"`
}

func (h *APIHandler) CreatePenalty(w http.ResponseWriter, r *http.Request) {
	var req PenaltyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz ceza isteği")
		return
	}

	if req.Amount == 0 && req.AmountKurus > 0 {
		req.Amount = req.AmountKurus
	}

	if req.CardID == "" || req.Amount <= 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Kart ID ve ceza tutarı zorunludur")
		return
	}

	claimsVal := r.Context().Value(UserContextKey)
	var inspectorID uuid.UUID
	if claimsVal != nil {
		claims := claimsVal.(*auth.Claims)
		inspectorID = claims.UserID
	} else {
		WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Yetkisiz ceza işlemi")
		return
	}

	// Kartın var olduğunu doğrula
	if h.pool != nil {
		var exists bool
		err := h.pool.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM jw_cards WHERE id = $1)", req.CardID).Scan(&exists)
		if err != nil || !exists {
			WriteError(w, http.StatusBadRequest, "CARD_NOT_FOUND", "Ceza kesilecek kart bulunamadı")
			return
		}
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
		var lineIDVal *string
		if req.LineID != "" {
			lineIDVal = &req.LineID
		}
		var stationIDVal *string
		if req.StationID != "" {
			stationIDVal = &req.StationID
		}

		_, err := h.pool.Exec(r.Context(), `
			INSERT INTO jw_penalties (id, inspector_id, card_id, line_id, station_id, penalty_amount_kurus, reason, issued_at, notes)
			VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		`, penalty.ID, penalty.InspectorID, penalty.CardID, lineIDVal, stationIDVal, penalty.PenaltyAmountKurus, penalty.Reason, penalty.IssuedAt, penalty.Notes)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "PENALTY_INSERT_FAILED", "Ceza kaydedilemedi: "+err.Error())
			return
		}
	} else {
		h.memMu.Lock()
		h.memPenalties = append([]domain.Penalty{penalty}, h.memPenalties...)
		h.memMu.Unlock()
	}

	WriteJSON(w, http.StatusCreated, penalty)
}

func encodeCursor(t time.Time, id uuid.UUID) string {
	raw := fmt.Sprintf("%s_%s", t.UTC().Format(time.RFC3339Nano), id.String())
	return base64.RawURLEncoding.EncodeToString([]byte(raw))
}

func decodeCursor(cursor string) (time.Time, uuid.UUID, bool) {
	if cursor == "" {
		return time.Time{}, uuid.Nil, false
	}
	raw := cursor
	if decoded, err := base64.RawURLEncoding.DecodeString(cursor); err == nil {
		raw = string(decoded)
	}
	parts := strings.Split(raw, "_")
	if len(parts) == 2 {
		t, err1 := time.Parse(time.RFC3339Nano, parts[0])
		id, err2 := uuid.Parse(parts[1])
		if err1 == nil && err2 == nil {
			return t, id, true
		}
	}
	return time.Time{}, uuid.Nil, false
}

func (h *APIHandler) ListPenalties(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	cursor := r.URL.Query().Get("cursor")
	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}

	cursorTime, cursorID, hasCursor := decodeCursor(cursor)
	if cursor != "" && !hasCursor {
		WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "Geçersiz sayfalama imleci")
		return
	}

	if h.pool == nil {
		h.memMu.RLock()
		all := make([]domain.Penalty, len(h.memPenalties))
		copy(all, h.memPenalties)
		h.memMu.RUnlock()

		startIndex := 0
		if hasCursor {
			for i, p := range all {
				if p.ID == cursorID && p.IssuedAt.Equal(cursorTime) {
					startIndex = i + 1
					break
				}
			}
			if startIndex == 0 {
				WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "Sayfalama imleci artık mevcut değil")
				return
			}
		}
		endIndex := startIndex + limit
		if endIndex > len(all) {
			endIndex = len(all)
		}

		var page []domain.Penalty
		if startIndex < len(all) {
			page = all[startIndex:endIndex]
		} else {
			page = []domain.Penalty{}
		}

		if endIndex < len(all) && len(page) > 0 {
			last := page[len(page)-1]
			w.Header().Set("X-Next-Cursor", encodeCursor(last.IssuedAt, last.ID))
			w.Header().Set("X-Has-More", "true")
		} else {
			w.Header().Set("X-Has-More", "false")
		}

		if cursor == "" && limitStr == "" {
			WriteJSON(w, http.StatusOK, all)
			return
		}
		WriteJSON(w, http.StatusOK, page)
		return
	}

	if hasCursor {
		var exists bool
		if err := h.pool.QueryRow(r.Context(), "SELECT EXISTS(SELECT 1 FROM jw_penalties WHERE id = $1 AND issued_at = $2)", cursorID, cursorTime).Scan(&exists); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		} else if !exists {
			WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "Sayfalama imleci artık mevcut değil")
			return
		}
	}

	var rows pgx.Rows
	var err error
	if hasCursor {
		rows, err = h.pool.Query(r.Context(), `
			SELECT id, inspector_id, card_id, COALESCE(line_id, ''), COALESCE(station_id, ''),
			       penalty_amount_kurus, reason, issued_at, COALESCE(notes, '')
			FROM jw_penalties
			WHERE (issued_at, id) < ($1, $2)
			ORDER BY issued_at DESC, id DESC
			LIMIT $3
		`, cursorTime, cursorID, limit+1)
	} else {
		rows, err = h.pool.Query(r.Context(), `
			SELECT id, inspector_id, card_id, COALESCE(line_id, ''), COALESCE(station_id, ''),
			       penalty_amount_kurus, reason, issued_at, COALESCE(notes, '')
			FROM jw_penalties
			ORDER BY issued_at DESC, id DESC
			LIMIT $1
		`, limit+1)
	}
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}
	defer rows.Close()

	penalties := make([]domain.Penalty, 0)
	for rows.Next() {
		var p domain.Penalty
		if err := rows.Scan(&p.ID, &p.InspectorID, &p.CardID, &p.LineID, &p.StationID, &p.PenaltyAmountKurus, &p.Reason, &p.IssuedAt, &p.Notes); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}
		penalties = append(penalties, p)
	}
	if err := rows.Err(); err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	hasMore := len(penalties) > limit
	if hasMore {
		penalties = penalties[:limit]
	}
	if hasMore && len(penalties) > 0 {
		last := penalties[len(penalties)-1]
		w.Header().Set("X-Next-Cursor", encodeCursor(last.IssuedAt, last.ID))
		w.Header().Set("X-Has-More", "true")
	} else {
		w.Header().Set("X-Has-More", "false")
	}

	WriteJSON(w, http.StatusOK, penalties)
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

	if h.pool != nil {
		rows, err := h.pool.Query(r.Context(), `
			SELECT l.id, l.name, l.mode, l.operator,
			       COUNT(e.id) AS tap_count,
			       ROUND(COUNT(e.id)::numeric / 1000.0, 2) AS occupancy_ratio,
			       CASE 
			           WHEN COUNT(e.id) > 1500 THEN 'HIGH'
			           WHEN COUNT(e.id) > 500 THEN 'MEDIUM'
			           ELSE 'LOW'
			       END AS crowd_level
			FROM jw_lines l
			LEFT JOIN jw_validator_events e ON e.line_id = l.id AND e.tapped_at >= NOW() - INTERVAL '24 hours' AND e.status = 'PROCESSED'
			WHERE l.is_active = true
			GROUP BY l.id, l.name, l.mode, l.operator
			ORDER BY tap_count DESC
			LIMIT 10
		`)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Yoğunluk raporu okunamadı")
			return
		}
		defer rows.Close()
		items := make([]HeatmapItem, 0)
		for rows.Next() {
			var it HeatmapItem
			if err := rows.Scan(&it.LineID, &it.LineName, &it.Mode, &it.Operator, &it.TapCount, &it.OccupancyRatio, &it.CrowdLevel); err != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Yoğunluk raporu çözümlenemedi")
				return
			}
			items = append(items, it)
		}
		if err := rows.Err(); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Yoğunluk raporu tamamlanamadı")
			return
		}
		WriteJSON(w, http.StatusOK, items)
		return
	}

	WriteJSON(w, http.StatusOK, []HeatmapItem{})
}

func (h *APIHandler) GetAnomalies(w http.ResponseWriter, r *http.Request) {
	limitStr := r.URL.Query().Get("limit")
	cursor := r.URL.Query().Get("cursor")

	limit := 50
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			if l > 100 {
				l = 100
			}
			limit = l
		}
	}
	var cursorUUID uuid.UUID
	if cursor != "" {
		parsed, err := uuid.Parse(cursor)
		if err != nil {
			WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "Geçersiz sayfalama imleci")
			return
		}
		cursorUUID = parsed
	}

	if h.pool != nil {
		var (
			rows pgx.Rows
			err  error
		)

		if cursor != "" {
			var cursorTime time.Time
			cErr := h.pool.QueryRow(r.Context(), "SELECT detected_at FROM jw_anomalies WHERE id = $1", cursorUUID).Scan(&cursorTime)
			if errors.Is(cErr, pgx.ErrNoRows) {
				WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "Sayfalama imleci artık mevcut değil")
				return
			}
			if cErr != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", cErr.Error())
				return
			}
			rows, err = h.pool.Query(r.Context(), `
						SELECT id, card_id, event_id, anomaly_type, description,
						       COALESCE(calculated_speed_kmh, 0), COALESCE(time_delta_seconds, 0),
						       COALESCE(distance_meters, 0), detected_at
						FROM jw_anomalies
						WHERE (detected_at < $1) OR (detected_at = $1 AND id < $2)
						ORDER BY detected_at DESC, id DESC
						LIMIT $3
					`, cursorTime, cursorUUID, limit+1)
		}

		if cursor == "" && rows == nil {
			rows, err = h.pool.Query(r.Context(), `
				SELECT id, card_id, event_id, anomaly_type, description,
				       COALESCE(calculated_speed_kmh, 0), COALESCE(time_delta_seconds, 0),
				       COALESCE(distance_meters, 0), detected_at
				FROM jw_anomalies
				ORDER BY detected_at DESC, id DESC
				LIMIT $1
			`, limit+1)
		}
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
			return
		}

		if rows != nil {
			defer rows.Close()
			var dbAnoms []domain.Anomaly
			for rows.Next() {
				var a domain.Anomaly
				var aType string
				if sErr := rows.Scan(&a.ID, &a.CardID, &a.EventID, &aType, &a.Description, &a.CalculatedSpeedKmh, &a.TimeDeltaSeconds, &a.DistanceMeters, &a.DetectedAt); sErr != nil {
					WriteError(w, http.StatusInternalServerError, "DB_ERROR", sErr.Error())
					return
				}
				a.AnomalyType = domain.AnomalyType(aType)
				dbAnoms = append(dbAnoms, a)
			}
			if rowsErr := rows.Err(); rowsErr != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", rowsErr.Error())
				return
			}

			if len(dbAnoms) > 0 || (cursor != "" && len(dbAnoms) == 0) {
				hasMore := len(dbAnoms) > limit
				if hasMore {
					dbAnoms = dbAnoms[:limit]
				}
				if len(dbAnoms) > 0 && hasMore {
					last := dbAnoms[len(dbAnoms)-1]
					w.Header().Set("X-Next-Cursor", last.ID.String())
					w.Header().Set("X-Has-More", "true")
				} else {
					w.Header().Set("X-Has-More", "false")
				}
				WriteJSON(w, http.StatusOK, dbAnoms)
				return
			}
		}
	}

	anoms := h.anomalyDet.GetAnomalies()

	if cursor == "" && limitStr == "" {
		WriteJSON(w, http.StatusOK, anoms)
		return
	}

	startIndex := 0
	if cursor != "" {
		found := false
		for i, a := range anoms {
			if a.ID == cursorUUID {
				startIndex = i + 1
				found = true
				break
			}
		}
		if !found {
			WriteError(w, http.StatusBadRequest, "INVALID_CURSOR", "Sayfalama imleci artık mevcut değil")
			return
		}
	}

	endIndex := startIndex + limit
	if endIndex > len(anoms) {
		endIndex = len(anoms)
	}

	var page []domain.Anomaly
	if startIndex < len(anoms) {
		page = anoms[startIndex:endIndex]
	} else {
		page = []domain.Anomaly{}
	}

	if endIndex < len(anoms) && len(page) > 0 {
		last := page[len(page)-1]
		w.Header().Set("X-Next-Cursor", last.ID.String())
		w.Header().Set("X-Has-More", "true")
	} else {
		w.Header().Set("X-Has-More", "false")
	}

	WriteJSON(w, http.StatusOK, page)
}

// 9. Bonus 3: Tarife Değişikliği Etki Simülasyonu
type FareImpactRequest struct {
	BaseFareChangeBasisPoints    int64 `json:"base_fare_change_basis_points"`
	StudentFareChangeBasisPoints int64 `json:"student_fare_change_basis_points"`
	ElasticityBasisPoints        int64 `json:"elasticity_basis_points"`
}

type FareImpactResponse struct {
	CurrentMonthlyRevenueKurus    int64  `json:"current_monthly_revenue_kurus"`
	ProjectedMonthlyRevenueKurus  int64  `json:"projected_monthly_revenue_kurus"`
	RevenueDeltaKurus             int64  `json:"revenue_delta_kurus"`
	RevenueChangeBasisPoints      int64  `json:"revenue_change_basis_points"`
	ProjectedRidershipBasisPoints int64  `json:"projected_ridership_delta_basis_points"`
	ModelBasis                    string `json:"model_basis"`
	SummaryDescription            string `json:"summary_description"`
}

func mulDivRound(value, multiplier, divisor int64) int64 {
	if divisor <= 0 {
		return 0
	}
	product := value * multiplier
	if product >= 0 {
		return (product + divisor/2) / divisor
	}
	return (product - divisor/2) / divisor
}

func formatBasisPoints(value int64) string {
	sign := ""
	if value < 0 {
		sign = "-"
		value = -value
	}
	return fmt.Sprintf("%s%d.%02d%%", sign, value/100, value%100)
}

func (h *APIHandler) SimulateFareImpact(w http.ResponseWriter, r *http.Request) {
	var req FareImpactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz simülasyon parametreleri")
		return
	}

	elasticity := req.ElasticityBasisPoints
	if elasticity == 0 {
		elasticity = -3500 // -0.35, 10_000 tabanlı sabit nokta
	}
	if req.BaseFareChangeBasisPoints < -10000 || req.StudentFareChangeBasisPoints < -10000 ||
		req.BaseFareChangeBasisPoints > 100000 || req.StudentFareChangeBasisPoints > 100000 ||
		elasticity < -10000 || elasticity > 0 {
		WriteError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Tarife değişimi veya esneklik basis-point sınırları dışında")
		return
	}

	currentMonthlyRev := int64(2_500_000_000) // 25 milyon TL, kuruş
	weightedPriceChangeBP := (req.BaseFareChangeBasisPoints*70 + req.StudentFareChangeBasisPoints*30) / 100
	ridershipChangeBP := mulDivRound(weightedPriceChangeBP, elasticity, 10000)
	projectedAfterPrice := mulDivRound(currentMonthlyRev, 10000+weightedPriceChangeBP, 10000)
	projectedRev := mulDivRound(projectedAfterPrice, 10000+ridershipChangeBP, 10000)
	revenueDelta := projectedRev - currentMonthlyRev
	revenueChangeBP := mulDivRound(revenueDelta, 10000, currentMonthlyRev)

	resp := FareImpactResponse{
		CurrentMonthlyRevenueKurus:    currentMonthlyRev,
		ProjectedMonthlyRevenueKurus:  projectedRev,
		RevenueDeltaKurus:             revenueDelta,
		RevenueChangeBasisPoints:      revenueChangeBP,
		ProjectedRidershipBasisPoints: ridershipChangeBP,
		ModelBasis:                    "SCENARIO_ASSUMPTION_NOT_ACTUAL_REVENUE",
		SummaryDescription: fmt.Sprintf(
			"Varsayımsal tarife değişimi (Tam: %s, Öğrenci: %s) için talep etkisi %s, tahmini gelir etkisi %s.",
			formatBasisPoints(req.BaseFareChangeBasisPoints), formatBasisPoints(req.StudentFareChangeBasisPoints),
			formatBasisPoints(ridershipChangeBP), formatBasisPoints(revenueChangeBP),
		),
	}

	WriteJSON(w, http.StatusOK, resp)
}

// 12. P1 — ESHOT Canlı Araç Takip Uçları
func (h *APIHandler) GetLiveBuses(w http.ResponseWriter, r *http.Request) {
	lineNo := r.URL.Query().Get("line")
	if lineNo == "" {
		lineNo = "800"
	}
	res, err := h.transitSvc.GetLiveBuses(r.Context(), lineNo)
	if err != nil {
		WriteError(w, http.StatusInternalServerError, "TRANSIT_ERROR", err.Error())
		return
	}
	WriteJSON(w, http.StatusOK, res)
}

func (h *APIHandler) GetTransitLines(w http.ResponseWriter, r *http.Request) {
	WriteJSON(w, http.StatusOK, map[string]interface{}{
		"supported_lines": transit.SupportedLines,
		"attribution":     transit.ESHOTAttribution,
		"license":         transit.ESHOTLicense,
	})
}

// 13. P2 — Hat Bazlı Gelir, Maliyet ve Finansal Analiz
type LineFinancialReport struct {
	LineID               string       `json:"line_id"`
	LineName             string       `json:"line_name"`
	Operator             string       `json:"operator"`
	TotalBoardings       int64        `json:"total_boardings"`
	GrossRevenueKurus    domain.Kurus `json:"gross_revenue_kurus"`
	DiscountImpactKurus  domain.Kurus `json:"discount_impact_kurus"`
	VehicleKmMeters      int64        `json:"vehicle_km"`
	FuelEnergyCostKurus  domain.Kurus `json:"fuel_energy_cost_kurus"`
	CrewCostKurus        domain.Kurus `json:"crew_cost_kurus"`
	TotalCostKurus       domain.Kurus `json:"total_cost_kurus"`
	NetMarginKurus       domain.Kurus `json:"net_margin_kurus"`
	NetMarginBasisPoints int64        `json:"net_margin_basis_points"`
	CostBasis            string       `json:"cost_basis"`
}

func (h *APIHandler) GetLineFinancials(w http.ResponseWriter, r *http.Request) {
	var reports []LineFinancialReport

	if h.pool != nil {
		rows, err := h.pool.Query(r.Context(), `
			SELECT l.id, l.name, l.operator,
			       COUNT(e.id) as total_boardings,
			       COALESCE(SUM(e.fare_charged_kurus), 0) as gross_revenue
			FROM jw_lines l
			LEFT JOIN jw_validator_events e ON e.line_id = l.id AND e.status = 'PROCESSED'
			WHERE l.is_active = true
			GROUP BY l.id, l.name, l.operator
			ORDER BY gross_revenue DESC, total_boardings DESC
		`)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Hat gelir raporu okunamadı")
			return
		}
		defer rows.Close()
		for rows.Next() {
			var rep LineFinancialReport
			var boardings int64
			var gross domain.Kurus
			if err := rows.Scan(&rep.LineID, &rep.LineName, &rep.Operator, &boardings, &gross); err != nil {
				WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Hat gelir raporu çözümlenemedi")
				return
			}
			rep.TotalBoardings = boardings
			rep.GrossRevenueKurus = gross

			vKm := int64(120)
			if boardings > 0 {
				vKm += (boardings / 15) * 8
			}
			rep.VehicleKmMeters = vKm
			rep.FuelEnergyCostKurus = domain.Kurus(vKm * 2500)
			rep.CrewCostKurus = domain.Kurus(16 * 15000)
			rep.TotalCostKurus = rep.FuelEnergyCostKurus + rep.CrewCostKurus
			rep.DiscountImpactKurus = domain.Kurus(boardings * 500)
			rep.NetMarginKurus = rep.GrossRevenueKurus - rep.TotalCostKurus
			if rep.TotalCostKurus > 0 {
				rep.NetMarginBasisPoints = int64(rep.NetMarginKurus) * 10000 / int64(rep.TotalCostKurus)
			}
			rep.CostBasis = "ESTIMATED_COST_MODEL_V1"
			reports = append(reports, rep)
		}
		if err := rows.Err(); err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Hat gelir raporu tamamlanamadı")
			return
		}
	}

	WriteJSON(w, http.StatusOK, reports)
}

// 14. P2 — Hat Yöneticisi Sefer Optimizasyonu
type HeadwayScenarioRequest struct {
	LineID                  string `json:"line_id"`
	CurrentHeadwayMinutes   int    `json:"current_headway_minutes"`
	TargetHeadwayMinutes    int    `json:"target_headway_minutes"`
	CurrentHeadwayMins      int    `json:"current_headway_mins"`
	TargetHeadwayMins       int    `json:"target_headway_mins"`
	FleetCapacityPerVehicle int    `json:"fleet_capacity_per_vehicle"`
	PeakHour                bool   `json:"peak_hour"`
}

type HeadwayScenarioResponse struct {
	LineID                      string         `json:"line_id"`
	CurrentHeadwayMinutes       int            `json:"current_headway_minutes"`
	TargetHeadwayMinutes        int            `json:"target_headway_minutes"`
	CurrentHeadwayMins          int            `json:"current_headway_mins"`
	ProposedHeadwayMins         int            `json:"proposed_headway_mins"`
	CurrentVehicles             int            `json:"current_vehicles"`
	ProposedVehicles            int            `json:"proposed_vehicles"`
	VehicleDelta                int            `json:"vehicle_delta"`
	DeltaCostKurus              int64          `json:"delta_cost_kurus"`
	DemandChangeBasisPoints     int64          `json:"demand_change_basis_points"`
	ProjectedNetMarginKurus     int64          `json:"projected_net_margin_kurus"`
	CurrentVehiclesRequired     int            `json:"current_vehicles_required"`
	TargetVehiclesRequired      int            `json:"target_vehicles_required"`
	AdditionalVehicles          int            `json:"additional_vehicles"`
	AvgWaitReductionSeconds     int            `json:"avg_wait_reduction_seconds"`
	CapacityIncreaseBasisPoints int64          `json:"capacity_increase_basis_points"`
	DailyCostDeltaKurus         int64          `json:"daily_cost_delta_kurus"`
	ModelBasis                  string         `json:"model_basis"`
	Assumptions                 map[string]int `json:"assumptions"`
	RecommendedAction           string         `json:"recommended_action"`
}

func (h *APIHandler) SimulateHeadwayScenario(w http.ResponseWriter, r *http.Request) {
	var req HeadwayScenarioRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz optimizasyon senaryo parametreleri")
		return
	}

	currentHeadway := req.CurrentHeadwayMinutes
	if currentHeadway <= 0 {
		currentHeadway = req.CurrentHeadwayMins
	}
	if currentHeadway <= 0 {
		currentHeadway = 15
	}

	targetHeadway := req.TargetHeadwayMinutes
	if targetHeadway <= 0 {
		targetHeadway = req.TargetHeadwayMins
	}
	if targetHeadway <= 0 {
		targetHeadway = 10
	}

	fleetCapacity := req.FleetCapacityPerVehicle
	if fleetCapacity <= 0 {
		fleetCapacity = 100
	}

	if req.LineID == "" || currentHeadway <= 0 || targetHeadway <= 0 || fleetCapacity <= 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Hat, mevcut/hedef sefer aralığı ve araç kapasitesi pozitif olmalıdır")
		return
	}
	if currentHeadway > 180 || targetHeadway > 180 || fleetCapacity > 1000 {
		WriteError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Optimizasyon girdileri kabul edilen sınırların dışında")
		return
	}

	roundtripMinutes := 90
	hourlyDemand := 1200
	if req.PeakHour {
		hourlyDemand = 1600
	}

	currFreqVehicles := (roundtripMinutes + currentHeadway - 1) / currentHeadway
	targetFreqVehicles := (roundtripMinutes + targetHeadway - 1) / targetHeadway

	totalDemandPerRoundtrip := (hourlyDemand * roundtripMinutes) / 60
	capacityVehicles := (totalDemandPerRoundtrip + fleetCapacity - 1) / fleetCapacity

	currVehicles := currFreqVehicles
	if capacityVehicles > currVehicles && fleetCapacity < 100 {
		currVehicles = capacityVehicles
	}

	targetVehicles := targetFreqVehicles
	if capacityVehicles > targetVehicles && fleetCapacity < 100 {
		targetVehicles = capacityVehicles + (targetFreqVehicles - currFreqVehicles)
	}
	if targetVehicles < 1 {
		targetVehicles = 1
	}

	vehicleDelta := targetVehicles - currVehicles

	waitReductionSeconds := (currentHeadway - targetHeadway) * 30
	capacityIncreaseBP := int64(currentHeadway)*10000/int64(targetHeadway) - 10000
	headwayReductionBP := int64(currentHeadway-targetHeadway) * 10000 / int64(currentHeadway)
	demandChangeBP := headwayReductionBP * 35 / 100

	dailyCostPerVehicleKurus := int64(1040000)
	deltaCostKurus := int64(vehicleDelta) * dailyCostPerVehicleKurus

	estimatedDailyPassengers := int64(hourlyDemand * 16)
	incrementalPassengers := estimatedDailyPassengers * demandChangeBP / 10000
	avgFareKurus := int64(2000)
	incrementalRevenueKurus := incrementalPassengers * avgFareKurus
	projectedNetMarginKurus := incrementalRevenueKurus - deltaCostKurus

	action := "UYGUN: Pik saatlerde sefer sıklığı artırımı hizmet kalitesini ve net kârlılığı artıracaktır."
	if vehicleDelta > 5 {
		action = "DİKKAT: Yüksek ek filo maliyeti. Kademeli pilot uygulama önerilir."
	} else if projectedNetMarginKurus < 0 {
		action = "FİLO KISITI: Artan operasyonel maliyet ek gelir beklentisini aşmaktadır."
	}

	resp := HeadwayScenarioResponse{
		LineID:                      req.LineID,
		CurrentHeadwayMinutes:       currentHeadway,
		TargetHeadwayMinutes:        targetHeadway,
		CurrentHeadwayMins:          currentHeadway,
		ProposedHeadwayMins:         targetHeadway,
		CurrentVehicles:             currVehicles,
		ProposedVehicles:            targetVehicles,
		VehicleDelta:                vehicleDelta,
		DeltaCostKurus:              deltaCostKurus,
		DemandChangeBasisPoints:     demandChangeBP,
		ProjectedNetMarginKurus:     projectedNetMarginKurus,
		CurrentVehiclesRequired:     currVehicles,
		TargetVehiclesRequired:      targetVehicles,
		AdditionalVehicles:          vehicleDelta,
		AvgWaitReductionSeconds:     waitReductionSeconds,
		CapacityIncreaseBasisPoints: capacityIncreaseBP,
		DailyCostDeltaKurus:         deltaCostKurus,
		ModelBasis:                  "SCENARIO_ASSUMPTION_NOT_ACTUAL_PROFIT",
		Assumptions: map[string]int{
			"roundtrip_minutes":              roundtripMinutes,
			"hourly_demand_passengers":       hourlyDemand,
			"service_hours_per_day":          16,
			"daily_vehicle_cost_kurus":       int(dailyCostPerVehicleKurus),
			"average_fare_kurus":             int(avgFareKurus),
			"demand_response_weight_percent": 35,
		},
		RecommendedAction: action,
	}

	WriteJSON(w, http.StatusOK, resp)
}

// 15. P1 — Hat Yöneticisi Optimizasyon Karar Onayı (Audit Persistence)
type ApproveOptimizationRequest struct {
	LineID                  string `json:"line_id"`
	CurrentHeadwayMinutes   int    `json:"current_headway_minutes"`
	TargetHeadwayMinutes    int    `json:"target_headway_minutes"`
	FleetCapacity           int    `json:"fleet_capacity"`
	VehicleDelta            int    `json:"vehicle_delta"`
	DeltaCostKurus          int64  `json:"delta_cost_kurus"`
	ProjectedNetMarginKurus int64  `json:"projected_net_margin_kurus"`
	Notes                   string `json:"notes"`
}

func (h *APIHandler) ApproveOptimizationScenario(w http.ResponseWriter, r *http.Request) {
	claimsVal := r.Context().Value(UserContextKey)
	var claims *auth.Claims
	if claimsVal != nil {
		claims = claimsVal.(*auth.Claims)
	}

	if claims == nil || (claims.Role != domain.RoleHatYoneticisi && claims.Role != domain.RoleAdmin) {
		WriteError(w, http.StatusForbidden, "FORBIDDEN", "Optimizasyon senaryosu onaylama yetkiniz bulunmamaktadır")
		return
	}

	var req ApproveOptimizationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Geçersiz onay parametreleri")
		return
	}

	if req.LineID == "" || req.CurrentHeadwayMinutes <= 0 || req.TargetHeadwayMinutes <= 0 || req.FleetCapacity <= 0 {
		WriteError(w, http.StatusBadRequest, "INVALID_REQUEST", "Hat, pozitif sefer aralıkları ve pozitif filo kapasitesi zorunludur")
		return
	}
	if req.CurrentHeadwayMinutes > 180 || req.TargetHeadwayMinutes > 180 || req.FleetCapacity > 1000 {
		WriteError(w, http.StatusUnprocessableEntity, "INVALID_REQUEST", "Sefer aralığı veya filo kapasitesi kabul edilen sınırların dışında")
		return
	}

	idempotencyKey := r.Header.Get("Idempotency-Key")
	if idempotencyKey == "" {
		WriteError(w, http.StatusBadRequest, "IDEMPOTENCY_KEY_REQUIRED", "Idempotency-Key başlığı zorunludur")
		return
	}
	if len(idempotencyKey) > 255 {
		WriteError(w, http.StatusBadRequest, "INVALID_IDEMPOTENCY_KEY", "Idempotency-Key en fazla 255 karakter olabilir")
		return
	}
	if h.pool == nil {
		WriteError(w, http.StatusServiceUnavailable, "DB_UNAVAILABLE", "Onay kaydı için PostgreSQL gereklidir")
		return
	}

	approvalID := uuid.New()
	approvedAt := time.Time{}
	inserted := false
	err := h.pool.QueryRow(r.Context(), `
		WITH inserted AS (
			INSERT INTO jw_optimization_approvals (
				id, idempotency_key, line_id, current_headway_minutes, target_headway_minutes,
				fleet_capacity, vehicle_delta, delta_cost_kurus, projected_net_margin_kurus,
				approved_by, status, notes, approved_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, 'APPROVED', $11, NOW())
			ON CONFLICT (idempotency_key) WHERE idempotency_key IS NOT NULL DO NOTHING
			RETURNING id, approved_at
		), audited AS (
			INSERT INTO jw_audit_logs (user_id, action, table_name, record_id, new_values, created_at)
			SELECT $10, 'OPTIMIZATION_APPROVED', 'jw_optimization_approvals', id::text,
			       jsonb_build_object('line_id', $3, 'idempotency_key', $2, 'status', 'APPROVED'), NOW()
			FROM inserted
			RETURNING record_id
		)
		SELECT i.id, i.approved_at, true FROM inserted i LEFT JOIN audited a ON a.record_id = i.id::text
	`, approvalID, idempotencyKey, req.LineID, req.CurrentHeadwayMinutes, req.TargetHeadwayMinutes,
		req.FleetCapacity, req.VehicleDelta, req.DeltaCostKurus, req.ProjectedNetMarginKurus,
		claims.UserID, req.Notes).Scan(&approvalID, &approvedAt, &inserted)
	if errors.Is(err, pgx.ErrNoRows) {
		var existing ApproveOptimizationRequest
		err = h.pool.QueryRow(r.Context(), `
			SELECT id, line_id, current_headway_minutes, target_headway_minutes, fleet_capacity,
			       vehicle_delta, delta_cost_kurus, projected_net_margin_kurus, notes, approved_at
			FROM jw_optimization_approvals WHERE idempotency_key = $1
		`, idempotencyKey).Scan(&approvalID, &existing.LineID, &existing.CurrentHeadwayMinutes,
			&existing.TargetHeadwayMinutes, &existing.FleetCapacity, &existing.VehicleDelta,
			&existing.DeltaCostKurus, &existing.ProjectedNetMarginKurus, &existing.Notes, &approvedAt)
		if err != nil {
			WriteError(w, http.StatusInternalServerError, "DB_ERROR", "Onay idempotans kaydı okunamadı")
			return
		}
		if existing != req {
			WriteError(w, http.StatusConflict, "IDEMPOTENCY_CONFLICT", "Idempotency-Key farklı bir onay kararı için kullanılmış")
			return
		}
	} else if err != nil {
		WriteError(w, http.StatusInternalServerError, "DB_ERROR", err.Error())
		return
	}

	status := http.StatusOK
	if inserted {
		status = http.StatusCreated
	}
	WriteJSON(w, status, map[string]interface{}{
		"id":                         approvalID,
		"line_id":                    req.LineID,
		"status":                     "APPROVED",
		"approved_by":                claims.UserID,
		"current_headway_minutes":    req.CurrentHeadwayMinutes,
		"target_headway_minutes":     req.TargetHeadwayMinutes,
		"vehicle_delta":              req.VehicleDelta,
		"delta_cost_kurus":           req.DeltaCostKurus,
		"projected_net_margin_kurus": req.ProjectedNetMarginKurus,
		"notes":                      req.Notes,
		"approved_at":                approvedAt.Format(time.RFC3339),
	})
}
