package pipelineconnect

import (
	"context"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

type executorRegistryStorage struct {
	pipelineservice.Storage
	executors *fakeExecutors
}

func (s *executorRegistryStorage) Executors() pipelineservice.ExecutorsStorage { return s.executors }

type fakeExecutors struct {
	pipelineservice.ExecutorsStorage
	rows     []*pipelineservice.Executor
	upserted []*pipelineservice.Executor
}

func (e *fakeExecutors) Find(context.Context, *pipelineservice.ExecutorFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Executor, error) {
	return e.rows, nil
}

func (e *fakeExecutors) Upsert(_ context.Context, executor *pipelineservice.Executor) error {
	e.upserted = append(e.upserted, executor)

	return nil
}

func TestExecutorHandler_ExecutorHeartbeat(t *testing.T) {
	valid := func() *executorv1.ExecutorHeartbeatRequest {
		return &executorv1.ExecutorHeartbeatRequest{
			ExecutorId:   "synclet-executor-0",
			ExecutorType: pipelinev1.ExecutorType_EXECUTOR_TYPE_K8S,
			Version:      "v0.5.0",
			Capacity:     0,
			Capabilities: &executorv1.ExecutorCapabilities{NativeDestinations: []string{"postgres"}},
			ResourceDefaults: &pipelinev1.ExecutorResourceDefaults{
				CpuRequest: "750m", CpuLimit: "750m", CpuExecutorDefault: true,
			},
		}
	}

	t.Run("empty executor id is rejected", func(t *testing.T) {
		executors := &fakeExecutors{}
		h := &ExecutorHandler{registerExecutor: pipelinejobs.NewRegisterExecutor(&executorRegistryStorage{executors: executors})}
		req := valid()
		req.ExecutorId = ""

		_, err := h.ExecutorHeartbeat(t.Context(), connect.NewRequest(req))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		assert.Empty(t, executors.upserted)
	})

	t.Run("unknown executor type is rejected", func(t *testing.T) {
		executors := &fakeExecutors{}
		h := &ExecutorHandler{registerExecutor: pipelinejobs.NewRegisterExecutor(&executorRegistryStorage{executors: executors})}
		req := valid()
		req.ExecutorType = pipelinev1.ExecutorType_EXECUTOR_TYPE_UNSPECIFIED

		_, err := h.ExecutorHeartbeat(t.Context(), connect.NewRequest(req))
		require.Error(t, err)
		assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
		assert.Empty(t, executors.upserted)
	})

	t.Run("a valid heartbeat registers the executor", func(t *testing.T) {
		executors := &fakeExecutors{}
		h := &ExecutorHandler{registerExecutor: pipelinejobs.NewRegisterExecutor(&executorRegistryStorage{executors: executors})}

		_, err := h.ExecutorHeartbeat(t.Context(), connect.NewRequest(valid()))
		require.NoError(t, err)
		require.Len(t, executors.upserted, 1)

		got := executors.upserted[0]
		assert.Equal(t, "synclet-executor-0", got.InstanceID)
		assert.Equal(t, pipelineservice.ExecutorTypeK8s, got.Type)
		assert.Equal(t, "v0.5.0", got.Version)
		assert.Equal(t, []string{"postgres"}, got.Capabilities.NativeDestinations)
		assert.Equal(t, pipelineservice.ExecutorResourceDefaults{CPURequest: "750m", CPULimit: "750m", CPUExecutorDefault: true}, got.ResourceDefaults)
		assert.WithinDuration(t, time.Now(), got.LastSeenAt, time.Minute)
	})
}
