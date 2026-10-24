package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

const authLimit = 10

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

// sharedRateLimitStore is an in-memory connectutil.RateLimitStore without
// windows, standing in for the database every replica shares.
type sharedRateLimitStore struct {
	mu     sync.Mutex
	counts map[string]int
}

func newSharedRateLimitStore() *sharedRateLimitStore {
	return &sharedRateLimitStore{counts: map[string]int{}}
}

func (s *sharedRateLimitStore) Hit(_ context.Context, key string, limit int, _ time.Duration) (bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.counts[key]++

	return s.counts[key] <= limit, nil
}

// newRateLimitedAuthClient serves the auth service behind the app's rate
// limiter on store. The test server's loopback peer is a trusted proxy, so
// each call's X-Forwarded-For header is the client IP the limiter sees.
func newRateLimitedAuthClient(t *testing.T, store connectutil.RateLimitStore) authv1connect.AuthServiceClient {
	t.Helper()

	limiter, err := newConnectRPCRateLimitInterceptor(store, &rateLimitConfig{TrustedProxies: []string{"127.0.0.1", "::1"}})
	require.NoError(t, err)

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

	for i := range authLimit {
		require.NoError(t, call(), "request %d is within the limit", i)
	}

	err := call()
	require.Error(t, err)
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err))
}

func TestRateLimitInterceptor_LimitsPasswordResetProcedures(t *testing.T) {
	client := newRateLimitedAuthClient(t, newSharedRateLimitStore())

	exhaust(t, func() error { return requestPasswordReset(client, "203.0.113.1") })
	exhaust(t, func() error { return resetPassword(client, "203.0.113.1") })
}

func TestRateLimitInterceptor_PasswordResetBucketIsPerIP(t *testing.T) {
	client := newRateLimitedAuthClient(t, newSharedRateLimitStore())

	exhaust(t, func() error { return requestPasswordReset(client, "203.0.113.1") })

	require.NoError(t, requestPasswordReset(client, "203.0.113.2"))
}

func TestRateLimitInterceptor_PasswordResetBucketIsPerProcedure(t *testing.T) {
	client := newRateLimitedAuthClient(t, newSharedRateLimitStore())

	exhaust(t, func() error { return requestPasswordReset(client, "203.0.113.1") })

	require.NoError(t, resetPassword(client, "203.0.113.1"))
	require.NoError(t, login(client, "203.0.113.1"))
}

func TestRateLimitInterceptor_ReplicasShareOneLimit(t *testing.T) {
	store := newSharedRateLimitStore()
	replicaA := newRateLimitedAuthClient(t, store)
	replicaB := newRateLimitedAuthClient(t, store)

	exhaust(t, func() error { return login(replicaA, "203.0.113.1") })

	err := login(replicaB, "203.0.113.1")
	require.Error(t, err)
	assert.Equal(t, connect.CodeResourceExhausted, connect.CodeOf(err), "the limit spent on one replica applies on the other")

	require.NoError(t, login(replicaB, "203.0.113.2"))
}
