// Package statsrollups_test reads stats rollups back from a real Postgres. It
// needs TEST_DATABASE_DSN.
package statsrollups_test

import (
	"context"
	"os"
	"testing"
	"time"

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

type attempt struct {
	connection int
	status     string
	age        time.Duration
	records    int64
	bytes      int64
	durationMs int64
}

type totals struct {
	syncs, succeeded, failed, records, bytes, durationMs int64
}

type fixture struct {
	db          *gorm.DB
	workspaceID uuid.UUID
	connIDs     []uuid.UUID
}

func setup(t *testing.T, connections int) fixture {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping stats rollup database test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, pipelinedbstate.MigrationsFS, "pipeline_goose_db_version")

	f := fixture{db: db, workspaceID: uuid.New()}
	connectorID, sourceID, destID := uuid.New(), uuid.New(), uuid.New()

	exec := func(sql string, args ...any) {
		t.Helper()
		require.NoError(t, db.Exec(sql, args...).Error)
	}

	exec(`INSERT INTO pipeline.managed_connectors (id, workspace_id, docker_image, docker_tag, name, connector_type) VALUES (?, ?, 'img', 'latest', 'c', 'source')`, connectorID, f.workspaceID)
	exec(`INSERT INTO pipeline.sources (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 's', ?)`, sourceID, f.workspaceID, connectorID)
	exec(`INSERT INTO pipeline.destinations (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 'd', ?)`, destID, f.workspaceID, connectorID)

	for range connections {
		connID := uuid.New()
		exec(`INSERT INTO pipeline.connections (id, workspace_id, name, source_id, destination_id) VALUES (?, ?, 'conn', ?, ?)`, connID, f.workspaceID, sourceID, destID)
		f.connIDs = append(f.connIDs, connID)
	}

	t.Cleanup(func() {
		require.NoError(t, db.Exec(`DELETE FROM pipeline.stats_rollups WHERE workspace_id = ?`, f.workspaceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.connections WHERE workspace_id = ?`, f.workspaceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.sources WHERE id = ?`, sourceID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.destinations WHERE id = ?`, destID).Error)
		require.NoError(t, db.Exec(`DELETE FROM pipeline.managed_connectors WHERE id = ?`, connectorID).Error)
	})

	return f
}

func (f fixture) insert(t *testing.T, now time.Time, attempts []attempt) {
	t.Helper()

	for _, a := range attempts {
		jobID := uuid.New()
		completedAt := now.Add(-a.age)
		require.NoError(t, f.db.Exec(
			`INSERT INTO pipeline.jobs (id, connection_id, status, job_type, completed_at, attempt) VALUES (?, ?, ?, 'sync', ?, 1)`,
			jobID, f.connIDs[a.connection], a.status, completedAt,
		).Error)
		require.NoError(t, f.db.Exec(
			`INSERT INTO pipeline.job_attempts (job_id, attempt_number, completed_at, sync_stats_json)
			VALUES (?, 1, ?, jsonb_build_object('records_read', ?::bigint, 'bytes_synced', ?::bigint, 'duration', ?::bigint))`,
			jobID, completedAt, a.records, a.bytes, a.durationMs*int64(time.Millisecond),
		).Error)
	}
}

// sum adds up the attempts of one connection, or of all of them when connection is negative.
func sum(attempts []attempt, connection int) totals {
	var s totals

	for _, a := range attempts {
		if connection >= 0 && a.connection != connection {
			continue
		}

		s.syncs++
		if a.status == "completed" {
			s.succeeded++
		} else {
			s.failed++
		}

		s.records += a.records
		s.bytes += a.bytes
		s.durationMs += a.durationMs
	}

	return s
}

func sumInts(values []int64) int64 {
	var s int64
	for _, v := range values {
		s += v
	}

	return s
}

// seedAttempts are seven runs with one failure over two connections, all
// completed between one and twenty hours ago, so every hourly and daily bucket
// they fall into starts inside both the 24h and the 7d window.
var seedAttempts = []attempt{
	{connection: 0, status: "completed", age: 1 * time.Hour, records: 100, bytes: 1000, durationMs: 1200},
	{connection: 0, status: "completed", age: 3 * time.Hour, records: 250, bytes: 2500, durationMs: 800},
	{connection: 0, status: "failed", age: 5 * time.Hour, records: 7, bytes: 70, durationMs: 300},
	{connection: 0, status: "completed", age: 9 * time.Hour, records: 500, bytes: 5000, durationMs: 2000},
	{connection: 1, status: "completed", age: 2 * time.Hour, records: 300, bytes: 3000, durationMs: 1500},
	{connection: 1, status: "completed", age: 12 * time.Hour, records: 150, bytes: 1500, durationMs: 900},
	{connection: 1, status: "completed", age: 20 * time.Hour, records: 50, bytes: 500, durationMs: 400},
}

func TestRollupQueriesEqualAttemptSums(t *testing.T) {
	ctx := context.Background()
	f := setup(t, 2)
	now := time.Now()
	f.insert(t, now, seedAttempts)

	stats := pipelinestorage.NewStatsStorage(f.db)
	require.NoError(t, stats.UpsertRollups(ctx, pipelineservice.BucketSizeHourly, now.Add(-30*24*time.Hour), "hour"))
	require.NoError(t, stats.UpsertRollups(ctx, pipelineservice.BucketSizeDaily, now.Add(-30*24*time.Hour), "day"))

	for _, tr := range []pipelineservice.TimeRange{pipelineservice.TimeRange24h, pipelineservice.TimeRange7d} {
		t.Run(string(tr), func(t *testing.T) {
			from := now.Add(-tr.Duration())
			size := tr.BucketSize()
			want := sum(seedAttempts, -1)

			ws, err := stats.QueryWorkspaceRollup(ctx, f.workspaceID, size, from, now)
			require.NoError(t, err)
			assert.Equal(t, pipelineservice.RollupTotals{
				SyncsTotal:     want.syncs,
				SyncsSucceeded: want.succeeded,
				SyncsFailed:    want.failed,
				RecordsRead:    want.records,
			}, ws, "workspace rollup")

			top, err := stats.QueryTopConnections(ctx, f.workspaceID, size, from, now, 5)
			require.NoError(t, err)

			topByConn := map[uuid.UUID]pipelineservice.TopConnectionRow{}
			for _, row := range top {
				topByConn[row.ConnectionID] = row
			}

			sparklines, err := stats.QueryConnectionSparklineBatch(ctx, f.connIDs, size, from, now, 1000)
			require.NoError(t, err)

			for i, connID := range f.connIDs {
				wantConn := sum(seedAttempts, i)

				conn, err := stats.QueryConnectionRollup(ctx, connID, size, from, now)
				require.NoError(t, err)
				assert.Equal(t, pipelineservice.ConnectionRollup{
					SyncsTotal:      wantConn.syncs,
					SyncsSucceeded:  wantConn.succeeded,
					SyncsFailed:     wantConn.failed,
					RecordsRead:     wantConn.records,
					TotalDurationMs: wantConn.durationMs,
				}, conn, "connection %d rollup", i)

				assert.Equal(t, wantConn.records, topByConn[connID].RecordsSynced, "connection %d top records", i)
				assert.Equal(t, wantConn.bytes, topByConn[connID].BytesSynced, "connection %d top bytes", i)

				chart, err := stats.QueryRecordsChart(ctx, connID, size, from, now)
				require.NoError(t, err)

				var chartRecords int64
				for _, row := range chart {
					chartRecords += row.RecordsRead
				}

				assert.Equal(t, wantConn.records, chartRecords, "connection %d records chart", i)

				sparkline, err := stats.QueryConnectionSparkline(ctx, connID, size, from, now, 1000)
				require.NoError(t, err)
				assert.Equal(t, wantConn.records, sumInts(sparkline), "connection %d sparkline", i)
				assert.Equal(t, wantConn.records, sumInts(sparklines[connID]), "connection %d sparkline batch", i)
			}
		})
	}
}

func TestTopConnectionsSkipDeletedConnections(t *testing.T) {
	ctx := context.Background()
	f := setup(t, 2)
	now := time.Now()
	f.insert(t, now, seedAttempts)

	stats := pipelinestorage.NewStatsStorage(f.db)
	require.NoError(t, stats.UpsertRollups(ctx, pipelineservice.BucketSizeHourly, now.Add(-30*24*time.Hour), "hour"))

	require.NoError(t, f.db.Exec(`DELETE FROM pipeline.connections WHERE id = ?`, f.connIDs[1]).Error)

	from := now.Add(-24 * time.Hour)

	top, err := stats.QueryTopConnections(ctx, f.workspaceID, pipelineservice.BucketSizeHourly, from, now, 5)
	require.NoError(t, err)
	require.Len(t, top, 1)
	assert.Equal(t, f.connIDs[0], top[0].ConnectionID)

	ws, err := stats.QueryWorkspaceRollup(ctx, f.workspaceID, pipelineservice.BucketSizeHourly, from, now)
	require.NoError(t, err)
	assert.Equal(t, sum(seedAttempts, -1).records, ws.RecordsRead, "the workspace totals keep the deleted connection's syncs")
}
