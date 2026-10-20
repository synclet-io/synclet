package pipelinestats_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
)

func TestComputeHealthStatus(t *testing.T) {
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	hourlyCron := "0 * * * *"
	at := func(d time.Duration) *time.Time {
		v := now.Add(d)

		return &v
	}

	t.Run("manual connection with a completed last sync is healthy", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", "", nil, now)
		assert.Equal(t, pipelineservice.HealthHealthy, got)
	})

	t.Run("manual connection with a failed last sync is failing", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("failed", "", nil, now)
		assert.Equal(t, pipelineservice.HealthFailing, got)
	})

	t.Run("last job failed produces failing regardless of timing", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("Failed", hourlyCron, at(time.Hour), now)
		assert.Equal(t, pipelineservice.HealthFailing, got)
	})

	t.Run("invalid cron expression produces warning", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", "every blue moon", at(time.Hour), now)
		assert.Equal(t, pipelineservice.HealthWarning, got)
	})

	t.Run("next run still ahead is healthy", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, at(30*time.Minute), now)
		assert.Equal(t, pipelineservice.HealthHealthy, got)
	})

	t.Run("a schedule just set on a connection last synced long ago is healthy", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", "*/5 * * * *", at(5*time.Minute), now)
		assert.Equal(t, pipelineservice.HealthHealthy, got)
	})

	t.Run("next run late by up to one interval is healthy", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, at(-time.Hour), now)
		assert.Equal(t, pipelineservice.HealthHealthy, got)
	})

	t.Run("next run late by more than one interval is warning", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, at(-2*time.Hour), now)
		assert.Equal(t, pipelineservice.HealthWarning, got)
	})

	t.Run("next run late by more than two intervals is failing", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, at(-3*time.Hour), now)
		assert.Equal(t, pipelineservice.HealthFailing, got)
	})

	t.Run("scheduled connection without a next run is healthy", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, nil, now)
		assert.Equal(t, pipelineservice.HealthHealthy, got)
	})
}

func TestCategorizeFailure(t *testing.T) {
	cases := []struct {
		name string
		msg  string
		want pipelineservice.FailureCategory
	}{
		{"timeout keyword", "operation timeout after 30s", pipelineservice.FailureCategoryTimeout},
		{"deadline exceeded", "context deadline exceeded", pipelineservice.FailureCategoryTimeout},
		{"oom keyword", "container killed (OOM)", pipelineservice.FailureCategoryOOM},
		{"out of memory phrase", "process ran out of memory", pipelineservice.FailureCategoryOOM},
		{"connector error", "source error: invalid credentials", pipelineservice.FailureCategoryConnector},
		{"docker infrastructure", "docker daemon refused connection", pipelineservice.FailureCategoryInfrastructure},
		{"kubernetes infrastructure", "kubernetes pod evicted", pipelineservice.FailureCategoryInfrastructure},
		{"unrecognised falls back to unknown", "some weird thing happened", pipelineservice.FailureCategoryUnknown},
		{"case-insensitive matching", "TIMEOUT", pipelineservice.FailureCategoryTimeout},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got := pipelinestats.CategorizeFailure(tc.msg)
			assert.Equal(t, tc.want, got)
		})
	}
}
