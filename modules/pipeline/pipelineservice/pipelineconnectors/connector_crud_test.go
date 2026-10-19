package pipelineconnectors_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

var errStorage = errors.New("storage failure")

func (m *mockManagedConnectorsStorage) Create(_ context.Context, connector *pipelineservice.ManagedConnector) (*pipelineservice.ManagedConnector, error) {
	cp := connector.Copy()
	m.connectors[connector.ID] = &cp

	out := cp.Copy()

	return &out, nil
}

func (m *mockManagedConnectorsStorage) Delete(_ context.Context, f *pipelineservice.ManagedConnectorFilter) error {
	for id, mc := range m.connectors {
		if matchManagedConnectorFilter(mc, f) {
			delete(m.connectors, id)
		}
	}

	return nil
}

// faultyManagedConnectors wraps the in-memory managed connector storage with per-method errors.
type faultyManagedConnectors struct {
	*mockManagedConnectorsStorage
	firstErr  error
	findErr   error
	createErr error
	updateErr error
	deleteErr error
	deletes   int
}

func (f *faultyManagedConnectors) First(ctx context.Context, flt *pipelineservice.ManagedConnectorFilter, opts ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ManagedConnector, error) {
	if f.firstErr != nil {
		return nil, f.firstErr
	}

	return f.mockManagedConnectorsStorage.First(ctx, flt, opts...)
}

func (f *faultyManagedConnectors) Find(ctx context.Context, flt *pipelineservice.ManagedConnectorFilter, opts ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.ManagedConnector, error) {
	if f.findErr != nil {
		return nil, f.findErr
	}

	return f.mockManagedConnectorsStorage.Find(ctx, flt, opts...)
}

func (f *faultyManagedConnectors) Create(ctx context.Context, connector *pipelineservice.ManagedConnector) (*pipelineservice.ManagedConnector, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	return f.mockManagedConnectorsStorage.Create(ctx, connector)
}

func (f *faultyManagedConnectors) Update(ctx context.Context, connector *pipelineservice.ManagedConnector) (*pipelineservice.ManagedConnector, error) {
	if f.updateErr != nil {
		return nil, f.updateErr
	}

	return f.mockManagedConnectorsStorage.Update(ctx, connector)
}

func (f *faultyManagedConnectors) Delete(ctx context.Context, flt *pipelineservice.ManagedConnectorFilter) error {
	f.deletes++

	if f.deleteErr != nil {
		return f.deleteErr
	}

	return f.mockManagedConnectorsStorage.Delete(ctx, flt)
}

type fakeSources struct {
	pipelineservice.SourcesStorage
	count int
	err   error
}

func (f *fakeSources) Count(_ context.Context, _ *pipelineservice.SourceFilter) (int, error) {
	return f.count, f.err
}

func (f *fakeSources) Find(_ context.Context, _ *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Source, error) {
	return nil, f.err
}

type fakeDestinations struct {
	pipelineservice.DestinationsStorage
	count int
	err   error
}

func (f *fakeDestinations) Count(_ context.Context, _ *pipelineservice.DestinationFilter) (int, error) {
	return f.count, f.err
}

func (f *fakeDestinations) Find(_ context.Context, _ *pipelineservice.DestinationFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	return nil, f.err
}

// fakeConnectorTasks stores tasks in memory; complete, when set, moves a task to its final state on creation.
type fakeConnectorTasks struct {
	pipelineservice.ConnectorTasksStorage
	tasks     map[uuid.UUID]*pipelineservice.ConnectorTask
	createErr error
	complete  func(task *pipelineservice.ConnectorTask)
}

func (f *fakeConnectorTasks) Create(_ context.Context, task *pipelineservice.ConnectorTask) (*pipelineservice.ConnectorTask, error) {
	if f.createErr != nil {
		return nil, f.createErr
	}

	if f.complete != nil {
		f.complete(task)
	}

	f.tasks[task.ID] = task

	return task, nil
}

func (f *fakeConnectorTasks) First(_ context.Context, flt *pipelineservice.ConnectorTaskFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConnectorTask, error) {
	task, ok := f.tasks[flt.ID.(*filter.EqualsFilter[uuid.UUID]).Value]
	if !ok {
		return nil, pipelineservice.ErrConnectorTaskNotFound
	}

	return task, nil
}

