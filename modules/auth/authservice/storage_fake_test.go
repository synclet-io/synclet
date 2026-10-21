package authservice

import (
	"context"
	"slices"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// fakeStorage is an in-memory Storage and OAuthStorage for use-case tests.
// The users, API keys and OAuth stores are backed; other accessors panic.
type fakeStorage struct {
	Storage

	mu                 sync.Mutex
	users              []*User
	apiKeys            []*APIKey
	oauthClients       []*OAuthClient
	oauthCodes         []*OAuthCode
	oauthGrants        []*OAuthGrant
	oauthRefreshTokens []*OAuthRefreshToken
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{}
}

func (s *fakeStorage) Users() UsersStorage { return &fakeUsersStorage{parent: s} }
func (s *fakeStorage) APIKeys() APIKeysStorage {
	return &fakeTable[APIKey, APIKeyFilter]{mu: &s.mu, rows: &s.apiKeys, id: func(k *APIKey) uuid.UUID { return k.ID }, match: matchAPIKey, notFound: ErrAPIKeyNotFound}
}

func (s *fakeStorage) OAuthClients() OAuthClientsStorage {
	return &fakeTable[OAuthClient, OAuthClientFilter]{mu: &s.mu, rows: &s.oauthClients, id: func(c *OAuthClient) uuid.UUID { return c.ID }, match: matchOAuthClient, notFound: ErrOAuthClientNotFound}
}

func (s *fakeStorage) OAuthCodes() OAuthCodesStorage {
	return &fakeTable[OAuthCode, OAuthCodeFilter]{mu: &s.mu, rows: &s.oauthCodes, id: func(c *OAuthCode) uuid.UUID { return c.ID }, match: matchOAuthCode, notFound: ErrOAuthCodeNotFound}
}

func (s *fakeStorage) OAuthGrants() OAuthGrantsStorage {
	return &fakeTable[OAuthGrant, OAuthGrantFilter]{mu: &s.mu, rows: &s.oauthGrants, id: func(g *OAuthGrant) uuid.UUID { return g.ID }, match: matchOAuthGrant, notFound: ErrOAuthGrantNotFound}
}

func (s *fakeStorage) OAuthRefreshTokens() OAuthRefreshTokensStorage {
	return &fakeTable[OAuthRefreshToken, OAuthRefreshTokenFilter]{mu: &s.mu, rows: &s.oauthRefreshTokens, id: func(t *OAuthRefreshToken) uuid.UUID { return t.ID }, match: matchOAuthRefreshToken, notFound: ErrOAuthRefreshTokenNotFound}
}

func (s *fakeStorage) ExecuteInTransaction(ctx context.Context, cb func(context.Context, Storage) error) error {
	return cb(ctx, s)
}

func (s *fakeStorage) ConsumeOAuthCode(_ context.Context, codeHash string) (*OAuthCode, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, c := range s.oauthCodes {
		if c.CodeHash == codeHash {
			s.oauthCodes = append(s.oauthCodes[:i], s.oauthCodes[i+1:]...)

			return c, nil
		}
	}

	return nil, ErrOAuthCodeNotFound
}

func (s *fakeStorage) MarkOAuthRefreshTokenUsed(_ context.Context, tokenHash string, at time.Time) (*OAuthRefreshToken, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, t := range s.oauthRefreshTokens {
		if t.TokenHash == tokenHash && t.UsedAt == nil {
			t.UsedAt = &at
			cp := *t

			return &cp, nil
		}
	}

	return nil, ErrOAuthRefreshTokenNotFound
}

func (s *fakeStorage) DeleteExpiredOAuthKeys(_ context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	live := make(map[uuid.UUID]bool)
	for _, t := range s.oauthRefreshTokens {
		if t.UsedAt == nil && !t.ExpiresAt.Before(before) {
			live[t.GrantID] = true
		}
	}

	latest := make(map[uuid.UUID]*APIKey)
	for _, k := range s.apiKeys {
		if k.OAuthGrantID != nil && (latest[*k.OAuthGrantID] == nil || !k.CreatedAt.Before(latest[*k.OAuthGrantID].CreatedAt)) {
			latest[*k.OAuthGrantID] = k
		}
	}

	s.apiKeys = slices.DeleteFunc(s.apiKeys, func(k *APIKey) bool {
		if k.OAuthGrantID == nil || k.ExpiresAt == nil || !k.ExpiresAt.Before(before) {
			return false
		}

		return !live[*k.OAuthGrantID] || latest[*k.OAuthGrantID] != k
	})

	return nil
}

func (s *fakeStorage) DeleteUnusedOAuthClients(_ context.Context, neverGrantedBefore, lastGrantedBefore time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.oauthClients = slices.DeleteFunc(s.oauthClients, func(c *OAuthClient) bool {
		if slices.ContainsFunc(s.oauthGrants, func(g *OAuthGrant) bool { return g.ClientID == c.ClientID }) {
			return false
		}

		if c.LastGrantedAt == nil {
			return c.CreatedAt.Before(neverGrantedBefore)
		}

		return c.LastGrantedAt.Before(lastGrantedBefore)
	})

	return nil
}

func (s *fakeStorage) DeleteLapsedOAuthGrants(_ context.Context, before time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.oauthRefreshTokens = slices.DeleteFunc(s.oauthRefreshTokens, func(t *OAuthRefreshToken) bool { return t.ExpiresAt.Before(before) })

	lapsed := map[uuid.UUID]bool{}
	s.oauthGrants = slices.DeleteFunc(s.oauthGrants, func(g *OAuthGrant) bool {
		lapsed[g.ID] = !slices.ContainsFunc(s.oauthRefreshTokens, func(t *OAuthRefreshToken) bool { return t.GrantID == g.ID })

		return lapsed[g.ID]
	})
	s.apiKeys = slices.DeleteFunc(s.apiKeys, func(k *APIKey) bool { return k.OAuthGrantID != nil && lapsed[*k.OAuthGrantID] })

	return nil
}

func (s *fakeStorage) seedUser(u *User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = append(s.users, u)
}

// matchValue reports whether v passes flt: a nil filter, Equals, In or Less
// of a time.
func matchValue[T comparable](flt filter.Filter[T], v T) bool {
	switch f := flt.(type) {
	case nil:
		return true
	case *filter.EqualsFilter[T]:
		return f.Value == v
	case *filter.InFilter[T]:
		return slices.Contains(f.Values, v)
	case *filter.LessFilter[T]:
		lt, ok := any(v).(time.Time)
		limit, _ := any(f.Value).(time.Time)

		return ok && lt.Before(limit)
	default:
		panic("fake storage: unsupported filter")
	}
}

func matchAPIKey(k *APIKey, f *APIKeyFilter) bool {
	return f == nil || matchValue(f.ID, k.ID) && matchValue(f.WorkspaceID, k.WorkspaceID) && matchValue(f.UserID, k.UserID) &&
		matchValue(f.KeyHash, k.KeyHash) && matchPtr(f.OAuthGrantID, k.OAuthGrantID)
}

func matchOAuthClient(c *OAuthClient, f *OAuthClientFilter) bool {
	return f == nil || matchValue(f.ID, c.ID) && matchValue(f.ClientID, c.ClientID)
}

func matchOAuthCode(c *OAuthCode, f *OAuthCodeFilter) bool {
	return f == nil || matchValue(f.ID, c.ID) && matchValue(f.CodeHash, c.CodeHash) && matchValue(f.ClientID, c.ClientID) && matchValue(f.ExpiresAt, c.ExpiresAt)
}

func matchOAuthGrant(g *OAuthGrant, f *OAuthGrantFilter) bool {
	return f == nil || matchValue(f.ID, g.ID) && matchValue(f.ClientID, g.ClientID) && matchValue(f.UserID, g.UserID) && matchValue(f.WorkspaceID, g.WorkspaceID)
}

func matchOAuthRefreshToken(t *OAuthRefreshToken, f *OAuthRefreshTokenFilter) bool {
	return f == nil || matchValue(f.ID, t.ID) && matchValue(f.GrantID, t.GrantID) && matchValue(f.TokenHash, t.TokenHash) && matchValue(f.ExpiresAt, t.ExpiresAt)
}

func matchPtr[T comparable](flt filter.Filter[*T], v *T) bool {
	f, ok := flt.(*filter.EqualsFilter[*T])
	if flt == nil {
		return true
	}

	if !ok {
		panic("fake storage: unsupported filter")
	}

	return f.Value == nil && v == nil || f.Value != nil && v != nil && *f.Value == *v
}

// fakeTable is an in-memory dbutil.EntityStorage over one slice of rows.
type fakeTable[T, F any] struct {
	dbutil.EntityStorage[T, F]

	mu       *sync.Mutex
	rows     *[]*T
	id       func(*T) uuid.UUID
	match    func(*T, *F) bool
	notFound error
}

func (t *fakeTable[T, F]) Create(_ context.Context, row *T) (*T, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	cp := *row
	*t.rows = append(*t.rows, &cp)

	return row, nil
}

func (t *fakeTable[T, F]) Update(_ context.Context, row *T) (*T, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	for i, existing := range *t.rows {
		if t.id(existing) == t.id(row) {
			cp := *row
			(*t.rows)[i] = &cp

			return row, nil
		}
	}

	return nil, t.notFound
}

func (t *fakeTable[T, F]) Find(_ context.Context, flt *F, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*T, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	var out []*T

	for _, row := range *t.rows {
		if t.match(row, flt) {
			cp := *row
			out = append(out, &cp)
		}
	}

	return out, nil
}

func (t *fakeTable[T, F]) First(ctx context.Context, flt *F, opts ...optionutil.Option[dbutil.SelectOptions]) (*T, error) {
	rows, _ := t.Find(ctx, flt, opts...)
	if len(rows) == 0 {
		return nil, t.notFound
	}

	return rows[0], nil
}

func (t *fakeTable[T, F]) Count(ctx context.Context, flt *F) (int, error) {
	rows, _ := t.Find(ctx, flt)

	return len(rows), nil
}

func (t *fakeTable[T, F]) Delete(_ context.Context, flt *F) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	*t.rows = slices.DeleteFunc(*t.rows, func(row *T) bool { return t.match(row, flt) })

	return nil
}

