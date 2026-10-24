package authhttp

import (
	"net/http"
	"net/url"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/gorilla/mux"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// oidcStateCookie holds the binding of the OIDC state to the browser that
// started the login.
const (
	oidcStateCookie     = "synclet_oidc_state"
	oidcStateCookiePath = "/auth/oidc/"
)

// OIDCHTTPHandler provides plain HTTP endpoints for OIDC login/callback flows.
type OIDCHTTPHandler struct {
	startOIDCLogin     *authservice.StartOIDCLogin
	handleOIDCCallback *authservice.HandleOIDCCallback
	frontendURL        string // Base URL to redirect to after callback (e.g. "https://synclet.mycompany.com")
	logger             *logging.Logger
	cookieConfig       connectutil.CookieConfig
	stateTTL           time.Duration
}

// NewOIDCHTTPHandler creates a new OIDC HTTP handler. stateTTL is how long a
// started login stays valid.
func NewOIDCHTTPHandler(startOIDCLogin *authservice.StartOIDCLogin, handleOIDCCallback *authservice.HandleOIDCCallback, frontendURL string, logger *logging.Logger, cookieConfig connectutil.CookieConfig, stateTTL time.Duration) *OIDCHTTPHandler {
	return &OIDCHTTPHandler{
		startOIDCLogin:     startOIDCLogin,
		handleOIDCCallback: handleOIDCCallback,
		frontendURL:        frontendURL,
		logger:             logger,
		cookieConfig:       cookieConfig,
		stateTTL:           stateTTL,
	}
}

func (h *OIDCHTTPHandler) Register(router *mux.Router) {
	router.HandleFunc("/auth/oidc/{provider}/login", h.handleLogin).Methods("GET")
	router.HandleFunc("/auth/oidc/{provider}/callback", h.handleCallback).Methods("GET")
}

func (h *OIDCHTTPHandler) handleLogin(w http.ResponseWriter, r *http.Request) {
	provider := mux.Vars(r)["provider"]

	authURL, stateBinding, err := h.startOIDCLogin.Execute(r.Context(), provider)
	if err != nil {
		h.logger.WithError(err).WithField("provider", provider).Error(r.Context(), "OIDC login failed")
		http.Error(w, "OIDC login failed", http.StatusBadRequest)

		return
	}

	h.setStateCookie(w, stateBinding, int(h.stateTTL.Seconds()))

	// authURL is built by the OIDC provider's oauth2Config from a finite
	// server-configured provider set — not user-controlled.
	http.Redirect(w, r, authURL, http.StatusFound) //nolint:gosec // G710: authURL is server-built from configured providers.
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

	var stateBinding string
	if cookie, err := r.Cookie(oidcStateCookie); err == nil {
		stateBinding = cookie.Value
	}

	h.setStateCookie(w, "", -1)

	tokens, err := h.handleOIDCCallback.Execute(r.Context(), provider, code, state, stateBinding)
	if err != nil {
		h.logger.WithError(err).WithField("provider", provider).Error(r.Context(), "OIDC callback failed")
		http.Redirect(w, r, h.frontendURL+"/login?error=oidc_failed", http.StatusFound)

		return
	}

	// Set auth cookies and redirect to frontend callback page.
	connectutil.SetAuthCookies(w.Header(), toAuthTokens(tokens), h.cookieConfig)
	http.Redirect(w, r, h.frontendURL+"/auth/oidc/callback", http.StatusFound)
}

func (h *OIDCHTTPHandler) setStateCookie(w http.ResponseWriter, value string, maxAge int) {
	http.SetCookie(w, &http.Cookie{ //nolint:gosec // G124: Secure intentionally driven by cfg for dev/prod parity.
		Name:     oidcStateCookie,
		Value:    value,
		Path:     oidcStateCookiePath,
		MaxAge:   maxAge,
		HttpOnly: true,
		Secure:   h.cookieConfig.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

func toAuthTokens(tp *authservice.TokenPair) *connectutil.AuthTokens {
	return &connectutil.AuthTokens{
		AccessToken:      tp.AccessToken,
		RefreshToken:     tp.RefreshToken,
		ExpiresAt:        tp.ExpiresAt,
		RefreshExpiresAt: tp.RefreshExpiresAt,
	}
}
