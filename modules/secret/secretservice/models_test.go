package secretservice

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

// TestSecret_CopyEquals exercises the generated Copy/Equals helpers so a
// regression in code generation surfaces in CI.
func TestSecret_CopyEquals(t *testing.T) {
	orig := &Secret{
		ID:             uuid.New(),
		EncryptedValue: []byte{0xde, 0xad, 0xbe, 0xef},
		Salt:           []byte{0x01, 0x02},
		Nonce:          []byte{0x03, 0x04, 0x05},
		KeyVersion:     7,
		OwnerType:      "channel",
		OwnerID:        uuid.New(),
	}
	clone := orig.Copy()

	assert.True(t, orig.Equals(&clone))
	clone.KeyVersion = 9
	assert.False(t, orig.Equals(&clone), "differing field must yield non-equal")
}

func TestErrors_Format(t *testing.T) {
	assert.Equal(t, "Secret not found", ErrSecretNotFound.Error())
	assert.Equal(t, "Secret already exists", ErrSecretAlreadyExists.Error())
	assert.Equal(t, "decryption failed: invalid key or corrupted data", ErrDecryptionFailed.Error())
}
