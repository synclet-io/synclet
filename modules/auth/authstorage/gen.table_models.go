package authstorage

import (
	time "time"

	uuid "github.com/google/uuid"

	authservice "github.com/synclet-io/synclet/modules/auth/authservice"
	// user code 'imports'
	// end user code 'imports'
)

type dbUser struct {
	ID           uuid.UUID `gorm:"column:id"`
	Email        string    `gorm:"column:email;type:text"`
	PasswordHash string    `gorm:"column:password_hash;type:text"`
	Name         string    `gorm:"column:name;type:text"`
	CreatedAt    time.Time `gorm:"column:created_at"`
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
	ID        uuid.UUID `gorm:"column:id"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	TokenHash string    `gorm:"column:token_hash;type:text"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time `gorm:"column:created_at"`
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

type dbPasswordResetToken struct {
	ID        uuid.UUID `gorm:"column:id"`
	UserID    uuid.UUID `gorm:"column:user_id"`
	TokenHash string    `gorm:"column:token_hash;type:text"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
	CreatedAt time.Time `gorm:"column:created_at"`
}

func convertPasswordResetTokenToDB(src *authservice.PasswordResetToken) (*dbPasswordResetToken, error) {
	result := &dbPasswordResetToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertPasswordResetTokenFromDB(src *dbPasswordResetToken) (*authservice.PasswordResetToken, error) {
	result := &authservice.PasswordResetToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbPasswordResetToken) TableName() string {
	return "auth.password_reset_tokens"
}

type dbAPIKey struct {
	ID           uuid.UUID  `gorm:"column:id"`
	WorkspaceID  uuid.UUID  `gorm:"column:workspace_id"`
	UserID       uuid.UUID  `gorm:"column:user_id"`
	Name         string     `gorm:"column:name;type:text"`
	KeyHash      string     `gorm:"column:key_hash;type:text"`
	LastUsedAt   *time.Time `gorm:"column:last_used_at"`
	ExpiresAt    *time.Time `gorm:"column:expires_at"`
	OAuthGrantID *uuid.UUID `gorm:"column:oauth_grant_id"`
	OAuthLevel   *string    `gorm:"column:oauth_level;type:text"`
	CreatedAt    time.Time  `gorm:"column:created_at"`
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
	if src.OAuthGrantID == nil {
		result.OAuthGrantID = nil
	} else {
		result.OAuthGrantID = toPtr(fromPtr(src.OAuthGrantID))
	}
	if src.OAuthLevel == nil {
		result.OAuthLevel = nil
	} else {
		tmp12, err := convertOAuthAccessLevelToDB(fromPtr(src.OAuthLevel))
		if err != nil {
			return nil, err
		}
		result.OAuthLevel = toPtr(tmp12)
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
	if src.OAuthGrantID == nil {
		result.OAuthGrantID = nil
	} else {
		result.OAuthGrantID = toPtr(fromPtr(src.OAuthGrantID))
	}
	if src.OAuthLevel == nil {
		result.OAuthLevel = nil
	} else {
		tmp26, err := convertOAuthAccessLevelFromDB(fromPtr(src.OAuthLevel))
		if err != nil {
			return nil, err
		}
		result.OAuthLevel = toPtr(tmp26)
	}
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbAPIKey) TableName() string {
	return "auth.api_keys"
}

type dbOAuthClient struct {
	ID            uuid.UUID        `gorm:"column:id"`
	ClientID      string           `gorm:"column:client_id;type:text"`
	Name          string           `gorm:"column:name;type:text"`
	RedirectURIs  stringSliceValue `gorm:"column:redirect_uris"`
	LastGrantedAt *time.Time       `gorm:"column:last_granted_at"`
	CreatedAt     time.Time        `gorm:"column:created_at"`
}

func convertOAuthClientToDB(src *authservice.OAuthClient) (*dbOAuthClient, error) {
	result := &dbOAuthClient{}
	result.ID = src.ID
	result.ClientID = src.ClientID
	result.Name = src.Name
	tmp3 := make(stringSliceValue, 0, len(src.RedirectURIs))
	for _, el := range src.RedirectURIs {
		tmp3 = append(tmp3, el)
	}
	result.RedirectURIs = tmp3
	if src.LastGrantedAt == nil {
		result.LastGrantedAt = nil
	} else {
		result.LastGrantedAt = toPtr((fromPtr(src.LastGrantedAt)).UTC())
	}
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertOAuthClientFromDB(src *dbOAuthClient) (*authservice.OAuthClient, error) {
	result := &authservice.OAuthClient{}
	result.ID = src.ID
	result.ClientID = src.ClientID
	result.Name = src.Name
	tmp11 := make([]string, 0, len(src.RedirectURIs))
	for _, el := range src.RedirectURIs {

		tmp11 = append(tmp11, el)
	}
	result.RedirectURIs = tmp11
	if src.LastGrantedAt == nil {
		result.LastGrantedAt = nil
	} else {
		result.LastGrantedAt = toPtr(fromPtr(src.LastGrantedAt))
	}
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbOAuthClient) TableName() string {
	return "auth.oauth_clients"
}

type dbOAuthCode struct {
	ID            uuid.UUID `gorm:"column:id"`
	CodeHash      string    `gorm:"column:code_hash;type:text"`
	ClientID      string    `gorm:"column:client_id;type:text"`
	RedirectURI   string    `gorm:"column:redirect_uri;type:text"`
	CodeChallenge string    `gorm:"column:code_challenge;type:text"`
	UserID        uuid.UUID `gorm:"column:user_id"`
	WorkspaceID   uuid.UUID `gorm:"column:workspace_id"`
	Level         string    `gorm:"column:level;type:text"`
	ExpiresAt     time.Time `gorm:"column:expires_at"`
	CreatedAt     time.Time `gorm:"column:created_at"`
}

func convertOAuthCodeToDB(src *authservice.OAuthCode) (*dbOAuthCode, error) {
	result := &dbOAuthCode{}
	result.ID = src.ID
	result.CodeHash = src.CodeHash
	result.ClientID = src.ClientID
	result.RedirectURI = src.RedirectURI
	result.CodeChallenge = src.CodeChallenge
	result.UserID = src.UserID
	result.WorkspaceID = src.WorkspaceID
	tmp7, err := convertOAuthAccessLevelToDB(src.Level)
	if err != nil {
		return nil, err
	}
	result.Level = tmp7
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertOAuthCodeFromDB(src *dbOAuthCode) (*authservice.OAuthCode, error) {
	result := &authservice.OAuthCode{}
	result.ID = src.ID
	result.CodeHash = src.CodeHash
	result.ClientID = src.ClientID
	result.RedirectURI = src.RedirectURI
	result.CodeChallenge = src.CodeChallenge
	result.UserID = src.UserID
	result.WorkspaceID = src.WorkspaceID
	tmp17, err := convertOAuthAccessLevelFromDB(src.Level)
	if err != nil {
		return nil, err
	}
	result.Level = tmp17
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbOAuthCode) TableName() string {
	return "auth.oauth_codes"
}

type dbOAuthGrant struct {
	ID          uuid.UUID `gorm:"column:id"`
	ClientID    string    `gorm:"column:client_id;type:text"`
	UserID      uuid.UUID `gorm:"column:user_id"`
	WorkspaceID uuid.UUID `gorm:"column:workspace_id"`
	Level       string    `gorm:"column:level;type:text"`
	CreatedAt   time.Time `gorm:"column:created_at"`
}

func convertOAuthGrantToDB(src *authservice.OAuthGrant) (*dbOAuthGrant, error) {
	result := &dbOAuthGrant{}
	result.ID = src.ID
	result.ClientID = src.ClientID
	result.UserID = src.UserID
	result.WorkspaceID = src.WorkspaceID
	tmp4, err := convertOAuthAccessLevelToDB(src.Level)
	if err != nil {
		return nil, err
	}
	result.Level = tmp4
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertOAuthGrantFromDB(src *dbOAuthGrant) (*authservice.OAuthGrant, error) {
	result := &authservice.OAuthGrant{}
	result.ID = src.ID
	result.ClientID = src.ClientID
	result.UserID = src.UserID
	result.WorkspaceID = src.WorkspaceID
	tmp10, err := convertOAuthAccessLevelFromDB(src.Level)
	if err != nil {
		return nil, err
	}
	result.Level = tmp10
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbOAuthGrant) TableName() string {
	return "auth.oauth_grants"
}

type dbOAuthRefreshToken struct {
	ID        uuid.UUID  `gorm:"column:id"`
	GrantID   uuid.UUID  `gorm:"column:grant_id"`
	TokenHash string     `gorm:"column:token_hash;type:text"`
	UsedAt    *time.Time `gorm:"column:used_at"`
	ExpiresAt time.Time  `gorm:"column:expires_at"`
	CreatedAt time.Time  `gorm:"column:created_at"`
}

func convertOAuthRefreshTokenToDB(src *authservice.OAuthRefreshToken) (*dbOAuthRefreshToken, error) {
	result := &dbOAuthRefreshToken{}
	result.ID = src.ID
	result.GrantID = src.GrantID
	result.TokenHash = src.TokenHash
	if src.UsedAt == nil {
		result.UsedAt = nil
	} else {
		result.UsedAt = toPtr((fromPtr(src.UsedAt)).UTC())
	}
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertOAuthRefreshTokenFromDB(src *dbOAuthRefreshToken) (*authservice.OAuthRefreshToken, error) {
	result := &authservice.OAuthRefreshToken{}
	result.ID = src.ID
	result.GrantID = src.GrantID
	result.TokenHash = src.TokenHash
	if src.UsedAt == nil {
		result.UsedAt = nil
	} else {
		result.UsedAt = toPtr(fromPtr(src.UsedAt))
	}
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt
	return result, nil
}
func (a dbOAuthRefreshToken) TableName() string {
	return "auth.oauth_refresh_tokens"
}

type dbOIDCIdentity struct {
	ID           uuid.UUID `gorm:"column:id"`
	UserID       uuid.UUID `gorm:"column:user_id"`
	ProviderSlug string    `gorm:"column:provider_slug;type:text"`
	Subject      string    `gorm:"column:subject;type:text"`
	Email        string    `gorm:"column:email;type:text"`
	CreatedAt    time.Time `gorm:"column:created_at"`
	LastLoginAt  time.Time `gorm:"column:last_login_at"`
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
	ID           uuid.UUID `gorm:"column:id"`
	State        string    `gorm:"column:state;type:text"`
	Verifier     string    `gorm:"column:verifier;type:text"`
	ProviderSlug string    `gorm:"column:provider_slug;type:text"`
	ExpiresAt    time.Time `gorm:"column:expires_at"`
	CreatedAt    time.Time `gorm:"column:created_at"`
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

type dbRateLimit struct {
	ID        uuid.UUID `gorm:"column:id"`
	Key       string    `gorm:"column:key;type:text"`
	Count     int       `gorm:"column:count"`
	ExpiresAt time.Time `gorm:"column:expires_at"`
}

func convertRateLimitToDB(src *authservice.RateLimit) (*dbRateLimit, error) {
	result := &dbRateLimit{}
	result.ID = src.ID
	result.Key = src.Key
	result.Count = src.Count
	result.ExpiresAt = (src.ExpiresAt).UTC()
	return result, nil
}

func convertRateLimitFromDB(src *dbRateLimit) (*authservice.RateLimit, error) {
	result := &authservice.RateLimit{}
	result.ID = src.ID
	result.Key = src.Key
	result.Count = src.Count
	result.ExpiresAt = src.ExpiresAt
	return result, nil
}
func (a dbRateLimit) TableName() string {
	return "auth.rate_limits"
}
