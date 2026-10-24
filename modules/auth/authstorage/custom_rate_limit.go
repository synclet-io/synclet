package authstorage

import (
	"context"
	"fmt"
	"time"

	"gorm.io/gorm"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// RateLimitStorage counts hits per key in auth.rate_limits.
type RateLimitStorage struct {
	db *gorm.DB
}

func NewRateLimitStorage(db *gorm.DB) *RateLimitStorage {
	return &RateLimitStorage{db: db}
}

var _ authservice.RateLimitStorage = (*RateLimitStorage)(nil)

func (s *RateLimitStorage) HitRateLimit(ctx context.Context, key string, now time.Time, window time.Duration) (int, error) {
	var count int

	err := s.db.WithContext(ctx).
		Raw(`INSERT INTO auth.rate_limits AS r (key, count, expires_at) VALUES (@key, 1, @expires_at)
			ON CONFLICT (key) DO UPDATE SET
			  count = CASE WHEN r.expires_at <= @now THEN 1 ELSE r.count + 1 END,
			  expires_at = CASE WHEN r.expires_at <= @now THEN EXCLUDED.expires_at ELSE r.expires_at END
			RETURNING count`,
			map[string]any{"key": key, "now": now.UTC(), "expires_at": now.Add(window).UTC()}).
		Scan(&count).Error
	if err != nil {
		return 0, fmt.Errorf("hitting rate limit: %w", wrapRateLimitQueryError(err))
	}

	return count, nil
}
