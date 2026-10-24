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

	// 3. CORS Yapılandırması (React Vite Frontend erişimi için)
	r.Use(cors.Handler(cors.Options{
		AllowedOrigins:   []string{"*"},
		AllowedMethods:   []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders:   []string{"Accept", "Authorization", "Content-Type", "X-CSRF-Token", "Idempotency-Key", "X-DDO-Stamp"},
		ExposedHeaders:   []string{"Link", "X-DDO-Stamp", "Idempotency-Key"},
		AllowCredentials: true,
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
		r.Get("/auth/me", h.GetMe)

		// Ulaşım Bilgileri (Herkese açık / Yolcu / Operasyon)
		r.Get("/lines", h.ListLines)
		r.Get("/stations", h.ListStations)
		r.Get("/tariffs", h.ListTariffs)
		r.Get("/reports/heatmap", h.GetHeatmap)

		// Kart Yönetimi (Yolcu & Gişe)
		r.Route("/cards", func(r chi.Router) {
			r.Get("/{id}", h.GetCard)
			r.Get("/{id}/ledger", h.GetCardLedger)
			r.Post("/{id}/recharge", h.RechargeCard)
		})

		// Validatör Olayları (Turnike & Araçlar)
		r.Route("/validators", func(r chi.Router) {
			r.Post("/tap", h.ProcessTap)
			r.Post("/batch-sync", h.BatchSyncValidators)
		})

		// Saha Kontrol & Cezalar (Kontrolör & Admin)
		r.Route("/penalties", func(r chi.Router) {
			r.With(RequireRole(domain.RoleKontrolor, domain.RoleOperasyon, domain.RoleAdmin)).Post("/", h.CreatePenalty)
			r.Get("/", h.ListPenalties)
		})

		// Gelir Paylaşımı ve Muhasebe (Muhasebe & Admin)
		r.Route("/settlements", func(r chi.Router) {
			r.With(RequireRole(domain.RoleMuhasebe, domain.RoleAdmin)).Post("/close", h.CloseSettlementPeriod)
			r.With(RequireRole(domain.RoleMuhasebe, domain.RoleAdmin)).Post("/storno", h.StornoSettlement)
		})

		// Operasyon & Güvenlik İzleme (Operasyon & Admin)
		r.Route("/anomalies", func(r chi.Router) {
			r.With(RequireRole(domain.RoleOperasyon, domain.RoleAdmin)).Get("/", h.GetAnomalies)
		})

		// Bonus 3: Tarife Değişikliği Etki Simülasyonu
		r.Post("/simulation/fare-impact", h.SimulateFareImpact)
	})

	return r
}
