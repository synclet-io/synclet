package connectutil

import (
	"context"
	"fmt"
	"net/http"
	"strings"

	"connectrpc.com/connect"
)

// TokenValidator validates JWT or API key tokens.
type TokenValidator interface {
	ValidateAccessToken(tokenString string) (userID, email string, err error)
	ValidateAPIKey(ctx context.Context, rawKey string) (userID, workspaceID string, err error)
}

// publicProcedures that don't require authentication.
var publicProcedures = map[string]bool{
	"/synclet.publicapi.auth.v1.AuthService/Register":                   true,
	"/synclet.publicapi.auth.v1.AuthService/Login":                      true,
	"/synclet.publicapi.auth.v1.AuthService/RefreshToken":               true,
	"/synclet.publicapi.auth.v1.AuthService/Logout":                     true,
	"/synclet.publicapi.auth.v1.AuthService/GetOIDCProviders":           true,
	"/synclet.publicapi.workspace.v1.WorkspaceService/GetInviteByToken": true,
	"/synclet.publicapi.workspace.v1.WorkspaceService/DeclineInvite":    true,
	"/synclet.publicapi.pipeline.v1.SourceService/GetSystemInfo":        true,
}

// AuthInterceptor validates JWT or API key tokens on incoming requests.
// It populates user ID, email, and workspace ID in the context.
// Role-based authorization is handled by the downstream RoleInterceptor.
type AuthInterceptor struct {
	validator TokenValidator
}

// NewAuthInterceptor creates a new auth interceptor that validates tokens.
func NewAuthInterceptor(validator TokenValidator) *AuthInterceptor {
	return &AuthInterceptor{validator: validator}
}

func (i *AuthInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		if publicProcedures[req.Spec().Procedure] {
			return next(ctx, req)
		}

		newCtx, err := i.authenticate(ctx, req.Header(), req.Header().Get("Workspace-Id"))
		if err != nil {
			return nil, err
		}

		return next(newCtx, req)
	}
}

func (i *AuthInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *AuthInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return func(ctx context.Context, conn connect.StreamingHandlerConn) error {
		if publicProcedures[conn.Spec().Procedure] {
			return next(ctx, conn)
		}

		newCtx, err := i.authenticate(ctx, conn.RequestHeader(), conn.RequestHeader().Get("Workspace-Id"))
		if err != nil {
			return err
		}

		return next(newCtx, conn)
	}
}

func (i *AuthInterceptor) authenticate(ctx context.Context, headers http.Header, workspaceHeader string) (context.Context, error) {
	authHeader := headers.Get("Authorization")

	// If no Authorization header, fall back to access token cookie.
	if authHeader == "" {
		cookieToken := ReadCookieFromHeaders(headers, accessTokenCookie)
		if cookieToken == "" {
			return nil, connect.NewError(connect.CodeUnauthenticated, nil)
		}

		return i.authenticateJWT(ctx, cookieToken, workspaceHeader)
	}

	token := strings.TrimPrefix(authHeader, "Bearer ")
	if token == authHeader {
		return nil, connect.NewError(connect.CodeUnauthenticated, nil)
	}

	// API key authentication.
	if strings.HasPrefix(token, "synclet_sk_") {
		userID, workspaceID, err := i.validator.ValidateAPIKey(ctx, token)
		if err != nil {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		uid, err := parseUUID(userID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
		}

		wsID, err := parseUUID(workspaceID)
		if err != nil {
			return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
		}

		ctx = ContextWithUserID(ctx, uid)
		ctx = ContextWithWorkspaceID(ctx, wsID)

		return ctx, nil
	}

	// JWT from Authorization header.
	return i.authenticateJWT(ctx, token, workspaceHeader)
}

func (i *AuthInterceptor) authenticateJWT(ctx context.Context, token, workspaceHeader string) (context.Context, error) {
	userID, email, err := i.validator.ValidateAccessToken(token)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	uid, err := parseUUID(userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeInternal, fmt.Errorf("internal error"))
	}

	ctx = ContextWithUserID(ctx, uid)
	ctx = ContextWithEmail(ctx, email)

	if workspaceHeader != "" {
		wsID, err := parseUUID(workspaceHeader)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		ctx = ContextWithWorkspaceID(ctx, wsID)
	}

	return ctx, nil
}
