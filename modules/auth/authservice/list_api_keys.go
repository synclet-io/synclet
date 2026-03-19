package authservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ListAPIKeys returns all API keys for a workspace.
type ListAPIKeys struct {
	storage Storage
}

// NewListAPIKeys creates a new ListAPIKeys use case.
func NewListAPIKeys(storage Storage) *ListAPIKeys {
	return &ListAPIKeys{storage: storage}
}

// Execute returns all API keys belonging to the given workspace.
func (uc *ListAPIKeys) Execute(ctx context.Context, workspaceID uuid.UUID) ([]*APIKey, error) {
	return uc.storage.APIKeys().Find(ctx, &APIKeyFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
}
