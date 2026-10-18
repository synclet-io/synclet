package app

import (
	"bytes"
	"encoding/base64"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	validJWTSecret     = "e2e-test-jwt-secret-key-that-is-long-enough" //nolint:gosec // test fixture.
	validEncryptionKey = "PjsJSmk5MpLbSjsXYdIeflREhQ57B7+Sz0b0/VreDRk="
)

func unsetEnv(t *testing.T, keys ...string) {
	t.Helper()

	for _, key := range keys {
		t.Setenv(key, "")
		require.NoError(t, os.Unsetenv(key))
	}
}

func TestEnsureSecrets(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("production without secrets fails naming both variables", func(t *testing.T) {
		unsetEnv(t, "AUTH_JWT_SECRET", "SECRET_ENCRYPTION_KEY")
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set AUTH_JWT_SECRET and SECRET_ENCRYPTION_KEY")
		assert.Contains(t, err.Error(), `"production"`)
	})

	t.Run("production without encryption key fails naming it", func(t *testing.T) {
		unsetEnv(t, "SECRET_ENCRYPTION_KEY")
		t.Setenv("AUTH_JWT_SECRET", "jwt")
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set SECRET_ENCRYPTION_KEY:")
		assert.NotContains(t, err.Error(), "AUTH_JWT_SECRET")
	})

	t.Run("production with both secrets succeeds", func(t *testing.T) {
		t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
		t.Setenv("SECRET_ENCRYPTION_KEY", validEncryptionKey)
		t.Setenv("ENVIRONMENT", "production")

		require.NoError(t, ensureSecrets(logger))
	})

	t.Run("encryption key that is not base64 fails naming the variable and format", func(t *testing.T) {
		t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
		t.Setenv("SECRET_ENCRYPTION_KEY", "not base64!")
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SECRET_ENCRYPTION_KEY")
		assert.Contains(t, err.Error(), "base64-encoded 32-byte key")
	})

	t.Run("encryption key of the wrong length fails naming the variable and format", func(t *testing.T) {
		t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
		t.Setenv("SECRET_ENCRYPTION_KEY", base64.StdEncoding.EncodeToString([]byte("short")))
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "SECRET_ENCRYPTION_KEY")
		assert.Contains(t, err.Error(), "base64-encoded 32-byte key")
		assert.Contains(t, err.Error(), "got 5 bytes")
	})

	t.Run("short JWT secret fails naming the variable and minimum length", func(t *testing.T) {
		t.Setenv("AUTH_JWT_SECRET", "short-dummy-secret")
		t.Setenv("SECRET_ENCRYPTION_KEY", validEncryptionKey)
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AUTH_JWT_SECRET")
		assert.Contains(t, err.Error(), "at least 32 bytes")
		assert.NotContains(t, err.Error(), "SECRET_ENCRYPTION_KEY")
	})

	t.Run("both malformed secrets are reported together", func(t *testing.T) {
		t.Setenv("AUTH_JWT_SECRET", "jwt")
		t.Setenv("SECRET_ENCRYPTION_KEY", "key")
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "AUTH_JWT_SECRET")
		assert.Contains(t, err.Error(), "SECRET_ENCRYPTION_KEY")
	})

	t.Run("development inside kubernetes refuses to generate", func(t *testing.T) {
		unsetEnv(t, "AUTH_JWT_SECRET")
		t.Setenv("SECRET_ENCRYPTION_KEY", "key")
		t.Setenv("ENVIRONMENT", "development")
		t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "inside a container")
	})

	t.Run("development generates and persists secrets", func(t *testing.T) {
		if isRunningInContainer() {
			t.Skip("secret generation is disabled inside a container")
		}

		home := t.TempDir()
		unsetEnv(t, "AUTH_JWT_SECRET", "SECRET_ENCRYPTION_KEY", "KUBERNETES_SERVICE_HOST")
		t.Setenv("ENVIRONMENT", "development")
		t.Setenv("HOME", home)
		t.Setenv("XDG_CONFIG_HOME", home)

		require.NoError(t, ensureSecrets(logger))
		assert.NotEmpty(t, os.Getenv("AUTH_JWT_SECRET"))

		key := os.Getenv("SECRET_ENCRYPTION_KEY")
		require.NotEmpty(t, key)

		persisted, path, err := loadOrCreateEncryptionKey()
		require.NoError(t, err)
		assert.Equal(t, key, persisted)
		assert.Contains(t, path, home)
	})
}

func TestApplyPlainEnvDefaultsSetsHTTPTimeouts(t *testing.T) {
	keys := []string{
		"PUBLIC_HTTP_SERVER_READ_HEADER_TIMEOUT",
		"PUBLIC_HTTP_SERVER_IDLE_TIMEOUT",
		"INTERNAL_HTTP_SERVER_READ_HEADER_TIMEOUT",
		"INTERNAL_HTTP_SERVER_IDLE_TIMEOUT",
	}
	unsetEnv(t, keys...)
	t.Setenv("PUBLIC_HTTP_SERVER_IDLE_TIMEOUT", "30s")
	require.NoError(t, os.Unsetenv("INTERNAL_HTTP_SERVER_IDLE_TIMEOUT"))

	applyPlainEnvDefaults()

	assert.Equal(t, "10s", os.Getenv("PUBLIC_HTTP_SERVER_READ_HEADER_TIMEOUT"))
	assert.Equal(t, "30s", os.Getenv("PUBLIC_HTTP_SERVER_IDLE_TIMEOUT"))
	assert.Equal(t, "10s", os.Getenv("INTERNAL_HTTP_SERVER_READ_HEADER_TIMEOUT"))
	assert.Equal(t, "120s", os.Getenv("INTERNAL_HTTP_SERVER_IDLE_TIMEOUT"))
}

