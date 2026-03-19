package pipelinesync

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// syncExecutor abstracts SyncExecutor for testability.
type syncExecutor interface {
	Execute(ctx context.Context, bundle *SyncBundle) (*pipelineservice.SyncStats, error)
}

// DockerSyncWorker is a jobber that polls for scheduled jobs, claims them,
// and runs sync execution using SyncExecutor (which uses pipelineroute.Run
// with DockerHandler). It uses ExecutorBackend for all server communication
// per D-14. Runs at ~1s interval via the jobber framework.
type DockerSyncWorker struct {
	backend         ExecutorBackend
	executor        syncExecutor
	metrics         *pipelinemetrics.MetricsCollector
	manager         *SyncWorkerManager
	maxSyncDuration time.Duration
	semaphore       chan struct{}
	workerID        string
	logger          *logging.Logger
}

// DockerSyncWorkerParams holds all constructor dependencies for DockerSyncWorker.
type DockerSyncWorkerParams struct {
	Backend           ExecutorBackend
	Executor          *SyncExecutor
	Metrics           *pipelinemetrics.MetricsCollector
	Manager           *SyncWorkerManager
	MaxSyncDuration   time.Duration
	MaxConcurrentJobs int
	Logger            *logging.Logger
}

// NewDockerSyncWorker creates a new DockerSyncWorker with all dependencies.
// The workerID is derived from hostname for container identification per D-11.
func NewDockerSyncWorker(params DockerSyncWorkerParams) *DockerSyncWorker {
	workerID := getWorkerID()

	concurrency := params.MaxConcurrentJobs
	if concurrency <= 0 {
		concurrency = 10
	}

	var logger *logging.Logger
	if params.Logger != nil {
		logger = params.Logger.Named("docker-sync-worker")
	}

	return &DockerSyncWorker{
		backend:         params.Backend,
		executor:        params.Executor,
		metrics:         params.Metrics,
		manager:         params.Manager,
		maxSyncDuration: params.MaxSyncDuration,
		semaphore:       make(chan struct{}, concurrency),
		workerID:        workerID,
		logger:          logger,
	}
}

// Execute polls for and claims a scheduled job, then spawns a goroutine
// to execute it. Returns immediately so the jobber timer resets.
// Checks the concurrency semaphore BEFORE claiming to avoid claiming jobs
// that cannot be executed (job stays in queue for the next poll cycle).
func (w *DockerSyncWorker) Execute(ctx context.Context) error {
	// Check concurrency limit before claiming.
	select {
	case w.semaphore <- struct{}{}:
		// Got a slot -- proceed to claim.
	default:
		// No slot available -- skip this poll cycle.
		return nil
	}

	result, err := w.backend.ClaimJob(ctx, w.workerID)
	if err != nil {
		<-w.semaphore
		return err
	}

	if result == nil {
		<-w.semaphore
		return nil
	}

	if w.metrics != nil {
		w.metrics.ObserveJobDequeued()
	}

	if w.logger != nil {
		w.logger.WithFields(map[string]interface{}{
			"worker_id":     w.workerID,
			"job_id":        result.Job.ID.String(),
			"job_type":      result.Job.JobType.String(),
			"connection_id": result.Job.ConnectionID.String(),
		}).Info(ctx, "claimed job")
	}

	w.manager.RunJob(func(jobCtx context.Context) {
		defer func() { <-w.semaphore }()
		w.executeJob(jobCtx, result)
	})

	return nil
}

