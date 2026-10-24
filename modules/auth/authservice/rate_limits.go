package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RateLimitStorage holds the rate-limit query the generated storage cannot
// express.
type RateLimitStorage interface {
	// HitRateLimit adds one hit to the counter of key and returns its count.
	// A counter that does not exist or expired at now restarts at 1 and
	// expires at now+window.
	HitRateLimit(ctx context.Context, key string, now time.Time, window time.Duration) (int, error)
}

// RateLimits keeps fixed-window counters per key in the database, so every
// replica sees one shared count.
type RateLimits struct {
	storage Storage
	counter RateLimitStorage
	now     func() time.Time
}

// NewRateLimits creates a new RateLimits use case.
func NewRateLimits(storage Storage, counter RateLimitStorage) *RateLimits {
	return &RateLimits{storage: storage, counter: counter, now: time.Now}
}

// Hit counts one hit against key and reports whether the count is still
// within limit for the current window.
func (uc *RateLimits) Hit(ctx context.Context, key string, limit int, window time.Duration) (bool, error) {
	count, err := uc.counter.HitRateLimit(ctx, key, uc.now(), window)
	if err != nil {
		return false, err
	}

	return count <= limit, nil
}

// Peek returns the count of key in its current window, 0 when it has none.
func (uc *RateLimits) Peek(ctx context.Context, key string) (int, error) {
	row, err := uc.storage.RateLimits().First(ctx, &RateLimitFilter{
		Key:       filter.Equals(key),
		ExpiresAt: filter.Greater(uc.now()),
	})
	if err != nil {
		if errors.Is(err, ErrRateLimitNotFound) {
			return 0, nil
		}

		return 0, fmt.Errorf("reading rate limit: %w", err)
	}

	return row.Count, nil
}

// Reset removes the counter of key.
func (uc *RateLimits) Reset(ctx context.Context, key string) error {
	if err := uc.storage.RateLimits().Delete(ctx, &RateLimitFilter{Key: filter.Equals(key)}); err != nil {
		return fmt.Errorf("resetting rate limit: %w", err)
	}

	return nil
}
