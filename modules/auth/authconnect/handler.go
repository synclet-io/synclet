package authconnect

import (
	"context"
	"errors"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/timestamppb"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// mapError maps auth domain errors to ConnectRPC error codes.
func mapError(err error) error {
	var notFound authservice.NotFoundError
	if errors.As(err, &notFound) {
		return connect.NewError(connect.CodeNotFound, err)
	}

	var alreadyExists authservice.AlreadyExistsError
	if errors.As(err, &alreadyExists) {
		return connect.NewError(connect.CodeAlreadyExists, err)
	}

	var validation *authservice.ValidationError
	if errors.As(err, &validation) {
		return connect.NewError(connect.CodeInvalidArgument, err)
	}

	if errors.Is(err, authservice.ErrInvalidCredentials) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	if errors.Is(err, authservice.ErrInvalidCurrentPassword) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	if errors.Is(err, authservice.ErrInvalidRefreshToken) || errors.Is(err, authservice.ErrRefreshTokenExpired) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	if errors.Is(err, authservice.ErrInvalidToken) || errors.Is(err, authservice.ErrUnexpectedSigningMethod) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	if errors.Is(err, authservice.ErrInvalidAPIKey) || errors.Is(err, authservice.ErrAPIKeyExpired) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	if errors.Is(err, authservice.ErrInvalidOrExpiredState) || errors.Is(err, authservice.ErrStateProviderMismatch) || errors.Is(err, authservice.ErrMissingIDToken) {
		return connect.NewError(connect.CodeUnauthenticated, err)
	}

	if errors.Is(err, authservice.ErrEmailNotVerified) || errors.Is(err, authservice.ErrInvalidEmailFormat) {
		return connect.NewError(connect.CodePermissionDenied, err)
	}

	return err
}

// RegistrationEnabled is a named type for FX injection of the registration toggle.
type RegistrationEnabled bool

// Handler implements the AuthService ConnectRPC handler.
type Handler struct {
	authv1connect.UnimplementedAuthServiceHandler
	registerAndLogin    *authservice.RegisterAndLogin
	loginWithUserInfo   *authservice.LoginWithUserInfo
	refreshToken        *authservice.RefreshTokenUC
	logout              *authservice.Logout
	getUserByID         *authservice.GetUserByID
	updateProfile       *authservice.UpdateProfile
	changePassword      *authservice.ChangePassword
	createAPIKey        *authservice.CreateAPIKey
	revokeAPIKey        *authservice.RevokeAPIKey
	listAPIKeys         *authservice.ListAPIKeys
	getOIDCProviders    *authservice.GetOIDCProviders
	registrationEnabled RegistrationEnabled
	cookieConfig        connectutil.CookieConfig
}

// NewHandler creates a new auth handler.
func NewHandler(
	registerAndLogin *authservice.RegisterAndLogin,
	loginWithUserInfo *authservice.LoginWithUserInfo,
	refreshToken *authservice.RefreshTokenUC,
	logout *authservice.Logout,
	getUserByID *authservice.GetUserByID,
	updateProfile *authservice.UpdateProfile,
	changePassword *authservice.ChangePassword,
	createAPIKey *authservice.CreateAPIKey,
	revokeAPIKey *authservice.RevokeAPIKey,
	listAPIKeys *authservice.ListAPIKeys,
	getOIDCProviders *authservice.GetOIDCProviders,
	registrationEnabled RegistrationEnabled,
	cookieConfig connectutil.CookieConfig,
) *Handler {
	return &Handler{
		registerAndLogin:    registerAndLogin,
		loginWithUserInfo:   loginWithUserInfo,
		refreshToken:        refreshToken,
		logout:              logout,
		getUserByID:         getUserByID,
		updateProfile:       updateProfile,
		changePassword:      changePassword,
		createAPIKey:        createAPIKey,
		revokeAPIKey:        revokeAPIKey,
		listAPIKeys:         listAPIKeys,
		getOIDCProviders:    getOIDCProviders,
		registrationEnabled: registrationEnabled,
		cookieConfig:        cookieConfig,
	}
}

func (h *Handler) Register(ctx context.Context, req *connect.Request[authv1.RegisterRequest]) (*connect.Response[authv1.RegisterResponse], error) {
	if !h.registrationEnabled {
		return nil, connect.NewError(connect.CodePermissionDenied, errors.New("registration is disabled"))
	}

	if req.Msg.GetEmail() == "" || req.Msg.GetPassword() == "" || req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email, password, and name are required"))
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "email", Value: req.Msg.GetEmail(), MaxLen: connectutil.MaxNameLength},
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := h.registerAndLogin.Execute(ctx, authservice.RegisterAndLoginParams{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
		Name:     req.Msg.GetName(),
	})
	if err != nil {
		return nil, mapError(err)
	}

	resp := connect.NewResponse(&authv1.RegisterResponse{
		ExpiresAt:             timestamppb.New(result.Tokens.ExpiresAt),
		User:                  userToProto(result.User),
		RefreshTokenExpiresAt: timestamppb.New(result.Tokens.RefreshExpiresAt),
	})
	connectutil.SetAuthCookies(resp.Header(), toAuthTokens(result.Tokens), h.cookieConfig)

	return resp, nil
}

