package workspaceservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// RevokeInvite revokes a pending workspace invite (D-06).
type RevokeInvite struct {
	storage Storage
}

// NewRevokeInvite creates a new RevokeInvite use case.
func NewRevokeInvite(storage Storage) *RevokeInvite {
	return &RevokeInvite{storage: storage}
}

// Execute revokes a pending invite by ID and workspace ID.
func (uc *RevokeInvite) Execute(ctx context.Context, inviteID, workspaceID uuid.UUID) error {
	invite, err := uc.storage.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{
		ID:          filter.Equals(inviteID),
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("finding invite: %w", err)
	}
	if invite == nil {
		return ErrWorkspaceInviteNotFound
	}

	if !invite.Status.IsPending() {
		return &ValidationError{Message: fmt.Sprintf("invite is %s, cannot revoke", invite.Status.String())}
	}

	invite.Status = InviteStatusRevoked
	invite.UpdatedAt = time.Now()

	if _, err := uc.storage.WorkspaceInvites().Update(ctx, invite); err != nil {
		return fmt.Errorf("revoking invite: %w", err)
	}

	return nil
}
