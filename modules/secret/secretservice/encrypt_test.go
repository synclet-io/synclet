package secretservice

import (
	"bytes"
	"crypto/rand"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func randomKey(t *testing.T) []byte {
	t.Helper()

	key := make([]byte, 32)
	_, err := rand.Read(key)
	require.NoError(t, err)

	return key
}

func TestEncryptDecrypt_RoundTrip(t *testing.T) {
	tests := []struct {
		name      string
		plaintext []byte
	}{
		{"short string", []byte("hunter2")},
		{"empty plaintext", []byte("")},
		{"binary payload", []byte{0x00, 0xff, 0x10, 0x20, 0xde, 0xad, 0xbe, 0xef}},
		{"long string", bytes.Repeat([]byte("a"), 4096)},
		{"json-shaped", []byte(`{"username":"admin","password":"s3cr3t!"}`)},
	}

	key := randomKey(t)

	for _, testCase := range tests {
		t.Run(testCase.name, func(t *testing.T) {
			ciphertext, salt, nonce, err := Encrypt(key, testCase.plaintext)
			require.NoError(t, err)
			require.NotEqual(t, testCase.plaintext, ciphertext, "ciphertext must differ from plaintext")
			require.Len(t, salt, saltLength)
			require.Len(t, nonce, 12)

			plaintext, err := Decrypt(key, salt, nonce, ciphertext)
			require.NoError(t, err)
			// AES-GCM returns nil for empty payloads; treat nil and empty as
			// equivalent so callers don't have to special-case.
			assert.Len(t, plaintext, len(testCase.plaintext))

			if len(testCase.plaintext) > 0 {
				assert.Equal(t, testCase.plaintext, plaintext)
			}
		})
	}
}

func TestEncrypt_ProducesUniqueCiphertextPerCall(t *testing.T) {
	// Same plaintext + same key MUST produce different ciphertexts on each call,
	// because the salt and nonce are randomly generated. Equal output would be a
	// catastrophic regression (nonce reuse in AES-GCM breaks confidentiality).
	key := randomKey(t)
	plaintext := []byte("repeat-me")

	cipherA, saltA, nonceA, err := Encrypt(key, plaintext)
	require.NoError(t, err)
	cipherB, saltB, nonceB, err := Encrypt(key, plaintext)
	require.NoError(t, err)

	assert.NotEqual(t, saltA, saltB, "salts must not collide")
	assert.NotEqual(t, nonceA, nonceB, "nonces must not collide")
	assert.NotEqual(t, cipherA, cipherB, "ciphertexts must not collide")
}

func TestDecrypt_WrongKeyFails(t *testing.T) {
	correctKey := randomKey(t)
	wrongKey := randomKey(t)

	ciphertext, salt, nonce, err := Encrypt(correctKey, []byte("sensitive"))
	require.NoError(t, err)

	_, err = Decrypt(wrongKey, salt, nonce, ciphertext)
	assert.Error(t, err, "decryption with the wrong key must fail")
}

func TestDecrypt_TamperedCiphertextFails(t *testing.T) {
	key := randomKey(t)

	ciphertext, salt, nonce, err := Encrypt(key, []byte("integrity-check"))
	require.NoError(t, err)

	// Flip a single bit in the ciphertext. GCM's auth tag must reject this.
	tampered := append([]byte(nil), ciphertext...)
	tampered[0] ^= 0x01

	_, err = Decrypt(key, salt, nonce, tampered)
	assert.Error(t, err, "tampered ciphertext must fail GCM authentication")
}

func TestDecrypt_TamperedNonceFails(t *testing.T) {
	key := randomKey(t)

	ciphertext, salt, nonce, err := Encrypt(key, []byte("nonce-bind"))
	require.NoError(t, err)

	tamperedNonce := append([]byte(nil), nonce...)
	tamperedNonce[0] ^= 0x01

	_, err = Decrypt(key, salt, tamperedNonce, ciphertext)
	assert.Error(t, err, "wrong nonce must fail GCM authentication")
}

func TestDecrypt_TamperedSaltFails(t *testing.T) {
	key := randomKey(t)

	ciphertext, salt, nonce, err := Encrypt(key, []byte("salt-bind"))
	require.NoError(t, err)

	tamperedSalt := append([]byte(nil), salt...)
	tamperedSalt[0] ^= 0x01

	// A different salt derives a different per-secret key via HKDF.
	_, err = Decrypt(key, tamperedSalt, nonce, ciphertext)
	assert.Error(t, err, "wrong salt must yield a derived key that fails decryption")
}

func TestDeriveKey_Deterministic(t *testing.T) {
	masterKey := randomKey(t)
	salt := bytes.Repeat([]byte{0x5a}, saltLength)

	keyA, err := deriveKey(masterKey, salt)
	require.NoError(t, err)
	keyB, err := deriveKey(masterKey, salt)
	require.NoError(t, err)

	assert.Equal(t, keyA, keyB, "HKDF with the same inputs must derive the same key")
	assert.Len(t, keyA, 32, "derived AES-256 key must be 32 bytes")
}

func TestDeriveKey_DifferentSaltsProduceDifferentKeys(t *testing.T) {
	masterKey := randomKey(t)

	keyA, err := deriveKey(masterKey, bytes.Repeat([]byte{0x00}, saltLength))
	require.NoError(t, err)
	keyB, err := deriveKey(masterKey, bytes.Repeat([]byte{0xff}, saltLength))
	require.NoError(t, err)

	assert.NotEqual(t, keyA, keyB, "different salts must produce different derived keys")
}
