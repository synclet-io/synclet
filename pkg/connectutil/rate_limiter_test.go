package connectutil

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// memRateLimitStore counts hits per key without windows.
type memRateLimitStore struct {
	mu     sync.Mutex
	counts map[string]int
	err    error
}

func newMemRateLimitStore() *memRateLimitStore {
	return &memRateLimitStore{counts: map[string]int{}}
}

func (s *memRateLimitStore) Hit(_ context.Context, key string, limit int, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.err != nil {
		return false, s.err
	}

	s.counts[key]++

	return s.counts[key] <= limit, nil
}

func TestRateLimitInterceptorAllowRequest_StoreErrorRefuses(t *testing.T) {
	store := newMemRateLimitStore()
	store.err = errors.New("database down")

	limiter := NewRateLimitInterceptor(store, map[string]RateLimitConfig{"/oauth/register": {Limit: 10, Window: time.Minute}})

	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/oauth/register", nil)
	assert.False(t, limiter.AllowRequest("/oauth/register", req))
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

	trusting := NewRateLimitInterceptor(newMemRateLimitStore(), nil, WithTrustedProxies(proxies))
	untrusting := NewRateLimitInterceptor(newMemRateLimitStore(), nil)

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

	limiter := NewRateLimitInterceptor(newMemRateLimitStore(), map[string]RateLimitConfig{"/oauth/register": {Limit: 2, Window: time.Hour}}, WithTrustedProxies(proxies))

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
	assert.False(t, limiter.AllowRequest("/oauth/register", request("198.51.100.1:1002", "")), "limit spent")
	assert.False(t, limiter.AllowRequest("/oauth/register", request("198.51.100.1:1003", "203.0.113.9")), "an untrusted peer cannot pick its IP")

	assert.True(t, limiter.AllowRequest("/oauth/register", request("198.51.100.2:1000", "")), "another client has its own counter")
	assert.True(t, limiter.AllowRequest("/oauth/register", request("10.1.1.1:1000", "203.0.113.7")))
	assert.True(t, limiter.AllowRequest("/oauth/register", request("10.1.1.2:1000", "203.0.113.7")))
	assert.False(t, limiter.AllowRequest("/oauth/register", request("10.1.1.3:1000", "203.0.113.7")), "behind a trusted proxy the forwarded client is limited")

	assert.True(t, limiter.AllowRequest("/other", request("198.51.100.1:1004", "")), "an unconfigured key is not limited")
}
