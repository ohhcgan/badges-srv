package auth

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	"github.com/golang-jwt/jwt/v5"
)

/**
 * ValidateSession parses and validates the JWT from the session cookie.
 */
func (h *AuthHandler) ValidateSession(r *http.Request) (*JWTPayload, error) {
	cookie, err := r.Cookie(sessionCookieName)
	if err != nil || cookie.Value == "" {
		return nil, errors.New("no session cookie")
	}

	token, err := jwt.ParseWithClaims(cookie.Value, &JWTPayload{}, func(jwtTkn *jwt.Token) (any, error) {
		if _, ok := jwtTkn.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", jwtTkn.Header["alg"])
		}
		return h.jwtSecret, nil
	})

	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid session: %w", err)
	}

	claims, ok := token.Claims.(*JWTPayload)
	if !ok {
		return nil, errors.New("malformed claims")
	}
	return claims, nil
}

/**
 * Main authentication middleware
 */
func (h *AuthHandler) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(
		func(w http.ResponseWriter, r *http.Request) {
			payload, err := h.ValidateSession(r)

			if err != nil {
				http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
				return
			}

			//
			// TODO: uncomment
			//
			// userID, err := uuid.Parse(payload.UserID)
			// if err != nil {
			// 	http.Error(w, `{"error":"unauthorized"}`, http.StatusUnauthorized)
			// 	return
			// }
			//

			ctx := r.Context()
			ctx = context.WithValue(ctx, ContextKeyUserID, payload.UserID)
			ctx = context.WithValue(ctx, ContextKeyLogin, payload.GithubLogin)

			next.ServeHTTP(w, r.WithContext(ctx))
		},
	)
}

/**
 * Main admin authentication middleware
 */
func RequireAdmin(adminKey string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if adminKey == "" {
				http.NotFound(w, r)
				return
			}

			if r.Header.Get("X-Admin-Key") != adminKey {
				http.Error(w, `{"error":"forbidden"}`, http.StatusForbidden)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

func UserIDFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyUserID).(string)
	return v, ok
}

func LoginFromContext(ctx context.Context) (string, bool) {
	v, ok := ctx.Value(ContextKeyLogin).(string)
	return v, ok
}
