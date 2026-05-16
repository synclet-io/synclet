package connectutil

import (
	"net/http"
	"strings"
)

// img-src permits arbitrary https: hosts because the connector catalog renders
// icons from third-party registries (Airbyte's CDN, GitHub raw, custom user
// registries). We don't know the icon hostnames in advance and an allowlist
// would need to grow every time a new registry is added.
//
//nolint:gochecknoglobals
var contentSecurityPolicy = strings.Join([]string{
	"default-src 'self'",
	"script-src 'self'",
	"style-src 'self' 'unsafe-inline' https://fonts.googleapis.com",
	"font-src 'self' https://fonts.gstatic.com",
	"img-src 'self' data: https:",
	"connect-src 'self'",
}, "; ")

// SecurityHeadersMiddleware wraps an http.Handler and sets security headers on every response.
func SecurityHeadersMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("Strict-Transport-Security", "max-age=63072000; includeSubDomains")
		w.Header().Set("Referrer-Policy", "strict-origin-when-cross-origin")
		w.Header().Set("X-XSS-Protection", "0")
		w.Header().Set("Content-Security-Policy", contentSecurityPolicy)
		next.ServeHTTP(w, r)
	})
}
