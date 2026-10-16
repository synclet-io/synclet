package pipelineworkspaces_test

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineworkspaces"
)

var errBoom = errors.New("boom")

type purgeStore struct {
	pipelineservice.Storage

	log []string

	connections  []*pipelineservice.Connection
	jobs         []*pipelineservice.Job
	sources      []*pipelineservice.Source
	destinations []*pipelineservice.Destination
	tasks        []*pipelineservice.ConnectorTask
	repositories []*pipelineservice.Repository

	jobFilter         *pipelineservice.JobFilter
	cancelledJobs     []*pipelineservice.Job
	sourceDeleteError error
}

func (s *purgeStore) Connections() pipelineservice.ConnectionsStorage {
	return &purgeConnections{store: s}
}

func (s *purgeStore) Jobs() pipelineservice.JobsStorage { return &purgeJobs{store: s} }

func (s *purgeStore) Sources() pipelineservice.SourcesStorage { return &purgeSources{store: s} }

func (s *purgeStore) Destinations() pipelineservice.DestinationsStorage {
	return &purgeDestinations{store: s}
}

func (s *purgeStore) ConnectorTasks() pipelineservice.ConnectorTasksStorage {
	return &purgeTasks{store: s}
}

func (s *purgeStore) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return &purgeManagedConnectors{store: s}
}

func (s *purgeStore) Repositories() pipelineservice.RepositoriesStorage {
	return &purgeRepositories{store: s}
}

func (s *purgeStore) WorkspaceSettings() pipelineservice.WorkspaceSettingsStorage {
	return &purgeSettings{store: s}
}

type purgeConnections struct {
	pipelineservice.ConnectionsStorage
	store *purgeStore
}

func (c *purgeConnections) Find(context.Context, *pipelineservice.ConnectionFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	return c.store.connections, nil
}

func (c *purgeConnections) Delete(context.Context, *pipelineservice.ConnectionFilter) error {
	c.store.log = append(c.store.log, "delete connections")
	c.store.connections = nil

	return nil
}

type purgeJobs struct {
	pipelineservice.JobsStorage
	store *purgeStore
}

func (j *purgeJobs) Find(_ context.Context, f *pipelineservice.JobFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Job, error) {
	j.store.jobFilter = f

	return j.store.jobs, nil
}

func (j *purgeJobs) UpdateIfStatus(_ context.Context, job *pipelineservice.Job, _ ...pipelineservice.JobStatus) (bool, error) {
	j.store.log = append(j.store.log, "cancel job")
	j.store.cancelledJobs = append(j.store.cancelledJobs, job)

	return true, nil
}

type purgeSources struct {
	pipelineservice.SourcesStorage
	store *purgeStore
}

func (s *purgeSources) Find(context.Context, *pipelineservice.SourceFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Source, error) {
	return s.store.sources, nil
}

func (s *purgeSources) Delete(context.Context, *pipelineservice.SourceFilter) error {
	if s.store.sourceDeleteError != nil {
		return s.store.sourceDeleteError
	}

	s.store.log = append(s.store.log, "delete sources")
	s.store.sources = nil

	return nil
}

type purgeDestinations struct {
	pipelineservice.DestinationsStorage
	store *purgeStore
}

func (d *purgeDestinations) Find(context.Context, *pipelineservice.DestinationFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Destination, error) {
	return d.store.destinations, nil
}

func (d *purgeDestinations) Delete(context.Context, *pipelineservice.DestinationFilter) error {
	d.store.log = append(d.store.log, "delete destinations")
	d.store.destinations = nil

	return nil
}

type purgeTasks struct {
	pipelineservice.ConnectorTasksStorage
	store *purgeStore
}

func (t *purgeTasks) Find(context.Context, *pipelineservice.ConnectorTaskFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.ConnectorTask, error) {
	return t.store.tasks, nil
}

func (t *purgeTasks) Delete(context.Context, *pipelineservice.ConnectorTaskFilter) error {
	t.store.log = append(t.store.log, "delete connector tasks")
	t.store.tasks = nil

	return nil
}

type purgeManagedConnectors struct {
	pipelineservice.ManagedConnectorsStorage
	store *purgeStore
}

func (m *purgeManagedConnectors) Delete(context.Context, *pipelineservice.ManagedConnectorFilter) error {
	m.store.log = append(m.store.log, "delete managed connectors")

	return nil
}

type purgeRepositories struct {
	pipelineservice.RepositoriesStorage
	store *purgeStore
}

func (r *purgeRepositories) Find(context.Context, *pipelineservice.RepositoryFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Repository, error) {
	return r.store.repositories, nil
}

func (r *purgeRepositories) Delete(context.Context, *pipelineservice.RepositoryFilter) error {
	r.store.log = append(r.store.log, "delete repositories")
	r.store.repositories = nil

	return nil
}

