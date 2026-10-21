package appmcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/modelcontextprotocol/go-sdk/auth"

	"github.com/synclet-io/synclet/pkg/connectutil"
)

// Principal is the user and workspace a bearer token acts for, with the
// token's expiry and access level (empty for no limit).
type Principal struct {
	UserID      uuid.UUID
	WorkspaceID uuid.UUID
	Expiration  time.Time
	Level       string
}

// BearerAuthenticator resolves one form of bearer token. claimed is false when
// the token is not of that form, so the next authenticator can try it. A token
// it claims but rejects returns an error wrapping
// connectutil.ErrInvalidCredential.
type BearerAuthenticator interface {
	Authenticate(ctx context.Context, token string) (principal Principal, claimed bool, err error)
}

// Access levels of a key issued to an OAuth client. A plain API key has none.
const (
	LevelRead    = "read"
	LevelActions = "actions"
	LevelWrite   = "write"
)

// MCPKey is what a workspace API key grants on /mcp. Level is empty for a
// plain API key.
type MCPKey struct {
	UserID      string
	WorkspaceID string
	Level       string
	ExpiresAt   *time.Time
}

// APIKeyValidator validates a raw workspace API key, plain or OAuth-issued.
type APIKeyValidator interface {
	ValidateMCPKey(ctx context.Context, rawKey string) (MCPKey, error)
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

	key, err := a.validator.ValidateMCPKey(ctx, token)
	if err != nil {
		return Principal{}, true, err
	}

	userID, workspaceID := key.UserID, key.WorkspaceID

	uid, err := uuid.Parse(userID)
	if err != nil {
		return Principal{}, true, fmt.Errorf("parsing API key user id: %w", err)
	}

	wsID, err := uuid.Parse(workspaceID)
	if err != nil {
		return Principal{}, true, fmt.Errorf("parsing API key workspace id: %w", err)
	}

	principal := Principal{UserID: uid, WorkspaceID: wsID, Level: key.Level}
	if key.ExpiresAt != nil {
		principal.Expiration = *key.ExpiresAt
	}

	return principal, true, nil
}

const (
	extraWorkspaceID = "workspace_id"
	extraLevel       = "level"
)

// tokenVerifier resolves a bearer token with the first of authenticators that
// claims it. A token none claims, or one rejected as an invalid credential, is
// auth.ErrInvalidToken alone, so the response does not say why; the reason is
// logged.
func tokenVerifier(authenticators []BearerAuthenticator) auth.TokenVerifier {
	return func(ctx context.Context, token string, _ *http.Request) (*auth.TokenInfo, error) {
		for _, authenticator := range authenticators {
			principal, claimed, err := authenticator.Authenticate(ctx, token)
			if !claimed {
				continue
			}

			if errors.Is(err, connectutil.ErrInvalidCredential) {
				slog.Default().InfoContext(ctx, "rejected MCP bearer token", "error", err)

				return nil, auth.ErrInvalidToken
			}

			if err != nil {
				slog.Default().ErrorContext(ctx, "authenticating MCP request", "error", err)

				return nil, errors.New("internal error")
			}

			return &auth.TokenInfo{
				UserID:     principal.UserID.String(),
				Expiration: principal.Expiration,
				Extra: map[string]any{
					extraWorkspaceID: principal.WorkspaceID,
					extraLevel:       principal.Level,
				},
			}, nil
		}

		return nil, auth.ErrInvalidToken
	}
}

// requireBearer admits a request whose bearer token one of authenticators
// accepts, with the token's user and workspace in its context, and answers
// 401 with the protected resource metadata URL to any other.
func requireBearer(resourceMetadataURL string, authenticators []BearerAuthenticator, next http.Handler) http.Handler {
	withPrincipal := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := auth.TokenInfoFromContext(r.Context())

		userID, err := uuid.Parse(info.UserID)
		if err != nil {
			http.Error(w, "internal error", http.StatusInternalServerError)

			return
		}

		workspaceID, _ := info.Extra[extraWorkspaceID].(uuid.UUID)
		ctx := connectutil.ContextWithUserID(r.Context(), userID)
		ctx = connectutil.ContextWithWorkspaceID(ctx, workspaceID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})

	return auth.RequireBearerToken(tokenVerifier(authenticators), &auth.RequireBearerTokenOptions{
		ResourceMetadataURL:    resourceMetadataURL,
		AllowMissingExpiration: true,
	})(withPrincipal)
}

// levelOf returns the access level of the request's token, empty for no limit.
func levelOf(ctx context.Context) string {
	info := auth.TokenInfoFromContext(ctx)
	if info == nil {
		return ""
	}

	level, _ := info.Extra[extraLevel].(string)

	return level
}

// allows reports whether a token of level may use a tool of kind.
func allows(level string, kind toolKind) bool {
	switch level {
	case "", LevelWrite:
		return true
	case LevelActions:
		return kind == kindRead || kind == kindAction
	default:
		return kind == kindRead
	}
}
