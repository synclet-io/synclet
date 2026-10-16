package app

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
)

// envDefaults are applied for any var still unset or empty after dotenv files have been loaded.
var envDefaults = map[string]string{
	"WATERMILL_TRANSPORT":                      "sql",
	"WATERMILL_SQL_DRIVER":                     "postgres",
	"WATERMILL_SQL_CONSUMER_GROUP":             "synclet",
	"PUBLIC_HTTP_SERVER_ADDR":                  "0.0.0.0:8080",
	"PUBLIC_HTTP_SERVER_READ_HEADER_TIMEOUT":   "10s",
	"PUBLIC_HTTP_SERVER_IDLE_TIMEOUT":          "120s",
	"INTERNAL_HTTP_SERVER_ADDR":                "0.0.0.0:8081",
	"INTERNAL_HTTP_SERVER_READ_HEADER_TIMEOUT": "10s",
	"INTERNAL_HTTP_SERVER_IDLE_TIMEOUT":        "120s",
}

// applyEnvDefaults fills plain defaults and the secrets the server and jobs
// need. When a secret is missing and cannot be generated safely, it logs which
// variable to set and exits the process with status 1.
func applyEnvDefaults(logger *slog.Logger) {
	applyPlainEnvDefaults()

	if err := ensureSecrets(logger); err != nil {
		logger.Error(err.Error())
		os.Exit(1)
	}
}

func applyPlainEnvDefaults() {
	for key, value := range envDefaults {
		if os.Getenv(key) != "" {
			continue
		}

		_ = os.Setenv(key, value)
	}
}

// ensureSecrets generates AUTH_JWT_SECRET and SECRET_ENCRYPTION_KEY only for
// local development: ENVIRONMENT unset or "development", outside a container.
// Anywhere else a missing secret is an error, because a generated JWT secret
// differs per replica and restart, and a key persisted inside a container is
// lost with it, leaving stored connector secrets unrecoverable.
func ensureSecrets(logger *slog.Logger) error {
	hasJWTSecret := os.Getenv("AUTH_JWT_SECRET") != ""
	hasEncryptionKey := os.Getenv("SECRET_ENCRYPTION_KEY") != ""

	if hasJWTSecret && hasEncryptionKey {
		return validateSecrets()
	}

	if reason := secretGenerationForbiddenReason(); reason != "" {
		var missing []string
		if !hasJWTSecret {
			missing = append(missing, "AUTH_JWT_SECRET")
		}

		if !hasEncryptionKey {
			missing = append(missing, "SECRET_ENCRYPTION_KEY")
		}

		return fmt.Errorf("set %s: %s", strings.Join(missing, " and "), reason)
	}

	if !hasJWTSecret {
		secret, err := randomHex(32)
		if err != nil {
			return fmt.Errorf("generate AUTH_JWT_SECRET: %w", err)
		}

		_ = os.Setenv("AUTH_JWT_SECRET", secret)

		logger.Warn("AUTH_JWT_SECRET not set; generated an ephemeral one — existing sessions will be invalidated on next restart")
	}

	if !hasEncryptionKey {
		key, path, err := loadOrCreateEncryptionKey()
		if err != nil {
			return fmt.Errorf("SECRET_ENCRYPTION_KEY is not set and no local key could be loaded or created: %w", err)
		}

		_ = os.Setenv("SECRET_ENCRYPTION_KEY", key)

		logger.Warn("SECRET_ENCRYPTION_KEY not set; using persisted key — back this file up", slog.String("path", path))
	}

	return validateSecrets()
}

const (
	minJWTSecretLen  = 32
	encryptionKeyLen = 32
)

// validateSecrets checks AUTH_JWT_SECRET and SECRET_ENCRYPTION_KEY against the
// formats the auth and secret modules require, reporting every malformed one.
func validateSecrets() error {
	var errs []error

	if n := len(os.Getenv("AUTH_JWT_SECRET")); n < minJWTSecretLen {
		errs = append(errs, fmt.Errorf("AUTH_JWT_SECRET must be at least %d bytes long, got %d (generate one with `openssl rand -base64 32`)", minJWTSecretLen, n))
	}

	key, err := base64.StdEncoding.DecodeString(os.Getenv("SECRET_ENCRYPTION_KEY"))

	switch {
	case err != nil:
		errs = append(errs, errors.New("SECRET_ENCRYPTION_KEY must be a standard base64-encoded 32-byte key, got a value that is not valid base64 (generate one with `openssl rand -base64 32`)"))
	case len(key) != encryptionKeyLen:
		errs = append(errs, fmt.Errorf("SECRET_ENCRYPTION_KEY must be a standard base64-encoded %d-byte key, got %d bytes (generate one with `openssl rand -base64 32`)", encryptionKeyLen, len(key)))
	}

	return errors.Join(errs...)
}

// secretGenerationForbiddenReason returns why secrets must not be generated
// for this process, or "" when generating them is allowed.
func secretGenerationForbiddenReason() string {
	if env := os.Getenv("ENVIRONMENT"); env != "" && env != "development" {
		return fmt.Sprintf("secrets are generated only when ENVIRONMENT is development, got %q", env)
	}

	if isRunningInContainer() {
		return "secrets are never generated inside a container"
	}

	return ""
}

func isRunningInContainer() bool {
	if isRunningInK8s() {
		return true
	}

	_, err := os.Stat("/.dockerenv")

	return err == nil
}

func randomHex(n int) (string, error) {
	buf := make([]byte, n)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("read random bytes: %w", err)
	}

	return hex.EncodeToString(buf), nil
}

// loadOrCreateEncryptionKey reads a base64-encoded 32-byte AES key from
// <UserConfigDir>/synclet/encryption.key, creating one on first run. Losing
// this file makes every stored connector secret unrecoverable.
func loadOrCreateEncryptionKey() (key, path string, err error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", "", fmt.Errorf("resolve user config dir: %w", err)
	}

	dir := filepath.Join(configDir, "synclet")
	path = filepath.Join(dir, "encryption.key")

	data, err := os.ReadFile(path) //nolint:gosec // path is fixed to <UserConfigDir>/synclet/encryption.key.
	if err == nil {
		return string(data), path, nil
	}

	if !errors.Is(err, fs.ErrNotExist) {
		return "", path, fmt.Errorf("read %s: %w", path, err)
	}

	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", path, fmt.Errorf("create %s: %w", dir, err)
	}

	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", path, fmt.Errorf("read random bytes: %w", err)
	}

	encoded := base64.StdEncoding.EncodeToString(raw)
	if err := os.WriteFile(path, []byte(encoded), 0o600); err != nil {
		return "", path, fmt.Errorf("write %s: %w", path, err)
	}

	return encoded, path, nil
}
