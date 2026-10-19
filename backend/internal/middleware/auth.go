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

func AuthMiddleware(jwtManager *security.JWTManager) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			tokenStr := ""
			authHeader := r.Header.Get("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenStr = strings.TrimPrefix(authHeader, "Bearer ")
			} else if cookie, err := r.Cookie("motoguard_token"); err == nil && cookie.Value != "" {
				tokenStr = cookie.Value
			}

			if tokenStr == "" {
				domain.NewUnauthorizedError("Authentication token is missing", r.URL.Path).WriteJSON(w)
				return
			}
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

func GetClaims(ctx context.Context) (*security.JWTClaims, bool) {
	claims, ok := ctx.Value(UserClaimsKey).(*security.JWTClaims)
	return claims, ok && claims != nil
}
