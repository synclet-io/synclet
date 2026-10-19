// Package connectorstars_test runs the managed connector list and the star
// use case against a real Postgres. It needs TEST_DATABASE_DSN.
package connectorstars_test

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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
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

type auditLog struct{ events []pipelineservice.AuditEvent }

func (a *auditLog) Record(_ context.Context, event pipelineservice.AuditEvent) {
	a.events = append(a.events, event)
}

type fixture struct {
	db      *gorm.DB
	storage *pipelinestorage.Storages
}

func setup(t *testing.T) *fixture {
	t.Helper()

	dsn := os.Getenv("TEST_DATABASE_DSN")
	if dsn == "" {
		t.Skip("TEST_DATABASE_DSN not set, skipping connector stars database test")
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{Logger: gormlogger.Discard})
	require.NoError(t, err)

	sqlDB, err := db.DB()
	require.NoError(t, err)
	t.Cleanup(func() { require.NoError(t, sqlDB.Close()) })

	testdb.Migrate(t, sqlDB, pipelinedbstate.MigrationsFS, "pipeline_goose_db_version")

	return &fixture{db: db, storage: pipelinestorage.NewStorages(db, &logging.Logger{Delegate: noopLogDelegate{}}, nil)}
}

func (f *fixture) exec(t *testing.T, sql string, args ...any) {
	t.Helper()
	require.NoError(t, f.db.Exec(sql, args...).Error)
}

func (f *fixture) connector(t *testing.T, workspaceID uuid.UUID, connectorType string) uuid.UUID {
	t.Helper()

	id := uuid.New()
	f.exec(t, `INSERT INTO pipeline.managed_connectors (id, workspace_id, docker_image, docker_tag, name, connector_type, spec) VALUES (?, ?, ?, 'latest', 'c', ?, '{}')`,
		id, workspaceID, "img/"+id.String(), connectorType)
	t.Cleanup(func() { f.exec(t, `DELETE FROM pipeline.managed_connectors WHERE id = ?`, id) })

	return id
}

func (f *fixture) source(t *testing.T, workspaceID, connectorID uuid.UUID) {
	t.Helper()

	id := uuid.New()
	f.exec(t, `INSERT INTO pipeline.sources (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 's', ?)`, id, workspaceID, connectorID)
	t.Cleanup(func() { f.exec(t, `DELETE FROM pipeline.sources WHERE id = ?`, id) })
}

func (f *fixture) destination(t *testing.T, workspaceID, connectorID uuid.UUID) {
	t.Helper()

	id := uuid.New()
	f.exec(t, `INSERT INTO pipeline.destinations (id, workspace_id, name, managed_connector_id) VALUES (?, ?, 'd', ?)`, id, workspaceID, connectorID)
	t.Cleanup(func() { f.exec(t, `DELETE FROM pipeline.destinations WHERE id = ?`, id) })
}

type flags struct{ starred, used bool }

func (f *fixture) list(t *testing.T, workspaceID uuid.UUID) map[uuid.UUID]flags {
	t.Helper()

	items, err := pipelineconnectors.NewListConnectorsWithUpdateInfo(f.storage).Execute(context.Background(),
		pipelineconnectors.ListConnectorsWithUpdateInfoParams{WorkspaceID: workspaceID})
	require.NoError(t, err)

	result := make(map[uuid.UUID]flags, len(items))
	for _, item := range items {
		result[item.Connector.ID] = flags{starred: item.Connector.Starred, used: item.Used}
	}

	return result
}

// TestListMarksUsedConnectors asserts that a managed connector is used when a
// source or a destination of the same workspace is built on it, and only then.
func TestListMarksUsedConnectors(t *testing.T) {
	f := setup(t)
	workspaceID, otherWorkspaceID := uuid.New(), uuid.New()

	bySource := f.connector(t, workspaceID, "source")
	byDestination := f.connector(t, workspaceID, "destination")
	unused := f.connector(t, workspaceID, "source")
	usedElsewhere := f.connector(t, workspaceID, "source")

	f.source(t, workspaceID, bySource)
	f.source(t, workspaceID, bySource)
	f.destination(t, workspaceID, byDestination)
	f.source(t, otherWorkspaceID, usedElsewhere)

	got := f.list(t, workspaceID)
	require.Len(t, got, 4)
	assert.Equal(t, flags{used: true}, got[bySource])
	assert.Equal(t, flags{used: true}, got[byDestination])
	assert.Equal(t, flags{}, got[unused])
	assert.Equal(t, flags{}, got[usedElsewhere])

	detail, err := pipelineconnectors.NewGetConnectorWithUpdateInfo(f.storage).Execute(context.Background(),
		pipelineconnectors.GetConnectorWithUpdateInfoParams{ID: byDestination, WorkspaceID: workspaceID})
	require.NoError(t, err)
	assert.True(t, detail.Used)
}

