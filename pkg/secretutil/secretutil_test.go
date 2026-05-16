package secretutil

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIsSecretRef(t *testing.T) {
	id := uuid.New()

	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{"empty", "", false},
		{"plain value", "hunter2", false},
		{"prefix only", SecretRefPrefix, true},
		{"valid ref", MakeSecretRef(id), true},
		{"mask is not a ref", SecretMask, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, IsSecretRef(tt.value))
		})
	}
}

func TestExtractSecretID_RoundTrip(t *testing.T) {
	id := uuid.New()
	ref := MakeSecretRef(id)

	got, err := ExtractSecretID(ref)
	require.NoError(t, err)
	assert.Equal(t, id, got)
}

func TestExtractSecretID_InvalidUUID(t *testing.T) {
	_, err := ExtractSecretID(SecretRefPrefix + "not-a-uuid")
	assert.Error(t, err)
}

func TestMakeSecretRef_HasExpectedPrefix(t *testing.T) {
	id := uuid.New()
	ref := MakeSecretRef(id)

	assert.True(t, IsSecretRef(ref))
	assert.Equal(t, SecretRefPrefix+id.String(), ref)
}
