package connectutil

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
)

type passwordResetHandler struct {
	authv1connect.UnimplementedAuthServiceHandler
}

func (passwordResetHandler) RequestPasswordReset(context.Context, *connect.Request[authv1.RequestPasswordResetRequest]) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.RequestPasswordResetResponse{}), nil
}

func (passwordResetHandler) ResetPassword(context.Context, *connect.Request[authv1.ResetPasswordRequest]) (*connect.Response[authv1.ResetPasswordResponse], error) {
	return connect.NewResponse(&authv1.ResetPasswordResponse{}), nil
}

func (passwordResetHandler) GetCurrentUser(context.Context, *connect.Request[authv1.GetCurrentUserRequest]) (*connect.Response[authv1.GetCurrentUserResponse], error) {
	return connect.NewResponse(&authv1.GetCurrentUserResponse{}), nil
}

func TestAuthInterceptor_PasswordResetProceduresArePublic(t *testing.T) {
	path, handler := authv1connect.NewAuthServiceHandler(passwordResetHandler{},
		connect.WithInterceptors(NewAuthInterceptor(&mockTokenValidator{})))

	mux := http.NewServeMux()
	mux.Handle(path, handler)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	client := authv1connect.NewAuthServiceClient(server.Client(), server.URL)
	ctx := context.Background()

	_, err := client.RequestPasswordReset(ctx, connect.NewRequest(&authv1.RequestPasswordResetRequest{Email: "jane@example.com"}))
	require.NoError(t, err)

	_, err = client.ResetPassword(ctx, connect.NewRequest(&authv1.ResetPasswordRequest{Token: "token", NewPassword: "new-password-456"}))
	require.NoError(t, err)

	_, err = client.GetCurrentUser(ctx, connect.NewRequest(&authv1.GetCurrentUserRequest{}))
	require.Error(t, err)
	assert.Equal(t, connect.CodeUnauthenticated, connect.CodeOf(err))
}
