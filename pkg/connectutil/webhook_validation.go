package connectutil

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/netip"
	"net/url"
)

// ValidateWebhookURL checks that a webhook URL is safe to call.
// Rejects non-HTTP(S) schemes and URLs that resolve to private/loopback/link-local IPs.
func ValidateWebhookURL(rawURL string) error {
	if rawURL == "" {
		return errors.New("webhook URL is required")
	}

	parsedURL, err := url.Parse(rawURL)
	if err != nil {
		return errors.New("invalid webhook URL")
	}

	if parsedURL.Scheme != "https" && parsedURL.Scheme != "http" {
		return errors.New("webhook URL must use http or https scheme")
	}

	hostname := parsedURL.Hostname()
	if hostname == "" {
		return errors.New("webhook URL must have a hostname")
	}

	// Check if hostname is a direct IP.
	if ip := net.ParseIP(hostname); ip != nil {
		if isBlockedIP(ip) {
			return errors.New("webhook URL must not point to a private or internal address")
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
			return errors.New("webhook URL must not resolve to a private or internal address")
		}
	}

	return nil
}

// blockedPrefixes lists the IANA special-purpose ranges that are not globally
// reachable, plus the IPv6 transition ranges whose embedded IPv4 address could
// reach an internal host.
var blockedPrefixes = []struct {
	prefix netip.Prefix
	rfc    string
}{
	{netip.MustParsePrefix("0.0.0.0/8"), "RFC 791: this network"},
	{netip.MustParsePrefix("10.0.0.0/8"), "RFC 1918: private use"},
	{netip.MustParsePrefix("100.64.0.0/10"), "RFC 6598: shared address space"},
	{netip.MustParsePrefix("127.0.0.0/8"), "RFC 1122: loopback"},
	{netip.MustParsePrefix("169.254.0.0/16"), "RFC 3927: link-local"},
	{netip.MustParsePrefix("172.16.0.0/12"), "RFC 1918: private use"},
	{netip.MustParsePrefix("192.0.0.0/24"), "RFC 6890: IETF protocol assignments"},
	{netip.MustParsePrefix("192.0.2.0/24"), "RFC 5737: documentation (TEST-NET-1)"},
	{netip.MustParsePrefix("192.88.99.0/24"), "RFC 7526: deprecated 6to4 relay anycast"},
	{netip.MustParsePrefix("192.168.0.0/16"), "RFC 1918: private use"},
	{netip.MustParsePrefix("198.18.0.0/15"), "RFC 2544: benchmarking"},
	{netip.MustParsePrefix("198.51.100.0/24"), "RFC 5737: documentation (TEST-NET-2)"},
	{netip.MustParsePrefix("203.0.113.0/24"), "RFC 5737: documentation (TEST-NET-3)"},
	{netip.MustParsePrefix("224.0.0.0/4"), "RFC 5771: multicast"},
	{netip.MustParsePrefix("240.0.0.0/4"), "RFC 1112: reserved"},
	{netip.MustParsePrefix("255.255.255.255/32"), "RFC 919: limited broadcast"},
	{netip.MustParsePrefix("::/128"), "RFC 4291: unspecified address"},
	{netip.MustParsePrefix("::1/128"), "RFC 4291: loopback"},
	{netip.MustParsePrefix("64:ff9b::/96"), "RFC 6052: NAT64 well-known prefix"},
	{netip.MustParsePrefix("64:ff9b:1::/48"), "RFC 8215: local-use NAT64"},
	{netip.MustParsePrefix("100::/64"), "RFC 6666: discard-only"},
	{netip.MustParsePrefix("2001::/23"), "RFC 2928: IETF protocol assignments, Teredo (RFC 4380)"},
	{netip.MustParsePrefix("2001:db8::/32"), "RFC 3849: documentation"},
	{netip.MustParsePrefix("2002::/16"), "RFC 3056: 6to4"},
	{netip.MustParsePrefix("3fff::/20"), "RFC 9637: documentation"},
	{netip.MustParsePrefix("5f00::/16"), "RFC 9602: SRv6 SIDs"},
	{netip.MustParsePrefix("fc00::/7"), "RFC 4193: unique local"},
	{netip.MustParsePrefix("fe80::/10"), "RFC 4291: link-local"},
	{netip.MustParsePrefix("ff00::/8"), "RFC 4291: multicast"},
}

// isBlockedIP reports whether ip lies in a range of blockedPrefixes. An
// IPv4-mapped IPv6 address (::ffff:0:0/96, RFC 4291) is checked as its IPv4 address.
func isBlockedIP(ip net.IP) bool {
	addr, ok := netip.AddrFromSlice(ip)
	if !ok {
		return true
	}

	addr = addr.Unmap()

	for _, blocked := range blockedPrefixes {
		if blocked.prefix.Contains(addr) {
			return true
		}
	}

	return false
}
