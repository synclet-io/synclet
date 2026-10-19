package pipelinejobs

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// bundleStorage serves one connection, its source and destination, a
// configured catalog and per-stream generations kept in memory.
type bundleStorage struct {
	pipelineservice.Storage
	job         *pipelineservice.Job
	conn        *pipelineservice.Connection
	catalog     string
	generations map[string]*pipelineservice.StreamGeneration
	jobCount    int
}

func newBundleStorage(t *testing.T, streams []protocol.ConfiguredAirbyteStream, generations map[string]int64) *bundleStorage {
	t.Helper()

	streamsJSON, err := json.Marshal(streams)
	require.NoError(t, err)

	syncID := int64(4711)
	conn := &pipelineservice.Connection{
		ID:            uuid.New(),
		WorkspaceID:   uuid.New(),
		SourceID:      uuid.New(),
		DestinationID: uuid.New(),
		MaxAttempts:   3,
	}
	s := &bundleStorage{
		conn: conn,
		job: &pipelineservice.Job{
			ID:           uuid.New(),
			ConnectionID: conn.ID,
			Status:       pipelineservice.JobStatusStarting,
			Attempt:      1,
			MaxAttempts:  3,
			SyncID:       &syncID,
		},
		catalog:     string(streamsJSON),
		generations: map[string]*pipelineservice.StreamGeneration{},
		jobCount:    10,
	}

	for name, gen := range generations {
		s.generations[name] = &pipelineservice.StreamGeneration{ConnectionID: conn.ID, StreamName: name, GenerationID: gen}
	}

	return s
}

func (s *bundleStorage) Jobs() pipelineservice.JobsStorage { return &bundleJobs{s: s} }

func (s *bundleStorage) Connections() pipelineservice.ConnectionsStorage {
	return &jobStateConnections{s: &jobStateStorage{conn: s.conn}}
}

func (s *bundleStorage) Sources() pipelineservice.SourcesStorage { return &bundleSources{s: s} }

func (s *bundleStorage) Destinations() pipelineservice.DestinationsStorage {
	return &bundleDestinations{s: s}
}

func (s *bundleStorage) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return &bundleManagedConnectors{}
}

func (s *bundleStorage) ConfiguredCatalogs() pipelineservice.ConfiguredCatalogsStorage {
	return &bundleConfiguredCatalogs{s: s}
}

func (s *bundleStorage) CatalogDiscoveries() pipelineservice.CatalogDiscoveriesStorage {
	return &bundleCatalogDiscoveries{}
}

func (s *bundleStorage) ConnectionStates() pipelineservice.ConnectionStatesStorage {
	return &bundleConnectionStates{}
}

func (s *bundleStorage) StreamGenerations() pipelineservice.StreamGenerationsStorage {
	return &bundleStreamGenerations{s: s}
}

func (s *bundleStorage) PipelineEvents() pipelineservice.PipelineEventsOutbox {
	return &jobStateOutbox{s: &jobStateStorage{}}
}

type bundleJobs struct {
	pipelineservice.JobsStorage
	s *bundleStorage
}

func (j *bundleJobs) ClaimNextScheduledJob(context.Context, string) (*pipelineservice.Job, error) {
	cp := *j.s.job

	return &cp, nil
}

func (j *bundleJobs) Count(context.Context, *pipelineservice.JobFilter) (int, error) {
	return j.s.jobCount, nil
}

type bundleSources struct {
	pipelineservice.SourcesStorage
	s *bundleStorage
}

func (b *bundleSources) First(context.Context, *pipelineservice.SourceFilter, ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	return &pipelineservice.Source{ID: b.s.conn.SourceID, WorkspaceID: b.s.conn.WorkspaceID, Config: "{}"}, nil
}

type bundleDestinations struct {
	pipelineservice.DestinationsStorage
	s *bundleStorage
}

func (b *bundleDestinations) First(context.Context, *pipelineservice.DestinationFilter, ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Destination, error) {
	return &pipelineservice.Destination{ID: b.s.conn.DestinationID, WorkspaceID: b.s.conn.WorkspaceID, Config: "{}"}, nil
}

type bundleManagedConnectors struct {
	pipelineservice.ManagedConnectorsStorage
}

func (bundleManagedConnectors) First(context.Context, *pipelineservice.ManagedConnectorFilter, ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ManagedConnector, error) {
	return &pipelineservice.ManagedConnector{DockerImage: "airbyte/destination-dev-null", DockerTag: "0.7.24"}, nil
}

type bundleConfiguredCatalogs struct {
	pipelineservice.ConfiguredCatalogsStorage
	s *bundleStorage
}

func (b *bundleConfiguredCatalogs) First(context.Context, *pipelineservice.ConfiguredCatalogFilter, ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConfiguredCatalog, error) {
	return &pipelineservice.ConfiguredCatalog{ConnectionID: b.s.conn.ID, StreamsJSON: b.s.catalog}, nil
}

type bundleCatalogDiscoveries struct {
	pipelineservice.CatalogDiscoveriesStorage
}

