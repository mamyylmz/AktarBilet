package api

import (
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/go-chi/cors"
	"github.com/mamyylmz/AktarBilet/internal/domain"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func SetupRouter(h *APIHandler, ddoStamp string) *chi.Mux {
	r := chi.NewRouter()

	// 1. Temel Altyapı Middleware'leri
	r.Use(middleware.Recoverer)
	r.Use(RequestLogger)

	// 2. Zorunlu Sistem Damgası (System Canary 3): Her yanıtta X-DDO-Stamp: f85f26
	r.Use(DDOStampMiddleware(ddoStamp))

	// 3. CORS Yapılandırması (Wildcard credentials açık bırakılmaz, güvenli kökenler tanımlanır)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:13000", "http://127.0.0.1:13000"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key", "X-DDO-Stamp"},
		ExposedHeaders:   []string{"Link", "X-DDO-Stamp", "Idempotency-Key"},
		AllowCredentials: false,
		MaxAge:           300,
	}))

	// 4. Kimlik Doğrulama Middleware'i
	r.Use(AuthMiddleware(h.authSvc))

	// 5. Herkese Açık / Sağlık Kontrolleri
	r.Get("/health", h.HealthCheck)
	// Zorunlu Sistem Sağlık Ucu (System Canary 4)
	r.Get("/_hz/f85f", h.DDOMandatoryHealthCheck)
	// Prometheus Metrik Ucu (Bölüm 9 gereksinimi)
	r.Handle("/metrics", promhttp.Handler())

	// 6. API v1 Yönlendirmeleri
	r.Route("/api/v1", func(r chi.Router) {
		// Kimlik
		r.Post("/auth/login", h.Login)
		r.With(RequireAuth()).Get("/auth/me", h.GetMe)
		r.With(RequireAuth()).Post("/auth/switch-role", h.SwitchRole)

		// Ulaşım Bilgileri (Herkese açık katalog ve hat yöneticisi yönetimi)
		r.Get("/lines", h.ListLines)
		r.With(RequireRole(h.pool, domain.RoleHatYoneticisi, domain.RoleAdmin)).Post("/lines", h.CreateLine)
		r.Get("/stations", h.ListStations)
		r.Get("/tariffs", h.ListTariffs)
		r.Get("/reports/heatmap", h.GetHeatmap)

		// Canlı Ulaşım & ESHOT Entegrasyonu (P1)
		r.Get("/transit/live-buses", h.GetLiveBuses)
		r.Get("/transit/lines", h.GetTransitLines)

		// Kart Yönetimi (Kimlik doğrulamalı & IDOR korumalı)
		r.Route("/cards", func(r chi.Router) {
			r.Get("/{id}", h.GetCard)
			r.Get("/{id}/ledger", h.GetCardLedger)
			r.Post("/{id}/recharge", h.RechargeCard)
		})

		// Validatör Olayları (Turnike & Cihaz Doğrulaması)
		r.Route("/validators", func(r chi.Router) {
			r.Post("/tap", h.ProcessTap)
			r.Post("/batch-sync", h.BatchSyncValidators)
		})

		// Saha Kontrol & Cezalar (Yalnız Kontrolör & Admin; Operasyon ve diğerlerine kapalı)
		r.Route("/penalties", func(r chi.Router) {
			r.With(RequireRole(h.pool, domain.RoleKontrolor, domain.RoleAdmin)).Post("/", h.CreatePenalty)
			r.With(RequireRole(h.pool, domain.RoleKontrolor, domain.RoleAdmin)).Get("/", h.ListPenalties)
		})

		// Gelir Paylaşımı ve Muhasebe (Muhasebe & Admin)
		r.Route("/settlements", func(r chi.Router) {
			r.With(RequireRole(h.pool, domain.RoleMuhasebe, domain.RoleAdmin)).Get("/", h.ListSettlements)
			r.With(RequireRole(h.pool, domain.RoleMuhasebe, domain.RoleAdmin)).Post("/close", h.CloseSettlementPeriod)
			r.With(RequireRole(h.pool, domain.RoleMuhasebe, domain.RoleAdmin)).Post("/reopen", h.ReopenSettlementPeriod)
			r.With(RequireRole(h.pool, domain.RoleMuhasebe, domain.RoleAdmin)).Post("/storno", h.StornoSettlement)
		})

		// Operasyon & Güvenlik İzleme (Operasyon & Admin)
		r.Route("/anomalies", func(r chi.Router) {
			r.With(RequireRole(h.pool, domain.RoleOperasyon, domain.RoleAdmin)).Get("/", h.GetAnomalies)
		})

		// Raporlama & Hat Karlılığı (Muhasebe, Hat Yöneticisi, Operasyon, Admin)
		r.With(RequireRole(h.pool, domain.RoleMuhasebe, domain.RoleHatYoneticisi, domain.RoleOperasyon, domain.RoleAdmin)).Get("/reports/line-financials", h.GetLineFinancials)

		// Hat Sıklık & Maliyet Optimizasyonu (Hat Yöneticisi, Admin)
		r.With(RequireRole(h.pool, domain.RoleHatYoneticisi, domain.RoleAdmin)).Post("/optimization/headway-scenario", h.SimulateHeadwayScenario)
		r.With(RequireRole(h.pool, domain.RoleHatYoneticisi, domain.RoleAdmin)).Post("/optimization/approve", h.ApproveOptimizationScenario)

		// Tarife Değişikliği Etki Simülasyonu (Yalnız Hat Yöneticisi & Admin)
		r.With(RequireRole(h.pool, domain.RoleHatYoneticisi, domain.RoleAdmin)).Post("/simulation/fare-impact", h.SimulateFareImpact)
	})

	return r
}
