package pipelineconnectors_test

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
)

// --- Mock storage infrastructure ---

// mockStorage implements a minimal pipelineservice.Storage for testing connector use cases.
type mockStorage struct {
	pipelineservice.Storage
	managedConnectors *mockManagedConnectorsStorage
	repoConnectors    *mockRepositoryConnectorsStorage
}

func newMockStorage() *mockStorage {
	return &mockStorage{
		managedConnectors: &mockManagedConnectorsStorage{
			connectors: make(map[uuid.UUID]*pipelineservice.ManagedConnector),
		},
		repoConnectors: &mockRepositoryConnectorsStorage{
			connectors: make(map[uuid.UUID]*pipelineservice.RepositoryConnector),
		},
	}
}

func (m *mockStorage) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return m.managedConnectors
}

func (m *mockStorage) RepositoryConnectors() pipelineservice.RepositoryConnectorsStorage {
	return m.repoConnectors
}

// mockManagedConnectorsStorage provides in-memory managed connector storage.
type mockManagedConnectorsStorage struct {
	pipelineservice.ManagedConnectorsStorage
	connectors map[uuid.UUID]*pipelineservice.ManagedConnector
}

func (m *mockManagedConnectorsStorage) First(_ context.Context, f *pipelineservice.ManagedConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ManagedConnector, error) {
	for _, mc := range m.connectors {
		if matchManagedConnectorFilter(mc, f) {
			cp := mc.Copy()

			return &cp, nil
		}
	}

	return nil, pipelineservice.ErrManagedConnectorNotFound
}

func (m *mockManagedConnectorsStorage) Update(_ context.Context, mc *pipelineservice.ManagedConnector) (*pipelineservice.ManagedConnector, error) {
	if _, ok := m.connectors[mc.ID]; !ok {
		return nil, pipelineservice.ErrManagedConnectorNotFound
	}

	cp := mc.Copy()
	m.connectors[mc.ID] = &cp

	return &cp, nil
}

func (m *mockManagedConnectorsStorage) Find(_ context.Context, f *pipelineservice.ManagedConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.ManagedConnector, error) {
	var result []*pipelineservice.ManagedConnector

	for _, mc := range m.connectors {
		if matchManagedConnectorFilter(mc, f) {
			cp := mc.Copy()
			result = append(result, &cp)
		}
	}

	return result, nil
}

func matchManagedConnectorFilter(connector *pipelineservice.ManagedConnector, connFilter *pipelineservice.ManagedConnectorFilter) bool {
	if connFilter.ID != nil {
		switch filterType := connFilter.ID.(type) {
		case *filter.EqualsFilter[uuid.UUID]:
			if connector.ID != filterType.Value {
				return false
			}
		case *filter.InFilter[uuid.UUID]:
			found := false

			for _, v := range filterType.Values {
				if connector.ID == v {
					found = true

					break
				}
			}

			if !found {
				return false
			}
		}
	}

	if connFilter.WorkspaceID != nil {
		if connector.WorkspaceID != connFilter.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]).Value {
			return false
		}
	}

	return true
}

// mockRepositoryConnectorsStorage provides in-memory repository connector storage.
type mockRepositoryConnectorsStorage struct {
	pipelineservice.RepositoryConnectorsStorage
	connectors map[uuid.UUID]*pipelineservice.RepositoryConnector
}

func (m *mockRepositoryConnectorsStorage) First(_ context.Context, f *pipelineservice.RepositoryConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.RepositoryConnector, error) {
	for _, rc := range m.connectors {
		if matchRepoConnectorFilter(rc, f) {
			cp := rc.Copy()

			return &cp, nil
		}
	}

	return nil, pipelineservice.ErrRepositoryConnectorNotFound
}

func (m *mockRepositoryConnectorsStorage) Find(_ context.Context, f *pipelineservice.RepositoryConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.RepositoryConnector, error) {
	var result []*pipelineservice.RepositoryConnector

	for _, rc := range m.connectors {
		if matchRepoConnectorFilter(rc, f) {
			cp := rc.Copy()
			result = append(result, &cp)
		}
	}

	return result, nil
}