func (bundleCatalogDiscoveries) First(context.Context, *pipelineservice.CatalogDiscoveryFilter, ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.CatalogDiscovery, error) {
	return nil, errors.New("not found")
}

type bundleConnectionStates struct {
	pipelineservice.ConnectionStatesStorage
}

func (bundleConnectionStates) First(context.Context, *pipelineservice.ConnectionStateFilter, ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConnectionState, error) {
	return nil, errors.New("not found")
}

type bundleStreamGenerations struct {
	pipelineservice.StreamGenerationsStorage
	s *bundleStorage
}

func (b *bundleStreamGenerations) Find(context.Context, *pipelineservice.StreamGenerationFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.StreamGeneration, error) {
	result := make([]*pipelineservice.StreamGeneration, 0, len(b.s.generations))
	for _, sg := range b.s.generations {
		cp := *sg
		result = append(result, &cp)
	}

	return result, nil
}

func (b *bundleStreamGenerations) Upsert(_ context.Context, sg *pipelineservice.StreamGeneration) error {
	cp := *sg
	b.s.generations[sg.StreamName] = &cp

	return nil
}

func newTestClaimJobBundle(storage *bundleStorage) *ClaimJobBundle {
	logger := &logging.Logger{Delegate: noopLogDelegate{}}

	return NewClaimJobBundle(
		NewClaimJob(storage, logger),
		NewUpdateJobStatus(storage),
		pipelinecatalog.NewGetConfiguredCatalog(storage),
		pipelinecatalog.NewPopulateGenerationIDs(storage),
		pipelinestate.NewGetSyncState(storage),
		storage,
		nil,
		logger,
	)
}

type catalogGenerationFields struct {
	Streams []struct {
		Stream struct {
			Name string `json:"name"`
		} `json:"stream"`
		GenerationID        *int64 `json:"generation_id"`
		MinimumGenerationID *int64 `json:"minimum_generation_id"`
		SyncID              *int64 `json:"sync_id"`
	} `json:"streams"`
}

type streamGeneration struct{ gen, minGen, syncID int64 }

func claimGenerations(t *testing.T, uc *ClaimJobBundle) map[string]streamGeneration {
	t.Helper()

	result, err := uc.Execute(context.Background(), ClaimJobParams{WorkerID: "worker-1"})
	require.NoError(t, err)
	require.NotNil(t, result)

	var catalog catalogGenerationFields
	require.NoError(t, json.Unmarshal(result.ConfiguredCatalog, &catalog))

	fields := make(map[string]streamGeneration, len(catalog.Streams))

	for _, stream := range catalog.Streams {
		require.NotNil(t, stream.GenerationID, "%s: generation_id missing", stream.Stream.Name)
		require.NotNil(t, stream.MinimumGenerationID, "%s: minimum_generation_id missing", stream.Stream.Name)
		require.NotNil(t, stream.SyncID, "%s: sync_id missing", stream.Stream.Name)
		fields[stream.Stream.Name] = streamGeneration{*stream.GenerationID, *stream.MinimumGenerationID, *stream.SyncID}
	}

	return fields
}

func generationTestStreams() []protocol.ConfiguredAirbyteStream {
	return []protocol.ConfiguredAirbyteStream{
		{
			Stream:              protocol.AirbyteStream{Name: "users"},
			SyncMode:            protocol.SyncModeFullRefresh,
			DestinationSyncMode: protocol.DestinationSyncModeOverwrite,
		},
		{
			Stream:              protocol.AirbyteStream{Name: "orders"},
			SyncMode:            protocol.SyncModeIncremental,
			DestinationSyncMode: protocol.DestinationSyncModeAppend,
		},
	}
}

func TestClaimJobBundleCatalogCarriesGenerationFields(t *testing.T) {
	storage := newBundleStorage(t, generationTestStreams(), map[string]int64{"users": 4, "orders": 2})

	fields := claimGenerations(t, newTestClaimJobBundle(storage))

	assert.Equal(t, map[string]streamGeneration{
		"users":  {gen: 5, minGen: 5, syncID: 4711},
		"orders": {gen: 2, minGen: 0, syncID: 4711},
	}, fields)
}

// A retry is the same job: sync_id stays the job's, even when old jobs of the
// connection are cleaned up between attempts. A full refresh stream gets a new
// generation per attempt, as Airbyte's AttemptHandler bumps it for every
// attempt of a non-resumable full refresh.
func TestClaimJobBundleRetryKeepsSyncID(t *testing.T) {
	storage := newBundleStorage(t, generationTestStreams(), map[string]int64{"users": 4, "orders": 2})
	uc := newTestClaimJobBundle(storage)

	first := claimGenerations(t, uc)

	storage.job.Attempt = 2
	storage.jobCount = 3

	second := claimGenerations(t, uc)

	assert.Equal(t, map[string]streamGeneration{
		"users":  {gen: 5, minGen: 5, syncID: 4711},
		"orders": {gen: 2, minGen: 0, syncID: 4711},
	}, first)
	assert.Equal(t, map[string]streamGeneration{
		"users":  {gen: 6, minGen: 6, syncID: 4711},
		"orders": {gen: 2, minGen: 0, syncID: 4711},
	}, second)
}
