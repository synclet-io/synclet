package authuserids_test

import (
	"context"
	"os"
	"sync"
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

type queryRecorder struct {
	mu      sync.Mutex
	queries []string
}

func (r *queryRecorder) record(db *gorm.DB) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.queries = append(r.queries, db.Statement.SQL.String())
}

func (r *queryRecorder) reset() {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.queries = nil
}

func TestExistingUserIDs_OneIDOnlyQuery(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping user id storage database test")
	}

	ctx := context.Background()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, authdbstate.MigrationsFS, "auth_goose_db_version")

	recorder := &queryRecorder{}
	require.NoError(t, db.Callback().Query().After("gorm:query").Register("authuserids:record", recorder.record))

	users := authstorage.NewStorages(db, nil, nil).Users()
	live := []uuid.UUID{uuid.New(), uuid.New()}

	for _, id := range live {
		_, err := users.Create(ctx, &authservice.User{
			ID:           id,
			Email:        id.String() + "@example.com",
			PasswordHash: "hash",
			CreatedAt:    time.Now().UTC(),
		})
		require.NoError(t, err)
	}

	t.Cleanup(func() {
		require.NoError(t, users.Delete(ctx, &authservice.UserFilter{ID: filter.In(live...)}))
	})

	recorder.reset()

	got, err := authstorage.NewUserIDStorage(db).ExistingUserIDs(ctx, append([]uuid.UUID{uuid.New()}, live...))
	require.NoError(t, err)

	assert.ElementsMatch(t, live, got)
	require.Len(t, recorder.queries, 1, "one query for the whole batch")
	assert.Equal(t, `SELECT "id" FROM "auth"."users" WHERE "id" IN ($1,$2,$3)`, recorder.queries[0])
}