func equalsValue[T comparable](flt filter.Filter[T]) (T, bool) {
	var zero T
	if flt == nil {
		return zero, false
	}

	if eq, ok := flt.(*filter.EqualsFilter[T]); ok {
		return eq.Value, true
	}

	return zero, false
}

type fakeUsersStorage struct {
	UsersStorage

	parent *fakeStorage
}

func (s *fakeUsersStorage) matches(u *User, flt *UserFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.ID); ok && u.ID != id {
		return false
	}

	if email, ok := equalsValue[string](flt.Email); ok && u.Email != email {
		return false
	}

	return true
}

func (s *fakeUsersStorage) First(_ context.Context, flt *UserFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*User, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, u := range s.parent.users {
		if s.matches(u, flt) {
			cp := *u

			return &cp, nil
		}
	}

	return nil, ErrUserNotFound
}

func (s *fakeUsersStorage) Update(_ context.Context, u *User) (*User, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for i, existing := range s.parent.users {
		if existing.ID == u.ID {
			cp := *u
			s.parent.users[i] = &cp

			return u, nil
		}
	}

	return nil, ErrUserNotFound
}

// capturingAuditRecorder records every event passed to Record.
type capturingAuditRecorder struct {
	events []AuditEvent
}

func (r *capturingAuditRecorder) Record(_ context.Context, event AuditEvent) {
	r.events = append(r.events, event)
}
