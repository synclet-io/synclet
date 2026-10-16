package pipelinelogs

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

const archiveReadBatch = 5000

// ArchiveJobLogs moves the log lines of a finished job from Postgres to the
// log archive, one gzipped NDJSON object per attempt.
type ArchiveJobLogs struct {
	storage pipelineservice.Storage
	archive pipelineservice.LogArchive
	enabled bool
}

// NewArchiveJobLogs creates a new ArchiveJobLogs use case.
func NewArchiveJobLogs(storage pipelineservice.Storage, archive pipelineservice.LogArchive, config pipelineservice.Config) *ArchiveJobLogs {
	return &ArchiveJobLogs{storage: storage, archive: archive, enabled: config.LogArchiveEnabled}
}

// Execute archives every attempt of the job that still has lines in Postgres.
// It does nothing when archiving is disabled or the job is missing or not
// finished. Each attempt's object is written before its rows are deleted, and
// the rows are deleted in the transaction that marks the attempt archived, so
// a rerun after any failure repeats only unfinished attempts. Lines that
// arrive after an attempt is archived stay in Postgres.
func (uc *ArchiveJobLogs) Execute(ctx context.Context, jobID uuid.UUID) error {
	if !uc.enabled {
		return nil
	}

	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{ID: filter.Equals(jobID)})
	if errors.Is(err, pipelineservice.ErrJobNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	if !isTerminal(job.Status) {
		return nil
	}

	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{ID: filter.Equals(job.ConnectionID)})
	if errors.Is(err, pipelineservice.ErrConnectionNotFound) {
		return nil
	}

	if err != nil {
		return fmt.Errorf("getting connection: %w", err)
	}

	attempts, err := uc.storage.JobLogs().ListUnarchivedAttempts(ctx, jobID)
	if err != nil {
		return err
	}

	for _, attempt := range attempts {
		key := pipelineservice.LogArchiveKey(conn.WorkspaceID, conn.ID, jobID, attempt)
		if err := uc.archiveAttempt(ctx, jobID, attempt, key); err != nil {
			return fmt.Errorf("archiving logs of job %s attempt %d: %w", jobID, attempt, err)
		}
	}

	return nil
}

func (uc *ArchiveJobLogs) archiveAttempt(ctx context.Context, jobID uuid.UUID, attempt int, key string) error {
	lastID, archived, err := uc.lockRange(ctx, jobID, attempt)
	if err != nil || archived || lastID == 0 {
		return err
	}

	var buf bytes.Buffer

	firstID, err := uc.encodeAttempt(ctx, &buf, jobID, attempt, lastID)
	if err != nil {
		return err
	}

	size := int64(buf.Len())

	if err := uc.archive.Put(ctx, key, &buf); err != nil {
		return fmt.Errorf("writing archive object: %w", err)
	}

	now := time.Now()
	segment := &pipelineservice.JobLogSegment{
		JobID:      jobID,
		Attempt:    attempt,
		ArchiveKey: &key,
		SizeBytes:  size,
		FirstLogID: &firstID,
		LastLogID:  &lastID,
		ArchivedAt: &now,
	}

	var marked bool

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		var markErr error

		marked, markErr = tx.JobLogSegments().MarkArchived(ctx, segment)
		if markErr != nil || !marked {
			return markErr
		}

		return tx.JobLogs().DeleteAttemptLogs(ctx, jobID, attempt, lastID)
	}); err != nil {
		return err
	}

	if !marked {
		return uc.dropOrphanObject(ctx, jobID, attempt, key)
	}

	return nil
}

// lockRange returns the highest log id of the attempt while holding the
// segment lock that appends take, so every line up to that id is committed.
func (uc *ArchiveJobLogs) lockRange(ctx context.Context, jobID uuid.UUID, attempt int) (lastID int64, archived bool, err error) {
	err = uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		segment, err := tx.JobLogSegments().Lock(ctx, jobID, attempt)
		if err != nil {
			return err
		}

		if segment.Status == pipelineservice.JobLogSegmentStatusArchived {
			archived = true

			return nil
		}

		lastID, err = tx.JobLogs().MaxLogID(ctx, jobID, attempt)

		return err
	})

	return lastID, archived, err
}

func (uc *ArchiveJobLogs) encodeAttempt(ctx context.Context, buf *bytes.Buffer, jobID uuid.UUID, attempt int, lastID int64) (int64, error) {
	writer := newArchiveWriter(buf)

	var afterID, firstID int64

	for {
		lines, err := uc.storage.JobLogs().ListAttemptLogs(ctx, jobID, attempt, afterID, lastID, archiveReadBatch)
		if err != nil {
			return 0, err
		}

		for _, line := range lines {
			if firstID == 0 {
				firstID = line.ID
			}

			if err := writer.Write(line); err != nil {
				return 0, fmt.Errorf("encoding log line: %w", err)
			}
		}

		if len(lines) < archiveReadBatch {
			break
		}

		afterID = lines[len(lines)-1].ID
	}

	if err := writer.Close(); err != nil {
		return 0, fmt.Errorf("compressing log lines: %w", err)
	}

	return firstID, nil
}

// dropOrphanObject deletes the object just written when its segment was
// removed with its job in the meantime.
func (uc *ArchiveJobLogs) dropOrphanObject(ctx context.Context, jobID uuid.UUID, attempt int, key string) error {
	_, err := uc.storage.JobLogSegments().First(ctx, &pipelineservice.JobLogSegmentFilter{
		JobID:   filter.Equals(jobID),
		Attempt: filter.Equals(attempt),
	})
	if err == nil {
		return nil
	}

	if !errors.Is(err, pipelineservice.ErrJobLogSegmentNotFound) {
		return fmt.Errorf("checking job log segment: %w", err)
	}

	return uc.archive.Delete(ctx, key)
}

func isTerminal(status pipelineservice.JobStatus) bool {
	return status == pipelineservice.JobStatusCompleted ||
		status == pipelineservice.JobStatusFailed ||
		status == pipelineservice.JobStatusCancelled
}
