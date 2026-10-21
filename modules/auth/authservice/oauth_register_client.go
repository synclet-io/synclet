package authservice

import (
	"context"
	"fmt"
	"net/url"
	"time"

	"github.com/google/uuid"
)

const (
	maxOAuthRedirectURIs      = 10
	maxOAuthRedirectURILength = 2000
	maxOAuthClientNameLength  = 255
	defaultOAuthClientName    = "MCP client"
)

// RegisterOAuthClient registers a public OAuth client (RFC 7591 dynamic client
// registration).
type RegisterOAuthClient struct {
	storage Storage
}

// NewRegisterOAuthClient creates a new RegisterOAuthClient use case.
func NewRegisterOAuthClient(storage Storage) *RegisterOAuthClient {
	return &RegisterOAuthClient{storage: storage}
}

// Execute stores a client with name and redirectURIs and returns it with a new
// client id. Every redirect URI is https, or http on localhost or 127.0.0.1;
// any other is a ValidationError. A name longer than 255 characters is cut.
func (uc *RegisterOAuthClient) Execute(ctx context.Context, name string, redirectURIs []string) (*OAuthClient, error) {
	if len(redirectURIs) == 0 {
		return nil, &ValidationError{Message: "at least one redirect URI is required"}
	}

	if len(redirectURIs) > maxOAuthRedirectURIs {
		return nil, &ValidationError{Message: fmt.Sprintf("at most %d redirect URIs are allowed", maxOAuthRedirectURIs)}
	}

	for _, redirectURI := range redirectURIs {
		if err := validateOAuthRedirectURI(redirectURI); err != nil {
			return nil, err
		}
	}

	if name == "" {
		name = defaultOAuthClientName
	}

	if runes := []rune(name); len(runes) > maxOAuthClientNameLength {
		name = string(runes[:maxOAuthClientNameLength])
	}

	clientID, err := generateRandomString(16)
	if err != nil {
		return nil, fmt.Errorf("generating client id: %w", err)
	}

	client, err := uc.storage.OAuthClients().Create(ctx, &OAuthClient{
		ID:           uuid.New(),
		ClientID:     clientID,
		Name:         name,
		RedirectURIs: redirectURIs,
		CreatedAt:    time.Now(),
	})
	if err != nil {
		return nil, fmt.Errorf("creating OAuth client: %w", err)
	}

	return client, nil
}

func validateOAuthRedirectURI(raw string) error {
	invalid := &ValidationError{Message: "redirect URI must be https, or http on localhost or 127.0.0.1: " + raw}

	if len(raw) > maxOAuthRedirectURILength {
		return invalid
	}

	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.Fragment != "" || parsed.User != nil {
		return invalid
	}

	switch parsed.Scheme {
	case "https":
		return nil
	case "http":
		if host := parsed.Hostname(); host == "localhost" || host == "127.0.0.1" {
			return nil
		}
	}

	return invalid
}
