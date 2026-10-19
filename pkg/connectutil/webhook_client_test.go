package connectutil

import (
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
