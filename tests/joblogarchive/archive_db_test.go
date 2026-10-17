// Package joblogarchive_test runs job log appends, archiving, reads and
// retention against a real Postgres. It needs TEST_DATABASE_DSN.
package joblogarchive_test

import (
	"context"
	"fmt"
	"os"
	"sync"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	gormlogger "gorm.io/gorm/logger"

	"github.com/synclet-io/synclet/modules/pipeline/pipelinedbstate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/pkg/logarchive"
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
	archive     *logarchive.Filesystem
	archiveDir  string
	workspaceID uuid.UUID
	connID      uuid.UUID
}

func setup(t *testing.T) *fixture {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping job log archive database test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, pipelinedbstate.MigrationsFS, "pipeline_goose_db_version")

	archiveDir := t.TempDir()
	archive, err := logarchive.NewFilesystem(archiveDir)
	require.NoError(t, err)

	f := &fixture{
		db:          db,
		storage:     pipelinestorage.NewStorages(db, &logging.Logger{Delegate: noopLogDelegate{}}, nil),
		archive:     archive,
		archiveDir:  archiveDir,
		workspaceID: uuid.New(),
		connID:      uuid.New(),
	}

	connectorID, sourceID, destID := uuid.New(), uuid.New(), uuid.New()
	for _, stmt := range []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO pipeline.managed_connectors (id, workspace_id, docker_image, docker_tag, name, connector_type) VALUES (?, ?, 'img', 'latest', 'c', 'source')`, []any{connectorID, f.workspaceID}},
		{`INSERT INTO pipeline.sources (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 's', ?)`, []any{sourceID, f.workspaceID, connectorID}},
		{`INSERT INTO pipeline.destinations (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 'd', ?)`, []any{destID, f.workspaceID, connectorID}},
		{`INSERT INTO pipeline.connections (id, workspace_id, name, source_id, destination_id) VALUES (?, ?, 'conn', ?, ?)`, []any{f.connID, f.workspaceID, sourceID, destID}},
	} {
		require.NoError(t, db.Exec(stmt.sql, stmt.args...).Error)
	}

	t.Cleanup(func() {
		require.NoError(t, db.Exec(`DELETE FROM pipeline.connections WHERE id = ?`, f.connID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.sources WHERE id = ?`, sourceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.destinations WHERE id = ?`, destID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.managed_connectors WHERE id = ?`, connectorID).Error)
	})

	return f
}

func (f *fixture) newJob(t *testing.T, status string) uuid.UUID {
	t.Helper()

	id := uuid.New()
	require.NoError(t, f.db.Exec(
		`INSERT INTO pipeline.jobs (id, connection_id, status, job_type, attempt) VALUES (?, ?, ?, 'sync', 1)`,
		id, f.connID, status,
	).Error)

	return id
}

