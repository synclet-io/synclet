package notifyadapt

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
)

// JobInfoLoader bridges the notify module's JobInfoLoader port to pipeline use
// cases. Notify never imports pipeline directly — this adapter is the seam.
type JobInfoLoader struct {
	getConnection   *pipelineconnections.GetConnection
	getSource       *pipelinesources.GetSource
	getDestination  *pipelinedestinations.GetDestination
	listJobAttempts *pipelinejobs.ListJobAttempts
}

// NewJobInfoLoader creates the adapter.
func NewJobInfoLoader(
	getConnection *pipelineconnections.GetConnection,
	getSource *pipelinesources.GetSource,
	getDestination *pipelinedestinations.GetDestination,
	listJobAttempts *pipelinejobs.ListJobAttempts,
) *JobInfoLoader {
	return &JobInfoLoader{
		getConnection:   getConnection,
		getSource:       getSource,
		getDestination:  getDestination,
		listJobAttempts: listJobAttempts,
	}
}

var _ notifyservice.JobInfoLoader = (*JobInfoLoader)(nil)

// LoadJobInfo loads the connection, source, destination, and latest attempt for
// the given job. Best-effort: returns a partially-populated JobInfo if some
// lookups fail, since webhook payloads should be sent even when enrichment is
// incomplete.
func (l *JobInfoLoader) LoadJobInfo(ctx context.Context, workspaceID, connectionID, jobID uuid.UUID) (*notifyservice.JobInfo, error) {
	info := &notifyservice.JobInfo{}

	conn, err := l.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          connectionID,
		WorkspaceID: workspaceID,
	})
	if err == nil && conn != nil {
		info.ConnectionName = conn.Name

		if src, srcErr := l.getSource.Execute(ctx, pipelinesources.GetSourceParams{
			ID:          conn.SourceID,
			WorkspaceID: workspaceID,
		}); srcErr == nil && src != nil {
			info.SourceName = src.Name
		}

		if dest, destErr := l.getDestination.Execute(ctx, pipelinedestinations.GetDestinationParams{
			ID:          conn.DestinationID,
			WorkspaceID: workspaceID,
		}); destErr == nil && dest != nil {
			info.DestinationName = dest.Name
		}
	}

	attempts, err := l.listJobAttempts.Execute(ctx, pipelinejobs.ListJobAttemptsParams{JobID: jobID})
	if err != nil {
		return info, fmt.Errorf("listing attempts: %w", err)
	}

	if len(attempts) > 0 {
		latest := attempts[len(attempts)-1]
		info.LatestAttempt = &notifyservice.JobAttemptInfo{
			AttemptNumber: latest.AttemptNumber,
		}

		if stats, statsErr := parseSyncStats(latest.SyncStatsJSON); statsErr == nil {
			info.LatestAttempt.RecordsRead = stats.RecordsRead
			info.LatestAttempt.BytesSynced = stats.BytesSynced
			info.LatestAttempt.DurationMs = stats.DurationMs
		}
	}

	return info, nil
}
