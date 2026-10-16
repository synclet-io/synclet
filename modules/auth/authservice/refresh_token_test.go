package authservice

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type refreshStore struct {
	*fakeStorage
	tokens *memRefreshTokens
}

func (s *refreshStore) RefreshTokens() RefreshTokensStorage { return s.tokens }

func (s *refreshStore) ExecuteInTransaction(ctx context.Context, cb func(context.Context, Storage) error) error {
	return cb(ctx, s)
}

type memRefreshTokens struct {
	RefreshTokensStorage

	tokens   []*RefreshToken
	firstErr error
}

func (s *memRefreshTokens) First(_ context.Context, flt *RefreshTokenFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*RefreshToken, error) {
	if s.firstErr != nil {
		return nil, s.firstErr
	}

	hash, _ := equalsValue[string](flt.TokenHash)
	for _, token := range s.tokens {
		if token.TokenHash == hash {
			cp := *token

			return &cp, nil
		}
	}

	return nil, ErrRefreshTokenNotFound
}

func (s *memRefreshTokens) Delete(_ context.Context, flt *RefreshTokenFilter) error {
	id, _ := equalsValue[uuid.UUID](flt.ID)
	for i, token := range s.tokens {
		if token.ID == id {
			s.tokens = append(s.tokens[:i], s.tokens[i+1:]...)

			return nil
		}
	}

	return nil
}

func (s *memRefreshTokens) Create(_ context.Context, token *RefreshToken) (*RefreshToken, error) {
	cp := *token
	s.tokens = append(s.tokens, &cp)

	return token, nil
}

func newRefreshStore(user *User, raw string, expiresAt time.Time) *refreshStore {
	store := &refreshStore{fakeStorage: newFakeStorage(), tokens: &memRefreshTokens{}}
	store.seedUser(user)
	store.tokens.tokens = append(store.tokens.tokens, &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: hashToken(raw),
		ExpiresAt: expiresAt,
	})

	return store
}

func TestRefreshToken(t *testing.T) {
	config := Config{JWTSecret: "secret", AccessTokenTTL: time.Minute, RefreshTokenTTL: time.Hour}
	user := &User{ID: uuid.New(), Email: "a@example.com"}

	t.Run("rotates the token and rejects its reuse", func(t *testing.T) {
		store := newRefreshStore(user, "raw-token", time.Now().Add(time.Hour))
		uc := NewRefreshTokenUC(store, config)

		pair, err := uc.Execute(context.Background(), "raw-token")
		require.NoError(t, err)
		assert.NotEmpty(t, pair.RefreshToken)
		require.Len(t, store.tokens.tokens, 1)
		assert.Equal(t, hashToken(pair.RefreshToken), store.tokens.tokens[0].TokenHash)

		_, err = uc.Execute(context.Background(), "raw-token")
		assert.ErrorIs(t, err, ErrInvalidRefreshToken)
	})

	t.Run("deletes an expired token", func(t *testing.T) {
		store := newRefreshStore(user, "raw-token", time.Now().Add(-time.Minute))

		_, err := NewRefreshTokenUC(store, config).Execute(context.Background(), "raw-token")
		require.ErrorIs(t, err, ErrRefreshTokenExpired)
		assert.Empty(t, store.tokens.tokens)
	})

	t.Run("wraps lookup failures other than not found", func(t *testing.T) {
		store := newRefreshStore(user, "raw-token", time.Now().Add(time.Hour))
		store.tokens.firstErr = errors.New("connection reset")

		_, err := NewRefreshTokenUC(store, config).Execute(context.Background(), "raw-token")
		require.Error(t, err)
		require.NotErrorIs(t, err, ErrInvalidRefreshToken)
		assert.ErrorContains(t, err, "connection reset")
	})
}
