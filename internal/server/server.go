package server

import (
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"github-badges-backend/internal/auth"
	"github-badges-backend/internal/config"
	"github-badges-backend/internal/middlewares"
	controllers "github-badges-backend/internal/server/controllers"

	"github.com/gin-gonic/gin"
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
	routerMux := gin.Default()

	routerMux.Use(RequestLogger(logger))
	routerMux.Use(middlewares.CorsMiddleware([]string{cfg.FrontendURL}))

	routerMux.GET("/health", cont.HealthCheck)

	routerMux.GET("/auth/login", authHandler.HandleLogin)
	routerMux.GET("/auth/logout", authHandler.HandleLogout)
	routerMux.GET("/auth/callback", authHandler.HandleCallback)

	apiGroup := routerMux.Group("/api")
	apiGroup.Use(authHandler.RequireAuth)
	apiGroup.GET("/me", cont.GetMe)
	apiGroup.GET("/stats", cont.GetStats)
	apiGroup.GET("/stats/history", cont.GetStatsHistory)

	adminGroup := apiGroup.Group("/admin")
	adminGroup.Use(auth.RequireAdmin(cfg.AdminKey))
	/** TODO:
	 * Make it a cron job or a cron job triggers this api
	 */
	adminGroup.POST("/trigger-monthly", cont.TriggerMonthly)

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

func RequestLogger(logger *zap.Logger) func(*gin.Context) {
	return func(ctx *gin.Context) {
		start := time.Now()
		ctx.Next()

		logger.Info("http request",
			zap.String("method", ctx.Request.Method),
			zap.String("path", ctx.Request.URL.Path),
			zap.Int("status", ctx.Request.Response.StatusCode),
			zap.Duration("duration", time.Since(start)),
			zap.String("remote", ctx.Request.RemoteAddr),
		)
	}
}
