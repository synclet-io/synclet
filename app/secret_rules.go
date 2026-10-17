package app

import (
	"encoding/base64"
	"fmt"
)

const (
	minJWTSecretLen  = 32
	encryptionKeyLen = 32

	generateSecretHint = "generate one with `openssl rand -base64 32`"
)

// validateJWTSecret checks the secret held by the environment variable name
// against the minimum length JWT signing requires.
func validateJWTSecret(name string, secret []byte) error {
	if len(secret) < minJWTSecretLen {
		return fmt.Errorf("%s must be at least %d bytes long, got %d (%s)", name, minJWTSecretLen, len(secret), generateSecretHint)
	}

	return nil
}

// decodeEncryptionKey decodes the value of the environment variable name as a
// standard base64-encoded AES-256 key.
func decodeEncryptionKey(name, value string) ([]byte, error) {
	key, err := base64.StdEncoding.DecodeString(value)
	if err != nil {
		return nil, fmt.Errorf("%s must be a standard base64-encoded %d-byte key, got a value that is not valid base64 (%s)", name, encryptionKeyLen, generateSecretHint)
	}

	if len(key) != encryptionKeyLen {
		return nil, fmt.Errorf("%s must be a standard base64-encoded %d-byte key, got %d bytes (%s)", name, encryptionKeyLen, len(key), generateSecretHint)
	}

	return key, nil
}
