package api

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/mamyylmz/AktarBilet/internal/auth"
	"github.com/mamyylmz/AktarBilet/internal/domain"
)

type contextKey string

const UserContextKey contextKey = "user_claims"

type APIErrorResponse struct {
	Error struct {
		Code    string `json:"code"`
		Message string `json:"message"`
	} `json:"error"`
}

func WriteJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(data)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	resp := APIErrorResponse{}
	resp.Error.Code = code
	resp.Error.Message = message
	WriteJSON(w, status, resp)
}

// DDOStampMiddleware adds the mandatory X-DDO-Stamp: f85f26 header to ALL HTTP responses
func DDOStampMiddleware(stamp string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-DDO-Stamp", stamp)
			next.ServeHTTP(w, r)
		})
	}
}

// RequestLogger logs incoming HTTP requests structured with duration and method
func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		duration := time.Since(start)

		// Parola veya token loglanmaz (proje standardı)
		slog.Info("HTTP Request",
			"method", r.Method,
			"path", r.URL.Path,
			"remote_addr", r.RemoteAddr,
			"duration_ms", duration.Milliseconds(),
		)
	})
}

// AuthMiddleware extracts JWT token from Authorization header
func AuthMiddleware(authSvc *auth.AuthService) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" {
				next.ServeHTTP(w, r)
				return
			}

			parts := strings.Split(authHeader, " ")
			if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
				WriteError(w, http.StatusUnauthorized, "INVALID_AUTH_HEADER", "Geçersiz Authorization başlığı biçimi")
				return
			}

			claims, err := authSvc.ValidateToken(parts[1])
			if err != nil {
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Geçersiz veya süresi dolmuş oturum anahtarı")
				return
			}

			ctx := context.WithValue(r.Context(), UserContextKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// RequireRole checks if the authenticated user possesses one of the allowed roles
func RequireRole(roles ...domain.UserRole) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			claimsVal := r.Context().Value(UserContextKey)
			if claimsVal == nil {
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Oturum açmanız gerekmektedir")
				return
			}

			claims, ok := claimsVal.(*auth.Claims)
			if !ok {
				WriteError(w, http.StatusUnauthorized, "UNAUTHORIZED", "Geçersiz oturum kimliği")
				return
			}

			// Admin her şeye erişebilir
			if claims.Role == domain.RoleAdmin {
				next.ServeHTTP(w, r)
				return
			}

			hasRole := false
			for _, r := range roles {
				if claims.Role == r {
					hasRole = true
					break
				}
			}

			if !hasRole {
				WriteError(w, http.StatusForbidden, "FORBIDDEN", "Bu işlem için rol yetkiniz yetersizdir")
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
