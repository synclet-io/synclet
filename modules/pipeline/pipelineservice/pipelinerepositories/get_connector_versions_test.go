package pipelinerepositories

import (
	"context"
	"slices"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type versionsStorage struct {
	pipelineservice.Storage

	repos      []*pipelineservice.Repository
	connectors []*pipelineservice.RepositoryConnector
}

func (s *versionsStorage) Repositories() pipelineservice.RepositoriesStorage {
	return &versionsRepositories{parent: s}
}

func (s *versionsStorage) RepositoryConnectors() pipelineservice.RepositoryConnectorsStorage {
	return &versionsConnectors{parent: s}
}

type versionsRepositories struct {
	pipelineservice.RepositoriesStorage

	parent *versionsStorage
}

func (r *versionsRepositories) Find(_ context.Context, flt *pipelineservice.RepositoryFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Repository, error) {
	workspaceID := flt.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]).Value

	var out []*pipelineservice.Repository

	for _, repo := range r.parent.repos {
		if repo.WorkspaceID == workspaceID {
			out = append(out, repo)
		}
	}

	return out, nil
}

type versionsConnectors struct {
	pipelineservice.RepositoryConnectorsStorage

	parent *versionsStorage
}

func (c *versionsConnectors) Find(_ context.Context, flt *pipelineservice.RepositoryConnectorFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.RepositoryConnector, error) {
	repoIDs := flt.RepositoryID.(*filter.InFilter[uuid.UUID]).Values
	image := flt.DockerRepository.(*filter.EqualsFilter[string]).Value

	var out []*pipelineservice.RepositoryConnector

	for _, conn := range c.parent.connectors {
		if slices.Contains(repoIDs, conn.RepositoryID) && conn.DockerRepository == image {
			out = append(out, conn)
		}
	}

	return out, nil
}

func TestGetConnectorVersions_ScopedToWorkspaceRepositories(t *testing.T) {
	workspaceID := uuid.New()
	ownRepo := &pipelineservice.Repository{ID: uuid.New(), WorkspaceID: workspaceID}
	otherRepo := &pipelineservice.Repository{ID: uuid.New(), WorkspaceID: uuid.New()}
	storage := &versionsStorage{
		repos: []*pipelineservice.Repository{ownRepo, otherRepo},
		connectors: []*pipelineservice.RepositoryConnector{
			{RepositoryID: otherRepo.ID, DockerRepository: "airbyte/source-pg", DockerImageTag: "9.9.9"},
			{RepositoryID: ownRepo.ID, DockerRepository: "airbyte/source-pg", DockerImageTag: "1.2.3"},
		},
	}
	uc := NewGetConnectorVersions(storage)

	got, err := uc.Execute(context.Background(), GetConnectorVersionsParams{WorkspaceID: workspaceID, ConnectorImage: "airbyte/source-pg"})
	require.NoError(t, err)
	assert.Equal(t, "1.2.3", got.LatestVersion)

	got, err = uc.Execute(context.Background(), GetConnectorVersionsParams{WorkspaceID: uuid.New(), ConnectorImage: "airbyte/source-pg"})
	require.NoError(t, err)
	assert.Empty(t, got.LatestVersion)
}
