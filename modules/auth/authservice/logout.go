package authservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// Logout invalidates a refresh token.
type Logout struct {
	storage Storage
	audit   AuditRecorder
}

// NewLogout creates a new Logout use case.
func NewLogout(storage Storage, audit AuditRecorder) *Logout {
	return &Logout{storage: storage, audit: audit}
}

// Execute deletes the refresh token matching the given raw token. An unknown
// token is a no-op.
func (uc *Logout) Execute(ctx context.Context, refreshToken string) error {
	tokenFilter := &RefreshTokenFilter{
		TokenHash: filter.Equals(hashToken(refreshToken)),
	}

	token, err := uc.storage.RefreshTokens().First(ctx, tokenFilter)
	if err != nil {
		if errors.Is(err, ErrRefreshTokenNotFound) {
			return nil
		}

		return fmt.Errorf("fetching refresh token: %w", err)
	}

	if err := uc.storage.RefreshTokens().Delete(ctx, tokenFilter); err != nil {
		return fmt.Errorf("deleting refresh token: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		ActorID:      token.UserID,
		Action:       "logout",
		ResourceType: "user",
		ResourceID:   token.UserID,
	})

	return nil
}
