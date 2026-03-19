package authconnect

import (
	"net/http"
	"net/url"

	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/gorilla/mux"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// OIDCHTTPHandler provides plain HTTP endpoints for OIDC login/callback flows.
type OIDCHTTPHandler struct {
	startOIDCLogin     *authservice.StartOIDCLogin
	handleOIDCCallback *authservice.HandleOIDCCallback
	frontendURL        string // Base URL to redirect to after callback (e.g. "https://synclet.mycompany.com")
	logger             *logging.Logger
	cookieConfig       connectutil.CookieConfig
}

// NewOIDCHTTPHandler creates a new OIDC HTTP handler.
func NewOIDCHTTPHandler(startOIDCLogin *authservice.StartOIDCLogin, handleOIDCCallback *authservice.HandleOIDCCallback, frontendURL string, logger *logging.Logger, cookieConfig connectutil.CookieConfig) *OIDCHTTPHandler {
	return &OIDCHTTPHandler{
		startOIDCLogin:     startOIDCLogin,
		handleOIDCCallback: handleOIDCCallback,
		frontendURL:        frontendURL,
		logger:             logger,
		cookieConfig:       cookieConfig,
	}
}

// RegisterRoutes returns a MuxHandlerRegistrar that registers OIDC routes.
func (h *OIDCHTTPHandler) RegisterRoutes() pnphttpserver.MuxHandlerRegistrar {
	return pnphttpserver.MuxHandlerRegistrarFunc(func(router *mux.Router) {
		router.HandleFunc("/auth/oidc/{provider}/login", h.handleLogin).Methods("GET")
		router.HandleFunc("/auth/oidc/{provider}/callback", h.handleCallback).Methods("GET")
	})
}

func (h *OIDCHTTPHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"]
	authURL, err := h.startOIDCLogin.Execute(r.Context(), provider)
	if err != nil {
		h.logger.Error(r.Context(), "OIDC login failed", "error", err, "provider", provider)
		http.Error(w, "OIDC login failed", http.StatusBadRequest)
		return
	}
	http.Redirect(w, r, authURL, http.StatusFound)
}

func (h *OIDCHTTPHandler) handleCallback(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"]
	code := r.URL.Query().Get("code")
	state := r.URL.Query().Get("state")

	if code == "" || state == "" {
		// Check for error response from provider.
		errMsg := r.URL.Query().Get("error")
		errDesc := r.URL.Query().Get("error_description")
		if errMsg != "" {
			http.Redirect(w, r, h.frontendURL+"/login?error="+url.QueryEscape(errMsg+": "+errDesc), http.StatusFound)
			return
		}
		http.Error(w, "missing code or state parameter", http.StatusBadRequest)
		return
	}

	tokens, err := h.handleOIDCCallback.Execute(r.Context(), provider, code, state)
	if err != nil {
		// Redirect to frontend login page with error.
		http.Redirect(w, r, h.frontendURL+"/login?error=oidc_failed", http.StatusFound)
		return
	}

	// Set auth cookies and redirect to frontend callback page.
	connectutil.SetAuthCookies(w.Header(), toAuthTokens(tokens), h.cookieConfig)
	http.Redirect(w, r, h.frontendURL+"/auth/oidc/callback", http.StatusFound)
}
