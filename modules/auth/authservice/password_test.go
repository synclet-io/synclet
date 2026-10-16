package authservice

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name     string
		password string
		wantErr  string
	}{
		{"empty password", "", "at least 8 characters"},
		{"too short - 7 chars", "abcde1!", "at least 8 characters"},
		{"multibyte runes counted as characters", "пароль1", "at least 8 characters"},
		{"letters only", strings.Repeat("a", 8), "one letter and one digit or symbol"},
		{"digits only", "12345678", "one letter and one digit or symbol"},
		{"letters and spaces", "abcd efgh", "one letter and one digit or symbol"},
		{"letters and digit", "abcdefg1", ""},
		{"letters and symbol", "abcdefg-", ""},
		{"long password - 100 chars", strings.Repeat("a", 99) + "1", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password, 8)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
