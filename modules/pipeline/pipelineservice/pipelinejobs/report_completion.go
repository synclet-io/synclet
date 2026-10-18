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
	// SourceExitCode and DestExitCode are the connectors' exit codes; 0 means
	// a clean exit or none reported.
	SourceExitCode int
	DestExitCode   int
}

// ReportCompletion records a job's final result and triggers retention cleanup
// for the job's workspace.
type ReportCompletion struct {
	wg              sync.WaitGroup
	updateJobStatus *UpdateJobStatus
	storage         pipelineservice.Storage
	cleanupOldJobs  *CleanupOldJobs
	logger          *logging.Logger
	cfg             pipelineservice.Config
}

// NewReportCompletion creates a new ReportCompletion use case.
func NewReportCompletion(
	updateJobStatus *UpdateJobStatus,
	storage pipelineservice.Storage,
	cleanupOldJobs *CleanupOldJobs,
	logger *logging.Logger,
	cfg pipelineservice.Config,
) *ReportCompletion {
	return &ReportCompletion{
		updateJobStatus: updateJobStatus,
		storage:         storage,
		cleanupOldJobs:  cleanupOldJobs,
		logger:          logger.Named("report-completion"),
		cfg:             cfg,
	}
}

// Wait blocks until all background goroutines spawned by Execute complete.
func (uc *ReportCompletion) Wait() {
	uc.wg.Wait()
}

// Execute updates the job status and starts best-effort retention cleanup in
// the background.
func (uc *ReportCompletion) Execute(ctx context.Context, params ReportCompletionParams) error {
	stats := &pipelineservice.SyncStats{
		RecordsRead: params.RecordsRead,
		BytesSynced: params.BytesSynced,
		Duration:    time.Duration(params.DurationMs) * time.Millisecond,
	}

	var syncErr error
	if !params.Success {
		syncErr = pipelineservice.NewReportedFailureError(params.ErrorMessage, params.SourceExitCode, params.DestExitCode)
	}

	if err := uc.updateJobStatus.Execute(ctx, UpdateJobStatusParams{
		ID:        params.JobID,
		SyncErr:   syncErr,
		SyncStats: stats,
	}); err != nil {
		return fmt.Errorf("updating job status: %w", err)
	}

	uc.logger.Info(ctx, "job completed",
		"job_id", params.JobID.String(),
		"success", params.Success,
		"records_read", params.RecordsRead)

	// Trigger retention cleanup for this workspace (best-effort, non-blocking).
	conn, connErr := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(params.ConnectionID),
	})
	if connErr != nil || conn == nil {
		uc.logger.WithError(connErr).Warn(ctx, "could not load connection for retention cleanup; skipping")

		return nil
	}

	uc.wg.Add(1)

	go func() { //nolint:gosec // intentional background context for async cleanup
		defer uc.wg.Done()

		bgCtx, cancel := context.WithTimeout(context.Background(), uc.cfg.EventEmitTimeout)
		defer cancel()

		if cleanupErr := uc.cleanupOldJobs.ExecuteForWorkspace(bgCtx, conn.WorkspaceID); cleanupErr != nil {
			uc.logger.WithError(cleanupErr).Warn(bgCtx, "post-completion retention cleanup failed")
		}
	}()

	return nil
}
