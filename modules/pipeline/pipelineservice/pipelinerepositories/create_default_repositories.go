package pipelinerepositories

import (
	"context"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// DefaultRegistry describes a registry seeded into every new workspace.
type DefaultRegistry struct {
	Name string
	URL  string
}

// DefaultRegistries enumerates the registries auto-installed on workspace
// creation. Operators can still add or remove registries manually afterwards.
var DefaultRegistries = []DefaultRegistry{
	{
		Name: "Airbyte OSS",
		URL:  "https://connectors.airbyte.com/files/registries/v0/oss_registry.json",
	},
	{
		Name: "Synclet",
		URL:  "https://synclet-io.github.io/synclet-connector-registry/registry.json",
	},
}

// CreateDefaultRepositories seeds a new workspace with the well-known connector
// registries. Each registry is added through AddRepository so its catalog is
// fetched and validated; failures on a single registry are logged and do not
// block the rest (a transient network error must not leave the workspace half
// initialised).
type CreateDefaultRepositories struct {
	storage pipelineservice.Storage
	addRepo *AddRepository
	logger  *logging.Logger
}

// NewCreateDefaultRepositories creates a new CreateDefaultRepositories use case.
func NewCreateDefaultRepositories(
	storage pipelineservice.Storage,
	addRepo *AddRepository,
	logger *logging.Logger,
) *CreateDefaultRepositories {
	return &CreateDefaultRepositories{
		storage: storage,
		addRepo: addRepo,
		logger:  logger.Named("create-default-repositories"),
	}
}

// Execute seeds the given workspace with every DefaultRegistries entry that is
// not already present. Idempotent: re-running after a partial seed only adds
// the missing rows.
func (uc *CreateDefaultRepositories) Execute(ctx context.Context, workspaceID uuid.UUID) error {
	existing, err := uc.storage.Repositories().Find(ctx, &pipelineservice.RepositoryFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		return fmt.Errorf("listing existing repositories: %w", err)
	}

	existingURLs := make(map[string]struct{}, len(existing))
	for _, repo := range existing {
		existingURLs[repo.URL] = struct{}{}
	}

	for _, reg := range DefaultRegistries {
		if _, ok := existingURLs[reg.URL]; ok {
			continue
		}

		_, err := uc.addRepo.Execute(ctx, AddRepositoryParams{
			WorkspaceID: workspaceID,
			Name:        reg.Name,
			URL:         reg.URL,
		})
		if err != nil {
			uc.logger.
				WithError(err).
				WithField("workspace_id", workspaceID.String()).
				WithField("registry_name", reg.Name).
				WithField("registry_url", reg.URL).
				Warn(ctx, "failed to seed default registry; continuing with the rest")

			continue
		}
	}

	return nil
}
