package authstorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	time "time"

	uuid "github.com/google/uuid"

	authservice "github.com/synclet-io/synclet/modules/auth/authservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonUser struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"password_hash"`
	Name         string    `json:"name"`
	CreatedAt    time.Time `json:"created_at"`
}

func (u *jsonUser) Scan(value any) error {
	return json.Unmarshal(value.([]byte), u)
}

func (u jsonUser) Value() (driver.Value, error) {
	return json.Marshal(u)
}

func convertUserToJsonModel(src *authservice.User) (*jsonUser, error) {
	result := &jsonUser{}
	result.ID = src.ID
	result.Email = src.Email
	result.PasswordHash = src.PasswordHash
	result.Name = src.Name
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertUserFromJsonModel(src *jsonUser) (*authservice.User, error) {
	result := &authservice.User{}
	result.ID = src.ID
	result.Email = src.Email
	result.PasswordHash = src.PasswordHash
	result.Name = src.Name
	result.CreatedAt = src.CreatedAt
	return result, nil
}

type jsonRefreshToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (r *jsonRefreshToken) Scan(value any) error {
	return json.Unmarshal(value.([]byte), r)
}

func (r jsonRefreshToken) Value() (driver.Value, error) {
	return json.Marshal(r)
}

func convertRefreshTokenToJsonModel(src *authservice.RefreshToken) (*jsonRefreshToken, error) {
	result := &jsonRefreshToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertRefreshTokenFromJsonModel(src *jsonRefreshToken) (*authservice.RefreshToken, error) {
	result := &authservice.RefreshToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt
	return result, nil
}

type jsonAPIKey struct {
	ID          uuid.UUID  `json:"id"`
	WorkspaceID uuid.UUID  `json:"workspace_id"`
	UserID      uuid.UUID  `json:"user_id"`
	Name        string     `json:"name"`
	KeyHash     string     `json:"key_hash"`
	LastUsedAt  *time.Time `json:"last_used_at"`
	ExpiresAt   *time.Time `json:"expires_at"`
	CreatedAt   time.Time  `json:"created_at"`
}

func (a *jsonAPIKey) Scan(value any) error {
	return json.Unmarshal(value.([]byte), a)
}

func (a jsonAPIKey) Value() (driver.Value, error) {
	return json.Marshal(a)
}

func convertAPIKeyToJsonModel(src *authservice.APIKey) (*jsonAPIKey, error) {
	result := &jsonAPIKey{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.UserID = src.UserID
	result.Name = src.Name
	result.KeyHash = src.KeyHash
	if src.LastUsedAt == nil {
		result.LastUsedAt = nil
	} else {
		result.LastUsedAt = toPtr((fromPtr(src.LastUsedAt)).UTC())
	}
	if src.ExpiresAt == nil {
		result.ExpiresAt = nil
	} else {
		result.ExpiresAt = toPtr((fromPtr(src.ExpiresAt)).UTC())
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertAPIKeyFromJsonModel(src *jsonAPIKey) (*authservice.APIKey, error) {
	result := &authservice.APIKey{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.UserID = src.UserID
	result.Name = src.Name
	result.KeyHash = src.KeyHash
	if src.LastUsedAt == nil {
		result.LastUsedAt = nil
	} else {
		result.LastUsedAt = toPtr(fromPtr(src.LastUsedAt))
	}
	if src.ExpiresAt == nil {
		result.ExpiresAt = nil
	} else {
		result.ExpiresAt = toPtr(fromPtr(src.ExpiresAt))
	}
	result.CreatedAt = src.CreatedAt
	return result, nil
}

type jsonOIDCIdentity struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	ProviderSlug string    `json:"provider_slug"`
	Subject      string    `json:"subject"`
	Email        string    `json:"email"`
	CreatedAt    time.Time `json:"created_at"`
	LastLoginAt  time.Time `json:"last_login_at"`
}

func (o *jsonOIDCIdentity) Scan(value any) error {
	return json.Unmarshal(value.([]byte), o)
}

func (o jsonOIDCIdentity) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func convertOIDCIdentityToJsonModel(src *authservice.OIDCIdentity) (*jsonOIDCIdentity, error) {
	result := &jsonOIDCIdentity{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.ProviderSlug = src.ProviderSlug
	result.Subject = src.Subject
	result.Email = src.Email
	result.CreatedAt = (src.CreatedAt).UTC()
	result.LastLoginAt = (src.LastLoginAt).UTC()
	return result, nil
}

func convertOIDCIdentityFromJsonModel(src *jsonOIDCIdentity) (*authservice.OIDCIdentity, error) {
	result := &authservice.OIDCIdentity{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.ProviderSlug = src.ProviderSlug
	result.Subject = src.Subject
	result.Email = src.Email
	result.CreatedAt = src.CreatedAt
	result.LastLoginAt = src.LastLoginAt
	return result, nil
}

type jsonOIDCState struct {
	ID           uuid.UUID `json:"id"`
	State        string    `json:"state"`
	Verifier     string    `json:"verifier"`
	ProviderSlug string    `json:"provider_slug"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

func (o *jsonOIDCState) Scan(value any) error {
	return json.Unmarshal(value.([]byte), o)
}

func (o jsonOIDCState) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func convertOIDCStateToJsonModel(src *authservice.OIDCState) (*jsonOIDCState, error) {
	result := &jsonOIDCState{}
	result.ID = src.ID
	result.State = src.State
	result.Verifier = src.Verifier
	result.ProviderSlug = src.ProviderSlug
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertOIDCStateFromJsonModel(src *jsonOIDCState) (*authservice.OIDCState, error) {
	result := &authservice.OIDCState{}
	result.ID = src.ID
	result.State = src.State
	result.Verifier = src.Verifier
	result.ProviderSlug = src.ProviderSlug
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt
	return result, nil
}
