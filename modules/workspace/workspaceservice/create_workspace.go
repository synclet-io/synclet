package workspaceservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/pkg/stringutil"
)

// CreateWorkspace creates a new workspace and adds the owner as admin member.
type CreateWorkspace struct {
	storage Storage
	audit   AuditRecorder
}

// NewCreateWorkspace creates a new CreateWorkspace use case.
func NewCreateWorkspace(storage Storage, audit AuditRecorder) *CreateWorkspace {
	return &CreateWorkspace{storage: storage, audit: audit}
}

// Execute creates a workspace, adds the owner as admin and publishes the
// workspace.created event in one transaction.
func (uc *CreateWorkspace) Execute(ctx context.Context, name string, ownerUserID uuid.UUID) (*Workspace, error) {
	now := time.Now()
	slug := stringutil.Slugify(name)

	workspace := &Workspace{
		ID:        uuid.New(),
		Name:      name,
		Slug:      slug,
		CreatedAt: now,
		UpdatedAt: now,
	}

	var created *Workspace

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		var err error

		created, err = tx.Workspaces().Create(ctx, workspace)
		if err != nil {
			return fmt.Errorf("creating workspace: %w", err)
		}

		member := &WorkspaceMember{
			ID:          uuid.New(),
			WorkspaceID: created.ID,
			UserID:      ownerUserID,
			Role:        MemberRoleAdmin,
			JoinedAt:    now,
		}

		if _, err := tx.WorkspaceMembers().Create(ctx, member); err != nil {
			return fmt.Errorf("adding owner member: %w", err)
		}

		if err := tx.WorkspaceEvents().Send(ctx, &WorkspaceEvent{
			EventType: WorkspaceEventTypeCreated,
			Data:      &WorkspaceCreatedEventData{Workspace: created},
		}); err != nil {
			return fmt.Errorf("publishing workspace.created event: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   created.ID,
		Action:        "create",
		ResourceType:  "workspace",
		ResourceID:    created.ID,
		ResourceLabel: created.Name,
		After:         map[string]any{"name": created.Name, "slug": created.Slug},
	})

	return created, nil
}
