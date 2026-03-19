package pipelinesettings_test

import (
	"context"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/idempotency"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// mockStorage is a minimal mock of pipelineservice.Storage for settings tests.
type mockStorage struct {
	workspaceSettings *mockWorkspaceSettingssStorage
}

func (m *mockStorage) ManagedConnectors() pipelineservice.ManagedConnectorsStorage { return nil }
func (m *mockStorage) Repositorys() pipelineservice.RepositorysStorage             { return nil }
func (m *mockStorage) RepositoryConnectors() pipelineservice.RepositoryConnectorsStorage {
	return nil
}
func (m *mockStorage) Sources() pipelineservice.SourcesStorage           { return nil }
func (m *mockStorage) Destinations() pipelineservice.DestinationsStorage { return nil }
func (m *mockStorage) Connections() pipelineservice.ConnectionsStorage   { return nil }
func (m *mockStorage) Jobs() pipelineservice.JobsStorage                 { return nil }
func (m *mockStorage) JobAttempts() pipelineservice.JobAttemptsStorage   { return nil }
func (m *mockStorage) CatalogDiscoverys() pipelineservice.CatalogDiscoverysStorage {
	return nil
}
func (m *mockStorage) ConfiguredCatalogs() pipelineservice.ConfiguredCatalogsStorage {
	return nil
}
func (m *mockStorage) JobLogs() pipelineservice.JobLogsStorage               { return nil }
func (m *mockStorage) ConnectionStates() pipelineservice.ConnectionStatesStorage { return nil }
func (m *mockStorage) WorkspaceSettingss() pipelineservice.WorkspaceSettingssStorage {
	return m.workspaceSettings
}
func (m *mockStorage) ConnectorTasks() pipelineservice.ConnectorTasksStorage   { return nil }
func (m *mockStorage) StreamGenerations() pipelineservice.StreamGenerationsStorage { return nil }
func (m *mockStorage) IdempotencyKeys() idempotency.Storage                 { return nil }
func (m *mockStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, m)
}
func (m *mockStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error { return nil }

// mockWorkspaceSettingssStorage implements pipelineservice.WorkspaceSettingssStorage (dbutil.EntityStorage).
type mockWorkspaceSettingssStorage struct {
	dbutil.EntityStorage[pipelineservice.WorkspaceSettings, pipelineservice.WorkspaceSettingsFilter]
	firstResult *pipelineservice.WorkspaceSettings
	firstErr    error
}

func (m *mockWorkspaceSettingssStorage) First(_ context.Context, _ *pipelineservice.WorkspaceSettingsFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.WorkspaceSettings, error) {
	return m.firstResult, m.firstErr
}
