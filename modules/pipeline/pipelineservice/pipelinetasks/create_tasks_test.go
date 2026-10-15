package pipelinetasks_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

func TestCreateCheckTask(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("rejects params without entity or config", func(t *testing.T) {
		audit := &recordingAudit{}

		_, err := pipelinetasks.NewCreateCheckTask(newTaskStore(), newFakeSecrets(), audit).Execute(ctx, pipelinetasks.CreateCheckTaskParams{WorkspaceID: ws})
		require.ErrorIs(t, err, pipelineservice.ErrMissingCheckTaskParams)
		assert.Empty(t, audit.events)
	})

	t.Run("direct config encrypts secrets and is not audited", func(t *testing.T) {
		store := newTaskStore()
		secrets := newFakeSecrets()
		audit := &recordingAudit{}
		mc := seedConnector(store, ws)

		got, err := pipelinetasks.NewCreateCheckTask(store, secrets, audit).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:        ws,
			ManagedConnectorID: &mc.ID,
			Config:             json.RawMessage(`{"host":"db","password":"hunter2"}`),
		})
		require.NoError(t, err)

		task := store.tasks.get(got.TaskID)
		require.NotNil(t, task)
		assert.Equal(t, pipelineservice.ConnectorTaskTypeCheck, task.TaskType)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusPending, task.Status)

		payload, ok := task.Payload.(*pipelineservice.CheckPayload)
		require.True(t, ok)
		assert.Equal(t, mc.ID, payload.ManagedConnectorID)
		require.NotNil(t, payload.Config)

		var cfg map[string]string
		require.NoError(t, json.Unmarshal([]byte(*payload.Config), &cfg))
		assert.Equal(t, "db", cfg["host"])
		assert.True(t, secretutil.IsSecretRef(cfg["password"]))
		assert.Empty(t, audit.events)
	})

	t.Run("direct config fails for unknown connector", func(t *testing.T) {
		_, err := pipelinetasks.NewCreateCheckTask(newTaskStore(), newFakeSecrets(), &recordingAudit{}).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:        ws,
			ManagedConnectorID: uuidPtr(uuid.New()),
			Config:             json.RawMessage(`{}`),
		})
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
	})

	t.Run("direct config fails on malformed config", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)

		_, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), &recordingAudit{}).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:        ws,
			ManagedConnectorID: &mc.ID,
			Config:             json.RawMessage(`not json`),
		})
		require.Error(t, err)
	})

	t.Run("stored source check records a test audit event", func(t *testing.T) {
		store := newTaskStore()
		audit := &recordingAudit{}
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, Name: "prod-postgres", ManagedConnectorID: uuid.New()}
		store.sources.records[src.ID] = src

		got, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), audit).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID: ws,
			SourceID:    &src.ID,
		})
		require.NoError(t, err)

		payload := store.tasks.get(got.TaskID).Payload.(*pipelineservice.CheckPayload)
		assert.Equal(t, src.ManagedConnectorID, payload.ManagedConnectorID)
		assert.Equal(t, &src.ID, payload.SourceID)

		require.Len(t, audit.events, 1)
		assert.Equal(t, pipelineservice.AuditEvent{
			WorkspaceID:   ws,
			Action:        "test",
			ResourceType:  "source",
			ResourceID:    src.ID,
			ResourceLabel: "prod-postgres",
		}, audit.events[0])
	})

	t.Run("stored destination check records a test audit event", func(t *testing.T) {
		store := newTaskStore()
		audit := &recordingAudit{}
		dst := &pipelineservice.Destination{ID: uuid.New(), WorkspaceID: ws, Name: "warehouse", ManagedConnectorID: uuid.New()}
		store.destinations.records[dst.ID] = dst

		got, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), audit).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:   ws,
			DestinationID: &dst.ID,
		})
		require.NoError(t, err)

		payload := store.tasks.get(got.TaskID).Payload.(*pipelineservice.CheckPayload)
		assert.Equal(t, &dst.ID, payload.DestinationID)

		require.Len(t, audit.events, 1)
		assert.Equal(t, pipelineservice.AuditEvent{
			WorkspaceID:   ws,
			Action:        "test",
			ResourceType:  "destination",
			ResourceID:    dst.ID,
			ResourceLabel: "warehouse",
		}, audit.events[0])
	})

	t.Run("source from another workspace is not found and not audited", func(t *testing.T) {
		store := newTaskStore()
		audit := &recordingAudit{}
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: uuid.New()}
		store.sources.records[src.ID] = src

		_, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), audit).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID: ws,
			SourceID:    &src.ID,
		})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
		assert.Empty(t, audit.events)
	})

	t.Run("missing destination is not found", func(t *testing.T) {
		_, err := pipelinetasks.NewCreateCheckTask(newTaskStore(), newFakeSecrets(), &recordingAudit{}).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID:   ws,
			DestinationID: uuidPtr(uuid.New()),
		})
		require.ErrorIs(t, err, pipelineservice.ErrDestinationNotFound)
	})

	t.Run("task create failure is not audited", func(t *testing.T) {
		store := newTaskStore()
		store.tasks.createErr = errBoom
		audit := &recordingAudit{}
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws}
		store.sources.records[src.ID] = src

		_, err := pipelinetasks.NewCreateCheckTask(store, newFakeSecrets(), audit).Execute(ctx, pipelinetasks.CreateCheckTaskParams{
			WorkspaceID: ws,
			SourceID:    &src.ID,
		})
		require.ErrorIs(t, err, errBoom)
		assert.Empty(t, audit.events)
	})
}

