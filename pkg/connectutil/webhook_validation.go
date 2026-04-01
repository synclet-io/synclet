package connectutil

import (
	"context"
	"fmt"
	"net"
	"net/url"
)

// ValidateWebhookURL checks that a webhook URL is safe to call.
// Rejects non-HTTP(S) schemes and URLs that resolve to private/loopback/link-local IPs.
func ValidateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return fmt.Errorf("webhook URL is required")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("invalid webhook URL")
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return fmt.Errorf("webhook URL must use http or https scheme")
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return fmt.Errorf("webhook URL must have a hostname")
	}

	// Check if hostname is a direct IP.
	if ip := net.ParseIP(hostname); ip != nil {
		if isBlockedIP(ip) {
			return fmt.Errorf("webhook URL must not point to a private or internal address")
		}

		return nil
	}

	// Resolve hostname and check all IPs.
	resolver := &net.Resolver{}

	ips, err := resolver.LookupIPAddr(context.Background(), hostname)
	if err != nil {
		return fmt.Errorf("cannot resolve webhook URL hostname: %s", hostname)
	}

	for _, ip := range ips {
		if isBlockedIP(ip.IP) {
			return fmt.Errorf("webhook URL must not resolve to a private or internal address")
		}
	}

	return nil
}

// ValidateWebhookURLAtDelivery re-validates a webhook URL at delivery time to prevent
// DNS rebinding attacks. The URL was validated at creation time, but DNS records may
// have changed since then. Call this before actually sending the webhook payload.
func ValidateWebhookURLAtDelivery(rawURL string) error {
	return ValidateWebhookURL(rawURL)
}

// isBlockedIP returns true for loopback, private, link-local, and unspecified IPs.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsPrivate() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsUnspecified()
}
