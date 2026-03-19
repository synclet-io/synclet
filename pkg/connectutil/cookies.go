package connectutil

import (
	"fmt"
	"net/http"
	"time"
)

// CookieConfig controls auth cookie behavior.
type CookieConfig struct {
	Secure bool // true in production (HTTPS), false in dev
}

// AuthTokens holds the values needed to set auth cookies.
type AuthTokens struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
}

const (
	accessTokenCookie  = "synclet_at"
	refreshTokenCookie = "synclet_rt"
	authMetaCookie     = "synclet_auth"
)

// SetAuthCookies sets the three auth cookies (access token, refresh token, metadata) on the response.
func SetAuthCookies(h http.Header, tokens *AuthTokens, cfg CookieConfig) {
	accessMaxAge := int(time.Until(tokens.ExpiresAt).Seconds())
	if accessMaxAge < 0 {
		accessMaxAge = 0
	}
	refreshMaxAge := int(time.Until(tokens.RefreshExpiresAt).Seconds())
	if refreshMaxAge < 0 {
		refreshMaxAge = 0
	}

	setCookie(h, &http.Cookie{
		Name:     accessTokenCookie,
		Value:    tokens.AccessToken,
		Path:     "/",
		MaxAge:   accessMaxAge,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})

	setCookie(h, &http.Cookie{
		Name:     refreshTokenCookie,
		Value:    tokens.RefreshToken,
		Path:     "/",
		MaxAge:   refreshMaxAge,
		HttpOnly: true,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})

	metaValue := fmt.Sprintf("access_expires=%d&refresh_expires=%d",
		tokens.ExpiresAt.Unix(), tokens.RefreshExpiresAt.Unix())
	setCookie(h, &http.Cookie{
		Name:     authMetaCookie,
		Value:    metaValue,
		Path:     "/",
		MaxAge:   refreshMaxAge,
		HttpOnly: false,
		Secure:   cfg.Secure,
		SameSite: http.SameSiteLaxMode,
	})
}

// ClearAuthCookies expires all three auth cookies.
func ClearAuthCookies(h http.Header, cfg CookieConfig) {
	for _, name := range []string{accessTokenCookie, refreshTokenCookie, authMetaCookie} {
		setCookie(h, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			MaxAge:   -1,
			HttpOnly: name != authMetaCookie,
			Secure:   cfg.Secure,
			SameSite: http.SameSiteLaxMode,
		})
	}
}

// ReadCookieFromHeaders extracts a cookie value from HTTP headers.
func ReadCookieFromHeaders(h http.Header, name string) string {
	r := &http.Request{Header: h}
	c, err := r.Cookie(name)
	if err != nil {
		return ""
	}
	return c.Value
}

func setCookie(h http.Header, c *http.Cookie) {
	h.Add("Set-Cookie", c.String())
}