type connectorStorage struct {
	*mockStorage
	managed      *faultyManagedConnectors
	sources      *fakeSources
	destinations *fakeDestinations
	tasks        *fakeConnectorTasks
}

func newConnectorStorage() *connectorStorage {
	base := newMockStorage()

	return &connectorStorage{
		mockStorage:  base,
		managed:      &faultyManagedConnectors{mockManagedConnectorsStorage: base.managedConnectors},
		sources:      &fakeSources{},
		destinations: &fakeDestinations{},
		tasks:        &fakeConnectorTasks{tasks: make(map[uuid.UUID]*pipelineservice.ConnectorTask)},
	}
}

func (s *connectorStorage) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return s.managed
}

func (s *connectorStorage) Sources() pipelineservice.SourcesStorage {
	return s.sources
}

func (s *connectorStorage) Destinations() pipelineservice.DestinationsStorage {
	return s.destinations
}

func (s *connectorStorage) ConnectorTasks() pipelineservice.ConnectorTasksStorage {
	return s.tasks
}

func (s *connectorStorage) addConnector(workspaceID uuid.UUID, connectorType pipelineservice.ConnectorType) *pipelineservice.ManagedConnector {
	mc := &pipelineservice.ManagedConnector{
		ID:            uuid.New(),
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "1.0.0",
		Name:          "Postgres",
		ConnectorType: connectorType,
		Spec:          `{"connectionSpecification":{}}`,
	}
	s.managedConnectors.connectors[mc.ID] = mc

	return mc
}

type recordingAudit struct {
	events []pipelineservice.AuditEvent
}

func (r *recordingAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	r.events = append(r.events, event)
}

// --- GetConnector ---

func TestGetConnector(t *testing.T) {
	store := newConnectorStorage()
	workspaceID := uuid.New()
	mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
	useCase := pipelineconnectors.NewGetConnector(store)

	got, err := useCase.Execute(context.Background(), pipelineconnectors.GetConnectorParams{ID: mc.ID, WorkspaceID: workspaceID})
	require.NoError(t, err)
	assert.Equal(t, mc.ID, got.ID)

	_, err = useCase.Execute(context.Background(), pipelineconnectors.GetConnectorParams{ID: mc.ID, WorkspaceID: uuid.New()})
	require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
}

// --- ListConnectors ---

func TestListConnectors(t *testing.T) {
	store := newConnectorStorage()
	workspaceID := uuid.New()
	store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
	store.addConnector(workspaceID, pipelineservice.ConnectorTypeDestination)
	store.addConnector(uuid.New(), pipelineservice.ConnectorTypeSource)
	useCase := pipelineconnectors.NewListConnectors(store)

	got, err := useCase.Execute(context.Background(), pipelineconnectors.ListConnectorsParams{WorkspaceID: workspaceID})
	require.NoError(t, err)
	assert.Len(t, got, 2)

	store.managed.findErr = errStorage

	_, err = useCase.Execute(context.Background(), pipelineconnectors.ListConnectorsParams{WorkspaceID: workspaceID})
	require.ErrorIs(t, err, errStorage)
}

// --- GetConnectorSpec ---

