package auditpending_test

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

	"github.com/synclet-io/synclet/modules/audit/auditdbstate"
	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/audit/auditstorage"
	"github.com/synclet-io/synclet/tests/testdb"
)

func openDB(t *testing.T) *gorm.DB {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping pending audit event database test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, auditdbstate.MigrationsFS, "audit_goose_db_version")

	return db
}

func userEvent(userID, workspaceID uuid.UUID, action auditservice.Action, at time.Time) *auditservice.AuditEvent {
	return &auditservice.AuditEvent{
		ID:           uuid.New(),
		WorkspaceID:  workspaceID,
		ActorType:    auditservice.ActorTypeUser,
		ActorID:      userID,
		Action:       action,
		ResourceType: auditservice.ResourceTypeUser,
		ResourceID:   userID,
		DiffJSON:     "[]",
		CreatedAt:    at,
	}
}

func workspaceOf(ctx context.Context, t *testing.T, storage *auditstorage.Storages, id uuid.UUID) uuid.UUID {
	t.Helper()

	event, err := storage.AuditEvents().First(ctx, &auditservice.AuditEventFilter{ID: filter.Equals(id)})
	require.NoError(t, err)

	return event.WorkspaceID
}

func TestAdoptPendingEvents(t *testing.T) {
	ctx := context.Background()
	db := openDB(t)
	storage := auditstorage.NewStorages(db, nil, nil)
	pending := auditstorage.NewPendingEventStorage(db)

	now := time.Now().UTC().Truncate(time.Microsecond)
	userID, otherUserID, formerMemberID := uuid.New(), uuid.New(), uuid.New()
	workspaceID := uuid.New()

	registered := userEvent(userID, uuid.Nil, auditservice.ActionCreate, now.Add(-time.Hour))
	loggedIn := userEvent(userID, uuid.Nil, auditservice.ActionLogin, now.Add(-time.Minute))
	loggedOut := userEvent(userID, uuid.Nil, auditservice.ActionLogout, now.Add(-time.Minute))
	laterLogin := userEvent(userID, uuid.Nil, auditservice.ActionLogin, now.Add(time.Minute))
	otherRegistered := userEvent(otherUserID, uuid.Nil, auditservice.ActionCreate, now.Add(-time.Hour))
	formerMemberLogin := userEvent(formerMemberID, uuid.Nil, auditservice.ActionLogin, now.Add(-time.Minute))

	for _, event := range []*auditservice.AuditEvent{registered, loggedIn, loggedOut, laterLogin, otherRegistered, formerMemberLogin} {
		_, err := storage.AuditEvents().Create(ctx, event)
		require.NoError(t, err)
	}

	moved, err := pending.AdoptPendingEvents(ctx, userID, workspaceID, now)
	require.NoError(t, err)
	assert.Equal(t, int64(2), moved)

	moved, err = pending.AdoptPendingEvents(ctx, userID, workspaceID, now)
	require.NoError(t, err)
	assert.Equal(t, int64(0), moved, "a second call moves nothing")

	moved, err = pending.AdoptPendingEvents(ctx, formerMemberID, workspaceID, now)
	require.NoError(t, err)
	assert.Equal(t, int64(0), moved, "no pending registration, nothing moves")

	assert.Equal(t, workspaceID, workspaceOf(ctx, t, storage, registered.ID))
	assert.Equal(t, workspaceID, workspaceOf(ctx, t, storage, loggedIn.ID))
	assert.Equal(t, uuid.Nil, workspaceOf(ctx, t, storage, loggedOut.ID))
	assert.Equal(t, uuid.Nil, workspaceOf(ctx, t, storage, laterLogin.ID))
	assert.Equal(t, uuid.Nil, workspaceOf(ctx, t, storage, otherRegistered.ID))
	assert.Equal(t, uuid.Nil, workspaceOf(ctx, t, storage, formerMemberLogin.ID))
}
