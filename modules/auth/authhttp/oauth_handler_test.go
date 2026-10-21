package authhttp

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

type registerFunc func(ctx context.Context, name string, redirectURIs []string) (*authservice.OAuthClient, error)

func (f registerFunc) Execute(ctx context.Context, name string, redirectURIs []string) (*authservice.OAuthClient, error) {
	return f(ctx, name, redirectURIs)
}

type exchangeFunc func(ctx context.Context, code, verifier, clientID, redirectURI string) (*authservice.OAuthTokens, error)

func (f exchangeFunc) Execute(ctx context.Context, code, verifier, clientID, redirectURI string) (*authservice.OAuthTokens, error) {
	return f(ctx, code, verifier, clientID, redirectURI)
}

type refreshFunc func(ctx context.Context, refreshToken, clientID string) (*authservice.OAuthTokens, error)

func (f refreshFunc) Execute(ctx context.Context, refreshToken, clientID string) (*authservice.OAuthTokens, error) {
	return f(ctx, refreshToken, clientID)
}

type allowAll struct{}

func (allowAll) AllowRequest(string, *http.Request) bool { return true }

// countingLimiter allows the first n requests to each key.
type countingLimiter struct {
	n    int
	seen map[string]int
}

func (l *countingLimiter) AllowRequest(key string, _ *http.Request) bool {
	l.seen[key]++

	return l.seen[key] <= l.n
}

var testURLs = OAuthURLs{Public: "https://mcp.example.com", UI: "https://synclet.example.com"}

func newOAuthServer(t *testing.T, urls OAuthURLs, register OAuthClientRegistrar, exchange OAuthCodeExchanger, refresh OAuthTokenRefresher) *httptest.Server {
	t.Helper()

	router := mux.NewRouter()
	NewOAuthHTTPHandler(urls, allowAll{}, register, exchange, refresh).Register(router)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return srv
}

type response struct {
	status int
	header http.Header
	body   map[string]any
}

func do(t *testing.T, method, target, contentType string, payload io.Reader) response {
	t.Helper()

	req, err := http.NewRequestWithContext(t.Context(), method, target, payload)
	require.NoError(t, err)

	if contentType != "" {
		req.Header.Set("Content-Type", contentType)
	}

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err)

	defer func() { require.NoError(t, resp.Body.Close()) }()

	out := response{status: resp.StatusCode, header: resp.Header}
	if resp.Header.Get("Content-Type") == "application/json" {
		require.NoError(t, json.NewDecoder(resp.Body).Decode(&out.body))
	}

	return out
}

func getJSON(t *testing.T, target string) response {
	t.Helper()

	return do(t, http.MethodGet, target, "", nil)
}

func postJSON(t *testing.T, target, body string) response {
	t.Helper()

	return do(t, http.MethodPost, target, "application/json", strings.NewReader(body))
}

func postForm(t *testing.T, target string, form url.Values) response {
	t.Helper()

	return do(t, http.MethodPost, target, "application/x-www-form-urlencoded", strings.NewReader(form.Encode()))
}

func TestOAuthMetadata(t *testing.T) {
	srv := newOAuthServer(t, testURLs, nil, nil, nil)

	for _, path := range []string{"/.well-known/oauth-protected-resource", "/.well-known/oauth-protected-resource/mcp"} {
		res := getJSON(t, srv.URL+path)
		require.Equal(t, http.StatusOK, res.status, path)
		assert.Equal(t, "https://mcp.example.com/mcp", res.body["resource"])
		assert.Equal(t, []any{"https://mcp.example.com"}, res.body["authorization_servers"])
	}

	res := getJSON(t, srv.URL+"/.well-known/oauth-authorization-server")
	require.Equal(t, http.StatusOK, res.status)
	assert.Equal(t, "https://mcp.example.com", res.body["issuer"])
	assert.Equal(t, "https://synclet.example.com/oauth/authorize", res.body["authorization_endpoint"])
	assert.Equal(t, "https://mcp.example.com/oauth/token", res.body["token_endpoint"])
	assert.Equal(t, "https://mcp.example.com/oauth/register", res.body["registration_endpoint"])
	assert.Equal(t, []any{"S256"}, res.body["code_challenge_methods_supported"])
	assert.Equal(t, []any{"code"}, res.body["response_types_supported"])
	assert.Equal(t, []any{"authorization_code", "refresh_token"}, res.body["grant_types_supported"])
	assert.Equal(t, []any{"none"}, res.body["token_endpoint_auth_methods_supported"])
	assert.NotContains(t, res.body, "jwks_uri", "an empty jwks_uri fails clients that validate it as a URL")

	noUI := newOAuthServer(t, OAuthURLs{Public: testURLs.Public}, nil, nil, nil)
	assert.Equal(t, http.StatusNotFound, getJSON(t, noUI.URL+"/.well-known/oauth-authorization-server").status)
}

