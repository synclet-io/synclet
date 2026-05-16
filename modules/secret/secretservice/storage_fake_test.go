package secretservice

import (
	"context"
	"sync"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	idempotency "github.com/saturn4er/boilerplate-go/lib/idempotency"
)

// fakeStorage is the shared in-memory Storage implementation for secret use
// case tests.
type fakeStorage struct {
	mu       sync.Mutex
	secrets  []*Secret
	secStore *fakeSecretsStorage
}

func newFakeStorage() *fakeStorage {
	s := &fakeStorage{}
	s.secStore = &fakeSecretsStorage{parent: s}

	return s
}

func (s *fakeStorage) Secrets() SecretsStorage { return s.secStore }
func (s *fakeStorage) IdempotencyKeys() idempotency.Storage {
	panic("IdempotencyKeys not implemented")
}

func (s *fakeStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx Storage) error) error {
	return cb(ctx, s)
}

func (s *fakeStorage) WithAdvisoryLock(_ context.Context, _ string, _ int64) error {
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

type fakeSecretsStorage struct {
	parent *fakeStorage
}

func (s *fakeSecretsStorage) matches(sec *Secret, flt *SecretFilter) bool {
	if flt == nil {
		return true
	}

	if id, ok := equalsValue[uuid.UUID](flt.ID); ok && sec.ID != id {
		return false
	}

	if ownerType, ok := equalsValue[string](flt.OwnerType); ok && sec.OwnerType != ownerType {
		return false
	}

	if ownerID, ok := equalsValue[uuid.UUID](flt.OwnerID); ok && sec.OwnerID != ownerID {
		return false
	}

	return true
}

func (s *fakeSecretsStorage) Create(_ context.Context, sec *Secret) (*Secret, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()
	s.parent.secrets = append(s.parent.secrets, sec)

	return sec, nil
}

func (s *fakeSecretsStorage) BatchCreate(_ context.Context, _ []*Secret) ([]*Secret, error) {
	panic("BatchCreate not implemented")
}

func (s *fakeSecretsStorage) Count(_ context.Context, flt *SecretFilter) (int, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	count := 0
	for _, sec := range s.parent.secrets {
		if s.matches(sec, flt) {
			count++
		}
	}

	return count, nil
}

func (s *fakeSecretsStorage) Update(_ context.Context, sec *Secret) (*Secret, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for i, existing := range s.parent.secrets {
		if existing.ID == sec.ID {
			s.parent.secrets[i] = sec
			return sec, nil
		}
	}

	return nil, ErrSecretNotFound
}

func (s *fakeSecretsStorage) Save(ctx context.Context, sec *Secret) (*Secret, error) {
	return s.Update(ctx, sec)
}

func (s *fakeSecretsStorage) First(_ context.Context, flt *SecretFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*Secret, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	for _, sec := range s.parent.secrets {
		if s.matches(sec, flt) {
			return sec, nil
		}
	}

	return nil, ErrSecretNotFound
}

func (s *fakeSecretsStorage) FirstOrCreate(ctx context.Context, _ *SecretFilter, model *Secret, _ ...optionutil.Option[dbutil.SelectOptions]) (*Secret, error) {
	return s.Create(ctx, model)
}

func (s *fakeSecretsStorage) Find(_ context.Context, flt *SecretFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*Secret, error) {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	out := []*Secret{}
	for _, sec := range s.parent.secrets {
		if s.matches(sec, flt) {
			out = append(out, sec)
		}
	}

	return out, nil
}

func (s *fakeSecretsStorage) Delete(_ context.Context, flt *SecretFilter) error {
	s.parent.mu.Lock()
	defer s.parent.mu.Unlock()

	kept := s.parent.secrets[:0]
	for _, sec := range s.parent.secrets {
		if !s.matches(sec, flt) {
			kept = append(kept, sec)
		}
	}
	s.parent.secrets = kept

	return nil
}

func (s *fakeSecretsStorage) WithAdvisoryLock(_ context.Context, _ int64) error {
	return nil
}
