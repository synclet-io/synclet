package connectutil

import (
	"errors"
	"fmt"
	"net"
	"net/http"
	"syscall"
	"time"
)

var ErrWebhookAddressBlocked = errors.New("webhook address is private or internal")

// NewWebhookHTTPClient returns an HTTP client that connects only to public
// addresses. The check runs on the IP each connection is dialed to, so it holds
// for every redirect and for a hostname whose DNS answer changes after
// ValidateWebhookURL ran. The client dials directly, without a proxy.
func NewWebhookHTTPClient(timeout time.Duration) *http.Client {
	dialer := &net.Dialer{
		Timeout: timeout,
		Control: refuseBlockedAddress,
	}

	transport := http.DefaultTransport.(*http.Transport).Clone() //nolint:forcetypeassert // DefaultTransport is always *http.Transport
	transport.Proxy = nil
	transport.DialContext = dialer.DialContext

	return &http.Client{
		Timeout:   timeout,
		Transport: transport,
	}
}

func refuseBlockedAddress(_, address string, _ syscall.RawConn) error {
	host, _, err := net.SplitHostPort(address)
	if err != nil {
		return fmt.Errorf("parsing dial address %q: %w", address, err)
	}

	ip := net.ParseIP(host)
	if ip == nil || isBlockedIP(ip) {
		return fmt.Errorf("dialing %s: %w", host, ErrWebhookAddressBlocked)
	}

	return nil
}
