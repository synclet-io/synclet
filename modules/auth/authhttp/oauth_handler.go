package authhttp

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"time"

	"github.com/gorilla/mux"
	"github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

const maxOAuthRequestBytes = 64 << 10

// OAuthClientRegistrar registers OAuth clients.
type OAuthClientRegistrar interface {
	Execute(ctx context.Context, name string, redirectURIs []string) (*authservice.OAuthClient, error)
}

// OAuthCodeExchanger exchanges authorization codes for tokens.
type OAuthCodeExchanger interface {
	Execute(ctx context.Context, code, verifier, clientID, redirectURI string) (*authservice.OAuthTokens, error)
}

// OAuthTokenRefresher rotates refresh tokens.
type OAuthTokenRefresher interface {
	Execute(ctx context.Context, refreshToken, clientID string) (*authservice.OAuthTokens, error)
}

// RequestLimiter rate-limits plain HTTP endpoints per client.
type RequestLimiter interface {
	// AllowRequest reports whether a request to the endpoint key may proceed.
	AllowRequest(key string, r *http.Request) bool
}

// OAuthURLs are the base URLs the OAuth metadata names.
type OAuthURLs struct {
	// Public is the base URL of the MCP server: the protected resource, the
	// issuer and the host of the token and registration endpoints.
	Public string
	// UI is the base URL of the main UI, which serves /oauth/authorize. Empty
	// leaves the authorization server metadata unserved.
	UI string
}

// OAuthHTTPHandler serves the OAuth discovery, registration and token endpoints.
type OAuthHTTPHandler struct {
	urls     OAuthURLs
	limiter  RequestLimiter
	register OAuthClientRegistrar
	exchange OAuthCodeExchanger
	refresh  OAuthTokenRefresher
}

// NewOAuthHTTPHandler serves the OAuth endpoints. limiter limits client
// registration, which needs no credentials, per client IP under the key
// RegisterPath.
func NewOAuthHTTPHandler(urls OAuthURLs, limiter RequestLimiter, register OAuthClientRegistrar, exchange OAuthCodeExchanger, refresh OAuthTokenRefresher) *OAuthHTTPHandler {
	return &OAuthHTTPHandler{urls: urls, limiter: limiter, register: register, exchange: exchange, refresh: refresh}
}

// RegisterPath is the dynamic client registration endpoint.
const RegisterPath = "/oauth/register"

func (h *OAuthHTTPHandler) Register(router *mux.Router) {
	resource := auth.ProtectedResourceMetadataHandler(&oauthex.ProtectedResourceMetadata{
		Resource:               h.urls.Public + "/mcp",
		AuthorizationServers:   []string{h.urls.Public},
		BearerMethodsSupported: []string{"header"},
		ResourceName:           "Synclet",
	})
	router.Handle("/.well-known/oauth-protected-resource", resource)
	router.Handle("/.well-known/oauth-protected-resource/mcp", resource)
	router.HandleFunc("/.well-known/oauth-authorization-server", h.handleServerMetadata).Methods(http.MethodGet, http.MethodOptions)
	router.HandleFunc(RegisterPath, h.handleRegister).Methods(http.MethodPost, http.MethodOptions)
	router.HandleFunc("/oauth/token", h.handleToken).Methods(http.MethodPost, http.MethodOptions)
}

// allowCORS lets browser-based clients call the endpoint and reports whether
// the request was a preflight, already answered.
func allowCORS(w http.ResponseWriter, r *http.Request) bool {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

	if r.Method == http.MethodOptions {
		w.WriteHeader(http.StatusNoContent)

		return true
	}

	return false
}

func (h *OAuthHTTPHandler) handleServerMetadata(w http.ResponseWriter, r *http.Request) {
	if allowCORS(w, r) {
		return
	}

	if h.urls.UI == "" {
		http.NotFound(w, r)

		return
	}

	writeJSON(r.Context(), w, http.StatusOK, &authServerMetadata{
		Issuer:                            h.urls.Public,
		AuthorizationEndpoint:             h.urls.UI + "/oauth/authorize",
		TokenEndpoint:                     h.urls.Public + "/oauth/token",
		RegistrationEndpoint:              h.urls.Public + "/oauth/register",
		ResponseTypesSupported:            []string{"code"},
		GrantTypesSupported:               []string{"authorization_code", "refresh_token"},
		TokenEndpointAuthMethodsSupported: []string{"none"},
		CodeChallengeMethodsSupported:     []string{"S256"},
	})
}

