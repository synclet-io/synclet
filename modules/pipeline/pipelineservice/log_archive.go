package pipelineservice

import (
	"context"
	"fmt"
	"io"

	"github.com/google/uuid"
)

// LogArchive stores archived job logs as objects addressed by key.
type LogArchive interface {
	// Put writes the object, replacing any object under the same key.
	Put(ctx context.Context, key string, r io.Reader) error
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Delete removes the objects; keys without an object are skipped.
	Delete(ctx context.Context, keys ...string) error
}

// LogArchiveKey returns the object key of one job attempt's archived log.
func LogArchiveKey(workspaceID, connectionID, jobID uuid.UUID, attempt int) string {
	return fmt.Sprintf("logs/%s/%s/%s/%d.ndjson.gz", workspaceID, connectionID, jobID, attempt)
}
