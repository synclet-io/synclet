package authservice

import (
	"fmt"

	"github.com/golang-jwt/jwt/v5"
)

// ValidateAccessToken validates a JWT access token and returns claims.
type ValidateAccessToken struct {
	config Config
}

// NewValidateAccessToken creates a new ValidateAccessToken use case.
func NewValidateAccessToken(config Config) *ValidateAccessToken {
	return &ValidateAccessToken{config: config}
}

// Execute parses and validates the JWT token string.
func (uc *ValidateAccessToken) Execute(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method")
		}
		return []byte(uc.config.JWTSecret), nil
	},
		jwt.WithIssuer("synclet"),
		jwt.WithAudience("synclet"),
	)
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	if !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}

	return claims, nil
}
