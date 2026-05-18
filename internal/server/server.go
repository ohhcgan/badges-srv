package server

import (
	"fmt"
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"go.uber.org/zap"

	"github-badges-backend/internal/auth"
	"github-badges-backend/internal/config"
	"github-badges-backend/internal/middlewares"
	controllers "github-badges-backend/internal/server/controllers"
)

/**
 * TODO:
 * Move from chi to gin.
 */
func New(
	cfg *config.Config,
	authHandler *auth.AuthHandler,
	cont *controllers.Controllers,
	logger *zap.Logger,
) *http.Server {
	routerMux := chi.NewRouter()

	routerMux.Use(middleware.RealIP)
	routerMux.Use(RequestLogger(logger))
	routerMux.Use(middleware.Recoverer)
	routerMux.Use(middlewares.CorsMiddleware([]string{cfg.FrontendURL}))

	routerMux.Get("/health", cont.HealthCheck)

	routerMux.Get("/auth/login", authHandler.HandleLogin)
	routerMux.Get("/auth/logout", authHandler.HandleLogout)
	routerMux.Get("/auth/callback", authHandler.HandleCallback)

	routerMux.Group(func(r chi.Router) {
		r.Use(authHandler.RequireAuth)

		r.Get("/api/me", cont.GetMe)
		r.Get("/api/stats", cont.GetStats)
		r.Get("/api/stats/history", cont.GetStatsHistory)
	})

	routerMux.Group(func(r chi.Router) {
		r.Use(auth.RequireAdmin(cfg.AdminKey))

		/** TODO:
		 * Make it a cron job or a cron job triggers this api
		 */
		r.Post("/api/admin/trigger-monthly", cont.TriggerMonthly)
	})

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", cfg.Port),
		Handler: routerMux,
		/**
		 * TODO: move ReadTimeout, WriteTimeout and IdleTimeout to cfg.
		 */
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}
}

func RequestLogger(logger *zap.Logger) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {

		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			ww := middleware.NewWrapResponseWriter(w, r.ProtoMajor)
			start := time.Now()
			next.ServeHTTP(ww, r)

			logger.Info("http request",
				zap.String("method", r.Method),
				zap.String("path", r.URL.Path),
				zap.Int("status", ww.Status()),
				zap.Duration("duration", time.Since(start)),
				zap.String("remote", r.RemoteAddr),
			)
		})
	}
}
