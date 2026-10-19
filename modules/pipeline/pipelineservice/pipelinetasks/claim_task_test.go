package pipelinetasks_test

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

func seedConnector(store *taskStore, ws uuid.UUID) *pipelineservice.ManagedConnector {
	mc := &pipelineservice.ManagedConnector{
		ID:          uuid.New(),
		WorkspaceID: ws,
		DockerImage: "airbyte/source-postgres",
		DockerTag:   "1.0.0",
		Spec:        `{"connectionSpecification":{"properties":{"password":{"type":"string","airbyte_secret":true}}}}`,
	}
	store.connectors.records[mc.ID] = mc

	return mc
}

func pendingTask(ws uuid.UUID, taskType pipelineservice.ConnectorTaskType, payload pipelineservice.ConnectorTaskPayload) *pipelineservice.ConnectorTask {
	return &pipelineservice.ConnectorTask{
		ID:          uuid.New(),
		WorkspaceID: ws,
		TaskType:    taskType,
		Status:      pipelineservice.ConnectorTaskStatusPending,
		Payload:     payload,
		CreatedAt:   time.Now(),
	}
}

func TestClaimTask(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("returns nil when nothing is pending", func(t *testing.T) {
		store := newTaskStore()

		got, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.NoError(t, err)
		assert.Nil(t, got)
	})

	t.Run("propagates storage claim error", func(t *testing.T) {
		store := newTaskStore()
		store.tasks.claimErr = errBoom

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("claims a spec task and resolves image", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, &pipelineservice.SpecPayload{ManagedConnectorID: mc.ID})
		store.tasks.records[task.ID] = task

		got, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.NoError(t, err)
		require.NotNil(t, got)
		assert.Equal(t, task.ID, got.TaskID)
		assert.Equal(t, pipelineservice.ConnectorTaskTypeSpec, got.TaskType)
		assert.Equal(t, "airbyte/source-postgres:1.0.0", got.Image)
		assert.Nil(t, got.Config)
		assert.Equal(t, ws, got.WorkspaceID)

		stored := store.tasks.get(task.ID)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusRunning, stored.Status)
		require.NotNil(t, stored.WorkerID)
		assert.Equal(t, "worker-1", *stored.WorkerID)
	})

	t.Run("spec task fails when connector belongs to another workspace", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, uuid.New())
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, &pipelineservice.SpecPayload{ManagedConnectorID: mc.ID})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
	})

	t.Run("check task with inline config decrypts secrets", func(t *testing.T) {
		store := newTaskStore()
		secrets := newFakeSecrets()
		mc := seedConnector(store, ws)
		ref, err := secrets.StoreSecret(ctx, "connector_task", uuid.New(), "hunter2")
		require.NoError(t, err)

		cfg := `{"host":"db","password":"` + ref + `"}`
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, Config: &cfg})
		store.tasks.records[task.ID] = task

		got, err := pipelinetasks.NewClaimTask(store, secrets).Execute(ctx, "worker-1")
		require.NoError(t, err)
		assert.JSONEq(t, `{"host":"db","password":"hunter2"}`, string(got.Config))
	})

	t.Run("check task loads stored source config", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: mc.ID, Config: `{"host":"src"}`}
		store.sources.records[src.ID] = src
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, SourceID: &src.ID})
		store.tasks.records[task.ID] = task

		got, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.NoError(t, err)
		assert.JSONEq(t, `{"host":"src"}`, string(got.Config))
	})

	t.Run("check task loads stored destination config", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		dst := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: mc.ID, Config: `{"host":"dst"}`}
		store.destinations.records[dst.ID] = dst
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, DestinationID: &dst.ID})
		store.tasks.records[task.ID] = task

		got, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.NoError(t, err)
		assert.JSONEq(t, `{"host":"dst"}`, string(got.Config))
	})

	t.Run("check task fails when referenced source is missing", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, SourceID: uuidPtr(uuid.New())})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	})

	t.Run("check task fails when referenced destination is missing", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, DestinationID: uuidPtr(uuid.New())})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
	})

	t.Run("check task fails when a secret cannot be resolved", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		cfg := `{"password":"$secret:` + uuid.NewString() + `"}`
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, Config: &cfg})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("check task fails when connector is missing", func(t *testing.T) {
		store := newTaskStore()
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{ManagedConnectorID: uuid.New()})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
	})

	t.Run("discover task resolves image and source config", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: mc.ID, Config: `{"db":"x"}`}
		store.sources.records[src.ID] = src
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{ManagedConnectorID: mc.ID, SourceID: src.ID})
		store.tasks.records[task.ID] = task

		got, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.NoError(t, err)
		assert.Equal(t, "airbyte/source-postgres:1.0.0", got.Image)
		assert.JSONEq(t, `{"db":"x"}`, string(got.Config))
	})

	t.Run("discover task fails when connector is missing", func(t *testing.T) {
		store := newTaskStore()
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{ManagedConnectorID: uuid.New(), SourceID: uuid.New()})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
	})

	t.Run("discover task fails when source is missing", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{ManagedConnectorID: mc.ID, SourceID: uuid.New()})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	})

	t.Run("discover task fails when source config is not JSON", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: mc.ID, Config: `not json`}
		store.sources.records[src.ID] = src
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{ManagedConnectorID: mc.ID, SourceID: src.ID})
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.Error(t, err)
	})

	t.Run("carries the entity's service account", func(t *testing.T) {
		withAccount := `{"service_account_name":"wi-sa"}`
		withoutAccount := `{"cpu_limit":"1"}`
		inline := `{"host":"db"}`

		tests := []struct {
			name     string
			taskType pipelineservice.ConnectorTaskType
			payload  func(mc *pipelineservice.ManagedConnector, src *pipelineservice.Source, dst *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload
			runtime  *string
			want     string
		}{
			{name: "check of a source", taskType: pipelineservice.ConnectorTaskTypeCheck, runtime: &withAccount, want: "wi-sa", payload: func(mc *pipelineservice.ManagedConnector, src *pipelineservice.Source, _ *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload {
				return &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, SourceID: &src.ID}
			}},
			{name: "check of a destination", taskType: pipelineservice.ConnectorTaskTypeCheck, runtime: &withAccount, want: "wi-sa", payload: func(mc *pipelineservice.ManagedConnector, _ *pipelineservice.Source, dst *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload {
				return &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, DestinationID: &dst.ID}
			}},
			{name: "discover of a source", taskType: pipelineservice.ConnectorTaskTypeDiscover, runtime: &withAccount, want: "wi-sa", payload: func(mc *pipelineservice.ManagedConnector, src *pipelineservice.Source, _ *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload {
				return &pipelineservice.DiscoverPayload{ManagedConnectorID: mc.ID, SourceID: src.ID}
			}},
			{name: "source without an account", taskType: pipelineservice.ConnectorTaskTypeDiscover, runtime: &withoutAccount, payload: func(mc *pipelineservice.ManagedConnector, src *pipelineservice.Source, _ *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload {
				return &pipelineservice.DiscoverPayload{ManagedConnectorID: mc.ID, SourceID: src.ID}
			}},
			{name: "source without runtime config", taskType: pipelineservice.ConnectorTaskTypeDiscover, payload: func(mc *pipelineservice.ManagedConnector, src *pipelineservice.Source, _ *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload {
				return &pipelineservice.DiscoverPayload{ManagedConnectorID: mc.ID, SourceID: src.ID}
			}},
			{name: "inline check", taskType: pipelineservice.ConnectorTaskTypeCheck, runtime: &withAccount, payload: func(mc *pipelineservice.ManagedConnector, _ *pipelineservice.Source, _ *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload {
				return &pipelineservice.CheckPayload{ManagedConnectorID: mc.ID, Config: &inline}
			}},
			{name: "spec", taskType: pipelineservice.ConnectorTaskTypeSpec, runtime: &withAccount, payload: func(mc *pipelineservice.ManagedConnector, _ *pipelineservice.Source, _ *pipelineservice.Destination) pipelineservice.ConnectorTaskPayload {
				return &pipelineservice.SpecPayload{ManagedConnectorID: mc.ID}
			}},
		}

		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				store := newTaskStore()
				mc := seedConnector(store, ws)
				src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: mc.ID, Config: `{}`, RuntimeConfig: tt.runtime}
				store.sources.records[src.ID] = src
				dst := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: mc.ID, Config: `{}`, RuntimeConfig: tt.runtime}
				store.destinations.records[dst.ID] = dst
				task := pendingTask(ws, tt.taskType, tt.payload(mc, src, dst))
				store.tasks.records[task.ID] = task

				got, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
				require.NoError(t, err)
				assert.Equal(t, tt.want, got.ServiceAccountName)
			})
		}
	})

	t.Run("rejects unknown payload type", func(t *testing.T) {
		store := newTaskStore()
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		store.tasks.records[task.ID] = task

		_, err := pipelinetasks.NewClaimTask(store, newFakeSecrets()).Execute(ctx, "worker-1")
		require.ErrorContains(t, err, "unknown payload type")
	})
}

