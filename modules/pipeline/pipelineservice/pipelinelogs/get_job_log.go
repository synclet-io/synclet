package pipelinelogs

import (
	"cmp"
	"context"
	"fmt"
	"slices"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"go.uber.org/multierr"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetJobLogParams holds parameters for retrieving job logs.
type GetJobLogParams struct {
	WorkspaceID uuid.UUID
	JobID       uuid.UUID
	AfterID     int64 // cursor: return logs with id > AfterID (0 = from beginning)
	Limit       int   // max lines to return (<= 0 = DefaultLogLimit, capped at MaxLogLimit)
}

const (
	DefaultLogLimit = 1000
	MaxLogLimit     = 5000
)

// GetJobLogResult holds the result of a job log query.
type GetJobLogResult struct {
	Lines   []string
	LastID  int64
	HasMore bool
	// Attempts lists, in attempt order, every attempt that produced log lines.
	Attempts []AttemptLogStats
}

// AttemptLogStats holds the line-cap counters of one job attempt.
type AttemptLogStats struct {
	Attempt      int
	DroppedLines int
}

// GetJobLog retrieves log lines for a job with cursor-based pagination.
type GetJobLog struct {
	storage pipelineservice.Storage
	archive pipelineservice.LogArchive
}

// NewGetJobLog creates a new GetJobLog use case.
func NewGetJobLog(storage pipelineservice.Storage, archive pipelineservice.LogArchive) *GetJobLog {
	return &GetJobLog{storage: storage, archive: archive}
}

// Execute returns log lines for the given job, supporting cursor-based
// pagination. Lines come from archived attempts and from Postgres, merged in
// id order; archived lines keep their job_logs ids, so a cursor stays valid
// when the job is archived between two calls.
func (uc *GetJobLog) Execute(ctx context.Context, params GetJobLogParams) (*GetJobLogResult, error) {
	if err := checkJobWorkspace(ctx, uc.storage, params.WorkspaceID, params.JobID); err != nil {
		return nil, err
	}

	segments, err := uc.storage.JobLogSegments().Find(ctx, &pipelineservice.JobLogSegmentFilter{
		JobID: filter.Equals(params.JobID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing job log segments: %w", err)
	}

	limit := params.Limit
	if limit <= 0 {
		limit = DefaultLogLimit
	}

	limit = min(limit, MaxLogLimit)

	// Fetch limit+1 to detect hasMore.
	lines, err := uc.archivedLines(ctx, segments, params.AfterID, limit+1)
	if err != nil {
		return nil, err
	}

	logs, err := uc.storage.JobLogs().GetLogs(ctx, params.JobID, params.AfterID, limit+1)
	if err != nil {
		return nil, err
	}

	for _, l := range logs {
		lines = append(lines, archivedLine{ID: l.ID, Timestamp: l.CreatedAt, Message: l.LogLine})
	}

	slices.SortFunc(lines, func(a, b archivedLine) int { return cmp.Compare(a.ID, b.ID) })

	hasMore := len(lines) > limit
	if hasMore {
		lines = lines[:limit]
	}

	result := &GetJobLogResult{Lines: make([]string, len(lines)), Attempts: attemptStats(segments)}

	for i, l := range lines {
		result.Lines[i] = l.Message
		result.LastID = l.ID
	}

	result.HasMore = hasMore

	return result, nil
}

func attemptStats(segments []*pipelineservice.JobLogSegment) []AttemptLogStats {
	stats := make([]AttemptLogStats, len(segments))
	for i, s := range segments {
		stats[i] = AttemptLogStats{Attempt: s.Attempt, DroppedLines: s.DroppedCount}
	}

	slices.SortFunc(stats, func(a, b AttemptLogStats) int { return cmp.Compare(a.Attempt, b.Attempt) })

	return stats
}

// archivedLines returns up to limit archived lines with id > afterID, in id order.
func (uc *GetJobLog) archivedLines(ctx context.Context, segments []*pipelineservice.JobLogSegment, afterID int64, limit int) ([]archivedLine, error) {
	segments = slices.DeleteFunc(slices.Clone(segments), func(s *pipelineservice.JobLogSegment) bool {
		return s.Status != pipelineservice.JobLogSegmentStatusArchived || s.ArchiveKey == nil || s.FirstLogID == nil || s.LastLogID == nil || *s.LastLogID <= afterID
	})
	slices.SortFunc(segments, func(a, b *pipelineservice.JobLogSegment) int { return cmp.Compare(*a.FirstLogID, *b.FirstLogID) })

	var lines []archivedLine

	for _, segment := range segments {
		if len(lines) >= limit && *segment.FirstLogID > lines[limit-1].ID {
			break
		}

		segmentLines, err := uc.readSegment(ctx, *segment.ArchiveKey, afterID, limit)
		if err != nil {
			return nil, err
		}

		lines = append(lines, segmentLines...)
		slices.SortFunc(lines, func(a, b archivedLine) int { return cmp.Compare(a.ID, b.ID) })
		lines = lines[:min(len(lines), limit)]
	}

	return lines, nil
}

func (uc *GetJobLog) readSegment(ctx context.Context, key string, afterID int64, limit int) (_ []archivedLine, rerr error) {
	body, err := uc.archive.Open(ctx, key)
	if err != nil {
		return nil, fmt.Errorf("opening archived job log: %w", err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(body))

	return readArchive(body, afterID, limit)
}
