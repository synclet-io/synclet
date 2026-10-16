package authservice

import (
	"context"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// PasswordResetTokenTTL is how long a password reset link stays valid.
const PasswordResetTokenTTL = time.Hour

// PasswordResetLink is a freshly issued, single-use password reset link.
type PasswordResetLink struct {
	User      *User
	URL       string
	ExpiresAt time.Time
}

// IssuePasswordResetLinkParams holds parameters for issuing a reset link.
type IssuePasswordResetLinkParams struct {
	Email string
	// Cooldown rejects the request with ErrPasswordResetCooldown when the
	// user's current token was issued less than this long ago. Zero disables it.
	Cooldown time.Duration
}

// IssuePasswordResetLink creates a password reset token for a user and
// returns the link that consumes it. Each user holds at most one active
// token: issuing a new one invalidates the previous one.
type IssuePasswordResetLink struct {
	storage Storage
	cfg     Config
}

// NewIssuePasswordResetLink creates a new IssuePasswordResetLink use case.
func NewIssuePasswordResetLink(storage Storage, cfg Config) *IssuePasswordResetLink {
	return &IssuePasswordResetLink{storage: storage, cfg: cfg}
}

// Execute issues a reset link for the user with the given email. It returns
// ErrUserNotFound when no such user exists.
func (uc *IssuePasswordResetLink) Execute(ctx context.Context, params IssuePasswordResetLinkParams) (*PasswordResetLink, error) {
	user, err := uc.storage.Users().First(ctx, &UserFilter{
		Email: filter.Equals(strings.ToLower(strings.TrimSpace(params.Email))),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	rawToken, err := generateRandomString(32)
	if err != nil {
		return nil, fmt.Errorf("generating reset token: %w", err)
	}

	now := time.Now()
	token := &PasswordResetToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashToken(rawToken),
		ExpiresAt: now.Add(PasswordResetTokenTTL),
		CreatedAt: now,
	}

	if err := uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		if params.Cooldown > 0 {
			current, err := tx.PasswordResetTokens().First(ctx, &PasswordResetTokenFilter{UserID: filter.Equals(user.ID)})
			if err != nil && !errors.Is(err, ErrPasswordResetTokenNotFound) {
				return fmt.Errorf("fetching current reset token: %w", err)
			}

			if current != nil && now.Sub(current.CreatedAt) < params.Cooldown {
				return ErrPasswordResetCooldown
			}
		}

		if err := tx.PasswordResetTokens().Delete(ctx, &PasswordResetTokenFilter{UserID: filter.Equals(user.ID)}); err != nil {
			return fmt.Errorf("deleting previous reset token: %w", err)
		}

		if _, err := tx.PasswordResetTokens().Create(ctx, token); err != nil {
			return fmt.Errorf("storing reset token: %w", err)
		}

		return nil
	}); err != nil {
		return nil, err
	}

	return &PasswordResetLink{
		User:      user,
		URL:       strings.TrimRight(uc.cfg.FrontendURL, "/") + "/reset-password?token=" + url.QueryEscape(rawToken),
		ExpiresAt: token.ExpiresAt,
	}, nil
}