func TestClaimTask_ConcurrentClaimersGetExactlyOneWinner(t *testing.T) {
	ws := uuid.New()
	store := newTaskStore()
	mc := seedConnector(store, ws)
	task := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, &pipelineservice.SpecPayload{ManagedConnectorID: mc.ID})
	store.tasks.records[task.ID] = task

	uc := pipelinetasks.NewClaimTask(store, newFakeSecrets())

	const claimers = 8

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		winners []string
		start   = make(chan struct{})
	)

	for i := range claimers {
		wg.Add(1)

		go func(workerID string) {
			defer wg.Done()

			<-start

			got, err := uc.Execute(context.Background(), workerID)
			assert.NoError(t, err)

			if got != nil {
				mu.Lock()

				winners = append(winners, workerID)
				mu.Unlock()
			}
		}(fmt.Sprintf("worker-%d", i))
	}

	close(start)
	wg.Wait()

	require.Len(t, winners, 1)

	stored := store.tasks.get(task.ID)
	assert.Equal(t, pipelineservice.ConnectorTaskStatusRunning, stored.Status)
	require.NotNil(t, stored.WorkerID)
	assert.Equal(t, winners[0], *stored.WorkerID)
}

func TestClaimTask_DirectConfigCheckClaimsItsServiceAccount(t *testing.T) {
	ctx := context.Background()
	ws := uuid.New()
	store := newTaskStore()
	secrets := newFakeSecrets()
	mc := seedConnector(store, ws)

	_, err := pipelinetasks.NewCreateCheckTask(store, secrets, &recordingAudit{}).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
		WorkspaceID:        ws,
		ManagedConnectorID: &mc.ID,
		Config:             json.RawMessage(`{"host":"db"}`),
		ServiceAccountName: "wi-sa",
	})
	require.NoError(t, err)

	got, err := pipelinetasks.NewClaimTask(store, secrets).Execute(ctx, "worker-1")
	require.NoError(t, err)
	require.NotNil(t, got)
	assert.Equal(t, "wi-sa", got.ServiceAccountName)
}
