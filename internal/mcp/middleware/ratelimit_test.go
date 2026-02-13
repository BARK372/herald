package middleware

import (
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/btouchard/herald/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var okHandler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
})

func TestRateLimit_WithinBurst_AllowsRequests(t *testing.T) {
	t.Parallel()

	mw := RateLimit(config.RateLimitConfig{RequestsPerMinute: 60, Burst: 5})
	handler := mw(okHandler)

	for i := range 5 {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer token-abc")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code, "request %d should succeed", i+1)
	}
}

func TestRateLimit_ExceedsBurst_Returns429(t *testing.T) {
	t.Parallel()

	mw := RateLimit(config.RateLimitConfig{RequestsPerMinute: 60, Burst: 3})
	handler := mw(okHandler)

	// Exhaust the burst
	for range 3 {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer token-xyz")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Next request should be rejected
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer token-xyz")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
}

func TestRateLimit_Returns429WithRetryAfterHeader(t *testing.T) {
	t.Parallel()

	mw := RateLimit(config.RateLimitConfig{RequestsPerMinute: 60, Burst: 1})
	handler := mw(okHandler)

	// Exhaust
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer token-hdr")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Trigger 429
	req = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer token-hdr")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	retryAfter := rec.Header().Get("Retry-After")
	assert.NotEmpty(t, retryAfter, "Retry-After header must be present")

	seconds, err := strconv.ParseFloat(retryAfter, 64)
	require.NoError(t, err)
	assert.Greater(t, seconds, 0.0, "Retry-After must be positive")
}

func TestRateLimit_DifferentTokens_HaveSeparateLimits(t *testing.T) {
	t.Parallel()

	mw := RateLimit(config.RateLimitConfig{RequestsPerMinute: 60, Burst: 2})
	handler := mw(okHandler)

	// Token A: exhaust burst
	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer token-a")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Token A should be limited
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer token-a")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Token B should still work
	req = httptest.NewRequest(http.MethodPost, "/mcp", nil)
	req.Header.Set("Authorization", "Bearer token-b")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_NoToken_PassesThrough(t *testing.T) {
	t.Parallel()

	mw := RateLimit(config.RateLimitConfig{RequestsPerMinute: 60, Burst: 1})
	handler := mw(okHandler)

	// Without auth header, should pass through (auth middleware handles rejection)
	req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestRateLimit_DisabledWhenZero(t *testing.T) {
	t.Parallel()

	mw := RateLimit(config.RateLimitConfig{RequestsPerMinute: 0, Burst: 0})
	handler := mw(okHandler)

	for range 100 {
		req := httptest.NewRequest(http.MethodPost, "/mcp", nil)
		req.Header.Set("Authorization", "Bearer token-unlimited")
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		assert.Equal(t, http.StatusOK, rec.Code)
	}
}

func TestIPRateLimit_ExceedsBurst_Returns429(t *testing.T) {
	t.Parallel()

	mw := IPRateLimit(10, 2)
	handler := mw(okHandler)

	// Exhaust burst
	for range 2 {
		req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
		req.RemoteAddr = "192.168.1.10:54321"
		rec := httptest.NewRecorder()
		handler.ServeHTTP(rec, req)
		require.Equal(t, http.StatusOK, rec.Code)
	}

	// Should be rate limited
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.RemoteAddr = "192.168.1.10:54321"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	assert.Equal(t, http.StatusTooManyRequests, rec.Code)
	assert.NotEmpty(t, rec.Header().Get("Retry-After"))
}

func TestIPRateLimit_DifferentIPs_HaveSeparateLimits(t *testing.T) {
	t.Parallel()

	mw := IPRateLimit(10, 1)
	handler := mw(okHandler)

	// IP 1: exhaust
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// IP 1: limited
	req = httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.RemoteAddr = "10.0.0.1:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// IP 2: should still work
	req = httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.RemoteAddr = "10.0.0.2:1234"
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestIPRateLimit_UsesXForwardedFor(t *testing.T) {
	t.Parallel()

	mw := IPRateLimit(10, 1)
	handler := mw(okHandler)

	// Exhaust with X-Forwarded-For IP
	req := httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.RemoteAddr = "127.0.0.1:1234"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.1")
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	require.Equal(t, http.StatusOK, rec.Code)

	// Same XFF IP should be limited
	req = httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.RemoteAddr = "127.0.0.1:5678"
	req.Header.Set("X-Forwarded-For", "203.0.113.50, 10.0.0.2")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusTooManyRequests, rec.Code)

	// Different XFF IP should work
	req = httptest.NewRequest(http.MethodPost, "/oauth/token", nil)
	req.RemoteAddr = "127.0.0.1:9999"
	req.Header.Set("X-Forwarded-For", "203.0.113.51")
	rec = httptest.NewRecorder()
	handler.ServeHTTP(rec, req)
	assert.Equal(t, http.StatusOK, rec.Code)
}

func TestClientIP_ExtractsCorrectIP(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		remoteAddr string
		xff        string
		wantIP     string
	}{
		{
			name:       "from RemoteAddr with port",
			remoteAddr: "192.168.1.1:12345",
			wantIP:     "192.168.1.1",
		},
		{
			name:       "from X-Forwarded-For single IP",
			remoteAddr: "127.0.0.1:1234",
			xff:        "10.0.0.5",
			wantIP:     "10.0.0.5",
		},
		{
			name:       "from X-Forwarded-For first of multiple",
			remoteAddr: "127.0.0.1:1234",
			xff:        "203.0.113.10, 10.0.0.1, 172.16.0.1",
			wantIP:     "203.0.113.10",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			assert.Equal(t, tt.wantIP, clientIP(req))
		})
	}
}