type purgeSettings struct {
	pipelineservice.WorkspaceSettingsStorage
	store *purgeStore
}

func (s *purgeSettings) Delete(context.Context, *pipelineservice.WorkspaceSettingsFilter) error {
	s.store.log = append(s.store.log, "delete workspace settings")

	return nil
}

type purgeStats struct {
	pipelineservice.StatsStorage
	store *purgeStore
}

func (s *purgeStats) DeleteWorkspaceRollups(_ context.Context, workspaceID uuid.UUID) error {
	s.store.log = append(s.store.log, "delete rollups "+workspaceID.String())

	return nil
}

type purgeSecrets struct {
	pipelineservice.SecretsProvider
	store *purgeStore
}

func (s *purgeSecrets) DeleteByOwner(_ context.Context, ownerType string, ownerID uuid.UUID) error {
	s.store.log = append(s.store.log, "secrets "+ownerType+" "+ownerID.String())

	return nil
}

func TestPurgeWorkspace(t *testing.T) {
	ctx := context.Background()
	workspaceID := uuid.New()

	t.Run("cancels jobs, then deletes secrets before their owners", func(t *testing.T) {
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: workspaceID}
		job := &pipelineservice.Job{ID: uuid.New(), ConnectionID: conn.ID, Status: pipelineservice.JobStatusRunning}
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: workspaceID}
		dest := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: workspaceID}
		config := "{}"
		directCheck := &pipelineservice.ConnectorTask{ID: uuid.New(), Payload: &pipelineservice.CheckPayload{Config: &config}}
		specTask := &pipelineservice.ConnectorTask{ID: uuid.New(), Payload: &pipelineservice.SpecPayload{}}
		repo := &pipelineservice.Repository{ID: uuid.New(), WorkspaceID: workspaceID}

		store := &purgeStore{
			connections:  []*pipelineservice.Connection{conn},
			jobs:         []*pipelineservice.Job{job},
			sources:      []*pipelineservice.Source{src},
			destinations: []*pipelineservice.Destination{dest},
			tasks:        []*pipelineservice.ConnectorTask{directCheck, specTask},
			repositories: []*pipelineservice.Repository{repo},
		}

		uc := pipelineworkspaces.NewPurgeWorkspace(store, &purgeStats{store: store}, &purgeSecrets{store: store})
		require.NoError(t, uc.Execute(ctx, workspaceID))

		assert.Equal(t, []string{
			"cancel job",
			"delete connections",
			"secrets source " + src.ID.String(),
			"delete sources",
			"secrets destination " + dest.ID.String(),
			"delete destinations",
			"secrets connector_task " + directCheck.ID.String(),
			"delete connector tasks",
			"delete managed connectors",
			"secrets repository " + repo.ID.String(),
			"delete repositories",
			"delete workspace settings",
			"delete rollups " + workspaceID.String(),
		}, store.log)

		require.Len(t, store.cancelledJobs, 1)
		assert.Equal(t, pipelineservice.JobStatusCancelled, store.cancelledJobs[0].Status)
		assert.NotNil(t, store.cancelledJobs[0].CompletedAt)
		require.NotNil(t, store.cancelledJobs[0].FailureReason)
		assert.Equal(t, "workspace deleted", *store.cancelledJobs[0].FailureReason)

		connFilter, ok := store.jobFilter.ConnectionID.(*filter.InFilter[uuid.UUID])
		require.True(t, ok)
		assert.Equal(t, []uuid.UUID{conn.ID}, connFilter.Values)
	})

	t.Run("rerun on an emptied workspace only issues the scoped deletes", func(t *testing.T) {
		store := &purgeStore{}

		uc := pipelineworkspaces.NewPurgeWorkspace(store, &purgeStats{store: store}, &purgeSecrets{store: store})
		require.NoError(t, uc.Execute(ctx, workspaceID))

		assert.Equal(t, []string{
			"delete sources",
			"delete destinations",
			"delete connector tasks",
			"delete managed connectors",
			"delete repositories",
			"delete workspace settings",
			"delete rollups " + workspaceID.String(),
		}, store.log)
	})

	t.Run("stops at the first failure so a retry can finish", func(t *testing.T) {
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: workspaceID}
		store := &purgeStore{sources: []*pipelineservice.Source{src}, sourceDeleteError: errBoom}

		uc := pipelineworkspaces.NewPurgeWorkspace(store, &purgeStats{store: store}, &purgeSecrets{store: store})
		require.ErrorIs(t, uc.Execute(ctx, workspaceID), errBoom)

		assert.NotContains(t, store.log, "delete destinations")
		assert.Len(t, store.sources, 1)
	})
}