func TestOAuthRegister(t *testing.T) {
	var gotName string

	var gotURIs []string

	register := registerFunc(func(_ context.Context, name string, uris []string) (*authservice.OAuthClient, error) {
		gotName, gotURIs = name, uris
		if strings.HasPrefix(uris[0], "http://example.com") {
			return nil, &authservice.ValidationError{Message: "redirect URI must be https"}
		}

		return &authservice.OAuthClient{ClientID: "client-1", Name: name, RedirectURIs: uris, CreatedAt: time.Now()}, nil
	})
	srv := newOAuthServer(t, testURLs, register, nil, nil)

	var res response

	res = postJSON(t, srv.URL+"/oauth/register", `{"client_name":"Claude","redirect_uris":["https://claude.ai/api/mcp/auth_callback"],"token_endpoint_auth_method":"none"}`)
	require.Equal(t, http.StatusCreated, res.status)
	assert.Equal(t, "*", res.header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, "Claude", gotName)
	assert.Equal(t, []string{"https://claude.ai/api/mcp/auth_callback"}, gotURIs)
	assert.Equal(t, "client-1", res.body["client_id"])
	assert.Equal(t, "none", res.body["token_endpoint_auth_method"])
	assert.Equal(t, []any{"https://claude.ai/api/mcp/auth_callback"}, res.body["redirect_uris"])
	assert.NotContains(t, res.body, "client_secret")

	res = postJSON(t, srv.URL+"/oauth/register", `{"client_name":"Bad","redirect_uris":["http://example.com/cb"]}`)
	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, "invalid_redirect_uri", res.body["error"])

	res = postJSON(t, srv.URL+"/oauth/register", `not json`)
	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, "invalid_client_metadata", res.body["error"])
}

func TestOAuthToken(t *testing.T) {
	tokens := &authservice.OAuthTokens{AccessToken: "access-2", RefreshToken: "refresh-2", ExpiresIn: time.Hour}

	var exchangeArgs, refreshArgs []string

	exchange := exchangeFunc(func(_ context.Context, code, verifier, clientID, redirectURI string) (*authservice.OAuthTokens, error) {
		exchangeArgs = []string{code, verifier, clientID, redirectURI}
		if code == "spent" {
			return nil, authservice.ErrInvalidGrant
		}

		return tokens, nil
	})
	refresh := refreshFunc(func(_ context.Context, refreshToken, clientID string) (*authservice.OAuthTokens, error) {
		refreshArgs = []string{refreshToken, clientID}
		if refreshToken == "reused" {
			return nil, authservice.ErrInvalidGrant
		}

		return tokens, nil
	})
	srv := newOAuthServer(t, testURLs, nil, exchange, refresh)

	var res response

	res = postForm(t, srv.URL+"/oauth/token", url.Values{
		"grant_type":    {"authorization_code"},
		"code":          {"code-1"},
		"code_verifier": {"verifier-1"},
		"client_id":     {"client-1"},
		"redirect_uri":  {"https://claude.ai/cb"},
	})
	require.Equal(t, http.StatusOK, res.status)
	assert.Equal(t, "no-store", res.header.Get("Cache-Control"))
	assert.Equal(t, "*", res.header.Get("Access-Control-Allow-Origin"))
	assert.Equal(t, []string{"code-1", "verifier-1", "client-1", "https://claude.ai/cb"}, exchangeArgs)
	assert.Equal(t, map[string]any{"access_token": "access-2", "token_type": "Bearer", "expires_in": float64(3600), "refresh_token": "refresh-2"}, res.body)

	res = postForm(t, srv.URL+"/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"refresh-1"}, "client_id": {"client-1"}})
	require.Equal(t, http.StatusOK, res.status)
	assert.Equal(t, []string{"refresh-1", "client-1"}, refreshArgs)
	assert.Equal(t, "refresh-2", res.body["refresh_token"])

	res = postForm(t, srv.URL+"/oauth/token", url.Values{"grant_type": {"refresh_token"}, "refresh_token": {"reused"}, "client_id": {"client-1"}})
	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, "invalid_grant", res.body["error"])

	res = postForm(t, srv.URL+"/oauth/token", url.Values{
		"grant_type": {"authorization_code"}, "code": {"spent"}, "code_verifier": {"v"}, "client_id": {"client-1"}, "redirect_uri": {"https://claude.ai/cb"},
	})
	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, "invalid_grant", res.body["error"])

	res = postForm(t, srv.URL+"/oauth/token", url.Values{"grant_type": {"authorization_code"}, "code": {"code-1"}, "client_id": {"client-1"}})
	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, "invalid_request", res.body["error"])

	res = postForm(t, srv.URL+"/oauth/token", url.Values{"grant_type": {"client_credentials"}})
	assert.Equal(t, http.StatusBadRequest, res.status)
	assert.Equal(t, "unsupported_grant_type", res.body["error"])
}

func TestOAuthRegister_RateLimited(t *testing.T) {
	calls := 0
	register := registerFunc(func(_ context.Context, name string, uris []string) (*authservice.OAuthClient, error) {
		calls++

		return &authservice.OAuthClient{ClientID: "client-1", Name: name, RedirectURIs: uris, CreatedAt: time.Now()}, nil
	})
	limiter := &countingLimiter{n: 2, seen: map[string]int{}}

	router := mux.NewRouter()
	NewOAuthHTTPHandler(testURLs, limiter, register, nil, nil).Register(router)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	const body = `{"client_name":"Claude","redirect_uris":["https://claude.ai/cb"]}`

	for range 2 {
		require.Equal(t, http.StatusCreated, postJSON(t, srv.URL+"/oauth/register", body).status)
	}

	res := postJSON(t, srv.URL+"/oauth/register", body)
	assert.Equal(t, http.StatusTooManyRequests, res.status)
	assert.Equal(t, "too_many_requests", res.body["error"])
	assert.Equal(t, "60", res.header.Get("Retry-After"))
	assert.Equal(t, 2, calls, "a limited request registers nothing")
	assert.Equal(t, map[string]int{"/oauth/register": 3}, limiter.seen, "only registration is limited")
}
