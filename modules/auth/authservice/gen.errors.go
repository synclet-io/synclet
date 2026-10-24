package authservice

import (
	fmt "fmt"
	// user code 'imports'
	// end user code 'imports'
)

type NotFoundError string

func (n NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", string(n))
}

type AlreadyExistsError string

func (a AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists", string(a))
}

const (
	ErrUserNotFound      = NotFoundError("User")
	ErrUserAlreadyExists = AlreadyExistsError("User")
)
const (
	ErrRefreshTokenNotFound      = NotFoundError("RefreshToken")
	ErrRefreshTokenAlreadyExists = AlreadyExistsError("RefreshToken")
)
const (
	ErrPasswordResetTokenNotFound      = NotFoundError("PasswordResetToken")
	ErrPasswordResetTokenAlreadyExists = AlreadyExistsError("PasswordResetToken")
)
const (
	ErrAPIKeyNotFound      = NotFoundError("APIKey")
	ErrAPIKeyAlreadyExists = AlreadyExistsError("APIKey")
)
const (
	ErrOAuthClientNotFound      = NotFoundError("OAuthClient")
	ErrOAuthClientAlreadyExists = AlreadyExistsError("OAuthClient")
)
const (
	ErrOAuthCodeNotFound      = NotFoundError("OAuthCode")
	ErrOAuthCodeAlreadyExists = AlreadyExistsError("OAuthCode")
)
const (
	ErrOAuthGrantNotFound      = NotFoundError("OAuthGrant")
	ErrOAuthGrantAlreadyExists = AlreadyExistsError("OAuthGrant")
)
const (
	ErrOAuthRefreshTokenNotFound      = NotFoundError("OAuthRefreshToken")
	ErrOAuthRefreshTokenAlreadyExists = AlreadyExistsError("OAuthRefreshToken")
)
const (
	ErrOIDCIdentityNotFound      = NotFoundError("OIDCIdentity")
	ErrOIDCIdentityAlreadyExists = AlreadyExistsError("OIDCIdentity")
)
const (
	ErrOIDCStateNotFound      = NotFoundError("OIDCState")
	ErrOIDCStateAlreadyExists = AlreadyExistsError("OIDCState")
)
const (
	ErrRateLimitNotFound      = NotFoundError("RateLimit")
	ErrRateLimitAlreadyExists = AlreadyExistsError("RateLimit")
)
