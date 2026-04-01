package workspaceservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// AcceptInvite accepts a workspace invite and creates a membership.
type AcceptInvite struct {
	storage Storage
}

// NewAcceptInvite creates a new AcceptInvite use case.
func NewAcceptInvite(storage Storage) *AcceptInvite {
	return &AcceptInvite{storage: storage}
}

// AcceptInviteResult holds the result of accepting an invite.
type AcceptInviteResult struct {
	WorkspaceID   uuid.UUID
	WorkspaceName string
}

// Execute accepts an invite by token. Validates the invite is pending, not expired,
// and the user's email matches the invite email (D-14).
func (uc *AcceptInvite) Execute(ctx context.Context, token string, userID uuid.UUID, userEmail string) (*AcceptInviteResult, error) {
	invite, err := uc.storage.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{
		Token: filter.Equals(token),
	})
	if err != nil {
		return nil, fmt.Errorf("finding invite by token: %w", err)
	}

	if invite == nil {
		return nil, ErrWorkspaceInviteNotFound
	}

	if !invite.Status.IsPending() {
		return nil, &ValidationError{Message: fmt.Sprintf("invite is %s, not pending", invite.Status.String())}
	}

	if time.Now().After(invite.ExpiresAt) {
		return nil, &ValidationError{Message: "invite has expired"}
	}

	if !strings.EqualFold(invite.Email, userEmail) {
		return nil, &ValidationError{Message: fmt.Sprintf("this invite is for %s", invite.Email)}
	}

	// Check if user is already a member (edge case: invited user joined via another invite).
	existingMember, err := uc.storage.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{
		WorkspaceID: filter.Equals(invite.WorkspaceID),
		UserID:      filter.Equals(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("checking existing membership: %w", err)
	}

	if existingMember != nil {
		// Already a member -- just mark invite as accepted.
		invite.Status = InviteStatusAccepted

		invite.UpdatedAt = time.Now()
		if _, err := uc.storage.WorkspaceInvites().Update(ctx, invite); err != nil {
			return nil, fmt.Errorf("updating invite status: %w", err)
		}

		workspace, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{ID: filter.Equals(invite.WorkspaceID)})
		if err != nil || workspace == nil {
			return nil, fmt.Errorf("loading workspace: %w", err)
		}

		return &AcceptInviteResult{WorkspaceID: workspace.ID, WorkspaceName: workspace.Name}, nil
	}

	// Create membership and mark invite accepted atomically to prevent invite reuse on partial failure.
	now := time.Now()
	var workspace *Workspace

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		member := &WorkspaceMember{
			ID:          uuid.New(),
			WorkspaceID: invite.WorkspaceID,
			UserID:      userID,
			Role:        invite.Role,
			JoinedAt:    now,
		}
		if _, err := tx.WorkspaceMembers().Create(ctx, member); err != nil {
			return fmt.Errorf("creating workspace member: %w", err)
		}

		invite.Status = InviteStatusAccepted

		invite.UpdatedAt = now
		if _, err := tx.WorkspaceInvites().Update(ctx, invite); err != nil {
			return fmt.Errorf("updating invite status: %w", err)
		}

		var txErr error

		workspace, txErr = tx.Workspaces().First(ctx, &WorkspaceFilter{ID: filter.Equals(invite.WorkspaceID)})
		if txErr != nil || workspace == nil {
			return fmt.Errorf("loading workspace: %w", txErr)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &AcceptInviteResult{WorkspaceID: workspace.ID, WorkspaceName: workspace.Name}, nil
}

// ValidationError represents a validation error in the workspace service.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}
