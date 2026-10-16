package pipelinetasks_test

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

func runningTask(ws uuid.UUID, taskType pipelineservice.ConnectorTaskType, payload pipelineservice.ConnectorTaskPayload) *pipelineservice.ConnectorTask {
	task := pendingTask(ws, taskType, payload)
	task.Status = pipelineservice.ConnectorTaskStatusRunning
	task.WorkerID = strPtr("worker-1")
	task.UpdatedAt = time.Now()

	return task
}

func TestReportTaskResult(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("completes a check task with a parsed result", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{})
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{"Success":true,"Message":"ok"}`),
		})
		require.NoError(t, err)

		got := store.tasks.get(task.ID)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusCompleted, got.Status)
		require.NotNil(t, got.CompletedAt)
		require.NotNil(t, got.Result)
		assert.Equal(t, &pipelineservice.CheckResult{Success: true, Message: "ok"}, *got.Result)
		assert.Nil(t, got.ErrorMessage)
	})

	t.Run("completes a spec task", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeSpec, &pipelineservice.SpecPayload{})
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{"DocumentationURL":"https://docs"}`),
		})
		require.NoError(t, err)

		got := store.tasks.get(task.ID)
		require.NotNil(t, got.Result)
		assert.Equal(t, "https://docs", (*got.Result).(*pipelineservice.SpecResult).DocumentationURL)
	})

	t.Run("completes without a result payload", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{})
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{TaskID: task.ID, Success: true})
		require.NoError(t, err)

		got := store.tasks.get(task.ID)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusCompleted, got.Status)
		assert.Nil(t, got.Result)
	})

	t.Run("fails a task with the reported error", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{})
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:       task.ID,
			ErrorMessage: "connection refused",
		})
		require.NoError(t, err)

		got := store.tasks.get(task.ID)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusFailed, got.Status)
		require.NotNil(t, got.ErrorMessage)
		assert.Equal(t, "connection refused", *got.ErrorMessage)
		require.NotNil(t, got.CompletedAt)
		assert.Nil(t, got.Result)
	})

	t.Run("deletes secrets owned by a direct-config check", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{Config: strPtr("{}")})
		store := newTaskStore(task)
		secrets := newFakeSecrets()

		err := pipelinetasks.NewReportTaskResult(store, secrets, testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:       task.ID,
			ErrorMessage: "connection refused",
		})
		require.NoError(t, err)
		assert.Equal(t, []uuid.UUID{task.ID}, secrets.deletedOwners())
	})

	t.Run("secret deletion failure does not fail the report", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{Config: strPtr("{}")})
		store := newTaskStore(task)
		secrets := newFakeSecrets()
		secrets.deleteErr = errBoom

		err := pipelinetasks.NewReportTaskResult(store, secrets, testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{TaskID: task.ID, Success: true})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusCompleted, store.tasks.get(task.ID).Status)
	})

	t.Run("leaves secrets alone for entity checks", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{SourceID: uuidPtr(uuid.New())})
		store := newTaskStore(task)
		secrets := newFakeSecrets()

		err := pipelinetasks.NewReportTaskResult(store, secrets, testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{TaskID: task.ID, Success: true})
		require.NoError(t, err)
		assert.Empty(t, secrets.deletedOwners())
	})

	t.Run("unknown task returns not found", func(t *testing.T) {
		err := pipelinetasks.NewReportTaskResult(newTaskStore(), newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  uuid.New(),
			Success: true,
		})
		require.ErrorIs(t, err, pipelineservice.ErrConnectorTaskNotFound)
	})

	t.Run("malformed result leaves the task untouched", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{})
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{`),
		})
		require.Error(t, err)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusRunning, store.tasks.get(task.ID).Status)
	})

	t.Run("malformed check and spec results are rejected", func(t *testing.T) {
		for _, taskType := range []pipelineservice.ConnectorTaskType{pipelineservice.ConnectorTaskTypeCheck, pipelineservice.ConnectorTaskTypeSpec} {
			task := runningTask(ws, taskType, nil)
			store := newTaskStore(task)

			err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
				TaskID:  task.ID,
				Success: true,
				Result:  []byte(`[]`),
			})
			require.Error(t, err, taskType.String())
		}
	})

	t.Run("unknown task type result is rejected", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskType(0), nil)
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{}`),
		})
		require.ErrorContains(t, err, "unknown task type")
	})

	t.Run("propagates update error", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{})
		store := newTaskStore(task)
		store.tasks.updateErr = errBoom

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{TaskID: task.ID, Success: true})
		require.ErrorIs(t, err, errBoom)
	})

	t.Run("successful discover persists the next catalog version", func(t *testing.T) {
		srcID := uuid.New()
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{SourceID: srcID})
		store := newTaskStore(task)
		store.catalogs.records = []*pipelineservice.CatalogDiscovery{
			{ID: uuid.New(), SourceID: srcID, Version: 2},
			{ID: uuid.New(), SourceID: uuid.New(), Version: 9},
		}

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{"Catalog":"{\"streams\":[]}"}`),
		})
		require.NoError(t, err)

		require.Len(t, store.catalogs.records, 3)
		created := store.catalogs.records[2]
		assert.Equal(t, srcID, created.SourceID)
		assert.Equal(t, 3, created.Version)
		assert.JSONEq(t, `{"streams":[]}`, created.CatalogJSON)
	})

	t.Run("first discover starts at version one", func(t *testing.T) {
		srcID := uuid.New()
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{SourceID: srcID})
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{"Catalog":"{}"}`),
		})
		require.NoError(t, err)
		require.Len(t, store.catalogs.records, 1)
		assert.Equal(t, 1, store.catalogs.records[0].Version)
	})

	t.Run("empty discover catalog is not persisted", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{SourceID: uuid.New()})
		store := newTaskStore(task)

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{"Catalog":""}`),
		})
		require.NoError(t, err)
		assert.Empty(t, store.catalogs.records)
	})

	t.Run("catalog persistence failure does not fail the report", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeDiscover, &pipelineservice.DiscoverPayload{SourceID: uuid.New()})
		store := newTaskStore(task)
		store.catalogs.createErr = errBoom

		err := pipelinetasks.NewReportTaskResult(store, newFakeSecrets(), testLogger()).Execute(ctx, pipelinetasks.ReportTaskResultParams{
			TaskID:  task.ID,
			Success: true,
			Result:  []byte(`{"Catalog":"{}"}`),
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusCompleted, store.tasks.get(task.ID).Status)
	})
}

func TestFailStaleTasks(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()
	cfg := pipelineservice.Config{
		ConnectorTaskPendingTimeout: 5 * time.Minute,
		ConnectorTaskRunningTimeout: 10 * time.Minute,
	}

	t.Run("fails stale pending and orphaned running tasks, keeps fresh ones", func(t *testing.T) {
		stalePending := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		stalePending.CreatedAt = time.Now().Add(-time.Hour)

		freshPending := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)

		staleRunning := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, nil)
		staleRunning.UpdatedAt = time.Now().Add(-time.Hour)

		freshRunning := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, nil)
		freshRunning.CreatedAt = time.Now().Add(-time.Hour)

		store := newTaskStore(stalePending, freshPending, staleRunning, freshRunning)

		require.NoError(t, pipelinetasks.NewFailStaleTasks(store, newFakeSecrets(), cfg).Execute(ctx))

		got := store.tasks.get(stalePending.ID)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusFailed, got.Status)
		assert.Equal(t, "timed out waiting for executor", *got.ErrorMessage)
		assert.NotNil(t, got.CompletedAt)

		got = store.tasks.get(staleRunning.ID)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusFailed, got.Status)
		assert.Equal(t, "executor did not report result (orphaned)", *got.ErrorMessage)

		assert.Equal(t, pipelineservice.ConnectorTaskStatusPending, store.tasks.get(freshPending.ID).Status)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusRunning, store.tasks.get(freshRunning.ID).Status)
	})

	t.Run("propagates lookup error", func(t *testing.T) {
		store := newTaskStore()
		store.tasks.findErr = errBoom

		require.ErrorIs(t, pipelinetasks.NewFailStaleTasks(store, newFakeSecrets(), cfg).Execute(ctx), errBoom)
	})

	t.Run("deletes secrets of failed direct-config checks only", func(t *testing.T) {
		directCheck := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{Config: strPtr("{}")})
		directCheck.UpdatedAt = time.Now().Add(-time.Hour)

		staleSpec := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, &pipelineservice.SpecPayload{})
		staleSpec.CreatedAt = time.Now().Add(-time.Hour)

		freshCheck := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{Config: strPtr("{}")})

		store := newTaskStore(directCheck, staleSpec, freshCheck)
		secrets := newFakeSecrets()

		require.NoError(t, pipelinetasks.NewFailStaleTasks(store, secrets, cfg).Execute(ctx))
		assert.Equal(t, []uuid.UUID{directCheck.ID}, secrets.deletedOwners())
	})

	t.Run("reports secret deletion failure after committing the timeouts", func(t *testing.T) {
		directCheck := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{Config: strPtr("{}")})
		directCheck.UpdatedAt = time.Now().Add(-time.Hour)

		store := newTaskStore(directCheck)
		secrets := newFakeSecrets()
		secrets.deleteErr = errBoom

		require.ErrorIs(t, pipelinetasks.NewFailStaleTasks(store, secrets, cfg).Execute(ctx), errBoom)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusFailed, store.tasks.get(directCheck.ID).Status)
	})

	t.Run("propagates update error for pending task", func(t *testing.T) {
		stale := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		stale.CreatedAt = time.Now().Add(-time.Hour)
		store := newTaskStore(stale)
		store.tasks.updateErr = errBoom

		require.ErrorIs(t, pipelinetasks.NewFailStaleTasks(store, newFakeSecrets(), cfg).Execute(ctx), errBoom)
	})

	t.Run("propagates update error for running task", func(t *testing.T) {
		stale := runningTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		stale.UpdatedAt = time.Now().Add(-time.Hour)
		store := newTaskStore(stale)
		store.tasks.updateErr = errBoom

		require.ErrorIs(t, pipelinetasks.NewFailStaleTasks(store, newFakeSecrets(), cfg).Execute(ctx), errBoom)
	})
}

func TestRetainTasks(t *testing.T) {
	ws := uuid.New()
	ctx := context.Background()

	t.Run("rejects retention below one hour", func(t *testing.T) {
		_, err := pipelinetasks.NewRetainTasks(newTaskStore(), newFakeSecrets(), pipelineservice.Config{ConnectorTaskRetention: time.Minute})
		require.Error(t, err)
	})

	t.Run("deletes old finished tasks and keeps recent or unfinished ones", func(t *testing.T) {
		old := time.Now().Add(-48 * time.Hour)
		recent := time.Now().Add(-time.Hour)

		oldCompleted := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		oldCompleted.Status = pipelineservice.ConnectorTaskStatusCompleted
		oldCompleted.CompletedAt = &old

		oldFailed := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		oldFailed.Status = pipelineservice.ConnectorTaskStatusFailed
		oldFailed.CompletedAt = &old

		recentCompleted := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		recentCompleted.Status = pipelineservice.ConnectorTaskStatusCompleted
		recentCompleted.CompletedAt = &recent

		oldRunning := runningTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		oldRunning.CreatedAt = old

		store := newTaskStore(oldCompleted, oldFailed, recentCompleted, oldRunning)

		uc, err := pipelinetasks.NewRetainTasks(store, newFakeSecrets(), pipelineservice.Config{ConnectorTaskRetention: 24 * time.Hour})
		require.NoError(t, err)
		require.NoError(t, uc.Execute(ctx))

		assert.Nil(t, store.tasks.get(oldCompleted.ID))
		assert.Nil(t, store.tasks.get(oldFailed.ID))
		assert.NotNil(t, store.tasks.get(recentCompleted.ID))
		assert.NotNil(t, store.tasks.get(oldRunning.ID))
	})

	t.Run("deletes secrets of expired direct-config checks before their rows", func(t *testing.T) {
		old := time.Now().Add(-48 * time.Hour)

		directCheck := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{Config: strPtr("{}")})
		directCheck.Status = pipelineservice.ConnectorTaskStatusCompleted
		directCheck.CompletedAt = &old

		sourceCheck := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{SourceID: uuidPtr(uuid.New())})
		sourceCheck.Status = pipelineservice.ConnectorTaskStatusFailed
		sourceCheck.CompletedAt = &old

		store := newTaskStore(directCheck, sourceCheck)
		secrets := newFakeSecrets()

		uc, err := pipelinetasks.NewRetainTasks(store, secrets, pipelineservice.Config{ConnectorTaskRetention: time.Hour})
		require.NoError(t, err)
		require.NoError(t, uc.Execute(ctx))

		assert.Equal(t, []uuid.UUID{directCheck.ID}, secrets.deletedOwners())
		assert.Nil(t, store.tasks.get(directCheck.ID))
		assert.Nil(t, store.tasks.get(sourceCheck.ID))
	})

	t.Run("keeps rows when secret deletion fails", func(t *testing.T) {
		old := time.Now().Add(-48 * time.Hour)

		directCheck := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, &pipelineservice.CheckPayload{Config: strPtr("{}")})
		directCheck.Status = pipelineservice.ConnectorTaskStatusCompleted
		directCheck.CompletedAt = &old

		store := newTaskStore(directCheck)
		secrets := newFakeSecrets()
		secrets.deleteErr = errBoom

		uc, err := pipelinetasks.NewRetainTasks(store, secrets, pipelineservice.Config{ConnectorTaskRetention: time.Hour})
		require.NoError(t, err)
		require.ErrorIs(t, uc.Execute(ctx), errBoom)
		assert.NotNil(t, store.tasks.get(directCheck.ID))
	})

	t.Run("propagates delete error", func(t *testing.T) {
		old := time.Now().Add(-48 * time.Hour)
		expired := pendingTask(ws, pipelineservice.ConnectorTaskTypeSpec, nil)
		expired.Status = pipelineservice.ConnectorTaskStatusCompleted
		expired.CompletedAt = &old

		store := newTaskStore(expired)
		store.tasks.deleteErr = errBoom

		uc, err := pipelinetasks.NewRetainTasks(store, newFakeSecrets(), pipelineservice.Config{ConnectorTaskRetention: time.Hour})
		require.NoError(t, err)
		require.ErrorIs(t, uc.Execute(ctx), errBoom)
	})
}

func TestWaitForTaskResult(t *testing.T) {
	ws := uuid.New()
	cfg := pipelineservice.Config{
		ConnectorTaskPollInterval: time.Millisecond,
		ConnectionCheckTimeout:    time.Second,
	}

	t.Run("returns immediately for an already finished task", func(t *testing.T) {
		var result pipelineservice.ConnectorTaskResult = &pipelineservice.CheckResult{Success: true}
		task := pendingTask(ws, pipelineservice.ConnectorTaskTypeCheck, nil)
		task.Status = pipelineservice.ConnectorTaskStatusCompleted
		task.Result = &result
		store := newTaskStore(task)

		got, err := pipelinetasks.NewWaitForTaskResult(store, cfg).Execute(context.Background(), pipelinetasks.WaitForTaskResultParams{TaskID: task.ID, WorkspaceID: ws})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusCompleted, got.Status)
		assert.Equal(t, result, got.Result)
	})

	t.Run("polls until the task fails", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, nil)
		store := newTaskStore(task)

		go func() {
			time.Sleep(10 * time.Millisecond)
			store.tasks.set(task.ID, func(rec *pipelineservice.ConnectorTask) {
				rec.Status = pipelineservice.ConnectorTaskStatusFailed
				rec.ErrorMessage = strPtr("bad credentials")
			})
		}()

		got, err := pipelinetasks.NewWaitForTaskResult(store, cfg).Execute(context.Background(), pipelinetasks.WaitForTaskResultParams{TaskID: task.ID, WorkspaceID: ws})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusFailed, got.Status)
		assert.Equal(t, "bad credentials", got.ErrorMessage)
		assert.Nil(t, got.Result)
	})

	t.Run("stops on context deadline", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, nil)
		store := newTaskStore(task)

		ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
		defer cancel()

		_, err := pipelinetasks.NewWaitForTaskResult(store, cfg).Execute(ctx, pipelinetasks.WaitForTaskResultParams{TaskID: task.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, context.DeadlineExceeded)
	})

	t.Run("stops on configured timeout", func(t *testing.T) {
		task := runningTask(ws, pipelineservice.ConnectorTaskTypeCheck, nil)
		store := newTaskStore(task)

		uc := pipelinetasks.NewWaitForTaskResult(store, pipelineservice.Config{
			ConnectorTaskPollInterval: time.Millisecond,
			ConnectionCheckTimeout:    20 * time.Millisecond,
		})

		_, err := uc.Execute(context.Background(), pipelinetasks.WaitForTaskResultParams{TaskID: task.ID, WorkspaceID: ws})
		require.ErrorContains(t, err, "timeout waiting for task")
	})

	t.Run("task in another workspace is not found", func(t *testing.T) {
		task := runningTask(uuid.New(), pipelineservice.ConnectorTaskTypeCheck, nil)
		store := newTaskStore(task)

		_, err := pipelinetasks.NewWaitForTaskResult(store, cfg).Execute(context.Background(), pipelinetasks.WaitForTaskResultParams{TaskID: task.ID, WorkspaceID: ws})
		require.ErrorIs(t, err, pipelineservice.ErrConnectorTaskNotFound)
	})
}
