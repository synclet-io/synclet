package app

import (
	"strings"
	"testing"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateJWTSecret(t *testing.T) {
	for name, tc := range map[string]struct {
		secret  string
		wantErr string
	}{
		"32 bytes": {secret: strings.Repeat("x", 32)},
		"64 bytes": {secret: strings.Repeat("x", 64)},
		"31 bytes": {secret: strings.Repeat("x", 31), wantErr: "AUTH_JWT_SECRET must be at least 32 bytes long, got 31"},
		"empty":    {secret: "", wantErr: "AUTH_JWT_SECRET must be at least 32 bytes long, got 0"},
	} {
		t.Run(name, func(t *testing.T) {
			err := validateJWTSecret("AUTH_JWT_SECRET", []byte(tc.secret))
			if tc.wantErr == "" {
				require.NoError(t, err)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Contains(t, err.Error(), "openssl rand -base64 32")
		})
	}
}

func TestDecodeEncryptionKey(t *testing.T) {
	for name, tc := range map[string]struct {
		value   string
		wantLen int
		wantErr string
	}{
		"valid":        {value: validEncryptionKey, wantLen: 32},
		"not base64":   {value: "not base64!", wantErr: "SECRET_ENCRYPTION_KEY must be a standard base64-encoded 32-byte key, got a value that is not valid base64"},
		"url alphabet": {value: strings.NewReplacer("+", "-", "/", "_").Replace(validEncryptionKey), wantErr: "not valid base64"},
		"too short":    {value: "aGVsbG8=", wantErr: "SECRET_ENCRYPTION_KEY must be a standard base64-encoded 32-byte key, got 5 bytes"},
		"too long":     {value: "QUJDREVGR0hJSktMTU5PUFFSU1RVVldYWVphYmNkZWZn", wantErr: "got 33 bytes"},
	} {
		t.Run(name, func(t *testing.T) {
			key, err := decodeEncryptionKey("SECRET_ENCRYPTION_KEY", tc.value)
			if tc.wantErr == "" {
				require.NoError(t, err)
				assert.Len(t, key, tc.wantLen)

				return
			}

			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
			assert.Contains(t, err.Error(), "openssl rand -base64 32")
		})
	}
}

func TestSecretConfig_NamesMalformedVariable(t *testing.T) {
	for name, tc := range map[string]struct {
		current, previous string
		mention, absent   string
	}{
		"current key malformed": {
			current: "aGVsbG8=", previous: validEncryptionKey,
			mention: "SECRET_ENCRYPTION_KEY", absent: "SECRET_ENCRYPTION_KEY_PREVIOUS",
		},
		"previous key malformed": {
			current: validEncryptionKey, previous: "aGVsbG8=",
			mention: "SECRET_ENCRYPTION_KEY_PREVIOUS",
		},
	} {
		t.Run(name, func(t *testing.T) {
			t.Setenv("SECRET_ENCRYPTION_KEY", tc.current)
			t.Setenv("SECRET_ENCRYPTION_KEY_PREVIOUS", tc.previous)

			_, err := configutil.NewPrefixedConfigProvider[secretConfig]("SECRET_")()
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.mention)

			if tc.absent != "" {
				assert.NotContains(t, err.Error(), tc.absent)
			}
		})
	}
}

func TestSecretConfig_AcceptsValidKeys(t *testing.T) {
	t.Setenv("SECRET_ENCRYPTION_KEY", validEncryptionKey)
	t.Setenv("SECRET_ENCRYPTION_KEY_PREVIOUS", validEncryptionKey)

	res, err := configutil.NewPrefixedConfigProvider[secretConfig]("SECRET_")()
	require.NoError(t, err)
	assert.Len(t, res.EncryptionKey, 32)
	assert.Len(t, res.EncryptionKeyPrevious, 32)
}

func TestAuthConfig_NamesShortJWTSecret(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", "short")

	_, err := configutil.NewPrefixedConfigProvider[authConfig]("AUTH_")()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "AUTH_JWT_SECRET must be at least 32 bytes long")
}
