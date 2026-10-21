package authadapt

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// TokenValidator adapts auth use cases to connectutil.TokenValidator.
type TokenValidator struct {
	validateAccessToken *authservice.ValidateAccessToken
	validateAPIKey      *authservice.ValidateAPIKey
}

// NewTokenValidator creates a new TokenValidator.
func NewTokenValidator(validateAccessToken *authservice.ValidateAccessToken, validateAPIKey *authservice.ValidateAPIKey) *TokenValidator {
	return &TokenValidator{
		validateAccessToken: validateAccessToken,
		validateAPIKey:      validateAPIKey,
	}
}

func (a *TokenValidator) ValidateAccessToken(tokenString string) (userID, email string, err error) {
	claims, err := a.validateAccessToken.Execute(tokenString)
	if err != nil {
		return "", "", err
	}

	return claims.UserID.String(), claims.Email, nil
}

// ValidateAPIKey accepts a plain API key only: a key issued to an OAuth client
// is limited by its level, which only /mcp enforces.
func (a *TokenValidator) ValidateAPIKey(ctx context.Context, rawKey string) (userID, workspaceID string, err error) {
	apiKey, err := a.validate(ctx, rawKey)
	if err != nil {
		return "", "", err
	}

	if apiKey.OAuthLevel != nil {
		return "", "", fmt.Errorf("%w: OAuth key outside /mcp", connectutil.ErrInvalidCredential)
	}

	return apiKey.UserID.String(), apiKey.WorkspaceID.String(), nil
}

// MCPKey is what an API key grants on /mcp. Level is "" for a plain API key,
// or the OAuth access level: "read", "actions" or "write".
type MCPKey struct {
	UserID      string
	WorkspaceID string
	Level       string
	ExpiresAt   *time.Time
}

// ValidateMCPKey accepts a plain API key and a key issued to an OAuth client.
func (a *TokenValidator) ValidateMCPKey(ctx context.Context, rawKey string) (MCPKey, error) {
	apiKey, err := a.validate(ctx, rawKey)
	if err != nil {
		return MCPKey{}, err
	}

	key := MCPKey{UserID: apiKey.UserID.String(), WorkspaceID: apiKey.WorkspaceID.String(), ExpiresAt: apiKey.ExpiresAt}
	if apiKey.OAuthLevel != nil {
		key.Level = mcpLevel(*apiKey.OAuthLevel)
	}

	return key, nil
}

func (a *TokenValidator) validate(ctx context.Context, rawKey string) (*authservice.APIKey, error) {
	apiKey, err := a.validateAPIKey.Execute(ctx, rawKey)
	if errors.Is(err, authservice.ErrInvalidAPIKey) || errors.Is(err, authservice.ErrAPIKeyExpired) {
		return nil, fmt.Errorf("%w: %w", connectutil.ErrInvalidCredential, err)
	}

	if err != nil {
		return nil, err
	}

	return apiKey, nil
}

func mcpLevel(level authservice.OAuthAccessLevel) string {
	switch level {
	case authservice.OAuthAccessLevelRead:
		return "read"
	case authservice.OAuthAccessLevelActions:
		return "actions"
	case authservice.OAuthAccessLevelWrite:
		return "write"
	default:
		return "read"
	}
}
