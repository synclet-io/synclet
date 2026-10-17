// Package streamgenerations_test runs stream generation bookkeeping against a
// real Postgres. It needs TEST_DATABASE_DSN.
package streamgenerations_test

import (
	"context"
	"os"
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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/pkg/protocol"
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

func setup(t *testing.T) (*pipelinestorage.Storages, uuid.UUID) {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping stream generation database test")
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

	return pipelinestorage.NewStorages(db, &logging.Logger{Delegate: noopLogDelegate{}}, nil), connID
}

func catalog() *protocol.ConfiguredAirbyteCatalog {
	return &protocol.ConfiguredAirbyteCatalog{
		Streams: []protocol.ConfiguredAirbyteStream{
			{
				Stream:              protocol.AirbyteStream{Name: "items"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
			},
			{
				Stream:              protocol.AirbyteStream{Name: "users", Namespace: "public"},
				SyncMode:            protocol.SyncModeFullRefresh,
				DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
			},
			{
				Stream:              protocol.AirbyteStream{Name: "events"},
				SyncMode:            protocol.SyncModeIncremental,
				DestinationSyncMode: protocol.DestinationSyncModeAppend,
			},
		},
	}
}

func TestPopulateGenerationIDs_RepeatedSyncs(t *testing.T) {
	storage, connID := setup(t)
	ctx := context.Background()
	uc := pipelinecatalog.NewPopulateGenerationIDs(storage)

	for syncID := int64(1); syncID <= 3; syncID++ {
		cat := catalog()
		require.NoError(t, uc.Execute(ctx, pipelinecatalog.PopulateGenerationIDsParams{
			ConnectionID: connID,
			Catalog:      cat,
			SyncID:       syncID,
		}), "sync %d", syncID)

		assert.Equal(t, syncID, cat.Streams[0].GenerationID, "empty-namespace full refresh, sync %d", syncID)
		assert.Equal(t, syncID, cat.Streams[1].GenerationID, "namespaced full refresh, sync %d", syncID)
		assert.Equal(t, int64(0), cat.Streams[2].GenerationID, "incremental, sync %d", syncID)
	}

	stored, err := storage.StreamGenerations().Find(ctx, &pipelineservice.StreamGenerationFilter{
		ConnectionID: filter.Equals(connID),
	})
	require.NoError(t, err)

	got := map[string]int64{}
	for _, sg := range stored {
		got[sg.StreamNamespace+"."+sg.StreamName] = sg.GenerationID
	}

	assert.Equal(t, map[string]int64{".items": 3, "public.users": 3, ".events": 0}, got)
}
