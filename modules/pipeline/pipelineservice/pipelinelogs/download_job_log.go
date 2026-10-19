package pipelinelogs

import (
	"bufio"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"go.uber.org/multierr"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DownloadJobLogParams selects one attempt of a job in the caller's workspace.
type DownloadJobLogParams struct {
	WorkspaceID uuid.UUID
	JobID       uuid.UUID
	Attempt     int
}

// DownloadJobLog writes the full log of one job attempt as plain text.
type DownloadJobLog struct {
	storage pipelineservice.Storage
	archive pipelineservice.LogArchive
}

// NewDownloadJobLog creates a new DownloadJobLog use case.
func NewDownloadJobLog(storage pipelineservice.Storage, archive pipelineservice.LogArchive) *DownloadJobLog {
	return &DownloadJobLog{storage: storage, archive: archive}
}

// Execute writes every line of the attempt to out, one per line, in id order:
// the archived object when the attempt is archived, then the attempt's lines
// still in Postgres. Nothing is written when the job is not in the workspace,
// which returns ErrJobNotFound.
func (uc *DownloadJobLog) Execute(ctx context.Context, params DownloadJobLogParams, out io.Writer) error {
	if err := checkJobWorkspace(ctx, uc.storage, params.WorkspaceID, params.JobID); err != nil {
		return err
	}

	segment, err := uc.storage.JobLogSegments().First(ctx, &pipelineservice.JobLogSegmentFilter{
		JobID:   filter.Equals(params.JobID),
		Attempt: filter.Equals(params.Attempt),
	})
	if err != nil && !errors.Is(err, pipelineservice.ErrJobLogSegmentNotFound) {
		return fmt.Errorf("getting job log segment: %w", err)
	}

	if segment != nil && segment.Status == pipelineservice.JobLogSegmentStatusArchived && segment.ArchiveKey != nil {
		if err := uc.writeArchived(ctx, *segment.ArchiveKey, out); err != nil {
			return err
		}
	}

	var afterID int64

	for {
		logs, err := uc.storage.JobLogs().ListAttemptLogs(ctx, params.JobID, params.Attempt, afterID, math.MaxInt64, archiveReadBatch)
		if err != nil {
			return fmt.Errorf("listing attempt logs: %w", err)
		}

		for _, l := range logs {
			if err := writeLine(out, l.LogLine); err != nil {
				return err
			}

			afterID = l.ID
		}

		if len(logs) < archiveReadBatch {
			return nil
		}
	}
}

func (uc *DownloadJobLog) writeArchived(ctx context.Context, key string, out io.Writer) (rerr error) {
	body, err := uc.archive.Open(ctx, key)
	if err != nil {
		return fmt.Errorf("opening archived job log: %w", err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(body))

	unzipped, err := gzip.NewReader(body)
	if err != nil {
		return fmt.Errorf("opening gzip stream: %w", err)
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(unzipped))

	dec := json.NewDecoder(bufio.NewReader(unzipped))

	for {
		var line archivedLine
		if err := dec.Decode(&line); errors.Is(err, io.EOF) {
			return nil
		} else if err != nil {
			return fmt.Errorf("decoding archived log line: %w", err)
		}

		if err := writeLine(out, line.Message); err != nil {
			return err
		}
	}
}

func writeLine(w io.Writer, line string) error {
	if _, err := io.WriteString(w, line+"\n"); err != nil {
		return fmt.Errorf("writing log line: %w", err)
	}

	return nil
}

// checkJobWorkspace returns ErrJobNotFound unless the job exists and belongs
// to the workspace.
func checkJobWorkspace(ctx context.Context, storage pipelineservice.Storage, workspaceID, jobID uuid.UUID) error {
	job, err := storage.Jobs().First(ctx, &pipelineservice.JobFilter{ID: filter.Equals(jobID)})
	if err != nil {
		return fmt.Errorf("getting job: %w", err)
	}

	conn, err := storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID:          filter.Equals(job.ConnectionID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil || conn == nil {
		return pipelineservice.ErrJobNotFound
	}

	return nil
}
