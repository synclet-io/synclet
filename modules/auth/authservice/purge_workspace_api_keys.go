package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// PurgeWorkspaceAPIKeys deletes every API key issued for a deleted workspace.
type PurgeWorkspaceAPIKeys struct {
	storage Storage
}

// NewPurgeWorkspaceAPIKeys creates a new PurgeWorkspaceAPIKeys use case.
func NewPurgeWorkspaceAPIKeys(storage Storage) *PurgeWorkspaceAPIKeys {
	return &PurgeWorkspaceAPIKeys{storage: storage}
}

// Execute deletes the workspace's API keys. Running it again is a no-op.
func (uc *PurgeWorkspaceAPIKeys) Execute(ctx context.Context, workspaceID uuid.UUID) error {
	if err := uc.storage.APIKeys().Delete(ctx, &APIKeyFilter{
		WorkspaceID: filter.Equals(workspaceID),
	}); err != nil {
		return fmt.Errorf("deleting API keys: %w", err)
	}

	return nil
}
