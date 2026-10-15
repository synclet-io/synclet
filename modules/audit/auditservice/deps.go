package auditservice

import (
	"context"
	"time"

	"github.com/google/uuid"
)

// RetentionStorage provides the bulk queries behind audit retention cleanup.
type RetentionStorage interface {
	// DistinctWorkspaceIDs returns every workspace that has at least one audit event.
	DistinctWorkspaceIDs(ctx context.Context) ([]uuid.UUID, error)
	// DeleteOlderThan removes the workspace's events created before cutoff and returns the count.
	DeleteOlderThan(ctx context.Context, workspaceID uuid.UUID, cutoff time.Time) (int64, error)
}
