package authservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

type UserField byte

const (
	UserFieldID UserField = iota + 1
	UserFieldEmail
	UserFieldPasswordHash
	UserFieldName
	UserFieldCreatedAt
)

type UserFilter struct {
	ID    filter.Filter[uuid.UUID]
	Email filter.Filter[string]
	Or    []*UserFilter
	And   []*UserFilter
}
type UserOrder order.Order[UserField]

type User struct {
	ID           uuid.UUID
	Email        string
	PasswordHash string
	Name         string
	CreatedAt    time.Time
}

// user code 'User methods'
// end user code 'User methods'

func (u *User) Copy() User {
	var result User
	result.ID = u.ID
	result.Email = u.Email
	result.PasswordHash = u.PasswordHash
	result.Name = u.Name
	result.CreatedAt = u.CreatedAt

	return result
}
func (u *User) Equals(to *User) bool {
	if (u == nil) != (to == nil) {
		return false
	}
	if u == nil && to == nil {
		return true
	}
	if u.ID != to.ID {
		return false
	}
	if u.Email != to.Email {
		return false
	}
	if u.PasswordHash != to.PasswordHash {
		return false
	}
	if u.Name != to.Name {
		return false
	}
	if u.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type RefreshTokenField byte

const (
	RefreshTokenFieldID RefreshTokenField = iota + 1
	RefreshTokenFieldUserID
	RefreshTokenFieldTokenHash
	RefreshTokenFieldExpiresAt
	RefreshTokenFieldCreatedAt
)

type RefreshTokenFilter struct {
	ID        filter.Filter[uuid.UUID]
	UserID    filter.Filter[uuid.UUID]
	TokenHash filter.Filter[string]
	Or        []*RefreshTokenFilter
	And       []*RefreshTokenFilter
}
type RefreshTokenOrder order.Order[RefreshTokenField]

type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// user code 'RefreshToken methods'
// end user code 'RefreshToken methods'

func (r *RefreshToken) Copy() RefreshToken {
	var result RefreshToken
	result.ID = r.ID
	result.UserID = r.UserID
	result.TokenHash = r.TokenHash
	result.ExpiresAt = r.ExpiresAt
	result.CreatedAt = r.CreatedAt

	return result
}
func (r *RefreshToken) Equals(to *RefreshToken) bool {
	if (r == nil) != (to == nil) {
		return false
	}
	if r == nil && to == nil {
		return true
	}
	if r.ID != to.ID {
		return false
	}
	if r.UserID != to.UserID {
		return false
	}
	if r.TokenHash != to.TokenHash {
		return false
	}
	if r.ExpiresAt != to.ExpiresAt {
		return false
	}
	if r.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type PasswordResetTokenField byte

const (
	PasswordResetTokenFieldID PasswordResetTokenField = iota + 1
	PasswordResetTokenFieldUserID
	PasswordResetTokenFieldTokenHash
	PasswordResetTokenFieldExpiresAt
	PasswordResetTokenFieldCreatedAt
)

type PasswordResetTokenFilter struct {
	ID        filter.Filter[uuid.UUID]
	UserID    filter.Filter[uuid.UUID]
	TokenHash filter.Filter[string]
	ExpiresAt filter.Filter[time.Time]
	Or        []*PasswordResetTokenFilter
	And       []*PasswordResetTokenFilter
}
type PasswordResetTokenOrder order.Order[PasswordResetTokenField]

type PasswordResetToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}

// user code 'PasswordResetToken methods'
// end user code 'PasswordResetToken methods'

func (p *PasswordResetToken) Copy() PasswordResetToken {
	var result PasswordResetToken
	result.ID = p.ID
	result.UserID = p.UserID
	result.TokenHash = p.TokenHash
	result.ExpiresAt = p.ExpiresAt
	result.CreatedAt = p.CreatedAt

	return result
}
func (p *PasswordResetToken) Equals(to *PasswordResetToken) bool {
	if (p == nil) != (to == nil) {
		return false
	}
	if p == nil && to == nil {
		return true
	}
	if p.ID != to.ID {
		return false
	}
	if p.UserID != to.UserID {
		return false
	}
	if p.TokenHash != to.TokenHash {
		return false
	}
	if p.ExpiresAt != to.ExpiresAt {
		return false
	}
	if p.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type APIKeyField byte

const (
	APIKeyFieldID APIKeyField = iota + 1
	APIKeyFieldWorkspaceID
	APIKeyFieldUserID
	APIKeyFieldName
	APIKeyFieldKeyHash
	APIKeyFieldLastUsedAt
	APIKeyFieldExpiresAt
	APIKeyFieldOAuthGrantID
	APIKeyFieldOAuthLevel
	APIKeyFieldCreatedAt
)

type APIKeyFilter struct {
	ID           filter.Filter[uuid.UUID]
	WorkspaceID  filter.Filter[uuid.UUID]
	UserID       filter.Filter[uuid.UUID]
	KeyHash      filter.Filter[string]
	OAuthGrantID filter.Filter[*uuid.UUID]
	Or           []*APIKeyFilter
	And          []*APIKeyFilter
}
type APIKeyOrder order.Order[APIKeyField]

type APIKey struct {
	ID           uuid.UUID
	WorkspaceID  uuid.UUID
	UserID       uuid.UUID
	Name         string
	KeyHash      string
	LastUsedAt   *time.Time
	ExpiresAt    *time.Time
	OAuthGrantID *uuid.UUID
	OAuthLevel   *OAuthAccessLevel
	CreatedAt    time.Time
}

// user code 'APIKey methods'
// end user code 'APIKey methods'

func (a *APIKey) Copy() APIKey {
	var result APIKey
	result.ID = a.ID
	result.WorkspaceID = a.WorkspaceID
	result.UserID = a.UserID
	result.Name = a.Name
	result.KeyHash = a.KeyHash
	if a.LastUsedAt != nil {
		var tmp time.Time
		tmp = (*a.LastUsedAt)
		result.LastUsedAt = &tmp
	}
	if a.ExpiresAt != nil {
		var tmp1 time.Time
		tmp1 = (*a.ExpiresAt)
		result.ExpiresAt = &tmp1
	}
	if a.OAuthGrantID != nil {
		var tmp2 uuid.UUID
		tmp2 = (*a.OAuthGrantID)
		result.OAuthGrantID = &tmp2
	}
	if a.OAuthLevel != nil {
		var tmp3 OAuthAccessLevel
		tmp3 = (*a.OAuthLevel) // enum
		result.OAuthLevel = &tmp3
	}
	result.CreatedAt = a.CreatedAt

	return result
}
func (a *APIKey) Equals(to *APIKey) bool {
	if (a == nil) != (to == nil) {
		return false
	}
	if a == nil && to == nil {
		return true
	}
	if a.ID != to.ID {
		return false
	}
	if a.WorkspaceID != to.WorkspaceID {
		return false
	}
	if a.UserID != to.UserID {
		return false
	}
	if a.Name != to.Name {
		return false
	}
	if a.KeyHash != to.KeyHash {
		return false
	}
	if (a.LastUsedAt == nil) != (to.LastUsedAt == nil) {
		return false
	}
	if a.LastUsedAt != nil && to.LastUsedAt != nil {
		if (*a.LastUsedAt) != (*to.LastUsedAt) {
			return false
		}
	}
	if (a.ExpiresAt == nil) != (to.ExpiresAt == nil) {
		return false
	}
	if a.ExpiresAt != nil && to.ExpiresAt != nil {
		if (*a.ExpiresAt) != (*to.ExpiresAt) {
			return false
		}
	}
	if (a.OAuthGrantID == nil) != (to.OAuthGrantID == nil) {
		return false
	}
	if a.OAuthGrantID != nil && to.OAuthGrantID != nil {
		if (*a.OAuthGrantID) != (*to.OAuthGrantID) {
			return false
		}
	}
	if (a.OAuthLevel == nil) != (to.OAuthLevel == nil) {
		return false
	}
	if a.OAuthLevel != nil && to.OAuthLevel != nil {
		if (*a.OAuthLevel) != (*to.OAuthLevel) {
			return false
		}
	}
	if a.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type OAuthClientField byte

const (
	OAuthClientFieldID OAuthClientField = iota + 1
	OAuthClientFieldClientID
	OAuthClientFieldName
	OAuthClientFieldRedirectURIs
	OAuthClientFieldLastGrantedAt
	OAuthClientFieldCreatedAt
)

type OAuthClientFilter struct {
	ID       filter.Filter[uuid.UUID]
	ClientID filter.Filter[string]
	Or       []*OAuthClientFilter
	And      []*OAuthClientFilter
}
type OAuthClientOrder order.Order[OAuthClientField]

type OAuthClient struct {
	ID            uuid.UUID
	ClientID      string
	Name          string
	RedirectURIs  []string
	LastGrantedAt *time.Time
	CreatedAt     time.Time
}

// user code 'OAuthClient methods'
// end user code 'OAuthClient methods'

func (o *OAuthClient) Copy() OAuthClient {
	var result OAuthClient
	result.ID = o.ID
	result.ClientID = o.ClientID
	result.Name = o.Name
	tmp := make([]string, 0, len(o.RedirectURIs))
	for _, i := range o.RedirectURIs {
		var itemCopy string
		itemCopy = i
		tmp = append(tmp, itemCopy)
	}
	result.RedirectURIs = tmp
	if o.LastGrantedAt != nil {
		var tmp1 time.Time
		tmp1 = (*o.LastGrantedAt)
		result.LastGrantedAt = &tmp1
	}
	result.CreatedAt = o.CreatedAt

	return result
}
func (o *OAuthClient) Equals(to *OAuthClient) bool {
	if (o == nil) != (to == nil) {
		return false
	}
	if o == nil && to == nil {
		return true
	}
	if o.ID != to.ID {
		return false
	}
	if o.ClientID != to.ClientID {
		return false
	}
	if o.Name != to.Name {
		return false
	}
	if len(o.RedirectURIs) != len(to.RedirectURIs) {
		return false
	}
	for idx := range o.RedirectURIs {
		if o.RedirectURIs[idx] != to.RedirectURIs[idx] {
			return false
		}
	}
	if (o.LastGrantedAt == nil) != (to.LastGrantedAt == nil) {
		return false
	}
	if o.LastGrantedAt != nil && to.LastGrantedAt != nil {
		if (*o.LastGrantedAt) != (*to.LastGrantedAt) {
			return false
		}
	}
	if o.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type OAuthCodeField byte

const (
	OAuthCodeFieldID OAuthCodeField = iota + 1
	OAuthCodeFieldCodeHash
	OAuthCodeFieldClientID
	OAuthCodeFieldRedirectURI
	OAuthCodeFieldCodeChallenge
	OAuthCodeFieldUserID
	OAuthCodeFieldWorkspaceID
	OAuthCodeFieldLevel
	OAuthCodeFieldExpiresAt
	OAuthCodeFieldCreatedAt
)

type OAuthCodeFilter struct {
	ID        filter.Filter[uuid.UUID]
	CodeHash  filter.Filter[string]
	ClientID  filter.Filter[string]
	ExpiresAt filter.Filter[time.Time]
	Or        []*OAuthCodeFilter
	And       []*OAuthCodeFilter
}
type OAuthCodeOrder order.Order[OAuthCodeField]

type OAuthCode struct {
	ID            uuid.UUID
	CodeHash      string
	ClientID      string
	RedirectURI   string
	CodeChallenge string
	UserID        uuid.UUID
	WorkspaceID   uuid.UUID
	Level         OAuthAccessLevel
	ExpiresAt     time.Time
	CreatedAt     time.Time
}

// user code 'OAuthCode methods'
// end user code 'OAuthCode methods'

func (o *OAuthCode) Copy() OAuthCode {
	var result OAuthCode
	result.ID = o.ID
	result.CodeHash = o.CodeHash
	result.ClientID = o.ClientID
	result.RedirectURI = o.RedirectURI
	result.CodeChallenge = o.CodeChallenge
	result.UserID = o.UserID
	result.WorkspaceID = o.WorkspaceID
	result.Level = o.Level // enum
	result.ExpiresAt = o.ExpiresAt
	result.CreatedAt = o.CreatedAt

	return result
}
func (o *OAuthCode) Equals(to *OAuthCode) bool {
	if (o == nil) != (to == nil) {
		return false
	}
	if o == nil && to == nil {
		return true
	}
	if o.ID != to.ID {
		return false
	}
	if o.CodeHash != to.CodeHash {
		return false
	}
	if o.ClientID != to.ClientID {
		return false
	}
	if o.RedirectURI != to.RedirectURI {
		return false
	}
	if o.CodeChallenge != to.CodeChallenge {
		return false
	}
	if o.UserID != to.UserID {
		return false
	}
	if o.WorkspaceID != to.WorkspaceID {
		return false
	}
	if o.Level != to.Level {
		return false
	}
	if o.ExpiresAt != to.ExpiresAt {
		return false
	}
	if o.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type OAuthGrantField byte

const (
	OAuthGrantFieldID OAuthGrantField = iota + 1
	OAuthGrantFieldClientID
	OAuthGrantFieldUserID
	OAuthGrantFieldWorkspaceID
	OAuthGrantFieldLevel
	OAuthGrantFieldCreatedAt
)

type OAuthGrantFilter struct {
	ID          filter.Filter[uuid.UUID]
	ClientID    filter.Filter[string]
	UserID      filter.Filter[uuid.UUID]
	WorkspaceID filter.Filter[uuid.UUID]
	Or          []*OAuthGrantFilter
	And         []*OAuthGrantFilter
}
type OAuthGrantOrder order.Order[OAuthGrantField]

type OAuthGrant struct {
	ID          uuid.UUID
	ClientID    string
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Level       OAuthAccessLevel
	CreatedAt   time.Time
}

// user code 'OAuthGrant methods'
// end user code 'OAuthGrant methods'

func (o *OAuthGrant) Copy() OAuthGrant {
	var result OAuthGrant
	result.ID = o.ID
	result.ClientID = o.ClientID
	result.UserID = o.UserID
	result.WorkspaceID = o.WorkspaceID
	result.Level = o.Level // enum
	result.CreatedAt = o.CreatedAt

	return result
}
func (o *OAuthGrant) Equals(to *OAuthGrant) bool {
	if (o == nil) != (to == nil) {
		return false
	}
	if o == nil && to == nil {
		return true
	}
	if o.ID != to.ID {
		return false
	}
	if o.ClientID != to.ClientID {
		return false
	}
	if o.UserID != to.UserID {
		return false
	}
	if o.WorkspaceID != to.WorkspaceID {
		return false
	}
	if o.Level != to.Level {
		return false
	}
	if o.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type OAuthRefreshTokenField byte

const (
	OAuthRefreshTokenFieldID OAuthRefreshTokenField = iota + 1
	OAuthRefreshTokenFieldGrantID
	OAuthRefreshTokenFieldTokenHash
	OAuthRefreshTokenFieldUsedAt
	OAuthRefreshTokenFieldExpiresAt
	OAuthRefreshTokenFieldCreatedAt
)

type OAuthRefreshTokenFilter struct {
	ID        filter.Filter[uuid.UUID]
	GrantID   filter.Filter[uuid.UUID]
	TokenHash filter.Filter[string]
	ExpiresAt filter.Filter[time.Time]
	Or        []*OAuthRefreshTokenFilter
	And       []*OAuthRefreshTokenFilter
}
type OAuthRefreshTokenOrder order.Order[OAuthRefreshTokenField]

type OAuthRefreshToken struct {
	ID        uuid.UUID
	GrantID   uuid.UUID
	TokenHash string
	UsedAt    *time.Time
	ExpiresAt time.Time
	CreatedAt time.Time
}

// user code 'OAuthRefreshToken methods'
// end user code 'OAuthRefreshToken methods'

func (o *OAuthRefreshToken) Copy() OAuthRefreshToken {
	var result OAuthRefreshToken
	result.ID = o.ID
	result.GrantID = o.GrantID
	result.TokenHash = o.TokenHash
	if o.UsedAt != nil {
		var tmp time.Time
		tmp = (*o.UsedAt)
		result.UsedAt = &tmp
	}
	result.ExpiresAt = o.ExpiresAt
	result.CreatedAt = o.CreatedAt

	return result
}
func (o *OAuthRefreshToken) Equals(to *OAuthRefreshToken) bool {
	if (o == nil) != (to == nil) {
		return false
	}
	if o == nil && to == nil {
		return true
	}
	if o.ID != to.ID {
		return false
	}
	if o.GrantID != to.GrantID {
		return false
	}
	if o.TokenHash != to.TokenHash {
		return false
	}
	if (o.UsedAt == nil) != (to.UsedAt == nil) {
		return false
	}
	if o.UsedAt != nil && to.UsedAt != nil {
		if (*o.UsedAt) != (*to.UsedAt) {
			return false
		}
	}
	if o.ExpiresAt != to.ExpiresAt {
		return false
	}
	if o.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type OIDCIdentityField byte

const (
	OIDCIdentityFieldID OIDCIdentityField = iota + 1
	OIDCIdentityFieldUserID
	OIDCIdentityFieldProviderSlug
	OIDCIdentityFieldSubject
	OIDCIdentityFieldEmail
	OIDCIdentityFieldCreatedAt
	OIDCIdentityFieldLastLoginAt
)

type OIDCIdentityFilter struct {
	ID           filter.Filter[uuid.UUID]
	UserID       filter.Filter[uuid.UUID]
	ProviderSlug filter.Filter[string]
	Subject      filter.Filter[string]
	Or           []*OIDCIdentityFilter
	And          []*OIDCIdentityFilter
}
type OIDCIdentityOrder order.Order[OIDCIdentityField]

type OIDCIdentity struct {
	ID           uuid.UUID
	UserID       uuid.UUID
	ProviderSlug string
	Subject      string
	Email        string
	CreatedAt    time.Time
	LastLoginAt  time.Time
}

// user code 'OIDCIdentity methods'
// end user code 'OIDCIdentity methods'

func (o *OIDCIdentity) Copy() OIDCIdentity {
	var result OIDCIdentity
	result.ID = o.ID
	result.UserID = o.UserID
	result.ProviderSlug = o.ProviderSlug
	result.Subject = o.Subject
	result.Email = o.Email
	result.CreatedAt = o.CreatedAt
	result.LastLoginAt = o.LastLoginAt

	return result
}
func (o *OIDCIdentity) Equals(to *OIDCIdentity) bool {
	if (o == nil) != (to == nil) {
		return false
	}
	if o == nil && to == nil {
		return true
	}
	if o.ID != to.ID {
		return false
	}
	if o.UserID != to.UserID {
		return false
	}
	if o.ProviderSlug != to.ProviderSlug {
		return false
	}
	if o.Subject != to.Subject {
		return false
	}
	if o.Email != to.Email {
		return false
	}
	if o.CreatedAt != to.CreatedAt {
		return false
	}
	if o.LastLoginAt != to.LastLoginAt {
		return false
	}

	return true
}

type OIDCStateField byte

const (
	OIDCStateFieldID OIDCStateField = iota + 1
	OIDCStateFieldState
	OIDCStateFieldVerifier
	OIDCStateFieldProviderSlug
	OIDCStateFieldExpiresAt
	OIDCStateFieldCreatedAt
)

type OIDCStateFilter struct {
	ID    filter.Filter[uuid.UUID]
	State filter.Filter[string]
	Or    []*OIDCStateFilter
	And   []*OIDCStateFilter
}
type OIDCStateOrder order.Order[OIDCStateField]

type OIDCState struct {
	ID           uuid.UUID
	State        string
	Verifier     string
	ProviderSlug string
	ExpiresAt    time.Time
	CreatedAt    time.Time
}

// user code 'OIDCState methods'
// end user code 'OIDCState methods'

func (o *OIDCState) Copy() OIDCState {
	var result OIDCState
	result.ID = o.ID
	result.State = o.State
	result.Verifier = o.Verifier
	result.ProviderSlug = o.ProviderSlug
	result.ExpiresAt = o.ExpiresAt
	result.CreatedAt = o.CreatedAt

	return result
}
func (o *OIDCState) Equals(to *OIDCState) bool {
	if (o == nil) != (to == nil) {
		return false
	}
	if o == nil && to == nil {
		return true
	}
	if o.ID != to.ID {
		return false
	}
	if o.State != to.State {
		return false
	}
	if o.Verifier != to.Verifier {
		return false
	}
	if o.ProviderSlug != to.ProviderSlug {
		return false
	}
	if o.ExpiresAt != to.ExpiresAt {
		return false
	}
	if o.CreatedAt != to.CreatedAt {
		return false
	}

	return true
}

type RateLimitField byte

const (
	RateLimitFieldID RateLimitField = iota + 1
	RateLimitFieldKey
	RateLimitFieldCount
	RateLimitFieldExpiresAt
)

type RateLimitFilter struct {
	ID        filter.Filter[uuid.UUID]
	Key       filter.Filter[string]
	ExpiresAt filter.Filter[time.Time]
	Or        []*RateLimitFilter
	And       []*RateLimitFilter
}
type RateLimitOrder order.Order[RateLimitField]

type RateLimit struct {
	ID        uuid.UUID
	Key       string
	Count     int
	ExpiresAt time.Time
}

// user code 'RateLimit methods'
// end user code 'RateLimit methods'

func (r *RateLimit) Copy() RateLimit {
	var result RateLimit
	result.ID = r.ID
	result.Key = r.Key
	result.Count = r.Count
	result.ExpiresAt = r.ExpiresAt

	return result
}
func (r *RateLimit) Equals(to *RateLimit) bool {
	if (r == nil) != (to == nil) {
		return false
	}
	if r == nil && to == nil {
		return true
	}
	if r.ID != to.ID {
		return false
	}
	if r.Key != to.Key {
		return false
	}
	if r.Count != to.Count {
		return false
	}
	if r.ExpiresAt != to.ExpiresAt {
		return false
	}

	return true
}
