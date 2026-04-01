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
}

// NewCreateWorkspace creates a new CreateWorkspace use case.
func NewCreateWorkspace(storage Storage) *CreateWorkspace {
	return &CreateWorkspace{storage: storage}
}

// Execute creates a workspace with the given name and adds the owner as admin.
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

	return created, nil
}
