package pipelinelogs

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// appendLines stores lines under the job's current attempt. Once the attempt
// holds maxLines lines, further lines are dropped and counted in one marker
// line. maxLines <= 0 stores every line.
func appendLines(ctx context.Context, storage pipelineservice.Storage, maxLines int, jobID uuid.UUID, lines []string) error {
	if len(lines) == 0 {
		return nil
	}

	return storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx pipelineservice.Storage) error {
		segment, err := tx.JobLogSegments().LockCurrent(ctx, jobID)
		if err != nil {
			return fmt.Errorf("locking job log segment: %w", err)
		}

		accepted := lines
		if maxLines > 0 {
			accepted = lines[:min(len(lines), max(maxLines-segment.LineCount, 0))]
		}

		if err := tx.JobLogs().InsertLines(ctx, jobID, segment.Attempt, accepted); err != nil {
			return fmt.Errorf("inserting job log lines: %w", err)
		}

		segment.LineCount += len(accepted)

		if dropped := len(lines) - len(accepted); dropped > 0 {
			segment.DroppedCount += dropped

			if err := writeTruncationMarker(ctx, tx, segment); err != nil {
				return err
			}
		}

		if _, err := tx.JobLogSegments().Update(ctx, segment); err != nil {
			return fmt.Errorf("updating job log segment: %w", err)
		}

		return nil
	})
}

func writeTruncationMarker(ctx context.Context, tx pipelineservice.Storage, segment *pipelineservice.JobLogSegment) error {
	marker := fmt.Sprintf("[synclet] log truncated: %d further lines dropped", segment.DroppedCount)

	if segment.MarkerLogID != nil {
		updated, err := tx.JobLogs().UpdateLine(ctx, *segment.MarkerLogID, marker)
		if err != nil {
			return fmt.Errorf("updating log truncation marker: %w", err)
		}

		if updated {
			return nil
		}
	}

	id, err := tx.JobLogs().InsertLine(ctx, segment.JobID, segment.Attempt, marker)
	if err != nil {
		return fmt.Errorf("inserting log truncation marker: %w", err)
	}

	segment.MarkerLogID = &id

	return nil
}