func (f *fixture) setJob(t *testing.T, jobID uuid.UUID, status string, attempt int) {
	t.Helper()
	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET status = ?, attempt = ? WHERE id = ?`, status, attempt, jobID).Error)
}

func (f *fixture) appendLines(t *testing.T, maxLines int, jobID uuid.UUID, lines ...string) {
	t.Helper()

	uc := pipelinelogs.NewBatchAppendJobLogs(f.storage, pipelineservice.Config{MaxLogLinesPerAttempt: maxLines}, pipelinemetrics.NewMetricsCollector())
	require.NoError(t, uc.Execute(context.Background(), pipelinelogs.BatchAppendJobLogsParams{JobID: jobID, LogLines: lines}))
}

func (f *fixture) readAll(t *testing.T, jobID uuid.UUID, pageSize int) []string {
	t.Helper()

	get := pipelinelogs.NewGetJobLog(f.storage, f.archive)

	var (
		lines  []string
		cursor int64
	)

	for {
		page, err := get.Execute(context.Background(), pipelinelogs.GetJobLogParams{
			WorkspaceID: f.workspaceID, JobID: jobID, AfterID: cursor, Limit: pageSize,
		})
		require.NoError(t, err)

		lines = append(lines, page.Lines...)
		if !page.HasMore {
			return lines
		}

		cursor = page.LastID
	}
}

func (f *fixture) countRows(t *testing.T, jobID uuid.UUID) int {
	t.Helper()

	var n int
	require.NoError(t, f.db.Raw(`SELECT COUNT(*) FROM pipeline.job_logs WHERE job_id = ?`, jobID).Scan(&n).Error)

	return n
}

func TestJobLogArchive_EndToEnd(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	jobID := f.newJob(t, "running")
	f.appendLines(t, 3, jobID, "a1", "a2", "a3", "a4", "a5")
	f.appendLines(t, 3, jobID, "a6")
	f.setJob(t, jobID, "scheduled", 2)
	f.appendLines(t, 3, jobID, "b1")

	want := make([]string, 0, 6)
	want = append(want, "a1", "a2", "a3", "[synclet] log truncated: 3 further lines dropped", "b1")
	assert.Equal(t, want, f.readAll(t, jobID, 2))

	f.setJob(t, jobID, "completed", 2)

	archiver := pipelinelogs.NewArchiveJobLogs(f.storage, f.archive, pipelineservice.Config{LogArchiveEnabled: true}, pipelinemetrics.NewMetricsCollector())
	require.NoError(t, archiver.Execute(ctx, jobID))
	require.NoError(t, archiver.Execute(ctx, jobID))

	assert.Zero(t, f.countRows(t, jobID))
	assert.Equal(t, want, f.readAll(t, jobID, 2))
	assert.Equal(t, want, f.readAll(t, jobID, 100))

	f.appendLines(t, 3, jobID, "late")
	assert.Equal(t, append(want, "late"), f.readAll(t, jobID, 2))

	var keys []string

	require.NoError(t, f.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		var err error

		keys, err = tx.JobLogSegments().LockWorkspaceArchiveKeys(ctx, f.workspaceID)

		return err
	}))
	assert.ElementsMatch(t, []string{
		pipelineservice.LogArchiveKey(f.workspaceID, f.connID, jobID, 1),
		pipelineservice.LogArchiveKey(f.workspaceID, f.connID, jobID, 2),
	}, keys)
}

func TestJobLogArchive_ConcurrentAppendsRespectCap(t *testing.T) {
	f := setup(t)
	jobID := f.newJob(t, "running")

	var wg sync.WaitGroup

	for i := range 10 {
		wg.Go(func() {
			lines := make([]string, 10)
			for j := range lines {
				lines[j] = fmt.Sprintf("%d-%d", i, j)
			}

			f.appendLines(t, 50, jobID, lines...)
		})
	}

	wg.Wait()

	lines := f.readAll(t, jobID, 1000)
	require.Len(t, lines, 51)
	assert.Equal(t, "[synclet] log truncated: 50 further lines dropped", lines[len(lines)-1])
}

func TestJobLogArchive_RetentionStorage(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	old := f.newJob(t, "completed")
	f.appendLines(t, 0, old, "x")
	require.NoError(t, pipelinelogs.NewArchiveJobLogs(f.storage, f.archive, pipelineservice.Config{LogArchiveEnabled: true}, pipelinemetrics.NewMetricsCollector()).Execute(ctx, old))

	newer := f.newJob(t, "failed")
	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET created_at = NOW() - INTERVAL '1 hour' WHERE id = ?`, old).Error)
	f.newJob(t, "running")

	retention := pipelinestorage.NewJobRetentionStorage(f.db)

	ids, err := retention.ListExcessTerminalJobIDs(ctx, f.workspaceID, 1, 10)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{old}, ids)

	ids, err = retention.ListExcessTerminalJobIDs(ctx, f.workspaceID, 0, 10)
	require.NoError(t, err)
	assert.Equal(t, []uuid.UUID{newer, old}, ids)

	require.NoError(t, f.storage.Jobs().Delete(ctx, &pipelineservice.JobFilter{ID: filter.Equals(old)}))

	var segments int
	require.NoError(t, f.db.Raw(`SELECT COUNT(*) FROM pipeline.job_log_segments WHERE job_id = ?`, old).Scan(&segments).Error)
	assert.Zero(t, segments, "segments cascade with their job")
}