func TestApplyPlainEnvDefaultsFillsEmptyValues(t *testing.T) {
	for key := range envDefaults {
		t.Setenv(key, "")
	}

	applyPlainEnvDefaults()

	for key, value := range envDefaults {
		assert.Equal(t, value, os.Getenv(key), key)
	}
}

func TestApplyPlainEnvDefaultsKeepsNonEmptyValues(t *testing.T) {
	for key := range envDefaults {
		t.Setenv(key, "custom-"+key)
	}

	applyPlainEnvDefaults()

	for key := range envDefaults {
		assert.Equal(t, "custom-"+key, os.Getenv(key), key)
	}
}

func TestEnsureSecretsTreatsEmptyAsMissing(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	t.Run("production with empty secrets fails naming both variables", func(t *testing.T) {
		t.Setenv("AUTH_JWT_SECRET", "")
		t.Setenv("SECRET_ENCRYPTION_KEY", "")
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set AUTH_JWT_SECRET and SECRET_ENCRYPTION_KEY")
	})

	t.Run("production with empty encryption key fails naming it", func(t *testing.T) {
		t.Setenv("AUTH_JWT_SECRET", validJWTSecret)
		t.Setenv("SECRET_ENCRYPTION_KEY", "")
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set SECRET_ENCRYPTION_KEY:")
		assert.NotContains(t, err.Error(), "AUTH_JWT_SECRET")
	})
}

func TestEnsureSecretsWarnsOnComposeDefaults(t *testing.T) {
	composeEnv, err := godotenv.Read("../.env.compose")
	require.NoError(t, err)

	composeJWTSecret := composeEnv["AUTH_JWT_SECRET"]
	composeEncryptionKey := composeEnv["SECRET_ENCRYPTION_KEY"]

	require.NotEmpty(t, composeJWTSecret)
	require.NotEmpty(t, composeEncryptionKey)

	tests := []struct {
		name          string
		jwtSecret     string
		encryptionKey string
		warned        []string
		notWarned     []string
	}{
		{
			name:          "both compose defaults",
			jwtSecret:     composeJWTSecret,
			encryptionKey: composeEncryptionKey,
			warned:        []string{"AUTH_JWT_SECRET", "SECRET_ENCRYPTION_KEY"},
		},
		{
			name:          "compose JWT secret only",
			jwtSecret:     composeJWTSecret,
			encryptionKey: validEncryptionKey,
			warned:        []string{"AUTH_JWT_SECRET"},
			notWarned:     []string{"SECRET_ENCRYPTION_KEY"},
		},
		{
			name:          "compose encryption key only",
			jwtSecret:     validJWTSecret,
			encryptionKey: composeEncryptionKey,
			warned:        []string{"SECRET_ENCRYPTION_KEY"},
			notWarned:     []string{"AUTH_JWT_SECRET"},
		},
		{
			name:          "own secrets",
			jwtSecret:     validJWTSecret,
			encryptionKey: validEncryptionKey,
			notWarned:     []string{"AUTH_JWT_SECRET", "SECRET_ENCRYPTION_KEY"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Setenv("AUTH_JWT_SECRET", tt.jwtSecret)
			t.Setenv("SECRET_ENCRYPTION_KEY", tt.encryptionKey)
			t.Setenv("ENVIRONMENT", "production")

			var logs bytes.Buffer

			require.NoError(t, ensureSecrets(slog.New(slog.NewTextHandler(&logs, nil))))

			warnings := make(map[string]bool)

			for line := range strings.Lines(logs.String()) {
				if !strings.Contains(line, "level=WARN") {
					continue
				}

				for _, name := range []string{"AUTH_JWT_SECRET", "SECRET_ENCRYPTION_KEY"} {
					if strings.Contains(line, name) {
						warnings[name] = true
					}
				}
			}

			for _, name := range tt.warned {
				assert.True(t, warnings[name], "expected a WARN line naming %s, logs:\n%s", name, logs.String())
			}

			for _, name := range tt.notWarned {
				assert.False(t, warnings[name], "unexpected WARN line naming %s, logs:\n%s", name, logs.String())
			}
		})
	}
}

func TestDropControlPlaneEnv(t *testing.T) {
	t.Setenv("DB_DSN", "postgres://control-plane/synclet")
	t.Setenv("WATERMILL_TRANSPORT", "sql")
	t.Setenv("EXECUTOR_API_URL", "http://server:8081")

	var buf bytes.Buffer

	dropControlPlaneEnv(slog.New(slog.NewTextHandler(&buf, nil)))

	assert.Empty(t, os.Getenv("DB_DSN"))
	assert.Empty(t, os.Getenv("WATERMILL_TRANSPORT"))
	assert.Equal(t, "http://server:8081", os.Getenv("EXECUTOR_API_URL"))
	assert.Contains(t, buf.String(), "ignoring DB_DSN: control-plane variable, executors reach the server through the executor API")
	assert.Contains(t, buf.String(), "ignoring WATERMILL_TRANSPORT")
}
