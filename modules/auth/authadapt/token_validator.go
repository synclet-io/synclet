package authadapt

import (
	"context"

	"github.com/synclet-io/synclet/modules/auth/authservice"
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

func (a *TokenValidator) ValidateAPIKey(ctx context.Context, rawKey string) (userID, workspaceID string, err error) {
	apiKey, err := a.validateAPIKey.Execute(ctx, rawKey)
	if err != nil {
		return "", "", err
	}

	return apiKey.UserID.String(), apiKey.WorkspaceID.String(), nil
}
