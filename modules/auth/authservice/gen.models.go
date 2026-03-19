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

type APIKeyField byte

const (
	APIKeyFieldID APIKeyField = iota + 1
	APIKeyFieldWorkspaceID
	APIKeyFieldUserID
	APIKeyFieldName
	APIKeyFieldKeyHash
	APIKeyFieldLastUsedAt
	APIKeyFieldExpiresAt
	APIKeyFieldCreatedAt
)

type APIKeyFilter struct {
	ID          filter.Filter[uuid.UUID]
	WorkspaceID filter.Filter[uuid.UUID]
	UserID      filter.Filter[uuid.UUID]
	KeyHash     filter.Filter[string]
	Or          []*APIKeyFilter
	And         []*APIKeyFilter
}
type APIKeyOrder order.Order[APIKeyField]

type APIKey struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	UserID      uuid.UUID
	Name        string
	KeyHash     string
	LastUsedAt  *time.Time
	ExpiresAt   *time.Time
	CreatedAt   time.Time
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
	if a.CreatedAt != to.CreatedAt {
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
