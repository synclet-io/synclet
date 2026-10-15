package pipelinetasks_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

func TestGetTaskResult(t *testing.T) {
	ws := uuid.New()

	t.Run("returns status, type, and result for a completed task", func(t *testing.T) {
		var result pipelineservice.ConnectorTaskResult = &pipelineservice.CheckResult{Success: true}
		task := &pipelineservice.ConnectorTask{
			ID:          uuid.New(),
			WorkspaceID: ws,
			TaskType:    pipelineservice.ConnectorTaskTypeCheck,
			Status:      pipelineservice.ConnectorTaskStatusCompleted,
			Result:      &result,
		}
		store := newTaskStore(task)

		uc := pipelinetasks.NewGetTaskResult(store)
		got, err := uc.Execute(context.Background(), pipelinetasks.GetTaskResultParams{
			TaskID:      task.ID,
			WorkspaceID: ws,
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusCompleted, got.Status)
		assert.Equal(t, pipelineservice.ConnectorTaskTypeCheck, got.TaskType)
		assert.NotNil(t, got.Result)
		assert.Empty(t, got.ErrorMessage)
	})

	t.Run("returns error message for a failed task and nil Result", func(t *testing.T) {
		task := &pipelineservice.ConnectorTask{
			ID:           uuid.New(),
			WorkspaceID:  ws,
			TaskType:     pipelineservice.ConnectorTaskTypeDiscover,
			Status:       pipelineservice.ConnectorTaskStatusFailed,
			ErrorMessage: strPtr("bad config"),
		}
		store := newTaskStore(task)

		uc := pipelinetasks.NewGetTaskResult(store)
		got, err := uc.Execute(context.Background(), pipelinetasks.GetTaskResultParams{
			TaskID:      task.ID,
			WorkspaceID: ws,
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectorTaskStatusFailed, got.Status)
		assert.Equal(t, "bad config", got.ErrorMessage)
		assert.Nil(t, got.Result)
	})

	t.Run("refuses cross-workspace lookup", func(t *testing.T) {
		task := &pipelineservice.ConnectorTask{
			ID:          uuid.New(),
			WorkspaceID: uuid.New(),
			TaskType:    pipelineservice.ConnectorTaskTypeSpec,
			Status:      pipelineservice.ConnectorTaskStatusCompleted,
		}
		store := newTaskStore(task)

		uc := pipelinetasks.NewGetTaskResult(store)
		_, err := uc.Execute(context.Background(), pipelinetasks.GetTaskResultParams{
			TaskID:      task.ID,
			WorkspaceID: ws,
		})
		require.Error(t, err)
	})
}
