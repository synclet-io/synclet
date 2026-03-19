package connectutil

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	})

	handler := SecurityHeadersMiddleware(inner)

	req := httptest.NewRequestWithContext(context.Background(), http.MethodGet, "/test", http.NoBody)
	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	resp := rec.Result()
	defer func() { _ = resp.Body.Close() }()

	t.Run("sets X-Frame-Options", func(t *testing.T) {
		assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	})

	t.Run("sets X-Content-Type-Options", func(t *testing.T) {
		assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	})

	t.Run("sets Strict-Transport-Security", func(t *testing.T) {
		assert.Equal(t, "max-age=63072000; includeSubDomains", resp.Header.Get("Strict-Transport-Security"))
	})

	t.Run("sets Referrer-Policy", func(t *testing.T) {
		assert.Equal(t, "strict-origin-when-cross-origin", resp.Header.Get("Referrer-Policy"))
	})

	t.Run("sets X-XSS-Protection", func(t *testing.T) {
		assert.Equal(t, "0", resp.Header.Get("X-XSS-Protection"))
	})

	t.Run("sets Content-Security-Policy", func(t *testing.T) {
		assert.Contains(t, resp.Header.Get("Content-Security-Policy"), "default-src 'self'")
	})

	t.Run("passes request to inner handler", func(t *testing.T) {
		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Equal(t, "ok", string(body))
		assert.Equal(t, http.StatusOK, resp.StatusCode)
	})
}
