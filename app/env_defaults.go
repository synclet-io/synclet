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
)

// envDefaults are applied for any var still unset after dotenv files have been loaded.
var envDefaults = map[string]string{
	"WATERMILL_TRANSPORT":          "sql",
	"WATERMILL_SQL_DRIVER":         "postgres",
	"WATERMILL_SQL_CONSUMER_GROUP": "synclet",
	"PUBLIC_HTTP_SERVER_ADDR":      "0.0.0.0:8080",
	"INTERNAL_HTTP_SERVER_ADDR":    "0.0.0.0:8081",
}

// applyEnvDefaults fills required env vars that have no value yet. Plain
// defaults come from envDefaults; secrets are generated (and persisted where
// losing them would be destructive).
func applyEnvDefaults(logger *slog.Logger) {
	for key, value := range envDefaults {
		if _, ok := os.LookupEnv(key); ok {
			continue
		}

		_ = os.Setenv(key, value)
	}

	if _, ok := os.LookupEnv("AUTH_JWT_SECRET"); !ok {
		secret, err := randomHex(32)
		if err != nil {
			logger.Error("can't generate default AUTH_JWT_SECRET", slog.Any("error", err))
		} else {
			_ = os.Setenv("AUTH_JWT_SECRET", secret)

			logger.Warn("AUTH_JWT_SECRET not set; generated an ephemeral one — existing sessions will be invalidated on next restart")
		}
	}

	if _, ok := os.LookupEnv("SECRET_ENCRYPTION_KEY"); !ok {
		key, path, err := loadOrCreateEncryptionKey()
		if err != nil {
			logger.Error("can't load or create SECRET_ENCRYPTION_KEY", slog.Any("error", err))
		} else {
			_ = os.Setenv("SECRET_ENCRYPTION_KEY", key)

			logger.Warn("SECRET_ENCRYPTION_KEY not set; using persisted key — back this file up", slog.String("path", path))
		}
	}
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
