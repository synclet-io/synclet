package pipelinetasks

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ReportTaskResultParams holds parameters for reporting connector task completion.
type ReportTaskResultParams struct {
	TaskID       uuid.UUID
	Success      bool
	ErrorMessage string
	Result       []byte // JSON-encoded result (type-specific)
}

// ReportTaskResult updates a connector task to completed or failed with the result.
type ReportTaskResult struct {
	storage pipelineservice.Storage
	logger  *logging.Logger
}

// NewReportTaskResult creates a new ReportTaskResult use case.
func NewReportTaskResult(storage pipelineservice.Storage, logger *logging.Logger) *ReportTaskResult {
	return &ReportTaskResult{
		storage: storage,
		logger:  logger,
	}
}

// Execute reports the result of a connector task.
func (uc *ReportTaskResult) Execute(ctx context.Context, params ReportTaskResultParams) error {
	// Find the task by ID (no workspace scoping -- executor is trusted internal caller).
	task, err := uc.storage.ConnectorTasks().First(ctx, &pipelineservice.ConnectorTaskFilter{
		ID: filter.Equals(params.TaskID),
	})
	if err != nil {
		return fmt.Errorf("finding task: %w", err)
	}

	now := time.Now()
	task.CompletedAt = &now
	task.UpdatedAt = now

	if params.Success {
		task.Status = pipelineservice.ConnectorTaskStatusCompleted

		// Parse the result JSON into the appropriate one_of variant based on task type.
		if params.Result != nil {
			result, err := parseTaskResult(task.TaskType, params.Result)
			if err != nil {
				return fmt.Errorf("parsing task result: %w", err)
			}
			task.Result = &result
		}
	} else {
		task.Status = pipelineservice.ConnectorTaskStatusFailed
		task.ErrorMessage = &params.ErrorMessage
	}

	if _, err := uc.storage.ConnectorTasks().Update(ctx, task); err != nil {
		return fmt.Errorf("updating task: %w", err)
	}

	// Persist catalog discovery result when a discover task completes successfully.
	if params.Success && task.TaskType == pipelineservice.ConnectorTaskTypeDiscover {
		uc.persistDiscoverResult(ctx, task)
	}

	return nil
}

// persistDiscoverResult stores the discover result in the CatalogDiscovery table.
func (uc *ReportTaskResult) persistDiscoverResult(ctx context.Context, task *pipelineservice.ConnectorTask) {
	discoverPayload, ok := task.Payload.(*pipelineservice.DiscoverPayload)
	if !ok || task.Result == nil {
		return
	}

	discoverResult, ok := (*task.Result).(*pipelineservice.DiscoverResult)
	if !ok || discoverResult.Catalog == "" {
		return
	}

	// Compute next version number.
	nextVersion := 1

	latest, err := uc.storage.CatalogDiscoverys().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(discoverPayload.SourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err == nil {
		nextVersion = latest.Version + 1
	}

	record := &pipelineservice.CatalogDiscovery{
		ID:           uuid.New(),
		SourceID:     discoverPayload.SourceID,
		Version:      nextVersion,
		CatalogJSON:  discoverResult.Catalog,
		DiscoveredAt: time.Now(),
	}

	if _, err := uc.storage.CatalogDiscoverys().Create(ctx, record); err != nil {
		uc.logger.WithError(err).Error(ctx, "failed to persist catalog discovery result")
	}
}

// parseTaskResult parses JSON result bytes into the appropriate one_of variant.
func parseTaskResult(taskType pipelineservice.ConnectorTaskType, data []byte) (pipelineservice.ConnectorTaskResult, error) {
	switch taskType {
	case pipelineservice.ConnectorTaskTypeCheck:
		var r pipelineservice.CheckResult
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("unmarshaling check result: %w", err)
		}
		return &r, nil
	case pipelineservice.ConnectorTaskTypeSpec:
		var r pipelineservice.SpecResult
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("unmarshaling spec result: %w", err)
		}
		return &r, nil
	case pipelineservice.ConnectorTaskTypeDiscover:
		var r pipelineservice.DiscoverResult
		if err := json.Unmarshal(data, &r); err != nil {
			return nil, fmt.Errorf("unmarshaling discover result: %w", err)
		}
		return &r, nil
	default:
		return nil, fmt.Errorf("unknown task type: %s", taskType)
	}
}
