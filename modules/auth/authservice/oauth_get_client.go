package authservice

import (
	"context"
	"errors"
	"fmt"
	"slices"

	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// GetOAuthClient returns a registered OAuth client for the consent page.
type GetOAuthClient struct {
	storage Storage
}

// NewGetOAuthClient creates a new GetOAuthClient use case.
func NewGetOAuthClient(storage Storage) *GetOAuthClient {
	return &GetOAuthClient{storage: storage}
}

// Execute returns the client with clientID, or ErrOAuthClientNotFound, and a
// ValidationError when redirectURI is not one of its registered redirect URIs.
func (uc *GetOAuthClient) Execute(ctx context.Context, clientID, redirectURI string) (*OAuthClient, error) {
	return getOAuthClient(ctx, uc.storage, clientID, redirectURI)
}

func getOAuthClient(ctx context.Context, storage Storage, clientID, redirectURI string) (*OAuthClient, error) {
	client, err := storage.OAuthClients().First(ctx, &OAuthClientFilter{ClientID: filter.Equals(clientID)})
	if errors.Is(err, ErrOAuthClientNotFound) {
		return nil, ErrOAuthClientNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("getting OAuth client: %w", err)
	}

	if !slices.Contains(client.RedirectURIs, redirectURI) {
		return nil, &ValidationError{Message: "redirect URI is not registered for this client"}
	}

	return client, nil
}