// TestStarIsWorkspaceWide asserts that starring flips one flag on the
// workspace's connector, which every list of that workspace returns, and that
// another workspace cannot star it.
func TestStarIsWorkspaceWide(t *testing.T) {
	f := setup(t)
	workspaceID, otherWorkspaceID := uuid.New(), uuid.New()
	starredID := f.connector(t, workspaceID, "source")
	plainID := f.connector(t, workspaceID, "destination")

	audit := &auditLog{}
	setStarred := pipelineconnectors.NewSetConnectorStarred(f.storage, audit)
	ctx := context.Background()

	_, err := setStarred.Execute(ctx, pipelineconnectors.SetConnectorStarredParams{
		ConnectorID: starredID, WorkspaceID: workspaceID, Starred: true,
	})
	require.NoError(t, err)

	got := f.list(t, workspaceID)
	assert.Equal(t, flags{starred: true}, got[starredID])
	assert.Equal(t, flags{}, got[plainID])

	_, err = setStarred.Execute(ctx, pipelineconnectors.SetConnectorStarredParams{
		ConnectorID: starredID, WorkspaceID: otherWorkspaceID, Starred: false,
	})
	require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
	assert.True(t, f.list(t, workspaceID)[starredID].starred)

	_, err = setStarred.Execute(ctx, pipelineconnectors.SetConnectorStarredParams{
		ConnectorID: starredID, WorkspaceID: workspaceID, Starred: false,
	})
	require.NoError(t, err)
	assert.Equal(t, flags{}, f.list(t, workspaceID)[starredID])

	require.Len(t, audit.events, 2)
	assert.Equal(t, "star", audit.events[0].Action)
	assert.Equal(t, "unstar", audit.events[1].Action)
	assert.Equal(t, starredID, audit.events[1].ResourceID)
	assert.Equal(t, "connector", audit.events[1].ResourceType)
}

// TestUsedManagedConnectorIDs asserts that the storage returns, once each, the
// asked IDs that a source or destination of the workspace is built on.
func TestUsedManagedConnectorIDs(t *testing.T) {
	f := setup(t)
	workspaceID, otherWorkspaceID := uuid.New(), uuid.New()

	bySource := f.connector(t, workspaceID, "source")
	byBoth := f.connector(t, workspaceID, "destination")
	unused := f.connector(t, workspaceID, "source")
	usedElsewhere := f.connector(t, workspaceID, "source")
	notAsked := f.connector(t, workspaceID, "source")

	f.source(t, workspaceID, bySource)
	f.source(t, workspaceID, bySource)
	f.source(t, workspaceID, byBoth)
	f.destination(t, workspaceID, byBoth)
	f.source(t, otherWorkspaceID, usedElsewhere)
	f.source(t, workspaceID, notAsked)

	got, err := f.storage.ManagedConnectors().UsedManagedConnectorIDs(context.Background(), workspaceID,
		[]uuid.UUID{bySource, byBoth, unused, usedElsewhere})
	require.NoError(t, err)
	assert.ElementsMatch(t, []uuid.UUID{bySource, byBoth}, got)
}

// TestUsedManagedConnectorIDsSkipsQueryForNoIDs asserts that an empty ID list
// returns nothing without reaching the database: a cancelled context would
// fail any query.
func TestUsedManagedConnectorIDsSkipsQueryForNoIDs(t *testing.T) {
	f := setup(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	got, err := f.storage.ManagedConnectors().UsedManagedConnectorIDs(ctx, uuid.New(), nil)
	require.NoError(t, err)
	assert.Empty(t, got)
}
