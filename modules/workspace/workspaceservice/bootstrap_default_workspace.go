package workspaceservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// BootstrapDefaultWorkspace creates the default workspace if none exists.
// Idempotent: no-op when a workspace with slug "default" already exists.
type BootstrapDefaultWorkspace struct {
	storage Storage
}

// NewBootstrapDefaultWorkspace creates a new BootstrapDefaultWorkspace use case.
func NewBootstrapDefaultWorkspace(storage Storage) *BootstrapDefaultWorkspace {
	return &BootstrapDefaultWorkspace{storage: storage}
}

// Execute creates the "Default" workspace with slug "default" if it does not
// exist, and publishes a workspace.created event so subscribers (e.g. the
// default-registries seeder) treat it like any other newly created workspace.
func (uc *BootstrapDefaultWorkspace) Execute(ctx context.Context) (*Workspace, error) {
	existing, err := uc.storage.Workspaces().First(ctx, &WorkspaceFilter{
		Slug: filter.Equals("default"),
	})
	if err != nil {
		var notFound NotFoundError
		if !errors.As(err, &notFound) {
			return nil, fmt.Errorf("checking for default workspace: %w", err)
		}
		// Not found -- fall through to create.
	} else {
		return existing, nil // Already bootstrapped.
	}

	now := time.Now()
	workspace := &Workspace{
		ID:        uuid.New(),
		Name:      "Default",
		Slug:      "default",
		CreatedAt: now,
		UpdatedAt: now,
	}

	created, err := uc.storage.Workspaces().Create(ctx, workspace)
	if err != nil {
		return nil, fmt.Errorf("creating default workspace: %w", err)
	}

	if err := uc.storage.WorkspaceEvents().Send(ctx, &WorkspaceEvent{
		EventType: WorkspaceEventTypeCreated,
		Data:      &WorkspaceCreatedEventData{Workspace: created},
	}); err != nil {
		return nil, fmt.Errorf("publishing workspace.created event: %w", err)
	}

	return created, nil
}
