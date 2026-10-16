package app

import (
	"io"
	"log/slog"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
		t.Setenv("AUTH_JWT_SECRET", "jwt")
		t.Setenv("SECRET_ENCRYPTION_KEY", "key")
		t.Setenv("ENVIRONMENT", "production")

		require.NoError(t, ensureSecrets(logger))
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
		t.Setenv("AUTH_JWT_SECRET", "jwt")
		t.Setenv("SECRET_ENCRYPTION_KEY", "")
		t.Setenv("ENVIRONMENT", "production")

		err := ensureSecrets(logger)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "set SECRET_ENCRYPTION_KEY:")
		assert.NotContains(t, err.Error(), "AUTH_JWT_SECRET")
	})
}
