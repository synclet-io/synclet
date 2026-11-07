package connectutil

import (
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestWebhookHTTPClientRefusesInternalAddresses(t *testing.T) {
	internal := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(internal.Close)

	client := NewWebhookHTTPClient(5 * time.Second)

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, internal.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}

	require.ErrorIs(t, err, ErrWebhookAddressBlocked)
}

func TestWebhookHTTPClientRefusesBlockedRedirectTarget(t *testing.T) {
	listener, err := (&net.ListenConfig{}).Listen(t.Context(), "tcp", "[::1]:0")
	if err != nil {
		t.Skipf("IPv6 loopback unavailable: %v", err)
	}

	target := httptest.NewUnstartedServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	require.NoError(t, target.Listener.Close())
	target.Listener = listener
	target.Start()
	t.Cleanup(target.Close)

	redirector := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	t.Cleanup(redirector.Close)

	client := newGuardedHTTPClient(5*time.Second, func(ip net.IP) bool {
		return ip.Equal(net.IPv6loopback)
	})

	req, err := http.NewRequestWithContext(t.Context(), http.MethodGet, redirector.URL, http.NoBody)
	require.NoError(t, err)

	resp, err := client.Do(req)
	if resp != nil {
		require.NoError(t, resp.Body.Close())
	}

	require.ErrorIs(t, err, ErrWebhookAddressBlocked)
}
