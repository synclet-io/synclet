package pipelinestats_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
)

func TestComputeHealthStatus(t *testing.T) {
	// Use an hourly cron so that last-sync-on-the-hour gives a deterministic
	// 1-hour interval regardless of phase quirks in cron.Schedule.Next.
	now := time.Date(2026, 5, 17, 12, 0, 0, 0, time.UTC)
	hourlyCron := "0 * * * *"

	t.Run("empty schedule is treated as disabled (manual-only)", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", "", now.Add(-time.Hour), now)
		assert.Equal(t, pipelineservice.HealthDisabled, got)
	})

	t.Run("last job failed produces failing regardless of timing", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("Failed", hourlyCron, now.Add(-10*time.Minute), now)
		assert.Equal(t, pipelineservice.HealthFailing, got)
	})

	t.Run("invalid cron expression produces warning", func(t *testing.T) {
		got := pipelinestats.ComputeHealthStatus("completed", "every blue moon", now.Add(-time.Hour), now)
		assert.Equal(t, pipelineservice.HealthWarning, got)
	})

	t.Run("within one expected interval is healthy", func(t *testing.T) {
		// Last sync 30 minutes ago on an hourly schedule — well inside one interval.
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, now.Add(-30*time.Minute), now)
		assert.Equal(t, pipelineservice.HealthHealthy, got)
	})

	t.Run("elapsed between 2× and 3× the interval is warning", func(t *testing.T) {
		// 2h30m elapsed on a 1h schedule — 2.5× the interval. Last sync at 09:30
		// gives sched.Next(09:30)=10:00 → interval=30m, elapsed=2h30m → 5× ✗.
		// Use 09:00 (on a boundary) so interval comes out as a full 1h.
		lastSync := time.Date(2026, 5, 17, 9, 0, 0, 0, time.UTC)
		nowAt := lastSync.Add(2*time.Hour + 30*time.Minute) // 2.5× interval
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, lastSync, nowAt)
		assert.Equal(t, pipelineservice.HealthWarning, got)
	})

	t.Run("elapsed beyond 3× the interval is failing", func(t *testing.T) {
		lastSync := time.Date(2026, 5, 17, 8, 0, 0, 0, time.UTC)
		nowAt := lastSync.Add(4 * time.Hour) // 4× interval
		got := pipelinestats.ComputeHealthStatus("completed", hourlyCron, lastSync, nowAt)
		assert.Equal(t, pipelineservice.HealthFailing, got)
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
