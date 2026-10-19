package connectutil

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateWebhookURL(t *testing.T) {
	tests := []struct {
		name    string
		url     string
		wantErr bool
	}{
		{"accepts https URL", "https://example.com/webhook", false},
		{"accepts http URL", "http://example.com/webhook", false},
		{"rejects ftp scheme", "ftp://example.com", true},
		{"rejects loopback IPv4", "http://127.0.0.1/webhook", true},
		{"rejects localhost", "http://localhost/webhook", true},
		{"rejects private 10.x", "http://10.0.0.1/webhook", true},
		{"rejects private 192.168.x", "http://192.168.1.1/webhook", true},
		{"rejects private 172.16.x", "http://172.16.0.1/webhook", true},
		{"rejects link-local metadata", "http://169.254.169.254/latest/meta-data", true},
		{"rejects empty string", "", true},
		{"rejects URL without host", "http:///path", true},
		{"rejects loopback IPv6", "http://[::1]/webhook", true},
		{"rejects unspecified address", "http://0.0.0.0/webhook", true},
	}

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			err := ValidateWebhookURL(testCase.url)
			if testCase.wantErr {
				assert.Error(t, err, "expected error for URL: %s", testCase.url)
			} else {
				assert.NoError(t, err, "expected no error for URL: %s", testCase.url)
			}
		})
	}
}

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		ip      string
		blocked bool
	}{
		{"0.0.0.0", true},
		{"0.255.255.255", true},
		{"10.0.0.1", true},
		{"10.255.255.255", true},
		{"100.64.0.1", true},
		{"100.127.255.254", true},
		{"127.0.0.1", true},
		{"127.255.255.254", true},
		{"169.254.169.254", true},
		{"172.16.0.1", true},
		{"172.31.255.254", true},
		{"192.0.0.8", true},
		{"192.0.2.1", true},
		{"192.88.99.1", true},
		{"192.168.1.1", true},
		{"198.18.0.1", true},
		{"198.19.255.254", true},
		{"198.51.100.1", true},
		{"203.0.113.1", true},
		{"224.0.0.1", true},
		{"239.255.255.250", true},
		{"240.0.0.1", true},
		{"255.255.255.254", true},
		{"255.255.255.255", true},
		{"::", true},
		{"::1", true},
		{"::ffff:127.0.0.1", true},
		{"::ffff:10.0.0.1", true},
		{"::ffff:169.254.169.254", true},
		{"64:ff9b::7f00:1", true},
		{"64:ff9b::808:808", true},
		{"64:ff9b:1::a00:1", true},
		{"100::1", true},
		{"2001::1", true},
		{"2001:1ff:ffff::1", true},
		{"2001:db8::1", true},
		{"2002:7f00:1::1", true},
		{"2002:808:808::1", true},
		{"fc00::1", true},
		{"fd12:3456::1", true},
		{"fe80::1", true},
		{"febf::1", true},
		{"ff02::1", true},
		{"ff0e::1", true},
		{"8.8.8.8", false},
		{"1.1.1.1", false},
		{"93.184.216.34", false},
		{"100.128.0.1", false},
		{"172.32.0.1", false},
		{"198.20.0.1", false},
		{"::ffff:8.8.8.8", false},
		{"2606:4700:4700::1111", false},
		{"2001:4860:4860::8888", false},
	}

	for _, testCase := range tests {
		t.Run(testCase.ip, func(t *testing.T) {
			ip := net.ParseIP(testCase.ip)
			require.NotNil(t, ip)
			assert.Equal(t, testCase.blocked, isBlockedIP(ip))
		})
	}
}
