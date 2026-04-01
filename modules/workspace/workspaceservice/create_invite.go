package workspaceservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// CreateInvite creates a workspace invite and sends an email notification.
type CreateInvite struct {
	storage     Storage
	emailSender EmailSender
	userLookup  UserLookup
	inviteTTL   time.Duration
	frontendURL string
	logger      *logging.Logger
}

// NewCreateInvite creates a new CreateInvite use case.
func NewCreateInvite(
	storage Storage,
	emailSender EmailSender,
	userLookup UserLookup,
	inviteTTL time.Duration,
	frontendURL string,
	logger *logging.Logger,
) *CreateInvite {
	return &CreateInvite{
		storage:     storage,
		emailSender: emailSender,
		userLookup:  userLookup,
		inviteTTL:   inviteTTL,
		frontendURL: frontendURL,
		logger:      logger.Named("create-invite"),
	}
}

// CreateInviteParams holds the parameters for creating an invite.
type CreateInviteParams struct {
	WorkspaceID   uuid.UUID
	InviterUserID uuid.UUID
	Email         string
	Role          MemberRole
}

// Execute creates a workspace invite. If a pending invite already exists for the same
// email and workspace, it is replaced (new role, reset TTL). Sends email asynchronously.
func (uc *CreateInvite) Execute(ctx context.Context, params CreateInviteParams) (*WorkspaceInvite, error) {
	email := strings.ToLower(strings.TrimSpace(params.Email))

	// Check if user is already a member of this workspace.
	user, err := uc.userLookup.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("looking up user by email: %w", err)
	}

	if user != nil {
		// User exists, check membership.
		existingMember, err := uc.storage.WorkspaceMembers().First(ctx, &WorkspaceMemberFilter{
			WorkspaceID: filter.Equals(params.WorkspaceID),
			UserID:      filter.Equals(user.ID),
		})
		if err != nil {
			return nil, fmt.Errorf("checking membership: %w", err)
		}

		if existingMember != nil {
			return nil, AlreadyExistsError("this user is already a member of this workspace")
		}
	}

	now := time.Now()
	expiresAt := now.Add(uc.inviteTTL)

	// Check for existing pending invite for this email + workspace (D-07: replace existing).
	existingInvite, err := uc.storage.WorkspaceInvites().First(ctx, &WorkspaceInviteFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
		Email:       filter.Equals(email),
		Status:      filter.Equals(InviteStatusPending),
	})
	if err != nil {
		return nil, fmt.Errorf("checking existing invite: %w", err)
	}

	var invite *WorkspaceInvite

	if existingInvite != nil {
		// Replace existing pending invite: update role, TTL, inviter.
		existingInvite.Role = params.Role
		existingInvite.InviterUserID = params.InviterUserID
		existingInvite.ExpiresAt = expiresAt
		existingInvite.UpdatedAt = now

		invite, err = uc.storage.WorkspaceInvites().Update(ctx, existingInvite)
		if err != nil {
			return nil, fmt.Errorf("updating existing invite: %w", err)
		}
	} else {
		// Generate a new token: 32 random bytes -> 64 hex chars.
		tokenBytes := make([]byte, 32)
		if _, err := rand.Read(tokenBytes); err != nil {
			return nil, fmt.Errorf("generating invite token: %w", err)
		}

		token := hex.EncodeToString(tokenBytes)

		invite = &WorkspaceInvite{
			ID:            uuid.New(),
			WorkspaceID:   params.WorkspaceID,
			InviterUserID: params.InviterUserID,
			Email:         email,
			Role:          params.Role,
			Token:         token,
			Status:        InviteStatusPending,
			ExpiresAt:     expiresAt,
			CreatedAt:     now,
			UpdatedAt:     now,
		}

		invite, err = uc.storage.WorkspaceInvites().Create(ctx, invite)
		if err != nil {
			return nil, fmt.Errorf("creating invite: %w", err)
		}
	}

	// Send email asynchronously to avoid blocking the request.
	uc.sendInviteEmailAsync(ctx, invite, params.InviterUserID)

	return invite, nil
}

// sendInviteEmailAsync sends the invite email in a goroutine.
func (uc *CreateInvite) sendInviteEmailAsync(ctx context.Context, invite *WorkspaceInvite, inviterUserID uuid.UUID) {
	// Gather workspace and inviter info for the email.
	workspace, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		ID: filter.Equals(invite.WorkspaceID),
	})
	if err != nil || workspace == nil {
		uc.logger.WithError(err).WithField("workspace_id", invite.WorkspaceID).Error(ctx, "failed to load workspace for invite email")

		return
	}

	inviter, err := uc.userLookup.GetUserByID(ctx, inviterUserID)
	if err != nil || inviter == nil {
		uc.logger.WithError(err).WithField("inviter_id", inviterUserID).Error(ctx, "failed to load inviter for invite email")

		return
	}

	acceptURL := fmt.Sprintf("%s/invite/%s", strings.TrimRight(uc.frontendURL, "/"), invite.Token)

	go func() { //nolint:gosec // intentional background context for async email
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
			uc.logger.WithError(sendErr).WithField("email", invite.Email).Error(sendCtx, "failed to send invite email")
		}
	}()
}
