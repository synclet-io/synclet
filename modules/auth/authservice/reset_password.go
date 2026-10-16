package authservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ResetPassword sets a new password from a password reset token, consumes
// the token and signs the user out of every session.
type ResetPassword struct {
	storage Storage
	cfg     Config
	audit   AuditRecorder
}

// NewResetPassword creates a new ResetPassword use case.
func NewResetPassword(storage Storage, cfg Config, audit AuditRecorder) *ResetPassword {
	return &ResetPassword{storage: storage, cfg: cfg, audit: audit}
}

// Execute validates the token and the new password, then stores the password,
// deletes the token and revokes all refresh tokens in one transaction.
// A missing, used or expired token returns ErrInvalidResetToken.
func (uc *ResetPassword) Execute(ctx context.Context, rawToken, newPassword string) error {
	token, err := uc.storage.PasswordResetTokens().First(ctx, &PasswordResetTokenFilter{
		TokenHash: filter.Equals(hashToken(rawToken)),
	})
	if err != nil {
		if errors.Is(err, ErrPasswordResetTokenNotFound) {
			return ErrInvalidResetToken
		}

		return fmt.Errorf("fetching reset token: %w", err)
	}

	if !time.Now().Before(token.ExpiresAt) {
		return ErrInvalidResetToken
	}

	if err := ValidatePassword(newPassword, uc.cfg.MinPasswordLength); err != nil {
		return err
	}

	user, err := uc.storage.Users().First(ctx, &UserFilter{ID: filter.Equals(token.UserID)})
	if err != nil {
		return fmt.Errorf("fetching user: %w", err)
	}

	hash, err := hashPassword(newPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.PasswordHash = hash

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		if err := tx.PasswordResetTokens().Delete(ctx, &PasswordResetTokenFilter{ID: filter.Equals(token.ID)}); err != nil {
			return fmt.Errorf("consuming reset token: %w", err)
		}

		if _, err := tx.Users().Update(ctx, user); err != nil {
			return fmt.Errorf("updating password: %w", err)
		}

		if err := tx.RefreshTokens().Delete(ctx, &RefreshTokenFilter{UserID: filter.Equals(user.ID)}); err != nil {
			return fmt.Errorf("revoking refresh tokens: %w", err)
		}

		return nil
	}); err != nil {
		return err
	}

	uc.audit.Record(ctx, AuditEvent{
		ActorID:       user.ID,
		ActorLabel:    user.Email,
		Action:        "update",
		ResourceType:  "user",
		ResourceID:    user.ID,
		ResourceLabel: user.Email,
		After:         map[string]any{"password_reset": true, "sessions_revoked": true},
	})

	return nil
}
