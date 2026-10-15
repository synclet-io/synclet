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
	audit   AuditRecorder
}

// NewRevokeInvite creates a new RevokeInvite use case.
func NewRevokeInvite(storage Storage, audit AuditRecorder) *RevokeInvite {
	return &RevokeInvite{storage: storage, audit: audit}
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

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   invite.WorkspaceID,
		Action:        "delete",
		ResourceType:  "workspace_invite",
		ResourceID:    invite.ID,
		ResourceLabel: invite.Email,
		Before:        map[string]any{"email": invite.Email, "role": invite.Role.String()},
	})

	return nil
}
