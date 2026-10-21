package authservice

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ExchangeOAuthCode exchanges an authorization code for an access key and a
// refresh token.
type ExchangeOAuthCode struct {
	storage      Storage
	oauthStorage OAuthStorage
	audit        AuditRecorder
}

// NewExchangeOAuthCode creates a new ExchangeOAuthCode use case.
func NewExchangeOAuthCode(storage Storage, oauthStorage OAuthStorage, audit AuditRecorder) *ExchangeOAuthCode {
	return &ExchangeOAuthCode{storage: storage, oauthStorage: oauthStorage, audit: audit}
}

// Execute redeems code once. A code that is unknown, expired, issued to
// another client or redirect URI, or whose PKCE challenge verifier does not
// match, returns ErrInvalidGrant and is spent all the same. A redeemed code
// starts a grant, recorded as the client's last grant, with an API key of the client's name and the approved level,
// expiring after OAuthAccessKeyTTL, and a refresh token.
func (uc *ExchangeOAuthCode) Execute(ctx context.Context, code, verifier, clientID, redirectURI string) (*OAuthTokens, error) {
	authCode, err := uc.oauthStorage.ConsumeOAuthCode(ctx, hashToken(code))
	if errors.Is(err, ErrOAuthCodeNotFound) {
		return nil, ErrInvalidGrant
	}

	if err != nil {
		return nil, fmt.Errorf("consuming authorization code: %w", err)
	}

	if time.Now().After(authCode.ExpiresAt) || authCode.ClientID != clientID || authCode.RedirectURI != redirectURI ||
		!pkceMatches(authCode.CodeChallenge, verifier) {
		return nil, ErrInvalidGrant
	}

	client, err := uc.storage.OAuthClients().First(ctx, &OAuthClientFilter{ClientID: filter.Equals(clientID)})
	if errors.Is(err, ErrOAuthClientNotFound) {
		return nil, ErrInvalidGrant
	}

	if err != nil {
		return nil, fmt.Errorf("getting OAuth client: %w", err)
	}

	var (
		tokens *OAuthTokens
		apiKey *APIKey
	)

	err = uc.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		now := time.Now()

		grant, err := tx.OAuthGrants().Create(ctx, &OAuthGrant{
			ID:          uuid.New(),
			ClientID:    clientID,
			UserID:      authCode.UserID,
			WorkspaceID: authCode.WorkspaceID,
			Level:       authCode.Level,
			CreatedAt:   now,
		})
		if err != nil {
			return fmt.Errorf("creating OAuth grant: %w", err)
		}

		client.LastGrantedAt = &now
		if _, err := tx.OAuthClients().Update(ctx, client); err != nil {
			return fmt.Errorf("recording the client's last grant: %w", err)
		}

		tokens, apiKey, err = mintOAuthTokens(ctx, tx, grant, client.Name)

		return err
	})
	if err != nil {
		return nil, err
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:   apiKey.WorkspaceID,
		ActorID:       apiKey.UserID,
		Action:        "create",
		ResourceType:  "api_key",
		ResourceID:    apiKey.ID,
		ResourceLabel: apiKey.Name,
		After:         map[string]any{"name": apiKey.Name, "oauth_level": apiKey.OAuthLevel.String()},
	})

	return tokens, nil
}

// pkceMatches reports whether verifier is a well-formed RFC 7636 code
// verifier, 43 to 128 unreserved characters, that hashes to the S256 challenge.
func pkceMatches(challenge, verifier string) bool {
	if !validCodeVerifier(verifier) {
		return false
	}

	sum := sha256.Sum256([]byte(verifier))
	computed := base64.RawURLEncoding.EncodeToString(sum[:])

	return subtle.ConstantTimeCompare([]byte(computed), []byte(challenge)) == 1
}

func validCodeVerifier(verifier string) bool {
	if len(verifier) < 43 || len(verifier) > 128 {
		return false
	}

	for _, c := range []byte(verifier) {
		switch {
		case c >= 'A' && c <= 'Z', c >= 'a' && c <= 'z', c >= '0' && c <= '9', c == '-', c == '.', c == '_', c == '~':
		default:
			return false
		}
	}

	return true
}
