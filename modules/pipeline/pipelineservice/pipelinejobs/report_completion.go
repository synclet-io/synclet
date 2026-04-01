package pipelinejobs

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ReportCompletionParams holds parameters for reporting job completion from orchestrator.
type ReportCompletionParams struct {
	JobID        uuid.UUID
	ConnectionID uuid.UUID
	Success      bool
	ErrorMessage string
	RecordsRead  int64
	BytesSynced  int64
	DurationMs   int64
}

// ReportCompletion handles job completion reports from the orchestrator.
// It updates the job status and emits sync events.
type ReportCompletion struct {
	wg              sync.WaitGroup
	updateJobStatus *UpdateJobStatus
	eventEmitter    pipelineservice.SyncEventEmitter
	storage         pipelineservice.Storage
	cleanupOldJobs  *CleanupOldJobs
	logger          *logging.Logger
}

// NewReportCompletion creates a new ReportCompletion use case.
func NewReportCompletion(
	updateJobStatus *UpdateJobStatus,
	eventEmitter pipelineservice.SyncEventEmitter,
	storage pipelineservice.Storage,
	cleanupOldJobs *CleanupOldJobs,
	logger *logging.Logger,
) *ReportCompletion {
	return &ReportCompletion{
		updateJobStatus: updateJobStatus,
		eventEmitter:    eventEmitter,
		storage:         storage,
		cleanupOldJobs:  cleanupOldJobs,
		logger:          logger.Named("report-completion"),
	}
}

// Wait blocks until all background goroutines spawned by Execute complete.
func (uc *ReportCompletion) Wait() {
	uc.wg.Wait()
}

// Execute updates job status and emits the appropriate sync event.
func (uc *ReportCompletion) Execute(ctx context.Context, params ReportCompletionParams) error {
	stats := &pipelineservice.SyncStats{
		RecordsRead: params.RecordsRead,
		BytesSynced: params.BytesSynced,
		Duration:    time.Duration(params.DurationMs) * time.Millisecond,
	}

	var syncErr error
	if !params.Success {
		syncErr = fmt.Errorf("%s", params.ErrorMessage)
	}

	if err := uc.updateJobStatus.Execute(ctx, UpdateJobStatusParams{
		ID:        params.JobID,
		SyncErr:   syncErr,
		SyncStats: stats,
	}); err != nil {
		return fmt.Errorf("updating job status: %w", err)
	}

	// Look up connection to get WorkspaceID for event emission and retention cleanup.
	var workspaceID uuid.UUID

	conn, connErr := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(params.ConnectionID),
	})
	if connErr != nil {
		uc.logger.WithError(connErr).Warn(ctx, "failed to look up connection for workspace ID, using nil")
	} else {
		workspaceID = conn.WorkspaceID
	}

	// Emit sync events asynchronously with WaitGroup tracking for graceful shutdown.
	if params.Success {
		uc.wg.Add(1)

		go func() {
			defer uc.wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if emitErr := uc.eventEmitter.EmitSyncCompleted(ctx, pipelineservice.SyncCompletedEvent{
				ConnectionID: params.ConnectionID,
				WorkspaceID:  workspaceID,
				JobID:        params.JobID,
				RecordsRead:  params.RecordsRead,
				Duration:     stats.Duration,
			}); emitErr != nil {
				uc.logger.WithError(emitErr).Warn(ctx, "emit sync.completed event failed")
			}
		}()
	} else {
		uc.wg.Add(1)

		go func() {
			defer uc.wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if emitErr := uc.eventEmitter.EmitSyncFailed(ctx, pipelineservice.SyncFailedEvent{
				ConnectionID: params.ConnectionID,
				WorkspaceID:  workspaceID,
				JobID:        params.JobID,
				Error:        params.ErrorMessage,
			}); emitErr != nil {
				uc.logger.WithError(emitErr).Warn(ctx, "emit sync.failed event failed")
			}
		}()
	}

	uc.logger.Info(ctx, "job completed",
		"job_id", params.JobID.String(),
		"success", params.Success,
		"records_read", params.RecordsRead)

	// Trigger retention cleanup for this workspace (best-effort, non-blocking).
	if conn != nil {
		uc.wg.Add(1)

		go func() {
			defer uc.wg.Done()

			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()

			if cleanupErr := uc.cleanupOldJobs.ExecuteForWorkspace(ctx, conn.WorkspaceID); cleanupErr != nil {
				uc.logger.WithError(cleanupErr).Warn(ctx, "post-completion retention cleanup failed")
			}
		}()
	}

	return nil
}
