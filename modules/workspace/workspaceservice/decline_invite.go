package workspaceservice

import (
	"context"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// DeclineInvite declines a workspace invite by token (D-10).
type DeclineInvite struct {
	storage Storage
}

// NewDeclineInvite creates a new DeclineInvite use case.
func NewDeclineInvite(storage Storage) *DeclineInvite {
	return &DeclineInvite{storage: storage}
}

// Execute declines an invite. The invite is marked as declined (not deleted).
func (uc *DeclineInvite) Execute(ctx context.Context, token string) error {
	invite, err := uc.storage.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{
		Token: filter.Equals(token),
	})
	if err != nil {
		return fmt.Errorf("finding invite by token: %w", err)
	}
	if invite == nil {
		return ErrWorkspaceInviteNotFound
	}

	if !invite.Status.IsPending() {
		return &ValidationError{Message: fmt.Sprintf("invite is %s, not pending", invite.Status.String())}
	}

	if time.Now().After(invite.ExpiresAt) {
		return &ValidationError{Message: "invite has expired"}
	}

	invite.Status = InviteStatusDeclined
	invite.UpdatedAt = time.Now()

	if _, err := uc.storage.WorkspaceInvites().Update(ctx, invite); err != nil {
		return fmt.Errorf("declining invite: %w", err)
	}

	return nil
}