func TestGetConnectorSpec(t *testing.T) {
	workspaceID := uuid.New()

	t.Run("not found", func(t *testing.T) {
		store := newConnectorStorage()
		useCase := pipelineconnectors.NewGetConnectorSpec(store)

		_, err := useCase.Execute(context.Background(), pipelineconnectors.GetConnectorSpecParams{ID: uuid.New(), WorkspaceID: workspaceID})
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
	})

	for _, spec := range []string{"", "{}"} {
		t.Run("no cached spec "+spec, func(t *testing.T) {
			store := newConnectorStorage()
			mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
			mc.Spec = spec
			useCase := pipelineconnectors.NewGetConnectorSpec(store)

			_, err := useCase.Execute(context.Background(), pipelineconnectors.GetConnectorSpecParams{ID: mc.ID, WorkspaceID: workspaceID})
			require.ErrorIs(t, err, pipelineservice.ErrNoCachedSpec)
		})
	}

	t.Run("custom connector has no docs", func(t *testing.T) {
		store := newConnectorStorage()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		useCase := pipelineconnectors.NewGetConnectorSpec(store)

		got, err := useCase.Execute(context.Background(), pipelineconnectors.GetConnectorSpecParams{ID: mc.ID, WorkspaceID: workspaceID})
		require.NoError(t, err)
		assert.Equal(t, mc.Spec, got.Spec)
		assert.Empty(t, got.ExternalDocumentationURLs)
	})

	t.Run("repository connector docs", func(t *testing.T) {
		store := newConnectorStorage()
		repoID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		mc.RepositoryID = &repoID
		docs := []pipelineservice.ExternalDocumentationURL{{Title: "API", Type: "api_reference", URL: "https://example.com/api"}}
		rcID := uuid.New()
		store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{
			ID:               rcID,
			RepositoryID:     repoID,
			DockerRepository: mc.DockerImage,
			Metadata:         pipelineservice.RepositoryConnectorMetadata{ExternalDocumentationURLs: docs},
		}
		useCase := pipelineconnectors.NewGetConnectorSpec(store)

		got, err := useCase.Execute(context.Background(), pipelineconnectors.GetConnectorSpecParams{ID: mc.ID, WorkspaceID: workspaceID})
		require.NoError(t, err)
		assert.Equal(t, docs, got.ExternalDocumentationURLs)
	})

	t.Run("missing repository connector is ignored", func(t *testing.T) {
		store := newConnectorStorage()
		repoID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		mc.RepositoryID = &repoID
		useCase := pipelineconnectors.NewGetConnectorSpec(store)

		got, err := useCase.Execute(context.Background(), pipelineconnectors.GetConnectorSpecParams{ID: mc.ID, WorkspaceID: workspaceID})
		require.NoError(t, err)
		assert.Equal(t, mc.Spec, got.Spec)
		assert.Empty(t, got.ExternalDocumentationURLs)
	})
}

// --- DeleteConnector ---

func TestDeleteConnector_Success(t *testing.T) {
	for _, connectorType := range []pipelineservice.ConnectorType{pipelineservice.ConnectorTypeSource, pipelineservice.ConnectorTypeDestination} {
		t.Run(connectorType.String(), func(t *testing.T) {
			store := newConnectorStorage()
			audit := &recordingAudit{}
			workspaceID := uuid.New()
			mc := store.addConnector(workspaceID, connectorType)
			useCase := pipelineconnectors.NewDeleteConnector(store, audit)

			err := useCase.Execute(context.Background(), pipelineconnectors.DeleteConnectorParams{ID: mc.ID, WorkspaceID: workspaceID})
			require.NoError(t, err)
			assert.NotContains(t, store.managedConnectors.connectors, mc.ID)
			require.Len(t, audit.events, 1)
			assert.Equal(t, "delete", audit.events[0].Action)
			assert.Equal(t, mc.ID, audit.events[0].ResourceID)
			assert.Equal(t, "Postgres", auditField(audit.events[0].Before, "name"))
		})
	}
}

func TestDeleteConnector_InUse(t *testing.T) {
	tests := []struct {
		name          string
		connectorType pipelineservice.ConnectorType
		setup         func(*connectorStorage)
		wantMessage   string
	}{
		{
			name:          "source",
			connectorType: pipelineservice.ConnectorTypeSource,
			setup:         func(s *connectorStorage) { s.sources.count = 2 },
			wantMessage:   "cannot delete connector: used by 2 source(s)",
		},
		{
			name:          "destination",
			connectorType: pipelineservice.ConnectorTypeDestination,
			setup:         func(s *connectorStorage) { s.destinations.count = 1 },
			wantMessage:   "cannot delete connector: used by 1 destination(s)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newConnectorStorage()
			audit := &recordingAudit{}
			workspaceID := uuid.New()
			mc := store.addConnector(workspaceID, tt.connectorType)
			tt.setup(store)
			useCase := pipelineconnectors.NewDeleteConnector(store, audit)

			err := useCase.Execute(context.Background(), pipelineconnectors.DeleteConnectorParams{ID: mc.ID, WorkspaceID: workspaceID})

			var validationErr *pipelineservice.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, tt.wantMessage, validationErr.Message)
			assert.Contains(t, store.managedConnectors.connectors, mc.ID)
			assert.Empty(t, audit.events)
		})
	}
}

