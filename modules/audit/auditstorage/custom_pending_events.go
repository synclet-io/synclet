package auditstorage

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
)

// PendingEventStorage moves pending audit events into a workspace.
type PendingEventStorage struct {
	db *gorm.DB
}

func NewPendingEventStorage(db *gorm.DB) *PendingEventStorage {
	return &PendingEventStorage{db: db}
}

var _ auditservice.PendingEventStorage = (*PendingEventStorage)(nil)

func (s *PendingEventStorage) AdoptPendingEvents(ctx context.Context, userID, workspaceID uuid.UUID, until time.Time) (int64, error) {
	result := s.db.WithContext(ctx).Exec(`
		UPDATE audit.audit_events SET workspace_id = @workspace
		WHERE workspace_id = @pending AND actor_type = 'user' AND actor_id = @user
		  AND resource_type = 'user' AND action IN ('create', 'login') AND created_at <= @until
		  AND EXISTS (
		    SELECT 1 FROM audit.audit_events
		    WHERE workspace_id = @pending AND actor_type = 'user' AND actor_id = @user
		      AND resource_type = 'user' AND action = 'create'
		  )`,
		sql.Named("workspace", workspaceID),
		sql.Named("pending", uuid.Nil),
		sql.Named("user", userID),
		sql.Named("until", until),
	)
	if result.Error != nil {
		return 0, fmt.Errorf("moving pending audit events into workspace: %w", result.Error)
	}

	return result.RowsAffected, nil
}
