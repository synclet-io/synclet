package connectutil

import (
	"testing"

	"github.com/stretchr/testify/assert"
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
