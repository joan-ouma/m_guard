package middleware

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"motoguard-backend/internal/domain"
)

func GetClientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return strings.TrimSpace(xri)
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err == nil {
		return host
	}
	return r.RemoteAddr
}

type rateLimiter struct {
	mu      sync.Mutex
	history map[string][]time.Time
	limit   int
	window  time.Duration
}

func RateLimitMiddleware(limit int, window time.Duration) func(http.Handler) http.Handler {
	rl := &rateLimiter{
		history: make(map[string][]time.Time),
		limit:   limit,
		window:  window,
	}

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			ip := GetClientIP(r)
			now := time.Now()

			rl.mu.Lock()
			timestamps := rl.history[ip]
			validFrom := now.Add(-rl.window)

			valid := timestamps[:0]
			for _, t := range timestamps {
				if t.After(validFrom) {
					valid = append(valid, t)
				}
			}

			if len(valid) >= rl.limit {
				rl.history[ip] = valid
				rl.mu.Unlock()
				domain.NewTooManyRequestsError("Too many attempts. Please try again later.", r.URL.Path).WriteJSON(w)
				return
			}

			valid = append(valid, now)
			rl.history[ip] = valid
			rl.mu.Unlock()

			next.ServeHTTP(w, r)
		})
	}
}
