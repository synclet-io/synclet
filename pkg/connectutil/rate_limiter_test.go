package connectutil

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"golang.org/x/time/rate"
)

func TestRateLimitInterceptor(t *testing.T) {
	configs := map[string]RateLimitConfig{
		"/test.Service/Login":    {Rate: rate.Every(time.Second), Burst: 2},
		"/test.Service/Register": {Rate: rate.Every(time.Second), Burst: 1},
	}

	t.Run("non-rate-limited procedure passes through", func(t *testing.T) {
		rl := NewRateLimitInterceptor(configs)
		defer rl.Stop()

		// Non-rate-limited: should always pass
		for i := 0; i < 20; i++ {
			ip := "1.2.3.4:5000"
			limiter := rl.getLimiter(extractIP(ip), RateLimitConfig{Rate: rate.Every(time.Second), Burst: 100})
			assert.True(t, limiter.Allow(), "request %d should pass", i)
		}
	})

	t.Run("rate-limited procedure allows requests within limit", func(t *testing.T) {
		rl := NewRateLimitInterceptor(configs)
		defer rl.Stop()

		ip := extractIP("1.2.3.4:5000")
		cfg := configs["/test.Service/Login"] // Burst: 2

		for i := 0; i < 2; i++ {
			limiter := rl.getLimiter(ip, cfg)
			assert.True(t, limiter.Allow(), "request %d should pass within burst", i)
		}
	})

	t.Run("rate-limited procedure rejects requests exceeding limit", func(t *testing.T) {
		rl := NewRateLimitInterceptor(configs)
		defer rl.Stop()

		ip := extractIP("1.2.3.4:5000")
		cfg := configs["/test.Service/Register"] // Burst: 1

		limiter := rl.getLimiter(ip, cfg)
		// First request should succeed
		assert.True(t, limiter.Allow())
		// Second request should be rejected
		assert.False(t, limiter.Allow())
	})

	t.Run("different IPs have independent rate limits", func(t *testing.T) {
		rl := NewRateLimitInterceptor(configs)
		defer rl.Stop()

		cfg := configs["/test.Service/Register"] // Burst: 1

		limiter1 := rl.getLimiter("1.1.1.1", cfg)
		limiter2 := rl.getLimiter("2.2.2.2", cfg)

		// IP 1 uses its burst
		assert.True(t, limiter1.Allow())
		// IP 1 exceeds limit
		assert.False(t, limiter1.Allow())
		// IP 2 should still work (independent limiter)
		assert.True(t, limiter2.Allow())
	})

	t.Run("cleanup removes stale entries", func(t *testing.T) {
		rl := NewRateLimitInterceptor(configs)
		defer rl.Stop()

		// Manually add a stale entry
		rl.mu.Lock()
		rl.limiters["stale-ip"] = &ipLimiter{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now().Add(-15 * time.Minute),
		}
		rl.limiters["fresh-ip"] = &ipLimiter{
			limiter:  rate.NewLimiter(1, 1),
			lastSeen: time.Now(),
		}
		rl.mu.Unlock()

		rl.cleanup()

		rl.mu.Lock()
		defer rl.mu.Unlock()
		_, staleExists := rl.limiters["stale-ip"]
		_, freshExists := rl.limiters["fresh-ip"]
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
