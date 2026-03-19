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
	JWTSecret          string
	AccessTokenExpiry  time.Duration
	RefreshTokenExpiry time.Duration
}

// DefaultConfig returns default auth config.
func DefaultConfig() Config {
	return Config{
		AccessTokenExpiry:  15 * time.Minute,
		RefreshTokenExpiry: 7 * 24 * time.Hour,
	}
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
	expiresAt := now.Add(config.AccessTokenExpiry)

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

	rt := &RefreshToken{
		ID:        uuid.New(),
		UserID:    user.ID,
		TokenHash: refreshTokenHash,
		ExpiresAt: now.Add(config.RefreshTokenExpiry),
		CreatedAt: now,
	}

	if _, err := storage.RefreshTokens().Create(ctx, rt); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &TokenPair{
		AccessToken:      accessToken,
		RefreshToken:     refreshTokenRaw,
		ExpiresAt:        expiresAt,
		RefreshExpiresAt: rt.ExpiresAt,
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
