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

// Execute creates a workspace, adds the owner as admin, and publishes the
// workspace.created event for downstream subscribers (e.g. seeding default
// connector registries).
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

	created, err := uc.storage.Workspaces().Create(ctx, workspace)
	if err != nil {
		return nil, fmt.Errorf("creating workspace: %w", err)
	}

	// Add owner as admin member.
	member := &WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: created.ID,
		UserID:      ownerUserID,
		Role:        MemberRoleAdmin,
		JoinedAt:    now,
	}

	if _, err := uc.storage.WorkspaceMembers().Create(ctx, member); err != nil {
		return nil, fmt.Errorf("adding owner member: %w", err)
	}

	if err := uc.storage.WorkspaceEvents().Send(ctx, &WorkspaceEvent{
		EventType: WorkspaceEventTypeCreated,
		Data:      &WorkspaceCreatedEventData{Workspace: created},
	}); err != nil {
		return nil, fmt.Errorf("publishing workspace.created event: %w", err)
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
