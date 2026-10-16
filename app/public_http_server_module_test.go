package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx/fxtest"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
)

const authLimiterBurst = 10

type limitedAuthHandler struct {
	authv1connect.UnimplementedAuthServiceHandler
}

func (limitedAuthHandler) RequestPasswordReset(context.Context, *connect.Request[authv1.RequestPasswordResetRequest]) (*connect.Response[authv1.RequestPasswordResetResponse], error) {
	return connect.NewResponse(&authv1.RequestPasswordResetResponse{}), nil
}

func (limitedAuthHandler) ResetPassword(context.Context, *connect.Request[authv1.ResetPasswordRequest]) (*connect.Response[authv1.ResetPasswordResponse], error) {
	return connect.NewResponse(&authv1.ResetPasswordResponse{}), nil
}

func (limitedAuthHandler) Login(context.Context, *connect.Request[authv1.LoginRequest]) (*connect.Response[authv1.LoginResponse], error) {
	return connect.NewResponse(&authv1.LoginResponse{}), nil
}

// newRateLimitedAuthClient serves the auth service behind the app's rate
// limiter. The test server's loopback peer is a trusted proxy, so each call's
// X-Forwarded-For header is the client IP the limiter sees.
func newRateLimitedAuthClient(t *testing.T) authv1connect.AuthServiceClient {
	t.Helper()

	lc := fxtest.NewLifecycle(t)

	limiter, err := newConnectRPCRateLimitInterceptor(lc, &rateLimitConfig{TrustedProxies: []string{"127.0.0.1", "::1"}})
	require.NoError(t, err)

	lc.RequireStart()
	t.Cleanup(lc.RequireStop)

	path, handler := authv1connect.NewAuthServiceHandler(limitedAuthHandler{}, connect.WithInterceptors(limiter.Value))

	mux := http.NewServeMux()
	mux.Handle(path, handler)

	server := httptest.NewServer(mux)
	t.Cleanup(server.Close)

	return authv1connect.NewAuthServiceClient(server.Client(), server.URL)
}

func fromIP[T any](msg *T, ip string) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("X-Forwarded-For", ip)

	return req
}

func requestPasswordReset(client authv1connect.AuthServiceClient, ip string) error {
	_, err := client.RequestPasswordReset(context.Background(), fromIP(&authv1.RequestPasswordResetRequest{Email: "jane@example.com"}, ip))

	return err
}

func resetPassword(client authv1connect.AuthServiceClient, ip string) error {
	_, err := client.ResetPassword(context.Background(), fromIP(&authv1.ResetPasswordRequest{Token: "token", NewPassword: "new-password-456"}, ip))

	return err
}

func login(client authv1connect.AuthServiceClient, ip string) error {
	_, err := client.Login(context.Background(), fromIP(&authv1.LoginRequest{Email: "jane@example.com", Password: "password-123"}, ip))

	return err
}

func exhaust(t *testing.T, call func() error) {
	t.Helper()

	for i := range authLimiterBurst {
		require.NoError(t, call(), "request %d is within the burst", i)
	}

	err := call()
	require.Error(t, err)
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

func TestRateLimitInterceptor_LimitsPasswordResetProcedures(t *testing.T) {
	client := newRateLimitedAuthClient(t)

	exhaust(t, func() error { return requestPasswordReset(client, "203.0.113.1") })
	exhaust(t, func() error { return resetPassword(client, "203.0.113.1") })
}

func TestRateLimitInterceptor_PasswordResetBucketIsPerIP(t *testing.T) {
	client := newRateLimitedAuthClient(t)

	exhaust(t, func() error { return requestPasswordReset(client, "203.0.113.1") })

	require.NoError(t, requestPasswordReset(client, "203.0.113.2"))
}

func TestRateLimitInterceptor_PasswordResetBucketIsPerProcedure(t *testing.T) {
	client := newRateLimitedAuthClient(t)

	exhaust(t, func() error { return requestPasswordReset(client, "203.0.113.1") })

	require.NoError(t, resetPassword(client, "203.0.113.1"))
	require.NoError(t, login(client, "203.0.113.1"))
}
