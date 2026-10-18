package pipelinejobs

import (
	"context"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
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

func TestListExecutors_SilenceTimeout(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		lastSeen time.Time
		timeout  time.Duration
		listed   bool
	}{
		{name: "seen now", lastSeen: now, timeout: 60 * time.Second, listed: true},
		{name: "seen 59s ago", lastSeen: now.Add(-59 * time.Second), timeout: 60 * time.Second, listed: true},
		{name: "seen 61s ago", lastSeen: now.Add(-61 * time.Second), timeout: 60 * time.Second, listed: false},
		{name: "seen 61s ago with a longer timeout", lastSeen: now.Add(-61 * time.Second), timeout: 120 * time.Second, listed: true},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			row := &pipelineservice.Executor{InstanceID: "executor-0", Type: pipelineservice.ExecutorTypeDocker, LastSeenAt: tc.lastSeen}
			uc := NewListExecutors(&executorRegistryStorage{executors: &fakeExecutors{rows: []*pipelineservice.Executor{row}}}, pipelineservice.Config{ExecutorSilenceTimeout: tc.timeout})
			uc.now = func() time.Time { return now }

			got, err := uc.Execute(context.Background())
			require.NoError(t, err)

			if tc.listed {
				assert.Equal(t, []*pipelineservice.Executor{row}, got)
			} else {
				assert.Empty(t, got)
			}
		})
	}
}

func TestRegisterExecutor_Upserts(t *testing.T) {
	now := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	executors := &fakeExecutors{}
	uc := NewRegisterExecutor(&executorRegistryStorage{executors: executors})
	uc.now = func() time.Time { return now }

	err := uc.Execute(context.Background(), RegisterExecutorParams{
		InstanceID:       "synclet-executor-0",
		Type:             pipelineservice.ExecutorTypeK8s,
		Version:          "v0.5.0",
		Capabilities:     pipelineservice.ExecutorCapabilities{NativeDestinations: []string{"postgres"}},
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{CPURequest: "750m", CPULimit: "750m", CPUExecutorDefault: true},
	})
	require.NoError(t, err)
	require.Len(t, executors.upserted, 1)

	got := executors.upserted[0]
	assert.NotZero(t, got.ID)
	assert.Equal(t, "synclet-executor-0", got.InstanceID)
	assert.Equal(t, pipelineservice.ExecutorTypeK8s, got.Type)
	assert.Equal(t, "v0.5.0", got.Version)
	assert.Equal(t, []string{"postgres"}, got.Capabilities.NativeDestinations)
	assert.True(t, got.ResourceDefaults.CPUExecutorDefault)
	assert.Equal(t, now, got.FirstSeenAt)
	assert.Equal(t, now, got.LastSeenAt)
}
