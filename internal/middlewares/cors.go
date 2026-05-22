package middlewares

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

func CorsMiddleware(allowedOrigins []string) func(*gin.Context) {
	allowed := make(map[string]bool)
	for _, origin := range allowedOrigins {
		allowed[origin] = true
	}

	return func(ctx *gin.Context) {
		origin := ctx.GetHeader("Origin")

		if allowed[origin] {
			ctx.Header("Access-Control-Allow-Origin", origin)
		}

		ctx.Header("Vary", "Origin")
		ctx.Header("Access-Control-Allow-Credentials", "true")
		ctx.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		ctx.Header("Access-Control-Allow-Headers", "Content-Type, X-Admin-Key")

		if ctx.Request.Method == http.MethodOptions {
			ctx.Writer.WriteHeader(http.StatusNoContent)
			return
		}

		ctx.Next()
	}
}
