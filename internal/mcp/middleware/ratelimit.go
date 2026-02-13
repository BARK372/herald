package middleware

import (
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/btouchard/herald/internal/config"
)

// tokenBucket implements a simple token bucket rate limiter.
type tokenBucket struct {
	tokens     float64
	maxTokens  float64
	refillRate float64 // tokens per second
	lastRefill time.Time
}

func newTokenBucket(maxTokens int, refillRate float64) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(maxTokens),
		maxTokens:  float64(maxTokens),
		refillRate: refillRate,
		lastRefill: time.Now(),
	}
}

// allow checks if a request is allowed and consumes a token if so.
// Returns true if allowed, and the time to wait before retrying if not.
func (b *tokenBucket) allow(now time.Time) (bool, time.Duration) {
	elapsed := now.Sub(b.lastRefill).Seconds()
	b.tokens += elapsed * b.refillRate
	if b.tokens > b.maxTokens {
		b.tokens = b.maxTokens
	}
	b.lastRefill = now

	if b.tokens >= 1 {
		b.tokens--
		return true, 0
	}

	wait := time.Duration((1 - b.tokens) / b.refillRate * float64(time.Second))
	return false, wait
}

// rateLimiter manages per-key token buckets with periodic cleanup.
type rateLimiter struct {
	mu      sync.Mutex
	buckets map[string]*tokenBucket
	burst   int
	rate    float64 // tokens per second
	cleanup time.Duration
}

func newRateLimiter(requestsPerMinute, burst int) *rateLimiter {
	rl := &rateLimiter{
		buckets: make(map[string]*tokenBucket),
		burst:   burst,
		rate:    float64(requestsPerMinute) / 60.0,
		cleanup: 10 * time.Minute,
	}
	go rl.cleanupLoop()
	return rl
}

// allow checks if the key is within its rate limit.
func (rl *rateLimiter) allow(key string) (bool, time.Duration) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	bucket, ok := rl.buckets[key]
	if !ok {
		bucket = newTokenBucket(rl.burst, rl.rate)
		rl.buckets[key] = bucket
	}

	return bucket.allow(time.Now())
}

// cleanupLoop removes stale entries that haven't been used recently.
func (rl *rateLimiter) cleanupLoop() {
	ticker := time.NewTicker(rl.cleanup)
	defer ticker.Stop()

	for range ticker.C {
		rl.mu.Lock()
		now := time.Now()
		for key, bucket := range rl.buckets {
			// Remove buckets that are full and haven't been used in a while.
			// A bucket is "stale" if its last refill was long enough ago
			// that it would be fully replenished.
			timeSinceUse := now.Sub(bucket.lastRefill)
			if timeSinceUse > rl.cleanup {
				delete(rl.buckets, key)
			}
		}
		rl.mu.Unlock()
	}
}

// RateLimit returns per-Bearer-token rate limiting middleware.
// It extracts the token from the Authorization header and applies limits per token.
func RateLimit(cfg config.RateLimitConfig) func(http.Handler) http.Handler {
	if cfg.RequestsPerMinute <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	rl := newRateLimiter(cfg.RequestsPerMinute, cfg.Burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := extractBearerToken(r)
			if key == "" {
				// No token — fall through to auth middleware which will reject it.
				next.ServeHTTP(w, r)
				return
			}

			allowed, retryAfter := rl.allow(key)
			if !allowed {
				slog.Warn("rate limit exceeded",
					"retry_after", retryAfter.Seconds())
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()+1))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// IPRateLimit returns per-IP rate limiting middleware for unauthenticated endpoints.
func IPRateLimit(requestsPerMinute, burst int) func(http.Handler) http.Handler {
	if requestsPerMinute <= 0 {
		return func(next http.Handler) http.Handler { return next }
	}

	rl := newRateLimiter(requestsPerMinute, burst)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			key := clientIP(r)

			allowed, retryAfter := rl.allow(key)
			if !allowed {
				slog.Warn("ip rate limit exceeded",
					"ip", key,
					"retry_after", retryAfter.Seconds())
				w.Header().Set("Retry-After", fmt.Sprintf("%.0f", retryAfter.Seconds()+1))
				http.Error(w, "rate limit exceeded", http.StatusTooManyRequests)
				return
			}

			next.ServeHTTP(w, r)
		})
	}
}

// extractBearerToken extracts the bearer token from the Authorization header.
func extractBearerToken(r *http.Request) string {
	header := r.Header.Get("Authorization")
	if header == "" {
		return ""
	}
	parts := strings.SplitN(header, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return ""
	}
	return parts[1]
}

// clientIP extracts the client IP address from the request.
// It checks X-Forwarded-For first (for Traefik), then falls back to RemoteAddr.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		if i := strings.IndexByte(xff, ','); i != -1 {
			return strings.TrimSpace(xff[:i])
		}
		return strings.TrimSpace(xff)
	}

	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
