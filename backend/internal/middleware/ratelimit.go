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

type IPTrackerState struct {
	ConsecutiveFailures int
	BlockedUntil        time.Time
	IsLockedOut         bool
	LastAttempt         time.Time
}

type TrackerRateLimiter struct {
	mu              sync.Mutex
	states          map[string]*IPTrackerState
	baseWindow      time.Duration
	maxFailures     int
	lockoutDuration time.Duration
}

func NewTrackerRateLimiter(baseWindow time.Duration, maxFailures int, lockoutDuration time.Duration) *TrackerRateLimiter {
	if baseWindow <= 0 {
		baseWindow = 1 * time.Minute
	}
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if lockoutDuration <= 0 {
		lockoutDuration = 24 * time.Hour
	}
	return &TrackerRateLimiter{
		states:          make(map[string]*IPTrackerState),
		baseWindow:      baseWindow,
		maxFailures:     maxFailures,
		lockoutDuration: lockoutDuration,
	}
}

// CheckAllowed checks if the IP is allowed to attempt verification.
// Returns (allowed, retryAfterSeconds, isLockedOut).
func (l *TrackerRateLimiter) CheckAllowed(ip string) (bool, int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	state, exists := l.states[ip]
	if !exists {
		return true, 0, false
	}

	now := time.Now()
	if state.IsLockedOut {
		return false, 86400, true
	}

	if now.Before(state.BlockedUntil) {
		remainingSec := int(state.BlockedUntil.Sub(now).Seconds()) + 1
		return false, remainingSec, false
	}

	return true, 0, false
}

// RecordFailure records a failed verification lookup for the given IP and applies exponential backoff.
// Returns (retryAfterSeconds, isLockedOut).
func (l *TrackerRateLimiter) RecordFailure(ip string) (int, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	state, exists := l.states[ip]
	if !exists {
		state = &IPTrackerState{}
		l.states[ip] = state
	}

	state.LastAttempt = now
	state.ConsecutiveFailures++

	if state.ConsecutiveFailures >= l.maxFailures {
		state.IsLockedOut = true
		state.BlockedUntil = now.Add(l.lockoutDuration)
		return int(l.lockoutDuration.Seconds()), true
	}

	// Exponential backoff: baseWindow * 2^(consecutiveFailures - 1)
	// 1st: 1 * baseWindow (e.g. 1m)
	// 2nd: 2 * baseWindow (e.g. 2m)
	// 3rd: 4 * baseWindow (e.g. 4m)
	// 4th: 8 * baseWindow (e.g. 8m)
	multiplier := 1 << (state.ConsecutiveFailures - 1)
	backoff := l.baseWindow * time.Duration(multiplier)
	state.BlockedUntil = now.Add(backoff)

	return int(backoff.Seconds()), false
}

// RecordSuccess records a successful verification, resetting failures and enforcing the baseline rate limit.
func (l *TrackerRateLimiter) RecordSuccess(ip string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	now := time.Now()
	state, exists := l.states[ip]
	if !exists {
		state = &IPTrackerState{}
		l.states[ip] = state
	}

	state.LastAttempt = now
	state.ConsecutiveFailures = 0
	state.IsLockedOut = false
	// Baseline: 1 request per minute even after success
	state.BlockedUntil = now.Add(l.baseWindow)
}
