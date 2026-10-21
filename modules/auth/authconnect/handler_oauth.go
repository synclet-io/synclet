package authconnect

import (
	"context"
	"fmt"
	"net/url"

	"connectrpc.com/connect"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

func (h *Handler) GetOAuthClient(ctx context.Context, req *connect.Request[authv1.GetOAuthClientRequest]) (*connect.Response[authv1.GetOAuthClientResponse], error) {
	client, err := h.getOAuthClient.Execute(ctx, req.Msg.GetClientId(), req.Msg.GetRedirectUri())
	if err != nil {
		return nil, mapError(fmt.Errorf("getting OAuth client: %w", err))
	}

	return connect.NewResponse(&authv1.GetOAuthClientResponse{
		ClientId:   client.ClientID,
		ClientName: client.Name,
	}), nil
}

func (h *Handler) AuthorizeOAuth(ctx context.Context, req *connect.Request[authv1.AuthorizeOAuthRequest]) (*connect.Response[authv1.AuthorizeOAuthResponse], error) {
	userID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	wsID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	code, err := h.issueOAuthCode.Execute(ctx, authservice.IssueOAuthCodeParams{
		ClientID:            req.Msg.GetClientId(),
		RedirectURI:         req.Msg.GetRedirectUri(),
		CodeChallenge:       req.Msg.GetCodeChallenge(),
		CodeChallengeMethod: req.Msg.GetCodeChallengeMethod(),
		UserID:              userID,
		WorkspaceID:         wsID,
		Level:               oauthLevelFromProto(req.Msg.GetLevel()),
	})
	if err != nil {
		return nil, mapError(fmt.Errorf("issuing OAuth code: %w", err))
	}

	redirectURL, err := url.Parse(req.Msg.GetRedirectUri())
	if err != nil {
		return nil, fmt.Errorf("parsing registered redirect URI: %w", err)
	}

	query := redirectURL.Query()
	query.Set("code", code)

	if state := req.Msg.GetState(); state != "" {
		query.Set("state", state)
	}

	redirectURL.RawQuery = query.Encode()

	return connect.NewResponse(&authv1.AuthorizeOAuthResponse{RedirectUrl: redirectURL.String()}), nil
}
