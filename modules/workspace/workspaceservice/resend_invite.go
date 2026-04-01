package workspaceservice

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ResendInvite resends a pending invite email and resets the TTL (D-06).
type ResendInvite struct {
	storage     Storage
	emailSender EmailSender
	userLookup  UserLookup
	inviteTTL   time.Duration
	frontendURL string
	logger      *logging.Logger
}

// NewResendInvite creates a new ResendInvite use case.
func NewResendInvite(
	storage Storage,
	emailSender EmailSender,
	userLookup UserLookup,
	inviteTTL time.Duration,
	frontendURL string,
	logger *logging.Logger,
) *ResendInvite {
	return &ResendInvite{
		storage:     storage,
		emailSender: emailSender,
		userLookup:  userLookup,
		inviteTTL:   inviteTTL,
		frontendURL: frontendURL,
		logger:      logger.Named("resend-invite"),
	}
}

// Execute resends the invite email and resets the expiration. Same token is reused (D-06).
func (uc *ResendInvite) Execute(ctx context.Context, inviteID, workspaceID uuid.UUID) error {
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
		return &ValidationError{Message: fmt.Sprintf("invite is %s, cannot resend", invite.Status.String())}
	}

	// Reset TTL.
	now := time.Now()
	invite.ExpiresAt = now.Add(uc.inviteTTL)
	invite.UpdatedAt = now

	invite, err = uc.storage.WorkspaceInvites().Update(ctx, invite)
	if err != nil {
		return fmt.Errorf("updating invite TTL: %w", err)
	}

	// Send email asynchronously.
	workspace, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		ID: filter.Equals(invite.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("loading workspace for resend email: %w", err)
	}

	if workspace == nil {
		return fmt.Errorf("workspace %s not found for resend email", invite.WorkspaceID)
	}

	inviter, err := uc.userLookup.GetUserByID(ctx, invite.InviterUserID)
	if err != nil {
		return fmt.Errorf("loading inviter for resend email: %w", err)
	}

	if inviter == nil {
		return fmt.Errorf("inviter %s not found for resend email", invite.InviterUserID)
	}

	acceptURL := fmt.Sprintf("%s/invite/%s", strings.TrimRight(uc.frontendURL, "/"), invite.Token)

	go func() {
		sendCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		sendErr := uc.emailSender.SendInviteEmail(sendCtx, SendInviteEmailParams{
			To:            invite.Email,
			WorkspaceName: workspace.Name,
			InviterName:   inviter.Name,
			Role:          invite.Role.String(),
			AcceptURL:     acceptURL,
			ExpiresAt:     invite.ExpiresAt.Format("January 2, 2006"),
		})
		if sendErr != nil {
			uc.logger.WithError(sendErr).WithField("email", invite.Email).Error(sendCtx, "failed to resend invite email")
		}
	}()

	return nil
}
