package authstorage

import (
	time "time"

	uuid "github.com/google/uuid"

	authservice "github.com/synclet-io/synclet/modules/auth/authservice"
	// user code 'imports'
	// end user code 'imports'
)

type dbUser struct {
	ID           uuid.UUID `gorm:"column:id;"`
	Email        string    `gorm:"column:email;type:text;"`
	PasswordHash string    `gorm:"column:password_hash;type:text;"`
	Name         string    `gorm:"column:name;type:text;"`
	CreatedAt    time.Time `gorm:"column:created_at;"`
}

func convertUserToDB(src *authservice.User) (*dbUser, error) {
	result := &dbUser{}
	result.ID = src.ID
	result.Email = src.Email
	result.PasswordHash = src.PasswordHash
	result.Name = src.Name
	result.CreatedAt = (src.CreatedAt).UTC()

	return result, nil
}

func convertUserFromDB(src *dbUser) (*authservice.User, error) {
	result := &authservice.User{}
	result.ID = src.ID
	result.Email = src.Email
	result.PasswordHash = src.PasswordHash
	result.Name = src.Name
	result.CreatedAt = src.CreatedAt

	return result, nil
}
func (a dbUser) TableName() string {
	return "auth.users"
}

type dbRefreshToken struct {
	ID        uuid.UUID `gorm:"column:id;"`
	UserID    uuid.UUID `gorm:"column:user_id;"`
	TokenHash string    `gorm:"column:token_hash;type:text;"`
	ExpiresAt time.Time `gorm:"column:expires_at;"`
	CreatedAt time.Time `gorm:"column:created_at;"`
}

func convertRefreshTokenToDB(src *authservice.RefreshToken) (*dbRefreshToken, error) {
	result := &dbRefreshToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()

	return result, nil
}

func convertRefreshTokenFromDB(src *dbRefreshToken) (*authservice.RefreshToken, error) {
	result := &authservice.RefreshToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt

	return result, nil
}
func (a dbRefreshToken) TableName() string {
	return "auth.refresh_tokens"
}

type dbAPIKey struct {
	ID          uuid.UUID  `gorm:"column:id;"`
	WorkspaceID uuid.UUID  `gorm:"column:workspace_id;"`
	UserID      uuid.UUID  `gorm:"column:user_id;"`
	Name        string     `gorm:"column:name;type:text;"`
	KeyHash     string     `gorm:"column:key_hash;type:text;"`
	LastUsedAt  *time.Time `gorm:"column:last_used_at;"`
	ExpiresAt   *time.Time `gorm:"column:expires_at;"`
	CreatedAt   time.Time  `gorm:"column:created_at;"`
}

func convertAPIKeyToDB(src *authservice.APIKey) (*dbAPIKey, error) {
	result := &dbAPIKey{}
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

func convertAPIKeyFromDB(src *dbAPIKey) (*authservice.APIKey, error) {
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
func (a dbAPIKey) TableName() string {
	return "auth.api_keys"
}

type dbOIDCIdentity struct {
	ID           uuid.UUID `gorm:"column:id;"`
	UserID       uuid.UUID `gorm:"column:user_id;"`
	ProviderSlug string    `gorm:"column:provider_slug;type:text;"`
	Subject      string    `gorm:"column:subject;type:text;"`
	Email        string    `gorm:"column:email;type:text;"`
	CreatedAt    time.Time `gorm:"column:created_at;"`
	LastLoginAt  time.Time `gorm:"column:last_login_at;"`
}

func convertOIDCIdentityToDB(src *authservice.OIDCIdentity) (*dbOIDCIdentity, error) {
	result := &dbOIDCIdentity{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.ProviderSlug = src.ProviderSlug
	result.Subject = src.Subject
	result.Email = src.Email
	result.CreatedAt = (src.CreatedAt).UTC()
	result.LastLoginAt = (src.LastLoginAt).UTC()

	return result, nil
}

func convertOIDCIdentityFromDB(src *dbOIDCIdentity) (*authservice.OIDCIdentity, error) {
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
func (a dbOIDCIdentity) TableName() string {
	return "auth.oidc_identities"
}

type dbOIDCState struct {
	ID           uuid.UUID `gorm:"column:id;"`
	State        string    `gorm:"column:state;type:text;"`
	Verifier     string    `gorm:"column:verifier;type:text;"`
	ProviderSlug string    `gorm:"column:provider_slug;type:text;"`
	ExpiresAt    time.Time `gorm:"column:expires_at;"`
	CreatedAt    time.Time `gorm:"column:created_at;"`
}

func convertOIDCStateToDB(src *authservice.OIDCState) (*dbOIDCState, error) {
	result := &dbOIDCState{}
	result.ID = src.ID
	result.State = src.State
	result.Verifier = src.Verifier
	result.ProviderSlug = src.ProviderSlug
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()

	return result, nil
}

func convertOIDCStateFromDB(src *dbOIDCState) (*authservice.OIDCState, error) {
	result := &authservice.OIDCState{}
	result.ID = src.ID
	result.State = src.State
	result.Verifier = src.Verifier
	result.ProviderSlug = src.ProviderSlug
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt

	return result, nil
}
func (a dbOIDCState) TableName() string {
	return "auth.oidc_states"
}
