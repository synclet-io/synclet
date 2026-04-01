package authservice

import (
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const testJWTSecret = "test-secret-key-for-jwt-testing"

func TestJWTClaimsIncludeIssuer(t *testing.T) {
	token := createTestAccessToken(t)
	claims := parseTestToken(t, token)
	assert.Equal(t, "synclet", claims.Issuer)
}

func TestJWTClaimsIncludeAudience(t *testing.T) {
	token := createTestAccessToken(t)
	claims := parseTestToken(t, token)
	assert.Equal(t, jwt.ClaimStrings{"synclet"}, claims.Audience)
}

func TestJWTValidateAcceptsCorrectToken(t *testing.T) {
	token := createTestAccessToken(t)
	uc := NewValidateAccessToken(Config{JWTSecret: testJWTSecret})
	claims, err := uc.Execute(token)
	require.NoError(t, err)
	assert.Equal(t, "synclet", claims.Issuer)
	assert.Equal(t, jwt.ClaimStrings{"synclet"}, claims.Audience)
}

func TestJWTValidateRejectsWrongIssuer(t *testing.T) {
	token := createTokenWithClaims(t, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Issuer:    "wrong-issuer",
			Audience:  jwt.ClaimStrings{"synclet"},
		},
		UserID: uuid.New(),
		Email:  "test@example.com",
	})
	uc := NewValidateAccessToken(Config{JWTSecret: testJWTSecret})
	_, err := uc.Execute(token)
	assert.Error(t, err)
}

func TestJWTValidateRejectsWrongAudience(t *testing.T) {
	token := createTokenWithClaims(t, &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   uuid.New().String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Issuer:    "synclet",
			Audience:  jwt.ClaimStrings{"wrong-audience"},
		},
		UserID: uuid.New(),
		Email:  "test@example.com",
	})
	uc := NewValidateAccessToken(Config{JWTSecret: testJWTSecret})
	_, err := uc.Execute(token)
	assert.Error(t, err)
}

// createTestAccessToken creates a token using the same Claims structure as generateTokenPair.
func createTestAccessToken(t *testing.T) string {
	t.Helper()

	userID := uuid.New()
	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(15 * time.Minute)),
			Issuer:    "synclet",
			Audience:  jwt.ClaimStrings{"synclet"},
		},
		UserID: userID,
		Email:  "test@example.com",
	}

	return createTokenWithClaims(t, claims)
}

func createTokenWithClaims(t *testing.T, claims *Claims) string {
	t.Helper()

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(testJWTSecret))
	require.NoError(t, err)

	return signed
}

func parseTestToken(t *testing.T, tokenString string) *Claims {
	t.Helper()

	claims := &Claims{}
	_, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return []byte(testJWTSecret), nil
	})
	require.NoError(t, err)

	return claims
}
