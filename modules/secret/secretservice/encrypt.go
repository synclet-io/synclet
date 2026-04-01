package secretservice

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"

	"github.com/synclet-io/synclet/pkg/secretutil"
)

var hkdfInfo = []byte("synclet-secret-v1")

const saltLength = 32

// Re-export ref utilities from pkg/secretutil for backward compatibility within this package.
const (
	SecretRefPrefix = secretutil.SecretRefPrefix
	SecretMask      = secretutil.SecretMask
)

var (
	IsSecretRef     = secretutil.IsSecretRef
	ExtractSecretID = secretutil.ExtractSecretID
	MakeSecretRef   = secretutil.MakeSecretRef
)

// Encrypt encrypts plaintext using AES-256-GCM with an HKDF-derived key.
// It generates a random salt and nonce internally.
func Encrypt(masterKey, plaintext []byte) (ciphertext, salt, nonce []byte, err error) {
	// Generate random salt
	salt = make([]byte, saltLength)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, nil, nil, fmt.Errorf("generating salt: %w", err)
	}

	// Derive per-secret key via HKDF
	derivedKey, err := deriveKey(masterKey, salt)
	if err != nil {
		return nil, nil, nil, err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("creating GCM: %w", err)
	}

	nonce = make([]byte, gcm.NonceSize()) // 12 bytes
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, nil, fmt.Errorf("generating nonce: %w", err)
	}

	ciphertext = gcm.Seal(nil, nonce, plaintext, nil)

	return ciphertext, salt, nonce, nil
}

// Decrypt decrypts ciphertext using AES-256-GCM with an HKDF-derived key.
func Decrypt(masterKey, salt, nonce, ciphertext []byte) ([]byte, error) {
	derivedKey, err := deriveKey(masterKey, salt)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("creating cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("creating GCM: %w", err)
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

func deriveKey(masterKey, salt []byte) ([]byte, error) {
	hkdfReader := hkdf.New(sha256.New, masterKey, salt, hkdfInfo)

	derivedKey := make([]byte, 32)
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return nil, fmt.Errorf("deriving key: %w", err)
	}

	return derivedKey, nil
}
