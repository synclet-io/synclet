package workspaceservice

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// GetInviteByToken retrieves invite info by token (public, no auth).
type GetInviteByToken struct {
	storage    Storage
	userLookup UserLookup
}

// NewGetInviteByToken creates a new GetInviteByToken use case.
func NewGetInviteByToken(storage Storage, userLookup UserLookup) *GetInviteByToken {
	return &GetInviteByToken{
		storage:    storage,
		userLookup: userLookup,
	}
}

// InviteByTokenResult holds the denormalized invite info for the accept page.
type InviteByTokenResult struct {
	Invite        *WorkspaceInvite
	WorkspaceName string
	InviterName   string
	IsExpired     bool
}

// Execute returns invite details including workspace name and inviter name.
// Reports expired status even if DB status is still Pending.
func (uc *GetInviteByToken) Execute(ctx context.Context, token string) (*InviteByTokenResult, error) {
	invite, err := uc.storage.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{
		Token: filter.Equals(token),
	})
	if err != nil {
		return nil, fmt.Errorf("finding invite by token: %w", err)
	}

	if invite == nil {
		return nil, ErrWorkspaceInviteNotFound
	}

	workspace, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		ID: filter.Equals(invite.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading workspace: %w", err)
	}

	var workspaceName string
	if workspace != nil {
		workspaceName = workspace.Name
	}

	var inviterName string

	inviter, err := uc.userLookup.GetUserByID(ctx, invite.InviterUserID)
	if err == nil && inviter != nil {
		inviterName = inviter.Name
	}

	isExpired := invite.Status.IsPending() && time.Now().After(invite.ExpiresAt)

	return &InviteByTokenResult{
		Invite:        invite,
		WorkspaceName: workspaceName,
		InviterName:   inviterName,
		IsExpired:     isExpired,
	}, nil
}
