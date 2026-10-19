package pipelineadapt

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

type claimTaskStorage struct {
	pipelineservice.Storage
	task      *pipelineservice.ConnectorTask
	connector *pipelineservice.ManagedConnector
	source    *pipelineservice.Source
}

func (s *claimTaskStorage) ConnectorTasks() pipelineservice.ConnectorTasksStorage {
	return claimTaskTasks{task: s.task}
}

func (s *claimTaskStorage) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return claimTaskConnectors{connector: s.connector}
}

func (s *claimTaskStorage) Sources() pipelineservice.SourcesStorage {
	return claimTaskSources{source: s.source}
}

type claimTaskTasks struct {
	pipelineservice.ConnectorTasksStorage
	task *pipelineservice.ConnectorTask
}

func (t claimTaskTasks) ClaimPendingTask(_ context.Context, _ string) (*pipelineservice.ConnectorTask, error) {
	return t.task, nil
}

type claimTaskConnectors struct {
	pipelineservice.ManagedConnectorsStorage
	connector *pipelineservice.ManagedConnector
}

func (c claimTaskConnectors) First(_ context.Context, _ *pipelineservice.ManagedConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ManagedConnector, error) {
	return c.connector, nil
}

type claimTaskSources struct {
	pipelineservice.SourcesStorage
	source *pipelineservice.Source
}

func (s claimTaskSources) First(_ context.Context, _ *pipelineservice.SourceFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Source, error) {
	return s.source, nil
}

func TestUseCaseExecutorBackend_ClaimConnectorTaskCarriesServiceAccount(t *testing.T) {
	runtimeConfig := `{"service_account_name":"wi-sa"}`
	source := &pipelineservice.Source{ID: uuid.New(), Config: `{}`, RuntimeConfig: &runtimeConfig}
	storage := &claimTaskStorage{
		task: &pipelineservice.ConnectorTask{
			ID:       uuid.New(),
			TaskType: pipelineservice.ConnectorTaskTypeDiscover,
			Payload:  &pipelineservice.DiscoverPayload{ManagedConnectorID: uuid.New(), SourceID: source.ID},
		},
		connector: &pipelineservice.ManagedConnector{DockerImage: "source", DockerTag: "1"},
		source:    source,
	}
	backend := &UseCaseExecutorBackend{claimTask: pipelinetasks.NewClaimTask(storage, nil)}

	result, err := backend.ClaimConnectorTask(t.Context(), "w")
	require.NoError(t, err)
	require.NotNil(t, result)
	assert.Equal(t, "wi-sa", result.ServiceAccountName)
}
