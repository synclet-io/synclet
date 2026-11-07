// Package executorgroups_test runs job and connector task claiming against a
// real Postgres to pin executor group routing. It needs TEST_DATABASE_DSN.
package executorgroups_test

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

type fixture struct {
	db          *gorm.DB
	storage     *pipelinestorage.Storages
	workspaceID uuid.UUID
	connID      uuid.UUID
}

func setup(t *testing.T) fixture {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping executor group database test")
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
		require.NoError(t, db.Exec(`DELETE FROM pipeline.connector_tasks WHERE workspace_id = ?`, workspaceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.jobs WHERE connection_id = ?`, connID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.connections WHERE id = ?`, connID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.sources WHERE id = ?`, sourceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.destinations WHERE id = ?`, destID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.managed_connectors WHERE id = ?`, connectorID).Error)
	})

	return fixture{db: db, storage: pipelinestorage.NewStorages(db, &logging.Logger{Delegate: noopLogDelegate{}}, nil), workspaceID: workspaceID, connID: connID}
}

func (f fixture) scheduleJob(t *testing.T, group string) uuid.UUID {
	t.Helper()

	id := uuid.New()
	require.NoError(t, f.db.Exec(
		`INSERT INTO pipeline.jobs (id, connection_id, status, job_type, attempt, max_attempts, created_at, executor_group, queue_reason)
		 VALUES (?, ?, 'scheduled', 'sync', 1, 3, '2000-01-01', ?, 'executors in group "x" are busy')`, id, f.connID, group).Error)

	return id
}

func (f fixture) pendingTask(t *testing.T, taskType pipelineservice.ConnectorTaskType, payload pipelineservice.ConnectorTaskPayload, group *string) uuid.UUID {
	t.Helper()

	id := uuid.New()
	_, err := f.storage.ConnectorTasks().Create(context.Background(), &pipelineservice.ConnectorTask{
		ID:            id,
		WorkspaceID:   f.workspaceID,
		TaskType:      taskType,
		Status:        pipelineservice.ConnectorTaskStatusPending,
		Payload:       payload,
		ExecutorGroup: group,
	})
	require.NoError(t, err)
	require.NoError(t, f.db.Exec(`UPDATE pipeline.connector_tasks SET created_at = '1990-01-01' WHERE id = ?`, id).Error)

	return id
}

// TestClaimJobByGroup asserts that a scheduled job is claimed only by an
// executor of its group, and that the claim clears its queue reason.
func TestClaimJobByGroup(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	us, eu := "us-"+uuid.NewString(), "eu-"+uuid.NewString()

	jobID := f.scheduleJob(t, us)

	none, err := f.storage.Jobs().ClaimNextScheduledJob(ctx, "worker-eu", eu)
	require.NoError(t, err)
	assert.Nil(t, none)

	job, err := f.storage.Jobs().ClaimNextScheduledJob(ctx, "worker-us", us)
	require.NoError(t, err)
	require.NotNil(t, job)
	assert.Equal(t, jobID, job.ID)
	assert.Equal(t, us, job.ExecutorGroup)
	assert.Nil(t, job.QueueReason)
}

// TestClaimTaskByGroup asserts that a check or discover task is claimed only
// by an executor of its group and a spec task by an executor of any group.
func TestClaimTaskByGroup(t *testing.T) {
	f := setup(t)
	ctx := context.Background()
	us, eu := "us-"+uuid.NewString(), "eu-"+uuid.NewString()

	checkID := f.pendingTask(t, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: uuid.New()}, &us)

	none, err := f.storage.ConnectorTasks().ClaimPendingTask(ctx, "worker-eu", eu)
	require.NoError(t, err)
	assert.Nil(t, none)

	task, err := f.storage.ConnectorTasks().ClaimPendingTask(ctx, "worker-us", us)
	require.NoError(t, err)
	require.NotNil(t, task)
	assert.Equal(t, checkID, task.ID)

	specID := f.pendingTask(t, pipelineservice.ConnectorTaskTypeSpec, &pipelineservice.SpecPayload{ManagedConnectorID: uuid.New()}, nil)

	spec, err := f.storage.ConnectorTasks().ClaimPendingTask(ctx, "worker-eu", eu)
	require.NoError(t, err)
	require.NotNil(t, spec)
	assert.Equal(t, specID, spec.ID)
	assert.Nil(t, spec.ExecutorGroup)
}

// TestMigrationBackfillsDefaultGroup asserts that rows written without a
// group carry the default group.
func TestMigrationBackfillsDefaultGroup(t *testing.T) {
	f := setup(t)

	var groups []string
	require.NoError(t, f.db.Raw(
		`SELECT s.executor_group FROM pipeline.connections c JOIN pipeline.sources s ON s.id = c.source_id WHERE c.id = ?
		 UNION ALL SELECT d.executor_group FROM pipeline.connections c JOIN pipeline.destinations d ON d.id = c.destination_id WHERE c.id = ?`,
		f.connID, f.connID).Scan(&groups).Error)
	assert.Equal(t, []string{"default", "default"}, groups)
}
