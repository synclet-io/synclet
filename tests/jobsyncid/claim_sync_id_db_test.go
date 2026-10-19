// Package jobsyncid_test runs job claiming against a real Postgres to pin the
// sync_id a job carries. It needs TEST_DATABASE_DSN.
package jobsyncid_test

import (
	"context"
	"os"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/synclet-io/synclet/modules/pipeline/pipelinedbstate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/tests/testdb"
)

type noopLogDelegate struct{}

func (noopLogDelegate) Info(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Warn(context.Context, string, ...interface{})         {}
func (noopLogDelegate) Debug(context.Context, string, ...interface{})        {}
func (noopLogDelegate) Error(context.Context, string, ...interface{})        {}
func (d noopLogDelegate) WithFields(map[string]interface{}) logging.Delegate { return d }
func (d noopLogDelegate) WithField(string, interface{}) logging.Delegate     { return d }
func (d noopLogDelegate) WithError(error) logging.Delegate                   { return d }
func (d noopLogDelegate) Named(string) logging.Delegate                      { return d }
func (d noopLogDelegate) SkipCallers(int) logging.Delegate                   { return d }

func setup(t *testing.T) (*gorm.DB, *pipelinestorage.Storages, uuid.UUID) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping job sync_id database test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, pipelinedbstate.MigrationsFS, "pipeline_goose_db_version")

	workspaceID, connID := uuid.New(), uuid.New()
	connectorID, sourceID, destID := uuid.New(), uuid.New(), uuid.New()

	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO pipeline.managed_connectors (id, workspace_id, docker_image, docker_tag, name, connector_type) VALUES (?, ?, 'img', 'latest', 'c', 'source')`, []any{connectorID, workspaceID}},
		{`INSERT INTO pipeline.sources (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 's', ?)`, []any{sourceID, workspaceID, connectorID}},
		{`INSERT INTO pipeline.destinations (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 'd', ?)`, []any{destID, workspaceID, connectorID}},
		{`INSERT INTO pipeline.connections (id, workspace_id, name, source_id, destination_id) VALUES (?, ?, 'conn', ?, ?)`, []any{connID, workspaceID, sourceID, destID}},
	} {
		require.NoError(t, db.Exec(stmt.sql, stmt.args...).Error)
	}

	t.Cleanup(func() {
		require.NoError(t, db.Exec(`DELETE FROM pipeline.connections WHERE id = ?`, connID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.sources WHERE id = ?`, sourceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.destinations WHERE id = ?`, destID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.managed_connectors WHERE id = ?`, connectorID).Error)
	})

	return db, pipelinestorage.NewStorages(db, &logging.Logger{Delegate: noopLogDelegate{}}, nil), connID
}

func scheduleJob(t *testing.T, db *gorm.DB, connID uuid.UUID) uuid.UUID {
	t.Helper()

	id := uuid.New()
	require.NoError(t, db.Exec(
		`INSERT INTO pipeline.jobs (id, connection_id, status, job_type, attempt, max_attempts, created_at)
		 VALUES (?, ?, 'scheduled', 'sync', 1, 3, '2000-01-01')`, id, connID).Error)

	return id
}

func claim(t *testing.T, storage *pipelinestorage.Storages, want uuid.UUID) *pipelineservice.Job {
	t.Helper()

	job, err := storage.Jobs().ClaimNextScheduledJob(context.Background(), "worker-1")
	require.NoError(t, err)
	require.NotNil(t, job)
	require.Equal(t, want, job.ID)
	require.NotNil(t, job.SyncID, "claimed job has no sync_id")

	return job
}

// TestClaimAssignsStableSyncID asserts that a job gets its sync_id at its first
// claim, keeps it when it is claimed again for a retry, and that a later job
// gets a larger one even after the connection's older jobs are deleted.
func TestClaimAssignsStableSyncID(t *testing.T) {
	db, storage, connID := setup(t)
	ctx := context.Background()

	firstID := scheduleJob(t, db, connID)
	first := claim(t, storage, firstID)
	syncID := *first.SyncID

	first.Status = pipelineservice.JobStatusScheduled
	first.Attempt = 2
	updated, err := storage.Jobs().UpdateIfStatus(ctx, first, pipelineservice.JobStatusStarting)
	require.NoError(t, err)
	require.True(t, updated)

	retry := claim(t, storage, firstID)
	assert.Equal(t, syncID, *retry.SyncID)
	assert.Equal(t, 2, retry.Attempt)

	require.NoError(t, db.Exec(`DELETE FROM pipeline.jobs WHERE id = ?`, firstID).Error)

	next := claim(t, storage, scheduleJob(t, db, connID))
	assert.Greater(t, *next.SyncID, syncID)
}
