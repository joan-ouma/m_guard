package middleware

import (
	"context"
	"net/http"
	"strings"

	"motoguard-backend/internal/domain"
	"motoguard-backend/internal/security"
)

type contextKey string

const UserClaimsKey contextKey = "userClaims"

// AuthMiddleware validates JWT Bearer tokens and attaches claims to the request context.
func AuthMiddleware(jwtManager *security.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
				domain.NewUnauthorizedError("Authentication token is missing", r.URL.Path).WriteJSON(w)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			claims, err := jwtManager.ValidateAccessToken(tokenStr)
			if err != nil {
				domain.NewUnauthorizedError("Authentication token is invalid or expired", r.URL.Path).WriteJSON(w)
				return
			}

			ctx := context.WithValue(r.Context(), UserClaimsKey, claims)
			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

// GetClaims retrieves validated JWTClaims from the request context if present.
func GetClaims(ctx context.Context) (*security.JWTClaims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*security.JWTClaims)
	return claims, ok && claims != nil
}
