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
		wantErr  bool
	}{
		{"empty password", "", true},
		{"too short - 1 char", "a", true},
		{"too short - 11 chars", strings.Repeat("a", 11), true},
		{"exactly 12 chars", strings.Repeat("a", 12), false},
		{"long password - 100 chars", strings.Repeat("a", 100), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), "at least 12 characters")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
