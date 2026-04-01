package pipelinesync

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockK8sJobCreator implements K8sJobCreator for testing.
type mockK8sJobCreator struct {
	createSyncJobFn func(ctx context.Context, opts K8sSyncJobOptions) (string, error)
	lastOpts        *K8sSyncJobOptions
}

func (m *mockK8sJobCreator) CreateSyncJob(ctx context.Context, opts K8sSyncJobOptions) (string, error) {
	m.lastOpts = &opts
	if m.createSyncJobFn != nil {
		return m.createSyncJobFn(ctx, opts)
	}

	return "synclet-sync-test-123", nil
}

func TestK8sSyncWorker_Execute_NoJobAvailable(t *testing.T) {
	backend := &mockBackend{claimResult: nil}

	worker := &K8sSyncWorker{
		backend:  backend,
		workerID: "test-worker",
	}

	err := worker.Execute(context.Background())
	require.NoError(t, err)
}

func TestK8sSyncWorker_K8sSyncJobOptions(t *testing.T) {
	// Verify K8sSyncJobOptions struct has all required fields for 3-container pod spec.
	opts := K8sSyncJobOptions{
		JobID:          uuid.New(),
		ConnectionID:   uuid.New(),
		SourceID:       uuid.New(),
		DestinationID:  uuid.New(),
		SourceImage:    "airbyte/source-postgres:0.1.0",
		DestImage:      "airbyte/destination-postgres:0.1.0",
		SourceConfig:   []byte(`{"host":"localhost"}`),
		DestConfig:     []byte(`{"host":"localhost"}`),
		SourceCatalog:  []byte(`{"streams":[]}`),
		DestCatalog:    []byte(`{"streams":[]}`),
		State:          []byte(`[{"type":"STREAM"}]`),
		RuntimeConfig:  `{"memory_limit":2147483648}`,
		InternalAPIURL: "http://synclet:8081",
	}

	assert.NotEqual(t, uuid.Nil, opts.JobID)
	assert.NotEqual(t, uuid.Nil, opts.ConnectionID)
	assert.NotEqual(t, uuid.Nil, opts.SourceID)
	assert.NotEqual(t, uuid.Nil, opts.DestinationID)
	assert.NotEmpty(t, opts.SourceImage)
	assert.NotEmpty(t, opts.DestImage)
	assert.NotEmpty(t, opts.SourceConfig)
	assert.NotEmpty(t, opts.DestConfig)
	assert.NotEmpty(t, opts.SourceCatalog)
	assert.NotEmpty(t, opts.DestCatalog)
	assert.NotEmpty(t, opts.State)
	assert.NotEmpty(t, opts.RuntimeConfig)
	assert.NotEmpty(t, opts.InternalAPIURL)
}

func TestK8sSyncWorker_CreateSyncJobFailure(t *testing.T) {
	// Verify that when k8sRunner.CreateSyncJob fails, the mock captures the error.
	runner := &mockK8sJobCreator{
		createSyncJobFn: func(ctx context.Context, opts K8sSyncJobOptions) (string, error) {
			return "", errors.New("k8s api unavailable")
		},
	}

	_, err := runner.CreateSyncJob(context.Background(), K8sSyncJobOptions{
		JobID:          uuid.New(),
		InternalAPIURL: "http://synclet:8081",
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "k8s api unavailable")
}

func TestK8sSyncWorker_CreateSyncJobSuccess(t *testing.T) {
	// Verify that successful creation returns the K8s job name.
	runner := &mockK8sJobCreator{
		createSyncJobFn: func(ctx context.Context, opts K8sSyncJobOptions) (string, error) {
			return "synclet-sync-" + opts.JobID.String()[:8], nil
		},
	}

	jobID := uuid.New()
	name, err := runner.CreateSyncJob(context.Background(), K8sSyncJobOptions{
		JobID:          jobID,
		SourceImage:    "airbyte/source-postgres:0.1.0",
		DestImage:      "airbyte/destination-postgres:0.1.0",
		InternalAPIURL: "http://synclet:8081",
	})

	require.NoError(t, err)
	assert.Contains(t, name, "synclet-sync-")
	assert.NotNil(t, runner.lastOpts)
	assert.Equal(t, "airbyte/source-postgres:0.1.0", runner.lastOpts.SourceImage)
	assert.Equal(t, "airbyte/destination-postgres:0.1.0", runner.lastOpts.DestImage)
}