func (h *Handler) Login(ctx context.Context, req *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	if req.Msg.GetEmail() == "" || req.Msg.GetPassword() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("email and password are required"))
	}

	result, err := h.loginWithUserInfo.Execute(ctx, authservice.LoginWithUserInfoParams{
		Email:    req.Msg.GetEmail(),
		Password: req.Msg.GetPassword(),
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid credentials"))
	}

	resp := connect.NewResponse(&authv1.LoginResponse{
		ExpiresAt:             timestamppb.New(result.Tokens.ExpiresAt),
		User:                  userToProto(result.User),
		RefreshTokenExpiresAt: timestamppb.New(result.Tokens.RefreshExpiresAt),
	})
	connectutil.SetAuthCookies(resp.Header(), toAuthTokens(result.Tokens), h.cookieConfig)

	return resp, nil
}

func (h *Handler) RefreshToken(ctx context.Context, req *connect.Request[authv1.RefreshTokenRequest]) (*connect.Response[authv1.RefreshTokenResponse], error) {
	refreshTokenValue := connectutil.ReadCookieFromHeaders(req.Header(), "synclet_rt")
	if refreshTokenValue == "" {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("missing refresh token cookie"))
	}

	tokens, err := h.refreshToken.Execute(ctx, refreshTokenValue)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, errors.New("invalid refresh token"))
	}

	resp := connect.NewResponse(&authv1.RefreshTokenResponse{
		ExpiresAt:             timestamppb.New(tokens.ExpiresAt),
		RefreshTokenExpiresAt: timestamppb.New(tokens.RefreshExpiresAt),
	})
	connectutil.SetAuthCookies(resp.Header(), toAuthTokens(tokens), h.cookieConfig)

	return resp, nil
}

func (h *Handler) Logout(ctx context.Context, req *connect.Request[authv1.LogoutRequest]) (*connect.Response[authv1.LogoutResponse], error) {
	refreshTokenValue := connectutil.ReadCookieFromHeaders(req.Header(), "synclet_rt")
	if refreshTokenValue != "" {
		// Best-effort: revoke the refresh token.
		_ = h.logout.Execute(ctx, refreshTokenValue)
	}

	resp := connect.NewResponse(&authv1.LogoutResponse{})
	connectutil.ClearAuthCookies(resp.Header(), h.cookieConfig)

	return resp, nil
}

func (h *Handler) GetCurrentUser(ctx context.Context, _ *connect.Request[authv1.GetCurrentUserRequest]) (*connect.Response[authv1.GetCurrentUserResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	user, err := h.getUserByID.Execute(ctx, userID)
	if err != nil {
		return nil, connect.NewError(connect.CodeNotFound, errors.New("user not found"))
	}

	return connect.NewResponse(&authv1.GetCurrentUserResponse{
		User: userToProto(user),
	}), nil
}

func (h *Handler) UpdateProfile(ctx context.Context, req *connect.Request[authv1.UpdateProfileRequest]) (*connect.Response[authv1.UpdateProfileResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	user, err := h.updateProfile.Execute(ctx, userID, req.Msg.GetName())
	if err != nil {
		return nil, mapError(fmt.Errorf("updating profile: %w", err))
	}

	return connect.NewResponse(&authv1.UpdateProfileResponse{
		User: userToProto(user),
	}), nil
}

func (h *Handler) ChangePassword(ctx context.Context, req *connect.Request[authv1.ChangePasswordRequest]) (*connect.Response[authv1.ChangePasswordResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if req.Msg.GetCurrentPassword() == "" || req.Msg.GetNewPassword() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("current_password and new_password are required"))
	}

	if err := h.changePassword.Execute(ctx, userID, req.Msg.GetCurrentPassword(), req.Msg.GetNewPassword()); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("changing password: %w", err))
	}

	return connect.NewResponse(&authv1.ChangePasswordResponse{}), nil
}