func TestDeleteConnector_Errors(t *testing.T) {
	tests := []struct {
		name          string
		connectorType pipelineservice.ConnectorType
		setup         func(*connectorStorage)
		wantMessage   string
	}{
		{
			name:          "lookup fails",
			connectorType: pipelineservice.ConnectorTypeSource,
			setup:         func(s *connectorStorage) { s.managed.firstErr = errStorage },
			wantMessage:   "getting managed connector",
		},
		{
			name:          "counting sources fails",
			connectorType: pipelineservice.ConnectorTypeSource,
			setup:         func(s *connectorStorage) { s.sources.err = errStorage },
			wantMessage:   "counting dependent resources",
		},
		{
			name:          "counting destinations fails",
			connectorType: pipelineservice.ConnectorTypeDestination,
			setup:         func(s *connectorStorage) { s.destinations.err = errStorage },
			wantMessage:   "counting dependent resources",
		},
		{
			name:          "delete fails",
			connectorType: pipelineservice.ConnectorTypeSource,
			setup:         func(s *connectorStorage) { s.managed.deleteErr = errStorage },
			wantMessage:   "deleting managed connector",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newConnectorStorage()
			audit := &recordingAudit{}
			workspaceID := uuid.New()
			mc := store.addConnector(workspaceID, tt.connectorType)
			tt.setup(store)
			useCase := pipelineconnectors.NewDeleteConnector(store, audit)

			err := useCase.Execute(context.Background(), pipelineconnectors.DeleteConnectorParams{ID: mc.ID, WorkspaceID: workspaceID})
			require.ErrorIs(t, err, errStorage)
			assert.Contains(t, err.Error(), tt.wantMessage)
			assert.Empty(t, audit.events)
		})
	}
}

// --- AddConnector ---

func newAddConnector(store *connectorStorage, audit pipelineservice.AuditRecorder) *pipelineconnectors.AddConnector {
	return pipelineconnectors.NewAddConnector(
		store,
		pipelinetasks.NewCreateSpecTask(store),
		pipelinetasks.NewWaitForTaskResult(store, pipelineservice.Config{
			ConnectorTaskPollInterval: time.Millisecond,
			ConnectionCheckTimeout:    50 * time.Millisecond,
		}),
		audit,
	)
}

func completeTask(result pipelineservice.ConnectorTaskResult) func(*pipelineservice.ConnectorTask) {
	return func(task *pipelineservice.ConnectorTask) {
		task.Status = pipelineservice.ConnectorTaskStatusCompleted
		if result != nil {
			task.Result = &result
		}
	}
}

func failTask(message string) func(*pipelineservice.ConnectorTask) {
	return func(task *pipelineservice.ConnectorTask) {
		task.Status = pipelineservice.ConnectorTaskStatusFailed
		task.ErrorMessage = &message
	}
}

func addConnectorParams(workspaceID uuid.UUID) pipelineconnectors.AddConnectorParams {
	repoID := uuid.New()

	return pipelineconnectors.AddConnectorParams{
		WorkspaceID:   workspaceID,
		DockerImage:   "airbyte/source-postgres",
		DockerTag:     "1.2.3",
		Name:          "Postgres",
		ConnectorType: pipelineservice.ConnectorTypeSource,
		RepositoryID:  &repoID,
	}
}

func TestAddConnector_StoresExtractedSpec(t *testing.T) {
	store := newConnectorStorage()
	audit := &recordingAudit{}
	store.tasks.complete = completeTask(&pipelineservice.SpecResult{
		DocumentationURL:              "https://docs.example.com",
		ChangelogURL:                  "https://changelog.example.com",
		ConnectionSpecification:       `{"type":"object","properties":{"host":{"type":"string"}}}`,
		SupportsIncremental:           true,
		SupportedDestinationSyncModes: `["append","overwrite"]`,
		AdvancedAuth:                  `not json`,
		ProtocolVersion:               "0.5.2",
	})
	workspaceID := uuid.New()
	params := addConnectorParams(workspaceID)

	got, err := newAddConnector(store, audit).Execute(context.Background(), params)
	require.NoError(t, err)

	assert.Equal(t, workspaceID, got.WorkspaceID)
	assert.Equal(t, "1.2.3", got.DockerTag)
	assert.Equal(t, params.RepositoryID, got.RepositoryID)
	assert.JSONEq(t, `{
		"documentationUrl": "https://docs.example.com",
		"changelogUrl": "https://changelog.example.com",
		"supportsIncremental": true,
		"supportsNormalization": false,
		"supportsDBT": false,
		"protocol_version": "0.5.2",
		"connectionSpecification": {"type":"object","properties":{"host":{"type":"string"}}},
		"supported_destination_sync_modes": ["append","overwrite"]
	}`, got.Spec)

	stored := store.managedConnectors.connectors[got.ID]
	require.NotNil(t, stored)
	assert.JSONEq(t, got.Spec, stored.Spec)

	require.Len(t, store.tasks.tasks, 1)

	for _, task := range store.tasks.tasks {
		assert.Equal(t, pipelineservice.ConnectorTaskTypeSpec, task.TaskType)
		assert.Equal(t, got.ID, task.Payload.(*pipelineservice.SpecPayload).ManagedConnectorID)
	}

	require.Len(t, audit.events, 1)
	assert.Equal(t, "create", audit.events[0].Action)
	assert.Equal(t, got.ID, audit.events[0].ResourceID)
	assert.Equal(t, "1.2.3", auditField(audit.events[0].After, "docker_tag"))
}

