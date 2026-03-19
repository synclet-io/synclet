package connectutil

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockTokenValidator is a test double for TokenValidator.
type mockTokenValidator struct {
	userID string
	email  string
	err    error

	apiKeyUserID      string
	apiKeyWorkspaceID string
	apiKeyErr         error
}

func (m *mockTokenValidator) ValidateAccessToken(_ string) (userID, email string, err error) {
	return m.userID, m.email, m.err
}

func (m *mockTokenValidator) ValidateAPIKey(_ context.Context, _ string) (userID, workspaceID string, err error) {
	return m.apiKeyUserID, m.apiKeyWorkspaceID, m.apiKeyErr
}

func makeHeaders(authHeader string) http.Header {
	h := http.Header{}
	if authHeader != "" {
		h.Set("Authorization", authHeader)
	}
	return h
}

func TestAuthInterceptor_authenticate(t *testing.T) {
	userID := uuid.New()
	wsID := uuid.New()

	t.Run("JWT with valid workspace header sets workspace in context", func(t *testing.T) {
		validator := &mockTokenValidator{userID: userID.String(), email: "test@example.com"}
		interceptor := NewAuthInterceptor(validator)

		ctx, err := interceptor.authenticate(context.Background(), makeHeaders("Bearer valid-token"), wsID.String())
		require.NoError(t, err)

		gotUserID, err := UserIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, gotUserID)

		gotWsID, err := WorkspaceIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, wsID, gotWsID)
	})

	t.Run("JWT without workspace header skips workspace context", func(t *testing.T) {
		validator := &mockTokenValidator{userID: userID.String(), email: "test@example.com"}
		interceptor := NewAuthInterceptor(validator)

		ctx, err := interceptor.authenticate(context.Background(), makeHeaders("Bearer valid-token"), "")
		require.NoError(t, err)

		gotUserID, err := UserIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, gotUserID)

		// No workspace should be in context.
		_, err = WorkspaceIDFromContext(ctx)
		require.Error(t, err)
	})

	t.Run("API key auth sets user and workspace in context", func(t *testing.T) {
		apiKeyWsID := uuid.New()
		validator := &mockTokenValidator{
			apiKeyUserID:      userID.String(),
			apiKeyWorkspaceID: apiKeyWsID.String(),
		}
		interceptor := NewAuthInterceptor(validator)

		ctx, err := interceptor.authenticate(context.Background(), makeHeaders("Bearer synclet_sk_testkey"), "")
		require.NoError(t, err)

		gotUserID, err := UserIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, gotUserID)

		gotWsID, err := WorkspaceIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, apiKeyWsID, gotWsID)
	})

	t.Run("JWT with invalid workspace header UUID returns CodeInvalidArgument", func(t *testing.T) {
		validator := &mockTokenValidator{userID: userID.String(), email: "test@example.com"}
		interceptor := NewAuthInterceptor(validator)

		_, err := interceptor.authenticate(context.Background(), makeHeaders("Bearer valid-token"), "not-a-uuid")
		require.Error(t, err)
		var connectErr *connect.Error
		require.True(t, errors.As(err, &connectErr))
		assert.Equal(t, connect.CodeInvalidArgument, connectErr.Code())
	})

	t.Run("missing auth header returns CodeUnauthenticated", func(t *testing.T) {
		validator := &mockTokenValidator{}
		interceptor := NewAuthInterceptor(validator)

		_, err := interceptor.authenticate(context.Background(), makeHeaders(""), "")
		require.Error(t, err)
		var connectErr *connect.Error
		require.True(t, errors.As(err, &connectErr))
		assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	})

	t.Run("invalid token returns CodeUnauthenticated", func(t *testing.T) {
		validator := &mockTokenValidator{err: errors.New("invalid token")}
		interceptor := NewAuthInterceptor(validator)

		_, err := interceptor.authenticate(context.Background(), makeHeaders("Bearer bad-token"), "")
		require.Error(t, err)
		var connectErr *connect.Error
		require.True(t, errors.As(err, &connectErr))
		assert.Equal(t, connect.CodeUnauthenticated, connectErr.Code())
	})

	t.Run("cookie fallback when no Authorization header", func(t *testing.T) {
		validator := &mockTokenValidator{userID: userID.String(), email: "test@example.com"}
		interceptor := NewAuthInterceptor(validator)

		h := http.Header{}
		h.Set("Cookie", "synclet_at=cookie-jwt-token")

		ctx, err := interceptor.authenticate(context.Background(), h, "")
		require.NoError(t, err)

		gotUserID, err := UserIDFromContext(ctx)
		require.NoError(t, err)
		assert.Equal(t, userID, gotUserID)
	})
}