func matchRepoConnectorFilter(repoConnector *pipelineservice.RepositoryConnector, connFilter *pipelineservice.RepositoryConnectorFilter) bool {
	if connFilter.RepositoryID != nil {
		switch filterType := connFilter.RepositoryID.(type) {
		case *filter.EqualsFilter[uuid.UUID]:
			if repoConnector.RepositoryID != filterType.Value {
				return false
			}
		case *filter.InFilter[uuid.UUID]:
			found := false

			for _, v := range filterType.Values {
				if repoConnector.RepositoryID == v {
					found = true

					break
				}
			}

			if !found {
				return false
			}
		}
	}

	if connFilter.DockerRepository != nil {
		if repoConnector.DockerRepository != connFilter.DockerRepository.(*filter.EqualsFilter[string]).Value {
			return false
		}
	}

	return true
}

// --- Tests ---

func TestUpdateManagedConnector_Success(t *testing.T) {
	store := newMockStorage()
	useCase := pipelineconnectors.NewUpdateManagedConnector(store)

	workspaceID := uuid.New()
	repoID := uuid.New()
	connectorID := uuid.New()

	// Set up managed connector with old version.
	store.managedConnectors.connectors[connectorID] = &pipelineservice.ManagedConnector{
		ID:            connectorID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "0.1.0",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         `{"old":"spec"}`,
		RepositoryID: &repoID,
	}

	// Set up repo connector with newer version.
	rcID := uuid.New()
	store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{
		ID:               rcID,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-postgres",
		DockerImageTag:   "0.2.0",
		Name:             "Postgres",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
		Spec:             `{"new":"spec"}`,
	}

	result, err := useCase.Execute(context.Background(), pipelineconnectors.UpdateManagedConnectorParams{
		ConnectorID: connectorID,
		WorkspaceID: workspaceID,
	})

	require.NoError(t, err)
	assert.Equal(t, "0.2.0", result.DockerTag)
	assert.JSONEq(t, `{"new":"spec"}`, result.Spec)
}

func TestUpdateManagedConnector_NoRepository(t *testing.T) {
	store := newMockStorage()
	useCase := pipelineconnectors.NewUpdateManagedConnector(store)

	workspaceID := uuid.New()
	connectorID := uuid.New()

	// Set up managed connector without repository link.
	store.managedConnectors.connectors[connectorID] = &pipelineservice.ManagedConnector{
		ID:            connectorID,
		WorkspaceID:   workspaceID,
		DockerImage:   "custom/connector",
		DockerTag:     "latest",
		Name:          "Custom",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         "{}",
		RepositoryID: nil,
	}

	_, err := useCase.Execute(context.Background(), pipelineconnectors.UpdateManagedConnectorParams{
		ConnectorID: connectorID,
		WorkspaceID: workspaceID,
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "not linked to a repository")
}

func TestUpdateManagedConnector_AlreadyUpToDate(t *testing.T) {
	store := newMockStorage()
	useCase := pipelineconnectors.NewUpdateManagedConnector(store)

	workspaceID := uuid.New()
	repoID := uuid.New()
	connectorID := uuid.New()

	// Set up managed connector with same version as repo.
	store.managedConnectors.connectors[connectorID] = &pipelineservice.ManagedConnector{
		ID:            connectorID,
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "0.2.0",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,

		Spec:         `{"old":"spec"}`,
		RepositoryID: &repoID,
	}

	rcID := uuid.New()
	store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{
		ID:               rcID,
		RepositoryID:     repoID,
		DockerRepository: "airbyte/source-postgres",
		DockerImageTag:   "0.2.0",
		Name:             "Postgres",
		ConnectorType:    pipelineservice.ConnectorTypeSource,
		Spec:             `{"same":"spec"}`,
	}

	// UpdateManagedConnector still updates even if same tag (no skip -- that's batch logic).
	result, err := useCase.Execute(context.Background(), pipelineconnectors.UpdateManagedConnectorParams{
		ConnectorID: connectorID,
		WorkspaceID: workspaceID,
	})

	require.NoError(t, err)
	assert.Equal(t, "0.2.0", result.DockerTag)
	assert.JSONEq(t, `{"same":"spec"}`, result.Spec)
}
