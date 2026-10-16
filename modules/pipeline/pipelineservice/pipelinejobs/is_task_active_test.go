package pipelinejobs

import (
	"context"
	"errors"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type taskActiveStorage struct {
	pipelineservice.Storage
	task *pipelineservice.ConnectorTask
	err  error
}

func (s *taskActiveStorage) ConnectorTasks() pipelineservice.ConnectorTasksStorage {
	return &taskActiveTasks{s: s}
}

type taskActiveTasks struct {
	pipelineservice.ConnectorTasksStorage
	s *taskActiveStorage
}

func (t *taskActiveTasks) First(_ context.Context, _ *pipelineservice.ConnectorTaskFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.ConnectorTask, error) {
	return t.s.task, t.s.err
}

func TestIsTaskActive(t *testing.T) {
	tests := []struct {
		name    string
		storage *taskActiveStorage
		active  bool
		wantErr bool
	}{
		{name: "running task is active", storage: &taskActiveStorage{task: &pipelineservice.ConnectorTask{Status: pipelineservice.ConnectorTaskStatusRunning}}, active: true},
		{name: "completed task is inactive", storage: &taskActiveStorage{task: &pipelineservice.ConnectorTask{Status: pipelineservice.ConnectorTaskStatusCompleted}}},
		{name: "missing task is inactive", storage: &taskActiveStorage{err: pipelineservice.ErrConnectorTaskNotFound}},
		{name: "lookup failure is returned", storage: &taskActiveStorage{err: errors.New("connection refused")}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			active, err := NewIsTaskActive(tt.storage).Execute(context.Background(), IsTaskActiveParams{ID: uuid.New()})
			if tt.wantErr {
				require.Error(t, err)

				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.active, active)
		})
	}
}
