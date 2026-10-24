package connectutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/netip"
	"strings"
	"time"

	"connectrpc.com/connect"
)

// RateLimitConfig allows Limit requests per Window for a procedure.
type RateLimitConfig struct {
	Limit  int
	Window time.Duration
}

// RateLimitStore counts requests per key in fixed windows shared by every
// replica.
type RateLimitStore interface {
	// Hit counts one request against key and reports whether the count is
	// still within limit for the current window.
	Hit(ctx context.Context, key string, limit int, window time.Duration) (bool, error)
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
// Each procedure has its own counter per IP, kept in the store.
type RateLimitInterceptor struct {
	store          RateLimitStore
	configs        map[string]RateLimitConfig // procedure -> config
	trustedProxies []netip.Prefix
}

// NewRateLimitInterceptor creates a rate limiting interceptor.
// configs maps procedure paths to their rate limit configuration.
func NewRateLimitInterceptor(store RateLimitStore, configs map[string]RateLimitConfig, opts ...RateLimitOption) *RateLimitInterceptor {
	rateLimiter := &RateLimitInterceptor{
		store:   store,
		configs: configs,
	}
	for _, opt := range opts {
		opt(rateLimiter)
	}

	return rateLimiter
}

// WrapUnary implements connect.Interceptor. It puts the client IP of every
// request into the context and counts the configured procedures. A store
// error refuses the request.
func (r *RateLimitInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		clientIP := r.clientIP(req.Peer().Addr, req.Header())
		ctx = ContextWithClientIP(ctx, clientIP)

		procedure := req.Spec().Procedure

		cfg, ok := r.configs[procedure]
		if !ok {
			return next(ctx, req)
		}

		allowed, err := r.store.Hit(ctx, procedure+"|"+clientIP, cfg.Limit, cfg.Window)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, errors.New("rate limit unavailable"))
		}

		if !allowed {
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
// proceed, counting it against the counter of the request's client IP. A key
// with no configuration is not limited; a store error refuses the request.
func (r *RateLimitInterceptor) AllowRequest(key string, req *http.Request) bool {
	cfg, ok := r.configs[key]
	if !ok {
		return true
	}

	allowed, err := r.store.Hit(req.Context(), key+"|"+r.clientIP(req.RemoteAddr, req.Header), cfg.Limit, cfg.Window)

	return err == nil && allowed
}