func (h *Handler) CreateAPIKey(ctx context.Context, req *connect.Request[authv1.CreateAPIKeyRequest]) (*connect.Response[authv1.CreateAPIKeyResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	if req.Msg.GetName() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("name is required"))
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var expiresAt *time.Time

	if req.Msg.GetExpiresAt() != nil {
		t := req.Msg.GetExpiresAt().AsTime()
		expiresAt = &t
	}

	rawKey, apiKey, err := h.createAPIKey.Execute(ctx, wsID, userID, req.Msg.GetName(), expiresAt)
	if err != nil {
		return nil, mapError(fmt.Errorf("creating API key: %w", err))
	}

	return connect.NewResponse(&authv1.CreateAPIKeyResponse{
		RawKey: rawKey,
		ApiKey: apiKeyToProto(apiKey),
	}), nil
}

func (h *Handler) RevokeAPIKey(ctx context.Context, req *connect.Request[authv1.RevokeAPIKeyRequest]) (*connect.Response[authv1.RevokeAPIKeyResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if req.Msg.GetId() == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("id is required"))
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid id"))
	}

	if err := h.revokeAPIKey.Execute(ctx, id, userID); err != nil {
		return nil, mapError(fmt.Errorf("revoking API key: %w", err))
	}

	return connect.NewResponse(&authv1.RevokeAPIKeyResponse{}), nil
}

func (h *Handler) ListAPIKeys(ctx context.Context, _ *connect.Request[authv1.ListAPIKeysRequest]) (*connect.Response[authv1.ListAPIKeysResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodePermissionDenied, err)
	}

	keys, err := h.listAPIKeys.Execute(ctx, wsID)
	if err != nil {
		return nil, mapError(fmt.Errorf("listing API keys: %w", err))
	}

	protoKeys := make([]*authv1.APIKeyInfo, len(keys))
	for i, apiKey := range keys {
		protoKeys[i] = apiKeyToProto(apiKey)
	}

	return connect.NewResponse(&authv1.ListAPIKeysResponse{
		ApiKeys: protoKeys,
	}), nil
}

func apiKeyToProto(apiKey *authservice.APIKey) *authv1.APIKeyInfo {
	info := &authv1.APIKeyInfo{
		Id:          apiKey.ID.String(),
		WorkspaceId: apiKey.WorkspaceID.String(),
		Name:        apiKey.Name,
		CreatedAt:   timestamppb.New(apiKey.CreatedAt),
	}
	if apiKey.ExpiresAt != nil {
		info.ExpiresAt = timestamppb.New(*apiKey.ExpiresAt)
	}

	if apiKey.LastUsedAt != nil {
		info.LastUsedAt = timestamppb.New(*apiKey.LastUsedAt)
	}

	return info
}

func toAuthTokens(tp *authservice.TokenPair) *connectutil.AuthTokens {
	return &connectutil.AuthTokens{
		AccessToken:      tp.AccessToken,
		RefreshToken:     tp.RefreshToken,
		ExpiresAt:        tp.ExpiresAt,
		RefreshExpiresAt: tp.RefreshExpiresAt,
	}
}

func userToProto(u *authservice.User) *authv1.UserInfo {
	return &authv1.UserInfo{
		Id:        u.ID.String(),
		Email:     u.Email,
		Name:      u.Name,
		CreatedAt: timestamppb.New(u.CreatedAt),
	}
}

func (h *Handler) GetOIDCProviders(ctx context.Context, _ *connect.Request[authv1.GetOIDCProvidersRequest]) (*connect.Response[authv1.GetOIDCProvidersResponse], error) {
	if h.getOIDCProviders == nil {
		return connect.NewResponse(&authv1.GetOIDCProvidersResponse{}), nil
	}

	providers := h.getOIDCProviders.Execute()

	protoProviders := make([]*authv1.OIDCProviderInfo, len(providers))
	for i, p := range providers {
		protoProviders[i] = &authv1.OIDCProviderInfo{
			Slug:        p.Slug,
			DisplayName: p.DisplayName,
		}
	}

	return connect.NewResponse(&authv1.GetOIDCProvidersResponse{
		Providers: protoProviders,
	}), nil
}
