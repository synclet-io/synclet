package workspaceservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// AutoAssignMember assigns a user to the default workspace.
// First member gets admin role, all subsequent members get viewer role.
type AutoAssignMember struct {
	storage Storage
}

// NewAutoAssignMember creates a new AutoAssignMember use case.
func NewAutoAssignMember(storage Storage) *AutoAssignMember {
	return &AutoAssignMember{storage: storage}
}

// Execute finds the default workspace and adds the user as a member.
// The first member receives admin role; all others receive viewer role.
func (uc *AutoAssignMember) Execute(ctx context.Context, userID uuid.UUID) error {
	// Find default workspace by slug.
	ws, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		Slug: filter.Equals("default"),
	})
	if err != nil {
		return fmt.Errorf("finding default workspace: %w", err)
	}

	// Determine role: first member is admin, rest are viewers.
	memberCount, err := uc.storage.WorkspaceMembers().Count(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(ws.ID),
	})
	if err != nil {
		return fmt.Errorf("counting workspace members: %w", err)
	}

	role := MemberRoleViewer
	if memberCount == 0 {
		role = MemberRoleAdmin
	}

	member := &WorkspaceMember{
		ID:          uuid.New(),
		WorkspaceID: ws.ID,
		UserID:      userID,
		Role:        role,
		JoinedAt:    time.Now(),
	}

	if _, err := uc.storage.WorkspaceMembers().Create(ctx, member); err != nil {
		return fmt.Errorf("creating workspace member: %w", err)
	}

	return nil
}
