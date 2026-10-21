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

type jsonPasswordResetToken struct {
	ID        uuid.UUID `json:"id"`
	UserID    uuid.UUID `json:"user_id"`
	TokenHash string    `json:"token_hash"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

func (p *jsonPasswordResetToken) Scan(value any) error {
	return json.Unmarshal(value.([]byte), p)
}

func (p jsonPasswordResetToken) Value() (driver.Value, error) {
	return json.Marshal(p)
}

func convertPasswordResetTokenToJsonModel(src *authservice.PasswordResetToken) (*jsonPasswordResetToken, error) {
	result := &jsonPasswordResetToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = (src.ExpiresAt).UTC()
	result.CreatedAt = (src.CreatedAt).UTC()
	return result, nil
}

func convertPasswordResetTokenFromJsonModel(src *jsonPasswordResetToken) (*authservice.PasswordResetToken, error) {
	result := &authservice.PasswordResetToken{}
	result.ID = src.ID
	result.UserID = src.UserID
	result.TokenHash = src.TokenHash
	result.ExpiresAt = src.ExpiresAt
	result.CreatedAt = src.CreatedAt
	return result, nil
}

type jsonAPIKey struct {
	ID           uuid.UUID  `json:"id"`
	WorkspaceID  uuid.UUID  `json:"workspace_id"`
	UserID       uuid.UUID  `json:"user_id"`
	Name         string     `json:"name"`
	KeyHash      string     `json:"key_hash"`
	LastUsedAt   *time.Time `json:"last_used_at"`
	ExpiresAt    *time.Time `json:"expires_at"`
	OAuthGrantID *uuid.UUID `json:"o_auth_grant_id"`
	OAuthLevel   *string    `json:"o_auth_level"`
	CreatedAt    time.Time  `json:"created_at"`
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

type jsonOAuthClient struct {
	ID            uuid.UUID        `json:"id"`
	ClientID      string           `json:"client_id"`
	Name          string           `json:"name"`
	RedirectURIs  stringSliceValue `json:"redirect_ur_is"`
	LastGrantedAt *time.Time       `json:"last_granted_at"`
	CreatedAt     time.Time        `json:"created_at"`
}

func (o *jsonOAuthClient) Scan(value any) error {
	return json.Unmarshal(value.([]byte), o)
}

func (o jsonOAuthClient) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func convertOAuthClientToJsonModel(src *authservice.OAuthClient) (*jsonOAuthClient, error) {
	result := &jsonOAuthClient{}
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

func convertOAuthClientFromJsonModel(src *jsonOAuthClient) (*authservice.OAuthClient, error) {
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

type jsonOAuthCode struct {
	ID            uuid.UUID `json:"id"`
	CodeHash      string    `json:"code_hash"`
	ClientID      string    `json:"client_id"`
	RedirectURI   string    `json:"redirect_uri"`
	CodeChallenge string    `json:"code_challenge"`
	UserID        uuid.UUID `json:"user_id"`
	WorkspaceID   uuid.UUID `json:"workspace_id"`
	Level         string    `json:"level"`
	ExpiresAt     time.Time `json:"expires_at"`
	CreatedAt     time.Time `json:"created_at"`
}

func (o *jsonOAuthCode) Scan(value any) error {
	return json.Unmarshal(value.([]byte), o)
}

func (o jsonOAuthCode) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func convertOAuthCodeToJsonModel(src *authservice.OAuthCode) (*jsonOAuthCode, error) {
	result := &jsonOAuthCode{}
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

func convertOAuthCodeFromJsonModel(src *jsonOAuthCode) (*authservice.OAuthCode, error) {
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

type jsonOAuthGrant struct {
	ID          uuid.UUID `json:"id"`
	ClientID    string    `json:"client_id"`
	UserID      uuid.UUID `json:"user_id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Level       string    `json:"level"`
	CreatedAt   time.Time `json:"created_at"`
}

func (o *jsonOAuthGrant) Scan(value any) error {
	return json.Unmarshal(value.([]byte), o)
}

func (o jsonOAuthGrant) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func convertOAuthGrantToJsonModel(src *authservice.OAuthGrant) (*jsonOAuthGrant, error) {
	result := &jsonOAuthGrant{}
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

func convertOAuthGrantFromJsonModel(src *jsonOAuthGrant) (*authservice.OAuthGrant, error) {
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

type jsonOAuthRefreshToken struct {
	ID        uuid.UUID  `json:"id"`
	GrantID   uuid.UUID  `json:"grant_id"`
	TokenHash string     `json:"token_hash"`
	UsedAt    *time.Time `json:"used_at"`
	ExpiresAt time.Time  `json:"expires_at"`
	CreatedAt time.Time  `json:"created_at"`
}

func (o *jsonOAuthRefreshToken) Scan(value any) error {
	return json.Unmarshal(value.([]byte), o)
}

func (o jsonOAuthRefreshToken) Value() (driver.Value, error) {
	return json.Marshal(o)
}

func convertOAuthRefreshTokenToJsonModel(src *authservice.OAuthRefreshToken) (*jsonOAuthRefreshToken, error) {
	result := &jsonOAuthRefreshToken{}
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

func convertOAuthRefreshTokenFromJsonModel(src *jsonOAuthRefreshToken) (*authservice.OAuthRefreshToken, error) {
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
