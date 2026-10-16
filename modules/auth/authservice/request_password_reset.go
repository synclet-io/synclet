package authservice

import (
	"context"
	"errors"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/samber/lo"
)

// PasswordResetEmailParams holds the data for a password reset email.
type PasswordResetEmailParams struct {
	To        string
	ResetURL  string
	ExpiresAt time.Time
}

// PasswordResetEmailSender delivers password reset emails.
type PasswordResetEmailSender interface {
	SendPasswordResetEmail(ctx context.Context, params PasswordResetEmailParams) error
}

const passwordResetEmailTimeout = 30 * time.Second

// RequestPasswordReset emails a password reset link to a user. It reports
// success whether or not an account with the email exists, and the email is
// sent in the background so response timing does not reveal it either.
type RequestPasswordReset struct {
	cfg    Config
	issue  *IssuePasswordResetLink
	sender PasswordResetEmailSender
	audit  AuditRecorder
	logger *logging.Logger
}

// NewRequestPasswordReset creates a new RequestPasswordReset use case.
func NewRequestPasswordReset(
	cfg Config,
	issue *IssuePasswordResetLink,
	sender PasswordResetEmailSender,
	audit AuditRecorder,
	logger *logging.Logger,
) *RequestPasswordReset {
	return &RequestPasswordReset{cfg: cfg, issue: issue, sender: sender, audit: audit, logger: logger}
}

// Execute issues a reset link for the email and sends it. Without email
// configured, for an unknown email, or within the cooldown of the user's
// previous request, it does nothing and returns nil.
func (uc *RequestPasswordReset) Execute(ctx context.Context, email string) error {
	if !uc.cfg.EmailEnabled {
		uc.logger.Info(ctx, "password reset requested but email is not configured")

		return nil
	}

	link, err := uc.issue.Execute(ctx, IssuePasswordResetLinkParams{Email: email, Cooldown: uc.cfg.PasswordResetCooldown})
	if err != nil {
		if _, ok := lo.ErrorsAs[NotFoundError](err); ok {
			return nil
		}

		if errors.Is(err, ErrPasswordResetCooldown) || errors.Is(err, ErrPasswordResetTokenAlreadyExists) {
			return nil
		}

		return err
	}

	uc.audit.Record(ctx, AuditEvent{
		ActorID:       link.User.ID,
		ActorLabel:    link.User.Email,
		Action:        "update",
		ResourceType:  "user",
		ResourceID:    link.User.ID,
		ResourceLabel: link.User.Email,
		After:         map[string]any{"password_reset_requested": true},
	})

	params := PasswordResetEmailParams{To: link.User.Email, ResetURL: link.URL, ExpiresAt: link.ExpiresAt}

	go func() { //nolint:gosec // intentional background context for async email
		sendCtx, cancel := context.WithTimeout(context.Background(), passwordResetEmailTimeout)
		defer cancel()

		if err := uc.sender.SendPasswordResetEmail(sendCtx, params); err != nil {
			uc.logger.WithError(err).WithField("user_id", link.User.ID).Error(sendCtx, "failed to send password reset email")
		}
	}()

	return nil
}