func TestAddConnector_EmptySpecFields(t *testing.T) {
	store := newConnectorStorage()
	store.tasks.complete = completeTask(&pipelineservice.SpecResult{ProtocolVersion: "0.2.0"})

	got, err := newAddConnector(store, &recordingAudit{}).Execute(context.Background(), addConnectorParams(uuid.New()))
	require.NoError(t, err)
	assert.JSONEq(t, `{
		"documentationUrl": "",
		"changelogUrl": "",
		"supportsIncremental": false,
		"supportsNormalization": false,
		"supportsDBT": false,
		"protocol_version": "0.2.0"
	}`, got.Spec)
}

func TestAddConnector_NoSpecResultKeepsEmptySpec(t *testing.T) {
	store := newConnectorStorage()
	store.tasks.complete = completeTask(nil)

	got, err := newAddConnector(store, &recordingAudit{}).Execute(context.Background(), addConnectorParams(uuid.New()))
	require.NoError(t, err)
	assert.Equal(t, "{}", got.Spec)
	assert.Contains(t, store.managedConnectors.connectors, got.ID)
}

func TestAddConnector_CreateFails(t *testing.T) {
	store := newConnectorStorage()
	audit := &recordingAudit{}
	store.managed.createErr = errStorage

	_, err := newAddConnector(store, audit).Execute(context.Background(), addConnectorParams(uuid.New()))
	require.ErrorIs(t, err, errStorage)
	assert.Contains(t, err.Error(), "creating managed connector")
	assert.Empty(t, store.tasks.tasks)
	assert.Empty(t, audit.events)
}

func TestAddConnector_SpecFailureRemovesConnector(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(*connectorStorage)
		wantMessage string
	}{
		{
			name:        "task creation fails",
			setup:       func(s *connectorStorage) { s.tasks.createErr = errStorage },
			wantMessage: "creating spec task",
		},
		{
			name:        "task never finishes",
			setup:       func(*connectorStorage) {},
			wantMessage: "waiting for spec task",
		},
		{
			name:        "task fails",
			setup:       func(s *connectorStorage) { s.tasks.complete = failTask("image not found") },
			wantMessage: "spec extraction failed: image not found",
		},
		{
			name: "storing spec fails",
			setup: func(s *connectorStorage) {
				s.tasks.complete = completeTask(&pipelineservice.SpecResult{})
				s.managed.updateErr = errStorage
			},
			wantMessage: "updating connector spec",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := newConnectorStorage()
			audit := &recordingAudit{}

			tt.setup(store)

			_, err := newAddConnector(store, audit).Execute(context.Background(), addConnectorParams(uuid.New()))
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantMessage)
			assert.Equal(t, 1, store.managed.deletes)
			assert.Empty(t, store.managedConnectors.connectors)
			assert.Empty(t, audit.events)
		})
	}
}

func TestAddConnector_RollbackFailureJoinsErrors(t *testing.T) {
	store := newConnectorStorage()
	store.tasks.complete = failTask("boom")
	store.managed.deleteErr = errStorage

	_, err := newAddConnector(store, &recordingAudit{}).Execute(context.Background(), addConnectorParams(uuid.New()))
	require.ErrorIs(t, err, errStorage)
	assert.Contains(t, err.Error(), "spec extraction failed: boom")
	assert.Contains(t, err.Error(), "deleting connector without spec")
	assert.Len(t, store.managedConnectors.connectors, 1)
}

// --- UpdateManagedConnector / BatchUpdateConnectors error paths ---

