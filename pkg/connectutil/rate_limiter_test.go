package connectutil

import (
	"net/http"
	"net/http/httptest"
	"net/netip"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"golang.org/x/time/rate"
)

func TestRateLimitInterceptor(t *testing.T) {
	configs := map[string]RateLimitConfig{
		"/test.Service/Login":    {Rate: rate.Every(time.Second), Burst: 2},
		"/test.Service/Register": {Rate: rate.Every(time.Second), Burst: 1},
	}

	t.Run("non-rate-limited procedure passes through", func(t *testing.T) {
		rateLimiter := NewRateLimitInterceptor(configs)
		defer rateLimiter.Stop()

		// Non-rate-limited: should always pass
		for i := range 20 {
			ip := "1.2.3.4:5000"
			limiter := rateLimiter.getLimiter(extractIP(ip), RateLimitConfig{Rate: rate.Every(time.Second), Burst: 100})
			assert.True(t, limiter.Allow(), "request %d should pass", i)
		}
	})

	t.Run("rate-limited procedure allows requests within limit", func(t *testing.T) {
		rateLimiter := NewRateLimitInterceptor(configs)
		defer rateLimiter.Stop()

		ip := extractIP("1.2.3.4:5000")
		cfg := configs["/test.Service/Login"] // Burst: 2

		for i := range 2 {
			limiter := rateLimiter.getLimiter(ip, cfg)
			assert.True(t, limiter.Allow(), "request %d should pass within burst", i)
		}
	})

	t.Run("rate-limited procedure rejects requests exceeding limit", func(t *testing.T) {
		rateLimiter := NewRateLimitInterceptor(configs)
		defer rateLimiter.Stop()

		ip := extractIP("1.2.3.4:5000")
		cfg := configs["/test.Service/Register"] // Burst: 1

		limiter := rateLimiter.getLimiter(ip, cfg)
		// First request should succeed
		assert.True(t, limiter.Allow())
		// Second request should be rejected
		assert.False(t, limiter.Allow())
	})

	t.Run("different IPs have independent rate limits", func(t *testing.T) {
		rateLimiter := NewRateLimitInterceptor(configs)
		defer rateLimiter.Stop()

		cfg := configs["/test.Service/Register"] // Burst: 1

		limiter1 := rateLimiter.getLimiter("1.1.1.1", cfg)
		limiter2 := rateLimiter.getLimiter("2.2.2.2", cfg)

		// IP 1 uses its burst
		assert.True(t, limiter1.Allow())
		// IP 1 exceeds limit
		assert.False(t, limiter1.Allow())
		// IP 2 should still work (independent limiter)
		assert.True(t, limiter2.Allow())
	})

	t.Run("cleanup removes stale entries", func(t *testing.T) {
		rateLimiter := NewRateLimitInterceptor(configs)
		defer rateLimiter.Stop()

		// Manually add a stale entry
		rateLimiter.mu.Lock()
		rateLimiter.limiters["stale-ip"] = &ipLimiter{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now().Add(-15 * time.Minute),
		}
		rateLimiter.limiters["fresh-ip"] = &ipLimiter{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now(),
		}
		rateLimiter.mu.Unlock()

		rateLimiter.cleanup()

		rateLimiter.mu.Lock()
		defer rateLimiter.mu.Unlock()

		_, staleExists := rateLimiter.limiters["stale-ip"]
		_, freshExists := rateLimiter.limiters["fresh-ip"]

		assert.False(t, staleExists, "stale entry should be removed")
		assert.True(t, freshExists, "fresh entry should remain")
	})
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		addr     string
		expected string
	}{
		{"192.168.1.1:8080", "192.168.1.1"},
		{"10.0.0.1:443", "10.0.0.1"},
		{"[::1]:8080", "::1"},
		{"192.168.1.1", "192.168.1.1"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.addr, func(t *testing.T) {
			result := extractIP(tt.addr)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRateLimitInterceptorClientIP(t *testing.T) {
	proxies, err := ParseTrustedProxies([]string{"10.0.0.0/8", "192.168.1.5"})
	require.NoError(t, err)

	headers := func(pairs ...[2]string) http.Header {
		h := http.Header{}
		for _, pair := range pairs {
			h.Add(pair[0], pair[1])
		}

		return h
	}

	trusting := NewRateLimitInterceptor(nil, WithTrustedProxies(proxies))
	defer trusting.Stop()

	untrusting := NewRateLimitInterceptor(nil)
	defer untrusting.Stop()

	tests := []struct {
		name     string
		limiter  *RateLimitInterceptor
		peer     string
		header   http.Header
		expected string
	}{
		{"headers ignored without trusted proxies", untrusting, "10.1.1.1:443", headers([2]string{"X-Forwarded-For", "203.0.113.7"}), "10.1.1.1"},
		{"headers ignored from untrusted peer", trusting, "198.51.100.1:443", headers([2]string{"X-Forwarded-For", "203.0.113.7"}), "198.51.100.1"},
		{"forwarded client from trusted peer", trusting, "10.1.1.1:443", headers([2]string{"X-Forwarded-For", "203.0.113.7"}), "203.0.113.7"},
		{"spoofed left entries skipped", trusting, "10.1.1.1:443", headers([2]string{"X-Forwarded-For", "1.1.1.1, 203.0.113.7, 10.2.2.2"}), "203.0.113.7"},
		{"multiple header lines joined", trusting, "192.168.1.5:443", headers([2]string{"X-Forwarded-For", "1.1.1.1"}, [2]string{"X-Forwarded-For", "203.0.113.7"}), "203.0.113.7"},
		{"x-real-ip fallback", trusting, "10.1.1.1:443", headers([2]string{"X-Real-IP", "203.0.113.8"}), "203.0.113.8"},
		{"malformed forwarded falls back to x-real-ip", trusting, "10.1.1.1:443", headers([2]string{"X-Forwarded-For", "garbage"}, [2]string{"X-Real-IP", "203.0.113.8"}), "203.0.113.8"},
		{"peer when no headers", trusting, "10.1.1.1:443", headers(), "10.1.1.1"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.limiter.clientIP(tt.peer, tt.header))
		})
	}
}

func TestParseTrustedProxies(t *testing.T) {
	prefixes, err := ParseTrustedProxies([]string{"10.0.0.0/8", " 127.0.0.1 ", "", "::1"})
	require.NoError(t, err)
	require.Len(t, prefixes, 3)
	assert.True(t, prefixes[1].Contains(netip.MustParseAddr("127.0.0.1")))
	assert.False(t, prefixes[1].Contains(netip.MustParseAddr("127.0.0.2")))

	_, err = ParseTrustedProxies([]string{"not-an-ip"})
	require.Error(t, err)
}

func TestRateLimitInterceptorAllowRequest(t *testing.T) {
	proxies, err := ParseTrustedProxies([]string{"10.0.0.0/8"})
	require.NoError(t, err)

	limiter := NewRateLimitInterceptor(map[string]RateLimitConfig{"/oauth/register": {Rate: rate.Every(time.Hour), Burst: 2}}, WithTrustedProxies(proxies))
	defer limiter.Stop()

	request := func(remoteAddr, forwardedFor string) *http.Request {
		req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/oauth/register", nil)
		req.RemoteAddr = remoteAddr

		if forwardedFor != "" {
			req.Header.Set("X-Forwarded-For", forwardedFor)
		}

		return req
	}

	assert.True(t, limiter.AllowRequest("/oauth/register", request("198.51.100.1:1000", "")))
	assert.True(t, limiter.AllowRequest("/oauth/register", request("198.51.100.1:1001", "")))
	assert.False(t, limiter.AllowRequest("/oauth/register", request("198.51.100.1:1002", "")), "burst spent")
	assert.False(t, limiter.AllowRequest("/oauth/register", request("198.51.100.1:1003", "203.0.113.9")), "an untrusted peer cannot pick its IP")

	assert.True(t, limiter.AllowRequest("/oauth/register", request("198.51.100.2:1000", "")), "another client has its own bucket")
	assert.True(t, limiter.AllowRequest("/oauth/register", request("10.1.1.1:1000", "203.0.113.7")))
	assert.True(t, limiter.AllowRequest("/oauth/register", request("10.1.1.2:1000", "203.0.113.7")))
	assert.False(t, limiter.AllowRequest("/oauth/register", request("10.1.1.3:1000", "203.0.113.7")), "behind a trusted proxy the forwarded client is limited")

	assert.True(t, limiter.AllowRequest("/other", request("198.51.100.1:1004", "")), "an unconfigured key is not limited")
}
