package pipelinestats

import (
	"strings"
	"time"

	"github.com/robfig/cron/v3"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

var cronParser = cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)

// ComputeHealthStatus determines an active connection's health badge from its last job status,
// cron schedule expression, and the time its next scheduled run is due. A failed last job is
// failing. A scheduled connection is overdue only when its next run is late: by more than one
// interval it is a warning, by more than two it is failing. A manual connection (empty
// schedule) is never overdue. This is a pure function for testability.
func ComputeHealthStatus(lastJobStatus, schedule string, nextScheduledAt *time.Time, now time.Time) pipelineservice.Health {
	if strings.EqualFold(lastJobStatus, "failed") {
		return pipelineservice.HealthFailing
	}

	if schedule == "" {
		return pipelineservice.HealthHealthy
	}

	sched, err := cronParser.Parse(schedule)
	if err != nil {
		// Invalid cron expression -- can't determine health accurately.
		return pipelineservice.HealthWarning
	}

	if nextScheduledAt == nil {
		return pipelineservice.HealthHealthy
	}

	interval := sched.Next(*nextScheduledAt).Sub(*nextScheduledAt)
	late := now.Sub(*nextScheduledAt)

	if late > 2*interval {
		return pipelineservice.HealthFailing
	}

	if late > interval {
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