// executeJob runs a sync job end-to-end: starts combined heartbeat+cancel
// polling, delegates to SyncExecutor, then handles completion via ExecutorBackend.
func (w *DockerSyncWorker) executeJob(ctx context.Context, result *pipelinejobs.ClaimJobBundleResult) {
	if w.metrics != nil {
		w.metrics.ObserveActiveSyncStarted()
		defer w.metrics.ObserveActiveSyncStopped()
	}

	// Build SyncBundle from ClaimJobBundleResult.
	catalog := &protocol.ConfiguredAirbyteCatalog{}
	if err := json.Unmarshal(result.ConfiguredCatalog, catalog); err != nil {
		w.failJob(ctx, result.Job.ID, result.WorkspaceID, result.ConnectionID, fmt.Errorf("unmarshaling catalog: %w", err))
		return
	}

	bundle := &SyncBundle{
		Job:                   result.Job,
		ConnectionID:          result.ConnectionID,
		WorkspaceID:           result.WorkspaceID,
		SourceID:              result.SourceID,
		DestinationID:         result.DestinationID,
		SourceImage:           result.SourceImage,
		SourceConfig:          result.SourceConfig,
		DestImage:             result.DestImage,
		DestConfig:            result.DestConfig,
		ConfiguredCatalog:     catalog,
		StateBlob:             result.StateBlob,
		SourceRuntimeConfig:   result.SourceRuntimeConfig,
		DestRuntimeConfig:     result.DestRuntimeConfig,
		NamespaceDefinition:   result.NamespaceDefinition,
		CustomNamespaceFormat: nilIfEmpty(result.CustomNamespaceFormat),
		StreamPrefix:          nilIfEmpty(result.StreamPrefix),
	}

	// Apply max sync duration via context.WithTimeout.
	// When timeout fires, context cancellation propagates to executor which stops containers.
	execCtx, cancelExec := context.WithTimeout(ctx, w.maxSyncDuration)
	defer cancelExec()

	// Start combined heartbeat + cancel polling loop (5s interval per D-05).
	go w.heartbeatAndCancel(execCtx, cancelExec, result.Job.ID)

	// Execute the sync with timeout-bounded context.
	stats, execErr := w.executor.Execute(execCtx, bundle)

	// Cancel heartbeat/cancel polling BEFORE updating status to avoid race conditions.
	cancelExec()

	// Distinguish timeout from manual cancellation.
	if execErr != nil && execCtx.Err() == context.DeadlineExceeded {
		reason := fmt.Sprintf("max sync duration exceeded (%s)", w.maxSyncDuration)
		execErr = fmt.Errorf("%s: %w", reason, execErr)
		if w.logger != nil {
			w.logger.WithFields(map[string]interface{}{"worker_id": w.workerID, "job_id": result.Job.ID.String(), "max_duration": w.maxSyncDuration}).Warn(ctx, "job timed out")
		}
	}

	// Check if context was cancelled (cancel detected via heartbeat response).
	// If so, the job was cancelled server-side -- just return without updating status.
	// Only match Canceled, not DeadlineExceeded (timeouts must be marked as failed).
	if execErr != nil && execCtx.Err() == context.Canceled {
		if w.logger != nil {
			w.logger.WithFields(map[string]interface{}{"worker_id": w.workerID, "job_id": result.Job.ID.String()}).Info(ctx, "job cancelled")
		}
		return
	}

	// Update job status via ExecutorBackend.
	var errMsg string
	var recordsRead, bytesSynced, durationMs int64
	if stats != nil {
		recordsRead = stats.RecordsRead
		bytesSynced = stats.BytesSynced
		durationMs = stats.Duration.Milliseconds()
	}
	if execErr != nil {
		errMsg = execErr.Error()
	}

	if completeErr := w.backend.UpdateJobStatus(ctx, UpdateJobStatusParams{
		JobID:        result.Job.ID,
		Success:      execErr == nil,
		ErrorMessage: errMsg,
		RecordsRead:  recordsRead,
		BytesSynced:  bytesSynced,
		DurationMs:   durationMs,
	}); completeErr != nil && w.logger != nil {
		w.logger.WithError(completeErr).WithFields(map[string]interface{}{"worker_id": w.workerID, "job_id": result.Job.ID.String()}).Error(ctx, "complete job failed")
	}

	// Record local prometheus metrics (events/notifications now handled server-side).
	w.recordMetrics(ctx, result, execErr, stats)
}

// recordMetrics records local prometheus metrics for the completed sync.
// Events and notifications are now handled server-side in UpdateJobStatus.
func (w *DockerSyncWorker) recordMetrics(ctx context.Context, result *pipelinejobs.ClaimJobBundleResult, execErr error, stats *pipelineservice.SyncStats) {
	if w.metrics == nil {
		return
	}

	wsID := result.WorkspaceID.String()
	connID := result.ConnectionID.String()

	if execErr != nil {
		w.metrics.ObserveSyncFailed(wsID, connID)
		if w.logger != nil {
			w.logger.WithError(execErr).WithFields(map[string]interface{}{"worker_id": w.workerID, "job_id": result.Job.ID.String()}).Error(ctx, "job failed")
		}
	} else {
		var recordsRead, bytesSynced int64
		var duration time.Duration
		if stats != nil {
			recordsRead = stats.RecordsRead
			bytesSynced = stats.BytesSynced
			duration = stats.Duration
		}

		w.metrics.ObserveSyncCompleted(wsID, connID, duration, recordsRead, bytesSynced)
		if recordsRead == 0 {
			w.metrics.ObserveZeroRecordSync(wsID, connID)
		}

		if w.logger != nil {
			w.logger.WithFields(map[string]interface{}{"worker_id": w.workerID, "job_id": result.Job.ID.String()}).Info(ctx, "job completed")
		}
	}
}

// failJob reports a job failure via ExecutorBackend and records failure metrics.
func (w *DockerSyncWorker) failJob(ctx context.Context, jobID, workspaceID, connectionID uuid.UUID, reason error) {
	if w.logger != nil {
		w.logger.WithError(reason).WithField("job_id", jobID.String()).Error(ctx, "docker sync worker: job failed")
	}

	if err := w.backend.UpdateJobStatus(ctx, UpdateJobStatusParams{
		JobID:        jobID,
		Success:      false,
		ErrorMessage: reason.Error(),
	}); err != nil && w.logger != nil {
		w.logger.WithError(err).WithField("job_id", jobID.String()).Error(ctx, "failed to update job status")
	}

	if w.metrics != nil {
		w.metrics.ObserveSyncFailed(workspaceID.String(), connectionID.String())
	}
}

// heartbeatAndCancel combines heartbeat and cancel polling into a single loop
// per D-05. Heartbeat response carries a cancelled flag for cancel detection.
func (w *DockerSyncWorker) heartbeatAndCancel(ctx context.Context, cancel context.CancelFunc, jobID uuid.UUID) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			result, err := w.backend.Heartbeat(ctx, jobID, 0, 0)
			if err != nil {
				// Tolerate heartbeat failures per D-12.
				if w.logger != nil {
					w.logger.WithError(err).WithField("job_id", jobID.String()).Warn(ctx, "heartbeat failed")
				}
				continue
			}
			if result.Cancelled {
				if w.logger != nil {
					w.logger.WithField("job_id", jobID.String()).Info(ctx, "cancel detected via heartbeat, stopping execution")
				}
				cancel()
				return
			}
		}
	}
}

// nilIfEmpty returns nil if the string is empty, otherwise returns a pointer to the string.
func nilIfEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// getWorkerID returns a generated ID for worker identification.
func getWorkerID() string {
	return uuid.New().String()[:8]
}
