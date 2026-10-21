package authservice

type baseErr string

func (e baseErr) Error() string { return string(e) }

const (
	ErrInvalidCredentials      baseErr = "invalid credentials" //nolint:gosec // error message, not a credential
	ErrInvalidToken            baseErr = "invalid token"
	ErrUnexpectedSigningMethod baseErr = "unexpected signing method"
	ErrInvalidRefreshToken     baseErr = "invalid refresh token"
	ErrRefreshTokenExpired     baseErr = "refresh token expired"
	ErrInvalidCurrentPassword  baseErr = "invalid current password"
	ErrInvalidAPIKey           baseErr = "invalid API key" //nolint:gosec // error message, not a credential
	ErrAPIKeyExpired           baseErr = "API key expired" //nolint:gosec // error message, not a credential
	ErrInvalidOrExpiredState   baseErr = "invalid or expired state"
	ErrStateProviderMismatch   baseErr = "state provider mismatch"
	ErrMissingIDToken          baseErr = "missing id_token in token response"
	ErrEmailNotVerified        baseErr = "email not verified by provider"
	ErrInvalidEmailFormat      baseErr = "invalid email format"
	ErrRegistrationDisabled    baseErr = "registration is disabled"
	ErrPasswordResetCooldown   baseErr = "password reset requested too recently"
	ErrInvalidResetToken       baseErr = "invalid or expired reset token" //nolint:gosec // error message, not a credential
	// ErrInvalidGrant is the OAuth invalid_grant error: an authorization code
	// or refresh token that is unknown, expired, already used or bound to
	// another client, redirect URI or code verifier.
	ErrInvalidGrant baseErr = "invalid grant"
)

// ValidationError represents an input validation error in the auth module.
type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return "validation error: " + e.Message
}
