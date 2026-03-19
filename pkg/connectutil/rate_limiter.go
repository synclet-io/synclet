package connectutil

import (
	"context"
	"fmt"
	"net"
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

// RateLimitInterceptor rate-limits specific ConnectRPC procedures by client IP.
type RateLimitInterceptor struct {
	mu       sync.Mutex
	limiters map[string]*ipLimiter
	configs  map[string]RateLimitConfig // procedure -> config
	stopCh   chan struct{}
}

// NewRateLimitInterceptor creates a rate limiting interceptor.
// configs maps procedure paths to their rate limit configuration.
func NewRateLimitInterceptor(configs map[string]RateLimitConfig) *RateLimitInterceptor {
	rl := &RateLimitInterceptor{
		limiters: make(map[string]*ipLimiter),
		configs:  configs,
		stopCh:   make(chan struct{}),
	}
	go rl.cleanupLoop()
	return rl
}

// Stop stops the cleanup goroutine.
func (r *RateLimitInterceptor) Stop() {
	close(r.stopCh)
}

func (r *RateLimitInterceptor) getLimiter(ip string, cfg RateLimitConfig) *rate.Limiter {
	r.mu.Lock()
	defer r.mu.Unlock()

	if l, ok := r.limiters[ip]; ok {
		l.lastSeen = time.Now()
		return l.limiter
	}

	l := &ipLimiter{
		limiter:  rate.NewLimiter(cfg.Rate, cfg.Burst),
		lastSeen: time.Now(),
	}
	r.limiters[ip] = l
	return l.limiter
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
		cfg, ok := r.configs[req.Spec().Procedure]
		if !ok {
			return next(ctx, req)
		}

		ip := extractIP(req.Peer().Addr)
		limiter := r.getLimiter(ip, cfg)
		if !limiter.Allow() {
			return nil, connect.NewError(connect.CodeResourceExhausted, fmt.Errorf("rate limit exceeded, try again later"))
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

// extractIP extracts the IP from an address string (ip:port or just ip).
func extractIP(addr string) string {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return addr
	}
	return host
}
