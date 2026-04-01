package authservice

import (
	"context"
	"fmt"
)

// LoginWithUserInfoParams holds parameters for login with user info.
type LoginWithUserInfoParams struct {
	Email    string
	Password string
}

// LoginWithUserInfoResult holds the result of login with user info.
type LoginWithUserInfoResult struct {
	Tokens *TokenPair
	User   *User
}

// LoginWithUserInfo authenticates a user and returns tokens along with full user info.
type LoginWithUserInfo struct {
	login               *Login
	validateAccessToken *ValidateAccessToken
	getUserByID         *GetUserByID
}

// NewLoginWithUserInfo creates a new LoginWithUserInfo use case.
func NewLoginWithUserInfo(login *Login, validateAccessToken *ValidateAccessToken, getUserByID *GetUserByID) *LoginWithUserInfo {
	return &LoginWithUserInfo{
		login:               login,
		validateAccessToken: validateAccessToken,
		getUserByID:         getUserByID,
	}
}

// Execute logs in and returns tokens with the full user profile.
func (uc *LoginWithUserInfo) Execute(ctx context.Context, params LoginWithUserInfoParams) (*LoginWithUserInfoResult, error) {
	tokens, err := uc.login.Execute(ctx, params.Email, params.Password)
	if err != nil {
		return nil, ErrInvalidCredentials
	}

	claims, err := uc.validateAccessToken.Execute(tokens.AccessToken)
	if err != nil {
		return nil, fmt.Errorf("validating access token: %w", err)
	}

	user, err := uc.getUserByID.Execute(ctx, claims.UserID)
	if err != nil {
		return nil, fmt.Errorf("getting user: %w", err)
	}

	return &LoginWithUserInfoResult{
		Tokens: tokens,
		User:   user,
	}, nil
}