func TestCreateDiscoverTask(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("creates a pending discover task", func(t *testing.T) {
		store := newTaskStore()
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: uuid.New()}
		store.sources.records[src.ID] = src

		got, err := pipelinetasks.NewCreateDiscoverTask(store).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})
		require.NoError(t, err)

		task := store.tasks.get(got.TaskID)
		assert.Equal(t, pipelineservice.ConnectorTaskTypeDiscover, task.TaskType)
		assert.Equal(t, &pipelineservice.DiscoverPayload{SourceID: src.ID, ManagedConnectorID: src.ManagedConnectorID}, task.Payload)
	})

	t.Run("reuses an in-flight discover task for the same source", func(t *testing.T) {
		store := newTaskStore()
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws, ManagedConnectorID: uuid.New()}
		store.sources.records[src.ID] = src

		other := pendingTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{SourceID: uuid.New()})
		existing := pendingTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{SourceID: src.ID})
		existing.Status = pipelineservice.ConnectorTaskStatusRunning
		store.tasks.records[other.ID] = other
		store.tasks.records[existing.ID] = existing

		got, err := pipelinetasks.NewCreateDiscoverTask(store).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})
		require.NoError(t, err)
		assert.Equal(t, existing.ID, got.TaskID)
		assert.Len(t, store.tasks.records, 2)
	})

	t.Run("ignores finished discover tasks for the same source", func(t *testing.T) {
		store := newTaskStore()
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws}
		store.sources.records[src.ID] = src

		done := pendingTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{SourceID: src.ID})
		done.Status = pipelineservice.ConnectorTaskStatusCompleted
		store.tasks.records[done.ID] = done

		got, err := pipelinetasks.NewCreateDiscoverTask(store).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})
		require.NoError(t, err)
		assert.NotEqual(t, done.ID, got.TaskID)
	})

	t.Run("fails for source in another workspace", func(t *testing.T) {
		store := newTaskStore()
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: uuid.New()}
		store.sources.records[src.ID] = src

		_, err := pipelinetasks.NewCreateDiscoverTask(store).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})
		require.ErrorIs(t, err, pipelineservice.ErrSourceNotFound)
	})

	t.Run("propagates dedup lookup error", func(t *testing.T) {
		store := newTaskStore()
		store.tasks.findErr = errBoom
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws}
		store.sources.records[src.ID] = src

		_, err := pipelinetasks.NewCreateDiscoverTask(store).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("propagates create error", func(t *testing.T) {
		store := newTaskStore()
		store.tasks.createErr = errBoom
		src := &pipelineservice.Source{ID: uuid.New(), WorkspaceID: ws}
		store.sources.records[src.ID] = src

		_, err := pipelinetasks.NewCreateDiscoverTask(store).Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{WorkspaceID: ws, SourceID: src.ID})
		require.ErrorIs(t, err, errBoom)
	})
}

func TestCreateSpecTask(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("creates a pending spec task", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, ws)

		got, err := pipelinetasks.NewCreateSpecTask(store).Execute(ctx, pipelinetasks.CreateSpecTaskParams{WorkspaceID: ws, ManagedConnectorID: mc.ID})
		require.NoError(t, err)

		task := store.tasks.get(got.TaskID)
		assert.Equal(t, pipelineservice.ConnectorTaskTypeSpec, task.TaskType)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusPending, task.Status)
		assert.Equal(t, &pipelineservice.SpecPayload{ManagedConnectorID: mc.ID}, task.Payload)
	})

	t.Run("fails for connector in another workspace", func(t *testing.T) {
		store := newTaskStore()
		mc := seedConnector(store, uuid.New())

		_, err := pipelinetasks.NewCreateSpecTask(store).Execute(ctx, pipelinetasks.CreateSpecTaskParams{WorkspaceID: ws, ManagedConnectorID: mc.ID})
		require.ErrorIs(t, err, pipelineservice.ErrManagedConnectorNotFound)
		assert.Empty(t, store.tasks.records)
	})

	t.Run("propagates create error", func(t *testing.T) {
		store := newTaskStore()
		store.tasks.createErr = errBoom
		mc := seedConnector(store, ws)

		_, err := pipelinetasks.NewCreateSpecTask(store).Execute(ctx, pipelinetasks.CreateSpecTaskParams{WorkspaceID: ws, ManagedConnectorID: mc.ID})
		require.ErrorIs(t, err, errBoom)
	})
}
