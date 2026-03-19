package workspaceservice

import (
	"context"

	"github.com/google/uuid"
)

// EmailSender sends invite emails. Implemented by adapter calling notify module.
type EmailSender interface {
	SendInviteEmail(ctx context.Context, params SendInviteEmailParams) error
}

// SendInviteEmailParams holds the parameters for sending an invite email.
type SendInviteEmailParams struct {
	To            string
	WorkspaceName string
	InviterName   string
	Role          string
	AcceptURL     string
	ExpiresAt     string
}

// UserLookup resolves users by email or ID. Implemented by adapter calling auth module.
type UserLookup interface {
	GetUserByEmail(ctx context.Context, email string) (*UserInfo, error)
	GetUserByID(ctx context.Context, id uuid.UUID) (*UserInfo, error)
}

// UserInfo holds basic user information for cross-module communication.
type UserInfo struct {
	ID    uuid.UUID
	Email string
	Name  string
}
