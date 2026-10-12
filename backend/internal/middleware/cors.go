package middleware

import (
	"net/http"
	"net/url"
	"strings"
)

// isOriginAllowed checks if a given origin matches any allowed origin pattern.
func isOriginAllowed(origin string, allowedOrigins []string) bool {
	if origin == "" {
		return false
	}

	cleanOrigin := strings.TrimRight(strings.TrimSpace(origin), "/")
	parsedOrigin, err := url.Parse(cleanOrigin)
	originHost := ""
	originScheme := ""
	if err == nil {
		originHost = strings.ToLower(parsedOrigin.Hostname())
		originScheme = strings.ToLower(parsedOrigin.Scheme)
	}

	for _, pattern := range allowedOrigins {
		pattern = strings.TrimRight(strings.TrimSpace(pattern), "/")
		if pattern == "" {
			continue
		}

		if pattern == "*" || strings.EqualFold(pattern, cleanOrigin) {
			return true
		}

		// Direct match on host if pattern has no scheme (e.g. "mxguard.jullie.tech")
		if originHost != "" && strings.EqualFold(pattern, originHost) {
			return true
		}

		// Handle wildcards: e.g. "*.jullie.tech", "https://*.jullie.tech"
		targetHostPattern := pattern
		if strings.Contains(targetHostPattern, "://") {
			if u, err := url.Parse(targetHostPattern); err == nil {
				if originScheme != "" && !strings.EqualFold(originScheme, u.Scheme) {
					continue
				}
				targetHostPattern = u.Hostname()
			}
		}

		if strings.HasPrefix(targetHostPattern, "*.") {
			rootDomain := strings.ToLower(strings.TrimPrefix(targetHostPattern, "*."))
			if originHost != "" && (originHost == rootDomain || strings.HasSuffix(originHost, "."+rootDomain)) {
				return true
			}
		} else if strings.HasPrefix(targetHostPattern, ".") {
			rootDomain := strings.ToLower(strings.TrimPrefix(targetHostPattern, "."))
			if originHost != "" && (originHost == rootDomain || strings.HasSuffix(originHost, "."+rootDomain)) {
				return true
			}
		}
	}

	return false
}

// CORSMiddleware handles Cross-Origin Resource Sharing headers and preflight OPTIONS requests.
func CORSMiddleware(allowedOrigins []string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")

			if origin != "" && isOriginAllowed(origin, allowedOrigins) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS, HEAD")

				reqHeaders := r.Header.Get("Access-Control-Request-Headers")
				if reqHeaders != "" {
					w.Header().Set("Access-Control-Allow-Headers", reqHeaders)
				} else {
					w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Origin, X-Requested-With, X-CSRF-Token, X-Device-Key")
				}

				w.Header().Set("Access-Control-Expose-Headers", "Content-Length, Content-Type, Authorization, Set-Cookie")
				w.Header().Set("Access-Control-Max-Age", "86400")
			}

			if r.Method == http.MethodOptions {
				if origin != "" && !isOriginAllowed(origin, allowedOrigins) {
					w.WriteHeader(http.StatusForbidden)
					return
				}
				w.WriteHeader(http.StatusNoContent)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}
