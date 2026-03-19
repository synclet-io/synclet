package authservice

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateBoundClaims(t *testing.T) {
	bound := map[string]string{
		"department": "engineering",
		"team":       "platform",
	}

	t.Run("matching claims passes", func(t *testing.T) {
		claims := map[string]interface{}{
			"department": "engineering",
			"team":       "platform",
			"extra":      "value",
		}
		assert.NoError(t, ValidateBoundClaims(bound, claims))
	})

	t.Run("missing claim fails", func(t *testing.T) {
		claims := map[string]interface{}{
			"department": "engineering",
		}
		err := ValidateBoundClaims(bound, claims)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "team")
	})

	t.Run("wrong value fails", func(t *testing.T) {
		claims := map[string]interface{}{
			"department": "marketing",
			"team":       "platform",
		}
		err := ValidateBoundClaims(bound, claims)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "department")
	})

	t.Run("nil bound claims passes", func(t *testing.T) {
		assert.NoError(t, ValidateBoundClaims(nil, map[string]interface{}{"any": "value"}))
	})
}

func TestValidateEmailDomain(t *testing.T) {
	t.Run("matching domain passes", func(t *testing.T) {
		err := ValidateEmailDomain("user@mycompany.com", true, []string{"mycompany.com"})
		assert.NoError(t, err)
	})

	t.Run("wrong domain fails", func(t *testing.T) {
		err := ValidateEmailDomain("user@evil.com", true, []string{"mycompany.com"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "evil.com")
	})

	t.Run("empty allowed list passes all", func(t *testing.T) {
		err := ValidateEmailDomain("user@anything.com", false, nil)
		assert.NoError(t, err)
	})

	t.Run("unverified email rejected with allowed domains", func(t *testing.T) {
		err := ValidateEmailDomain("user@mycompany.com", false, []string{"mycompany.com"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not verified")
	})
}

func TestMapRole_StringClaim(t *testing.T) {
	roleMapping := map[string]string{"admin": "admin-group"}
	claims := map[string]interface{}{"groups": "admin-group"}

	role := MapRole("groups", roleMapping, "viewer", claims)
	assert.Equal(t, "admin", role)
}

func TestMapRole_ArrayClaim(t *testing.T) {
	roleMapping := map[string]string{"admin": "admin-group"}
	claims := map[string]interface{}{
		"groups": []interface{}{"dev", "admin-group"},
	}

	role := MapRole("groups", roleMapping, "viewer", claims)
	assert.Equal(t, "admin", role)
}

func TestMapRole_Default(t *testing.T) {
	roleMapping := map[string]string{"admin": "admin-group"}
	claims := map[string]interface{}{"groups": "dev-group"}

	role := MapRole("groups", roleMapping, "viewer", claims)
	assert.Equal(t, "viewer", role)
}

func TestMapRole_NoClaim(t *testing.T) {
	roleMapping := map[string]string{"admin": "admin-group"}
	claims := map[string]interface{}{"other": "value"}

	role := MapRole("groups", roleMapping, "viewer", claims)
	assert.Equal(t, "viewer", role)
}
