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

// EventPageStorage lists audit events with row-offset pagination.
type EventPageStorage interface {
	// FindPage returns up to limit events matching flt, newest first, after skipping offset of them.
	FindPage(ctx context.Context, flt *AuditEventFilter, offset, limit int) ([]*AuditEvent, error)
}

// UserDirectory looks up user accounts.
type UserDirectory interface {
	// UserExists reports whether the user account still exists.
	UserExists(ctx context.Context, id uuid.UUID) (bool, error)
	// UserEmail returns the user's email, or "" when the account does not exist.
	UserEmail(ctx context.Context, id uuid.UUID) (string, error)
}
