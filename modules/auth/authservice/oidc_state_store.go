package authservice

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// StateStore is a database-backed store for OIDC state parameters and PKCE verifiers.
type StateStore struct {
	storage Storage
}

// NewStateStore creates a new database-backed state store.
func NewStateStore(storage Storage) *StateStore {
	return &StateStore{storage: storage}
}

// generateState creates a crypto-random state string.
func generateState() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

// Set stores a PKCE verifier and provider slug keyed by state with a TTL.
func (s *StateStore) Set(ctx context.Context, state, verifier, provider string, ttl time.Duration) error {
	now := time.Now()
	_, err := s.storage.OIDCStates().Create(ctx, &OIDCState{
		ID:           uuid.New(),
		State:        state,
		Verifier:     verifier,
		ProviderSlug: provider,
		ExpiresAt:    now.Add(ttl),
		CreatedAt:    now,
	})
	if err != nil {
		return fmt.Errorf("creating OIDC state: %w", err)
	}
	return nil
}

// Get retrieves and deletes a state entry. Returns verifier, provider, and ok.
// One-time use: entry is deleted after retrieval within a transaction to prevent races.
func (s *StateStore) Get(ctx context.Context, state string) (verifier, provider string, ok bool) {
	err := s.storage.ExecuteInTransaction(ctx, func(ctx context.Context, tx Storage) error {
		entry, err := tx.OIDCStates().First(ctx, &OIDCStateFilter{
			State: filter.Equals(state),
		}, dbutil.WithForUpdate())
		if err != nil {
			return err
		}

		// Delete immediately (one-time use).
		if err := tx.OIDCStates().Delete(ctx, &OIDCStateFilter{ID: filter.Equals(entry.ID)}); err != nil {
			return err
		}

		// Check expiry.
		if time.Now().After(entry.ExpiresAt) {
			return nil
		}

		verifier = entry.Verifier
		provider = entry.ProviderSlug
		ok = true

		return nil
	})
	if err != nil {
		return "", "", false
	}

	return verifier, provider, ok
}

// CleanupExpired removes all OIDC state entries that have expired.
func (s *StateStore) CleanupExpired(ctx context.Context) error {
	entries, err := s.storage.OIDCStates().Find(ctx, &OIDCStateFilter{})
	if err != nil {
		return fmt.Errorf("listing OIDC states: %w", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if now.After(entry.ExpiresAt) {
			if err := s.storage.OIDCStates().Delete(ctx, &OIDCStateFilter{ID: filter.Equals(entry.ID)}); err != nil {
				return fmt.Errorf("deleting expired OIDC state: %w", err)
			}
		}
	}

	return nil
}
