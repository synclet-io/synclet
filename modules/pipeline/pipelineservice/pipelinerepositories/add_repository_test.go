package pipelinerepositories

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type addRepoStorage struct {
	pipelineservice.Storage

	repos   map[uuid.UUID]*pipelineservice.Repository
	created int
}

func (s *addRepoStorage) Repositories() pipelineservice.RepositoriesStorage {
	return &addRepoRepositories{parent: s}
}

func (s *addRepoStorage) RepositoryConnectors() pipelineservice.RepositoryConnectorsStorage {
	return addRepoConnectors{}
}

func (s *addRepoStorage) ManagedConnectors() pipelineservice.ManagedConnectorsStorage {
	return addRepoManagedConnectors{}
}

func (s *addRepoStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

type addRepoRepositories struct {
	pipelineservice.RepositoriesStorage

	parent *addRepoStorage
}

func (r *addRepoRepositories) Create(_ context.Context, repo *pipelineservice.Repository) (*pipelineservice.Repository, error) {
	stored := *repo
	r.parent.repos[repo.ID] = &stored
	r.parent.created++

	return repo, nil
}

func (r *addRepoRepositories) First(_ context.Context, f *pipelineservice.RepositoryFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Repository, error) {
	for _, repo := range r.parent.repos {
		if filterID(f) == repo.ID {
			stored := *repo

			return &stored, nil
		}
	}

	return nil, pipelineservice.ErrRepositoryNotFound
}

func (r *addRepoRepositories) Update(_ context.Context, repo *pipelineservice.Repository) (*pipelineservice.Repository, error) {
	stored := *repo
	r.parent.repos[repo.ID] = &stored

	return repo, nil
}

func (r *addRepoRepositories) Delete(_ context.Context, f *pipelineservice.RepositoryFilter) error {
	for id := range r.parent.repos {
		if filterID(f) == id {
			delete(r.parent.repos, id)
		}
	}

	return nil
}

func filterID(f *pipelineservice.RepositoryFilter) uuid.UUID {
	if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok {
		return eq.Value
	}

	return uuid.Nil
}

type addRepoConnectors struct {
	pipelineservice.RepositoryConnectorsStorage
}

func (addRepoConnectors) Find(context.Context, *pipelineservice.RepositoryConnectorFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.RepositoryConnector, error) {
	return nil, nil
}

type addRepoManagedConnectors struct {
	pipelineservice.ManagedConnectorsStorage
}

func (addRepoManagedConnectors) Find(context.Context, *pipelineservice.ManagedConnectorFilter, ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.ManagedConnector, error) {
	return nil, nil
}

type addRepoSecrets struct {
	pipelineservice.SecretsProvider
}

func (addRepoSecrets) DeleteByOwner(context.Context, string, uuid.UUID) error {
	return nil
}

func newAddRepositoryForTest(storage *addRepoStorage, allowPrivate bool) *AddRepository {
	cfg := pipelineservice.Config{RegistryFetchTimeout: 5 * time.Second, RegistryAllowPrivateAddresses: allowPrivate}
	syncRepo := NewSyncRepository(storage, NewRegistryFetcher(cfg), nil, nil, pipelineservice.NoopAuditRecorder{})

	return NewAddRepository(storage, syncRepo, addRepoSecrets{}, pipelineservice.NoopAuditRecorder{}, cfg)
}

func TestAddRepository_RejectsPrivateAddresses(t *testing.T) {
	for _, url := range []string{"http://localhost/", "http://169.254.169.254/"} {
		t.Run(url, func(t *testing.T) {
			storage := &addRepoStorage{repos: map[uuid.UUID]*pipelineservice.Repository{}}

			_, err := newAddRepositoryForTest(storage, false).Execute(context.Background(), AddRepositoryParams{
				WorkspaceID: uuid.New(),
				Name:        "internal",
				URL:         url,
			})

			var validationErr *pipelineservice.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Equal(t, "url", validationErr.Field)
			assert.Zero(t, storage.created, "no repository is stored")
		})
	}
}

func TestAddRepository_AllowsPrivateAddressesWhenEnabled(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"sources": [], "destinations": []}`))
	}))
	defer server.Close()

	storage := &addRepoStorage{repos: map[uuid.UUID]*pipelineservice.Repository{}}

	repo, err := newAddRepositoryForTest(storage, true).Execute(context.Background(), AddRepositoryParams{
		WorkspaceID: uuid.New(),
		Name:        "lan",
		URL:         server.URL,
	})
	require.NoError(t, err)
	assert.Equal(t, pipelineservice.RepositoryStatusSynced, repo.Status)
	assert.NotNil(t, repo.LastSyncedAt)
	assert.Len(t, storage.repos, 1)
}
