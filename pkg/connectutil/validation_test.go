package connectutil

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidateStringLengths(t *testing.T) {
	t.Run("returns nil for valid-length strings", func(t *testing.T) {
		err := ValidateStringLengths(
			StringValidation{Field: "name", Value: "short", MaxLen: MaxNameLength},
			StringValidation{Field: "url", Value: "https://example.com", MaxLen: MaxURLLength},
		)
		assert.NoError(t, err)
	})

	t.Run("returns error for string exceeding MaxLen with field name", func(t *testing.T) {
		longStr := strings.Repeat("a", MaxNameLength+1)
		err := ValidateStringLengths(
			StringValidation{Field: "name", Value: longStr, MaxLen: MaxNameLength},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")
		assert.Contains(t, err.Error(), "255")
	})

	t.Run("returns error for first failing field when multiple validations", func(t *testing.T) {
		longStr := strings.Repeat("a", 300)
		err := ValidateStringLengths(
			StringValidation{Field: "first_field", Value: longStr, MaxLen: MaxNameLength},
			StringValidation{Field: "second_field", Value: longStr, MaxLen: MaxNameLength},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "first_field")
	})

	t.Run("empty string passes validation", func(t *testing.T) {
		err := ValidateStringLengths(
			StringValidation{Field: "name", Value: "", MaxLen: MaxNameLength},
		)
		assert.NoError(t, err)
	})

	t.Run("string at exactly MaxLen passes validation", func(t *testing.T) {
		exactStr := strings.Repeat("a", MaxNameLength)
		err := ValidateStringLengths(
			StringValidation{Field: "name", Value: exactStr, MaxLen: MaxNameLength},
		)
		assert.NoError(t, err)
	})

	t.Run("string at MaxLen+1 fails validation", func(t *testing.T) {
		overStr := strings.Repeat("a", MaxNameLength+1)
		err := ValidateStringLengths(
			StringValidation{Field: "name", Value: overStr, MaxLen: MaxNameLength},
		)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "name")
	})
}
