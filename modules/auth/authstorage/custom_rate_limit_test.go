package authstorage_test

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/synclet-io/synclet/modules/auth/authdbstate"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/auth/authstorage"
	"github.com/synclet-io/synclet/tests/testdb"
)

// openReplica opens its own connection pool, as a separate server replica would.
func openReplica(t *testing.T, dsn string) *gorm.DB {
	t.Helper()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	return db
}

// TestRateLimitStorage_SharedAcrossReplicas needs TEST_DATABASE_DSN.
func TestRateLimitStorage_SharedAcrossReplicas(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping rate limit database test")
	}

	replicaA := openReplica(t, dsn)
	replicaB := openReplica(t, dsn)

	sqlDB, err := replicaA.DB()
	require.NoError(t, err)
	testdb.Migrate(t, sqlDB, authdbstate.MigrationsFS, "auth_goose_db_version")

	ctx := t.Context()
	limitsA := authstorage.NewRateLimitStorage(replicaA)
	limitsB := authstorage.NewRateLimitStorage(replicaB)
	storage := authstorage.NewStorages(replicaA, nil, nil)

	key := "test|" + uuid.NewString()
	now := time.Now()

	t.Cleanup(func() {
		require.NoError(t, storage.RateLimits().Delete(context.Background(), &authservice.RateLimitFilter{Key: filter.Equals(key)}))
	})

	for i, limits := range []*authstorage.RateLimitStorage{limitsA, limitsB, limitsA} {
		count, err := limits.HitRateLimit(ctx, key, now, time.Minute)
		require.NoError(t, err)
		assert.Equal(t, i+1, count, "hit %d sees the hits of both replicas", i+1)
	}

	count, err := limitsB.HitRateLimit(ctx, key, now.Add(time.Minute), time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 1, count, "the count restarts after the window")

	count, err = limitsA.HitRateLimit(ctx, key, now.Add(time.Minute+time.Second), time.Minute)
	require.NoError(t, err)
	assert.Equal(t, 2, count, "the new window keeps counting")

	expiredKey := "test|" + uuid.NewString()
	_, err = limitsA.HitRateLimit(ctx, expiredKey, now.Add(-time.Hour), time.Minute)
	require.NoError(t, err)

	require.NoError(t, authservice.NewCleanupExpiredTokens(storage, authservice.NewStateStore(storage)).Execute(ctx))

	_, err = storage.RateLimits().First(ctx, &authservice.RateLimitFilter{Key: filter.Equals(expiredKey)})
	require.ErrorIs(t, err, authservice.ErrRateLimitNotFound, "cleanup removes the expired row")

	live, err := storage.RateLimits().First(ctx, &authservice.RateLimitFilter{Key: filter.Equals(key)})
	require.NoError(t, err, "cleanup keeps the live row")
	assert.Equal(t, 2, live.Count)
}
