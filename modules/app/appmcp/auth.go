package appmcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/pkg/connectutil"
)

// Principal is the user and workspace a bearer token acts for.
type Principal struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
}

// BearerAuthenticator resolves one form of bearer token. claimed is false when
// the token is not of that form, so the next authenticator can try it. A token
// it claims but rejects returns an error wrapping
// connectutil.ErrInvalidCredential.
type BearerAuthenticator interface {
	Authenticate(ctx context.Context, token string) (principal Principal, claimed bool, err error)
}

// APIKeyValidator validates a raw workspace API key.
type APIKeyValidator interface {
	ValidateAPIKey(ctx context.Context, rawKey string) (userID, workspaceID string, err error)
}

const apiKeyPrefix = "synclet_sk_" //nolint:gosec // key prefix, not a credential

// APIKeyAuthenticator authenticates workspace API keys.
type APIKeyAuthenticator struct {
	validator APIKeyValidator
}

func NewAPIKeyAuthenticator(validator APIKeyValidator) *APIKeyAuthenticator {
	return &APIKeyAuthenticator{validator: validator}
}

func (a *APIKeyAuthenticator) Authenticate(ctx context.Context, token string) (Principal, bool, error) {
	if !strings.HasPrefix(token, apiKeyPrefix) {
		return Principal{}, false, nil
	}

	userID, workspaceID, err := a.validator.ValidateAPIKey(ctx, token)
	if err != nil {
		return Principal{}, true, err
	}

	uid, err := uuid.Parse(userID)
	if err != nil {
		return Principal{}, true, fmt.Errorf("parsing API key user id: %w", err)
	}

	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return Principal{}, true, fmt.Errorf("parsing API key workspace id: %w", err)
	}

	return Principal{UserID: uid, WorkspaceID: wsID}, true, nil
}

// requireBearer admits a request whose bearer token one of authenticators
// accepts, with the principal's user and workspace in its context, and
// answers 401 to any other.
func requireBearer(authenticators []BearerAuthenticator, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token, ok := strings.CutPrefix(r.Header.Get("Authorization"), "Bearer ")
		if !ok || token == "" {
			unauthorized(w)

			return
		}

		for _, authenticator := range authenticators {
			principal, claimed, err := authenticator.Authenticate(r.Context(), token)
			if !claimed {
				continue
			}

			if errors.Is(err, connectutil.ErrInvalidCredential) {
				unauthorized(w)

				return
			}

			if err != nil {
				slog.Default().ErrorContext(r.Context(), "authenticating MCP request", "error", err)
				http.Error(w, "internal error", http.StatusInternalServerError)

				return
			}

			ctx := connectutil.ContextWithUserID(r.Context(), principal.UserID)
			ctx = connectutil.ContextWithWorkspaceID(ctx, principal.WorkspaceID)
			next.ServeHTTP(w, r.WithContext(ctx))

			return
		}

		unauthorized(w)
	})
}

func unauthorized(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", "Bearer")
	http.Error(w, "unauthorized", http.StatusUnauthorized)
}
