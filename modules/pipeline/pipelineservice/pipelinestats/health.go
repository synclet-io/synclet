package pipelinestats

import (
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ComputeHealthStatus determines a connection's health badge from its last job status,
// cron schedule expression, and time of last sync. This is a pure function for testability.
func ComputeHealthStatus(lastJobStatus, schedule string, lastSyncAt, now time.Time) pipelineservice.Health {
	// No schedule means manual-only connection -- treated as disabled.
	if schedule == "" {
		return pipelineservice.HealthDisabled
	}

	// If the last sync failed, the connection is failing.
	if strings.EqualFold(lastJobStatus, "Failed") || strings.EqualFold(lastJobStatus, "failed") {
		return pipelineservice.HealthFailing
	}

	sched, err := cronParser.Parse(schedule)
	if err != nil {
		// Invalid cron expression -- can't determine health accurately.
		return pipelineservice.HealthWarning
	}

	// Determine expected interval between syncs.
	expectedNext := sched.Next(lastSyncAt)
	interval := expectedNext.Sub(lastSyncAt)

	elapsed := now.Sub(lastSyncAt)

	// More than 3x the expected interval -- failing.
	if elapsed > 3*interval {
		return pipelineservice.HealthFailing
	}

	// More than 2x the expected interval -- warning (overdue).
	if elapsed > 2*interval {
		return pipelineservice.HealthWarning
	}

	return pipelineservice.HealthHealthy
}

// CategorizeFailure classifies a failure error string into a category.
func CategorizeFailure(errorStr string) pipelineservice.FailureCategory {
	lower := strings.ToLower(errorStr)

	switch {
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "deadline exceeded") || strings.Contains(lower, "context deadline"):
		return pipelineservice.FailureCategoryTimeout
	case strings.Contains(lower, "oom") || strings.Contains(lower, "out of memory") || strings.Contains(lower, "memory limit"):
		return pipelineservice.FailureCategoryOOM
	case strings.Contains(lower, "connector") || strings.Contains(lower, "source error") || strings.Contains(lower, "destination error") || strings.Contains(lower, "spec error"):
		return pipelineservice.FailureCategoryConnector
	case strings.Contains(lower, "infrastructure") || strings.Contains(lower, "docker") || strings.Contains(lower, "k8s") || strings.Contains(lower, "kubernetes") || strings.Contains(lower, "container"):
		return pipelineservice.FailureCategoryInfrastructure
	default:
		return pipelineservice.FailureCategoryUnknown
	}
}
