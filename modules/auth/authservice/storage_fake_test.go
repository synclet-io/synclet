package authservice

import (
	"context"
	"sync"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// fakeStorage is an in-memory Storage for use-case tests. Only the users and
// API keys stores are backed; other accessors panic.
type fakeStorage struct {
	Storage

	mu      sync.Mutex
	users   []*User
	apiKeys []*APIKey
}

func newFakeStorage() *fakeStorage {
	return &fakeStorage{}
}

func (s *fakeStorage) Users() UsersStorage     { return &fakeUsersStorage{parent: s} }
func (s *fakeStorage) APIKeys() APIKeysStorage { return &fakeAPIKeysStorage{parent: s} }

func (s *fakeStorage) seedUser(u *User) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = append(s.users, u)
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

type fakeAPIKeysStorage struct {
	APIKeysStorage

	parent *fakeStorage
}

func (s *fakeAPIKeysStorage) Create(_ context.Context, k *APIKey) (*APIKey, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	cp := *k
	s.parent.apiKeys = append(s.parent.apiKeys, &cp)

	return k, nil
}

// capturingAuditRecorder records every event passed to Record.
type capturingAuditRecorder struct {
	events []AuditEvent
}

func (r *capturingAuditRecorder) Record(_ context.Context, event AuditEvent) {
	r.events = append(r.events, event)
}
