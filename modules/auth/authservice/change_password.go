package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ChangePasswordParams holds parameters for changing a password.
type ChangePasswordParams struct {
	UserID          uuid.UUID
	CurrentPassword string
	NewPassword     string
	// KeepRefreshToken is the caller's own refresh token, which stays valid;
	// every other refresh token of the user is revoked. Empty revokes all.
	KeepRefreshToken string
}

// ChangePassword changes a user's password after verifying the current one
// and signs the user out of every other session.
type ChangePassword struct {
	storage Storage
	cfg     Config
	audit   AuditRecorder
}

// NewChangePassword creates a new ChangePassword use case.
func NewChangePassword(storage Storage, cfg Config, audit AuditRecorder) *ChangePassword {
	return &ChangePassword{storage: storage, cfg: cfg, audit: audit}
}

// Execute verifies the current password, stores the new one and revokes the
// user's other refresh tokens in one transaction.
func (uc *ChangePassword) Execute(ctx context.Context, params ChangePasswordParams) error {
	user, err := uc.storage.Users().First(ctx, &UserFilter{
		ID: filter.Equals(params.UserID),
	})
	if err != nil {
		return fmt.Errorf("fetching user: %w", err)
	}

	validCurrentPassword, err := comparePasswordWithHash(params.CurrentPassword, user.PasswordHash)
	if err != nil {
		return fmt.Errorf("comparing current password: %w", err)
	}

	if !validCurrentPassword {
		return ErrInvalidCurrentPassword
	}

	if err := ValidatePassword(params.NewPassword, uc.cfg.MinPasswordLength); err != nil {
		return err
	}

	hash, err := hashPassword(params.NewPassword)
	if err != nil {
		return fmt.Errorf("hashing password: %w", err)
	}

	user.PasswordHash = hash

	revoke := &RefreshTokenFilter{UserID: filter.Equals(user.ID)}
	if params.KeepRefreshToken != "" {
		revoke.TokenHash = filter.NotEquals(hashToken(params.KeepRefreshToken))
	}

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		if _, err := tx.Users().Update(ctx, user); err != nil {
			return fmt.Errorf("updating password: %w", err)
		}

		if err := tx.RefreshTokens().Delete(ctx, revoke); err != nil {
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
		After:         map[string]any{"password_changed": true, "other_sessions_revoked": true},
	})

	return nil
}
