package pipelinetasks_test

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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

type taskStore struct {
	pipelineservice.Storage
	tasks *taskStorage
}

func (s *taskStore) ConnectorTasks() pipelineservice.ConnectorTasksStorage {
	return s.tasks
}

type taskStorage struct {
	pipelineservice.ConnectorTasksStorage
	records map[uuid.UUID]*pipelineservice.ConnectorTask
}

func (s *taskStorage) First(_ context.Context, f *pipelineservice.ConnectorTaskFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConnectorTask, error) {
	for _, rec := range s.records {
		if f.ID != nil {
			if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.ID != eq.Value {
				continue
			}
		}

		if f.WorkspaceID != nil {
			if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && rec.WorkspaceID != eq.Value {
				continue
			}
		}

		cp := *rec

		return &cp, nil
	}

	return nil, pipelineservice.ErrConnectorTaskNotFound
}

func newTaskStore(records ...*pipelineservice.ConnectorTask) *taskStore {
	m := make(map[uuid.UUID]*pipelineservice.ConnectorTask, len(records))
	for _, r := range records {
		m[r.ID] = r
	}

	return &taskStore{tasks: &taskStorage{records: m}}
}

func strPtr(s string) *string {
	return &s
}

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