func TestUpdateManagedConnector_Errors(t *testing.T) {
	workspaceID := uuid.New()

	t.Run("connector not found", func(t *testing.T) {
		store := newConnectorStorage()
		useCase := pipelineconnectors.NewUpdateManagedConnector(store, &recordingAudit{})

		_, err := useCase.Execute(context.Background(), pipelineconnectors.UpdateManagedConnectorParams{ConnectorID: uuid.New(), WorkspaceID: workspaceID})
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
	})

	t.Run("repository connector missing", func(t *testing.T) {
		store := newConnectorStorage()
		repoID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		mc.RepositoryID = &repoID
		useCase := pipelineconnectors.NewUpdateManagedConnector(store, &recordingAudit{})

		_, err := useCase.Execute(context.Background(), pipelineconnectors.UpdateManagedConnectorParams{ConnectorID: mc.ID, WorkspaceID: workspaceID})
		require.ErrorIs(t, err, pipelineservice.ErrRepositoryConnectorNotFound)
	})

	t.Run("update fails", func(t *testing.T) {
		store := newConnectorStorage()
		repoID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		mc.RepositoryID = &repoID
		rcID := uuid.New()
		store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{ID: rcID, RepositoryID: repoID, DockerRepository: mc.DockerImage, DockerImageTag: "2.0.0"}
		store.managed.updateErr = errStorage
		audit := &recordingAudit{}
		useCase := pipelineconnectors.NewUpdateManagedConnector(store, audit)

		_, err := useCase.Execute(context.Background(), pipelineconnectors.UpdateManagedConnectorParams{ConnectorID: mc.ID, WorkspaceID: workspaceID})
		require.ErrorIs(t, err, errStorage)
		assert.Empty(t, audit.events)
	})
}

func TestUpdateManagedConnector_EmptyRepositorySpecAndAudit(t *testing.T) {
	store := newConnectorStorage()
	workspaceID := uuid.New()
	repoID := uuid.New()
	mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
	mc.RepositoryID = &repoID
	rcID := uuid.New()
	store.repoConnectors.connectors[rcID] = &pipelineservice.RepositoryConnector{ID: rcID, RepositoryID: repoID, DockerRepository: mc.DockerImage, DockerImageTag: "2.0.0"}
	audit := &recordingAudit{}
	useCase := pipelineconnectors.NewUpdateManagedConnector(store, audit)

	got, err := useCase.Execute(context.Background(), pipelineconnectors.UpdateManagedConnectorParams{ConnectorID: mc.ID, WorkspaceID: workspaceID})
	require.NoError(t, err)
	assert.Equal(t, "{}", got.Spec)
	require.Len(t, audit.events, 1)
	assert.Equal(t, "1.0.0", auditField(audit.events[0].Before, "docker_tag"))
	assert.Equal(t, "2.0.0", auditField(audit.events[0].After, "docker_tag"))
}

func TestBatchUpdateConnectors_Errors(t *testing.T) {
	workspaceID := uuid.New()

	t.Run("listing fails", func(t *testing.T) {
		store := newConnectorStorage()
		store.managed.findErr = errStorage
		useCase := pipelineconnectors.NewBatchUpdateConnectors(pipelineconnectors.NewUpdateManagedConnector(store, &recordingAudit{}), store, nil)

		_, err := useCase.Execute(context.Background(), pipelineconnectors.BatchUpdateConnectorsParams{WorkspaceID: workspaceID, ConnectorIDs: []uuid.UUID{uuid.New()}})
		require.ErrorIs(t, err, errStorage)
	})

	t.Run("failed connector is skipped", func(t *testing.T) {
		store := newConnectorStorage()
		repoID := uuid.New()
		mc := store.addConnector(workspaceID, pipelineservice.ConnectorTypeSource)
		mc.RepositoryID = &repoID
		useCase := pipelineconnectors.NewBatchUpdateConnectors(pipelineconnectors.NewUpdateManagedConnector(store, &recordingAudit{}), store, nil)

		got, err := useCase.Execute(context.Background(), pipelineconnectors.BatchUpdateConnectorsParams{WorkspaceID: workspaceID, ConnectorIDs: []uuid.UUID{mc.ID}})
		require.NoError(t, err)
		assert.Zero(t, got.UpdatedCount)
		assert.Empty(t, got.UpdatedConnectors)
	})
}

func auditField(payload any, key string) any {
	return payload.(map[string]any)[key]
}
