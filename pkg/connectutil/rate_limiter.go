package connectutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"sync"
	"time"

	"connectrpc.com/connect"
	"golang.org/x/time/rate"
)

type ipLimiter struct {
	limiter  *rate.Limiter
	lastSeen time.Time
}

// RateLimitConfig defines rate limit settings for a specific procedure.
type RateLimitConfig struct {
	Rate  rate.Limit // requests per second
	Burst int
}

// RateLimitOption configures a RateLimitInterceptor.
type RateLimitOption func(*RateLimitInterceptor)

// WithTrustedProxies makes the interceptor resolve the client IP from
// X-Forwarded-For (then X-Real-IP) when the TCP peer is one of the given
// proxies. Requests from any other peer are keyed on the peer address, so
// clients cannot spoof their IP by sending the headers directly.
func WithTrustedProxies(proxies []netip.Prefix) RateLimitOption {
	return func(r *RateLimitInterceptor) {
		r.trustedProxies = proxies
	}
}

// ParseTrustedProxies parses IP addresses and CIDR ranges.
func ParseTrustedProxies(values []string) ([]netip.Prefix, error) {
	prefixes := make([]netip.Prefix, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}

		if strings.Contains(value, "/") {
			prefix, err := netip.ParsePrefix(value)
			if err != nil {
				return nil, fmt.Errorf("parse trusted proxy %q: %w", value, err)
			}

			prefixes = append(prefixes, prefix.Masked())

			continue
		}

		addr, err := netip.ParseAddr(value)
		if err != nil {
			return nil, fmt.Errorf("parse trusted proxy %q: %w", value, err)
		}

		addr = addr.Unmap()
		prefixes = append(prefixes, netip.PrefixFrom(addr, addr.BitLen()))
	}

	return prefixes, nil
}

// RateLimitInterceptor rate-limits specific ConnectRPC procedures per client IP.
// Each procedure has its own bucket per IP.
type RateLimitInterceptor struct {
	mu             sync.Mutex
	limiters       map[string]*ipLimiter
	configs        map[string]RateLimitConfig // procedure -> config
	trustedProxies []netip.Prefix
	stopCh         chan struct{}
}

// NewRateLimitInterceptor creates a rate limiting interceptor.
// configs maps procedure paths to their rate limit configuration.
func NewRateLimitInterceptor(configs map[string]RateLimitConfig, opts ...RateLimitOption) *RateLimitInterceptor {
	rateLimiter := &RateLimitInterceptor{
		limiters: make(map[string]*ipLimiter),
		configs:  configs,
		stopCh:   make(chan struct{}),
	}
	for _, opt := range opts {
		opt(rateLimiter)
	}

	go rateLimiter.cleanupLoop()

	return rateLimiter
}

// Stop stops the cleanup goroutine.
func (r *RateLimitInterceptor) Stop() {
	close(r.stopCh)
}

func (r *RateLimitInterceptor) getLimiter(key string, cfg RateLimitConfig) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if limiter, ok := r.limiters[key]; ok {
		limiter.lastSeen = time.Now()

		return limiter.limiter
	}

	limiter := &ipLimiter{
		limiter:  rate.NewLimiter(cfg.Rate, cfg.Burst),
		lastSeen: time.Now(),
	}
	r.limiters[key] = limiter

	return limiter.limiter
}

func (r *RateLimitInterceptor) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			r.cleanup()
		case <-r.stopCh:
			return
		}
	}
}

func (r *RateLimitInterceptor) cleanup() {
	r.mu.Lock()
	defer r.mu.Unlock()

	for key, l := range r.limiters {
		if time.Since(l.lastSeen) > 10*time.Minute {
			delete(r.limiters, key)
		}
	}
}

// WrapUnary implements connect.Interceptor.
func (r *RateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		procedure := req.Spec().Procedure

		cfg, ok := r.configs[procedure]
		if !ok {
			return next(ctx, req)
		}

		ip := r.clientIP(req.Peer().Addr, req.Header())

		limiter := r.getLimiter(procedure+"|"+ip, cfg)
		if !limiter.Allow() {
			return nil, connect.NewError(connect.CodeResourceExhausted, errors.New("rate limit exceeded, try again later"))
		}

		return next(ctx, req)
	}
}

// WrapStreamingClient implements connect.Interceptor.
func (r *RateLimitInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

// WrapStreamingHandler implements connect.Interceptor.
func (r *RateLimitInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next // No streaming auth endpoints
}

// clientIP returns the address of the client that sent the request. Behind a
// trusted proxy it is the right-most X-Forwarded-For entry that is not itself
// a trusted proxy, falling back to X-Real-IP and then the peer address.
func (r *RateLimitInterceptor) clientIP(peerAddr string, header http.Header) string {
	peer := extractIP(peerAddr)
	if !r.isTrustedProxy(peer) {
		return peer
	}

	forwarded := strings.Split(strings.Join(header.Values("X-Forwarded-For"), ","), ",")
	for i := len(forwarded) - 1; i >= 0; i-- {
		addr, err := netip.ParseAddr(strings.TrimSpace(forwarded[i]))
		if err != nil {
			break
		}

		if !r.isTrustedProxy(addr.String()) {
			return addr.Unmap().String()
		}
	}

	if addr, err := netip.ParseAddr(strings.TrimSpace(header.Get("X-Real-IP"))); err == nil {
		return addr.Unmap().String()
	}

	return peer
}

func (r *RateLimitInterceptor) isTrustedProxy(ip string) bool {
	if len(r.trustedProxies) == 0 {
		return false
	}

	addr, err := netip.ParseAddr(ip)
	if err != nil {
		return false
	}

	addr = addr.Unmap()
	for _, prefix := range r.trustedProxies {
		if prefix.Contains(addr) {
			return true
		}
	}

	return false
}

// extractIP extracts the IP from an address string (ip:port or just ip).
func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}

	return host
}

// AllowRequest reports whether a plain HTTP request to the endpoint key may
// proceed, counting it against the bucket of the request's client IP. A key
// with no configuration is not limited.
func (r *RateLimitInterceptor) AllowRequest(key string, req *http.Request) bool {
	cfg, ok := r.configs[key]
	if !ok {
		return true
	}

	return r.getLimiter(key+"|"+r.clientIP(req.RemoteAddr, req.Header), cfg).Allow()
}
