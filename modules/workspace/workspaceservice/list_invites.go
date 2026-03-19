package workspaceservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ListInvites lists all invites for a workspace (D-19).
type ListInvites struct {
	storage Storage
}

// NewListInvites creates a new ListInvites use case.
func NewListInvites(storage Storage) *ListInvites {
	return &ListInvites{storage: storage}
}

// Execute returns all invites for the given workspace, all statuses.
func (uc *ListInvites) Execute(ctx context.Context, workspaceID uuid.UUID) ([]*WorkspaceInvite, error) {
	invites, err := uc.storage.WorkspaceInvites().Find(ctx, &WorkspaceInviteFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("listing invites: %w", err)
	}

	return invites, nil
}
