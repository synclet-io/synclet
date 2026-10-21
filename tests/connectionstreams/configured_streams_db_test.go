// Package connectionstreams_test runs the configured-stream queries of the
// connections storage against a real Postgres. It needs TEST_DATABASE_DSN.
package connectionstreams_test

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
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/modules/workspace/workspacedbstate"
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

func TestConfiguredStreamsQueries(t *testing.T) {
	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping configured streams database test")
	}

	ctx := context.Background()

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, workspacedbstate.MigrationsFS, "workspace_goose_db_version")
	testdb.Migrate(t, sqlDB, pipelinedbstate.MigrationsFS, "pipeline_goose_db_version")

	tx := db.Begin()

	t.Cleanup(func() { tx.Rollback() })

	workspaceID, connectorID, sourceID, destinationID := uuid.New(), uuid.New(), uuid.New(), uuid.New()
	exec := func(sql string, args ...any) {
		t.Helper()
		require.NoError(t, tx.Exec(sql, args...).Error)
	}

	exec(`INSERT INTO workspace.workspaces (id, name, slug) VALUES (?, 'ws', ?)`, workspaceID, "ws-"+workspaceID.String())
	exec(`INSERT INTO pipeline.managed_connectors (id, workspace_id, name, connector_type, docker_image, docker_tag)
	      VALUES (?, ?, 'pg', 'source', 'img', 'latest')`, connectorID, workspaceID)
	exec(`INSERT INTO pipeline.sources (id, workspace_id, managed_connector_id, name) VALUES (?, ?, ?, 'src')`, sourceID, workspaceID, connectorID)
	exec(`INSERT INTO pipeline.destinations (id, workspace_id, managed_connector_id, name) VALUES (?, ?, ?, 'dst')`, destinationID, workspaceID, connectorID)

	connections := map[string]uuid.UUID{}

	for _, name := range []string{"configured", "empty", "null", "missing"} {
		id := uuid.New()
		connections[name] = id
		exec(`INSERT INTO pipeline.connections (id, workspace_id, name, source_id, destination_id, status, schedule, next_scheduled_at)
		      VALUES (?, ?, ?, ?, ?, 'active', '* * * * *', NOW() - INTERVAL '1 minute')`,
			id, workspaceID, name, sourceID, destinationID)
	}

	exec(`INSERT INTO pipeline.configured_catalogs (id, connection_id, streams_json) VALUES (?, ?, '[{"stream":{"name":"users"}}]')`, uuid.New(), connections["configured"])
	exec(`INSERT INTO pipeline.configured_catalogs (id, connection_id, streams_json) VALUES (?, ?, '[]')`, uuid.New(), connections["empty"])
	exec(`INSERT INTO pipeline.configured_catalogs (id, connection_id, streams_json) VALUES (?, ?, 'null')`, uuid.New(), connections["null"])

	storage := pipelinestorage.NewStorages(tx, &logging.Logger{Delegate: noopLogDelegate{}}, nil).Connections()

	ids := []uuid.UUID{connections["configured"], connections["empty"], connections["null"], connections["missing"]}

	configured, err := storage.WithConfiguredStreams(ctx, ids)
	require.NoError(t, err)
	assert.Equal(t, map[uuid.UUID]bool{connections["configured"]: true}, configured)

	due, err := storage.FindDueConnections(ctx, 1000)
	require.NoError(t, err)

	dueIDs := make([]uuid.UUID, 0, len(due))
	for _, conn := range due {
		dueIDs = append(dueIDs, conn.ConnectionID)
	}

	assert.Contains(t, dueIDs, connections["configured"])
	assert.NotContains(t, dueIDs, connections["empty"], "a connection with an empty stream list is not scheduled")
	assert.NotContains(t, dueIDs, connections["null"])
	assert.NotContains(t, dueIDs, connections["missing"], "a connection without a configured catalog is not scheduled")
}
