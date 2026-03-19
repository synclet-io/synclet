package secretutil

import (
	"strings"

	"github.com/google/uuid"
)

const (
	SecretRefPrefix = "$secret:"
	SecretMask      = "********"
)

// IsSecretRef returns true if the value is a secret reference.
func IsSecretRef(value string) bool {
	return strings.HasPrefix(value, SecretRefPrefix)
}

// ExtractSecretID parses a secret reference and returns the UUID.
func ExtractSecretID(ref string) (uuid.UUID, error) {
	return uuid.Parse(strings.TrimPrefix(ref, SecretRefPrefix))
}

// MakeSecretRef creates a secret reference string from a UUID.
func MakeSecretRef(id uuid.UUID) string {
	return SecretRefPrefix + id.String()
}