func (h *OAuthHTTPHandler) handleRegister(w http.ResponseWriter, r *http.Request) {
	if allowCORS(w, r) {
		return
	}

	if !h.limiter.AllowRequest(RegisterPath, r) {
		w.Header().Set("Retry-After", "60")
		writeJSON(r.Context(), w, http.StatusTooManyRequests, &oauthex.ClientRegistrationError{
			ErrorCode:        "too_many_requests",
			ErrorDescription: "too many client registrations, try again later",
		})

		return
	}

	var metadata oauthex.ClientRegistrationMetadata
	if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, maxOAuthRequestBytes)).Decode(&metadata); err != nil {
		writeJSON(r.Context(), w, http.StatusBadRequest, &oauthex.ClientRegistrationError{ErrorCode: "invalid_client_metadata", ErrorDescription: "request body is not client metadata"})

		return
	}

	client, err := h.register.Execute(r.Context(), metadata.ClientName, metadata.RedirectURIs)
	if validationErr, ok := errors.AsType[*authservice.ValidationError](err); ok {
		writeJSON(r.Context(), w, http.StatusBadRequest, &oauthex.ClientRegistrationError{ErrorCode: "invalid_redirect_uri", ErrorDescription: validationErr.Message})

		return
	}

	if err != nil {
		slog.Default().ErrorContext(r.Context(), "registering OAuth client", "error", err)
		writeJSON(r.Context(), w, http.StatusInternalServerError, &oauthex.ClientRegistrationError{ErrorCode: "server_error"})

		return
	}

	writeJSON(r.Context(), w, http.StatusCreated, &oauthex.ClientRegistrationResponse{
		ClientRegistrationMetadata: oauthex.ClientRegistrationMetadata{
			RedirectURIs:            client.RedirectURIs,
			TokenEndpointAuthMethod: "none",
			GrantTypes:              []string{"authorization_code", "refresh_token"},
			ResponseTypes:           []string{"code"},
			ClientName:              client.Name,
		},
		ClientID:         client.ClientID,
		ClientIDIssuedAt: client.CreatedAt,
	})
}

// authServerMetadata is the RFC 8414 authorization server metadata this
// server publishes. It has no jwks_uri: the server issues no signed tokens.
type authServerMetadata struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RegistrationEndpoint              string   `json:"registration_endpoint"`
	ResponseTypesSupported            []string `json:"response_types_supported"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
	CodeChallengeMethodsSupported     []string `json:"code_challenge_methods_supported"`
}

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int64  `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
}

type tokenError struct {
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description,omitempty"`
}

func (h *OAuthHTTPHandler) handleToken(w http.ResponseWriter, r *http.Request) {
	if allowCORS(w, r) {
		return
	}

	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")

	r.Body = http.MaxBytesReader(w, r.Body, maxOAuthRequestBytes)
	if err := r.ParseForm(); err != nil {
		writeJSON(r.Context(), w, http.StatusBadRequest, tokenError{Error: "invalid_request", ErrorDescription: "malformed form body"})

		return
	}

	clientID := r.PostForm.Get("client_id")
	if clientID == "" {
		clientID, _, _ = r.BasicAuth()
	}

	var (
		tokens  *authservice.OAuthTokens
		err     error
		missing bool
	)

	switch r.PostForm.Get("grant_type") {
	case "authorization_code":
		code, verifier, redirectURI := r.PostForm.Get("code"), r.PostForm.Get("code_verifier"), r.PostForm.Get("redirect_uri")
		if missing = code == "" || verifier == "" || redirectURI == "" || clientID == ""; !missing {
			tokens, err = h.exchange.Execute(r.Context(), code, verifier, clientID, redirectURI)
		}
	case "refresh_token":
		refreshToken := r.PostForm.Get("refresh_token")
		if missing = refreshToken == "" || clientID == ""; !missing {
			tokens, err = h.refresh.Execute(r.Context(), refreshToken, clientID)
		}
	default:
		writeJSON(r.Context(), w, http.StatusBadRequest, tokenError{Error: "unsupported_grant_type"})

		return
	}

	switch {
	case missing:
		writeJSON(r.Context(), w, http.StatusBadRequest, tokenError{Error: "invalid_request", ErrorDescription: "a required parameter is missing"})
	case errors.Is(err, authservice.ErrInvalidGrant):
		writeJSON(r.Context(), w, http.StatusBadRequest, tokenError{Error: "invalid_grant"})
	case err != nil:
		slog.Default().ErrorContext(r.Context(), "issuing OAuth tokens", "error", err)
		writeJSON(r.Context(), w, http.StatusInternalServerError, tokenError{Error: "server_error"})
	default:
		writeJSON(r.Context(), w, http.StatusOK, tokenResponse{
			AccessToken:  tokens.AccessToken,
			TokenType:    "Bearer",
			ExpiresIn:    int64(tokens.ExpiresIn / time.Second),
			RefreshToken: tokens.RefreshToken,
		})
	}
}

func writeJSON(ctx context.Context, w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(body); err != nil {
		slog.Default().WarnContext(ctx, "writing OAuth response", "error", err)
	}
}
