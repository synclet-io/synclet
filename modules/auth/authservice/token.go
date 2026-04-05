package authservice

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

// Config holds auth service configuration.
type Config struct {
	JWTSecret           string
	AccessTokenTTL      time.Duration
	RefreshTokenTTL     time.Duration
	SingleWorkspaceMode bool
	RegistrationEnabled bool
	OIDCCallbackBaseURL string
	OIDCStateTTL        time.Duration
	MinPasswordLength   int
}

// TokenPair represents an access + refresh token pair.
type TokenPair struct {
	AccessToken      string
	RefreshToken     string
	ExpiresAt        time.Time
	RefreshExpiresAt time.Time
}

// Claims represents JWT claims.
type Claims struct {
	jwt.RegisteredClaims
	UserID uuid.UUID `json:"user_id"`
	Email  string    `json:"email"`
}

// generateTokenPair creates a new access + refresh token pair for a user.
func generateTokenPair(ctx context.Context, storage Storage, config Config, user *User) (*TokenPair, error) {
	now := time.Now()
	expiresAt := now.Add(config.AccessTokenTTL)

	claims := &Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			Issuer:    "synclet",
			Audience:  jwt.ClaimStrings{"synclet"},
		},
		UserID: user.ID,
		Email:  user.Email,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

	accessToken, err := token.SignedString([]byte(config.JWTSecret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	// Generate refresh token.
	refreshTokenRaw, err := generateRandomString(64)
	if err != nil {
		return nil, fmt.Errorf("generating refresh token: %w", err)
	}

	refreshTokenHash := hashToken(refreshTokenRaw)

	refreshToken := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: now.Add(config.RefreshTokenTTL),
		CreatedAt: now,
	}

	if _, err := storage.RefreshTokens().Create(ctx, refreshToken); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshTokenRaw,
		ExpiresAt:        expiresAt,
		RefreshExpiresAt: refreshToken.ExpiresAt,
	}, nil
}

// hashToken returns the SHA-256 hex hash of a token string.
func hashToken(token string) string {
	h := sha256.Sum256([]byte(token))

	return hex.EncodeToString(h[:])
}

// generateRandomString generates a cryptographically random hex string of n bytes.
func generateRandomString(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}

	return hex.EncodeToString(b), nil
}
