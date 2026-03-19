package pipelinerepositories

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// ListRepositoryConnectors lists connectors for a specific repository with optional filters.
type ListRepositoryConnectors struct {
	storage pipelineservice.Storage
}

// NewListRepositoryConnectors creates a new ListRepositoryConnectors use case.
func NewListRepositoryConnectors(storage pipelineservice.Storage) *ListRepositoryConnectors {
	return &ListRepositoryConnectors{storage: storage}
}

// ListRepositoryConnectorsParams holds parameters for listing repository connectors.
type ListRepositoryConnectorsParams struct {
	RepositoryID  uuid.UUID
	WorkspaceID   uuid.UUID
	ConnectorType string // optional: "source" or "destination"
	Search        string // optional: case-insensitive name/docker_repository search
	SupportLevel  string // optional: "community" or "certified"
	License       string // optional: "ELv2" or "MIT"
	SourceType    string // optional: "api", "database", or "file"
}

// Execute returns connectors for the given repository, optionally filtered by type and search term.
// Verifies that the repository belongs to the specified workspace before listing connectors.
func (uc *ListRepositoryConnectors) Execute(ctx context.Context, params ListRepositoryConnectorsParams) ([]*pipelineservice.RepositoryConnector, error) {
	// Verify repository belongs to the workspace.
	if params.WorkspaceID != (uuid.UUID{}) {
		_, err := uc.storage.Repositorys().First(ctx, &pipelineservice.RepositoryFilter{
			ID:          filter.Equals(params.RepositoryID),
			WorkspaceID: filter.Equals(params.WorkspaceID),
		})
		if err != nil {
			return nil, fmt.Errorf("verifying repository ownership: %w", err)
		}
	}

	f := &pipelineservice.RepositoryConnectorFilter{
		RepositoryID: filter.Equals(params.RepositoryID),
	}

	if params.ConnectorType != "" {
		f.ConnectorType = filter.Equals(parseConnectorType(params.ConnectorType))
	}
	if params.SupportLevel != "" {
		f.SupportLevel = filter.Equals(parseSupportLevel(params.SupportLevel))
	}
	if params.License != "" {
		f.License = filter.Equals(params.License)
	}
	if params.SourceType != "" {
		f.SourceType = filter.Equals(parseSourceType(params.SourceType))
	}

	connectors, err := uc.storage.RepositoryConnectors().Find(ctx, f)
	if err != nil {
		return nil, fmt.Errorf("listing repository connectors: %w", err)
	}

	// Apply search filter in-memory (case-insensitive name/docker_repository match).
	if params.Search != "" {
		search := strings.ToLower(params.Search)
		var filtered []*pipelineservice.RepositoryConnector
		for _, c := range connectors {
			if strings.Contains(strings.ToLower(c.Name), search) ||
				strings.Contains(strings.ToLower(c.DockerRepository), search) {
				filtered = append(filtered, c)
			}
		}
		return filtered, nil
	}

	return connectors, nil
}
