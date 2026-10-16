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
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

type fakeSyncStorage struct {
	pipelineservice.Storage

	repo *pipelineservice.Repository
	inTx bool
}

func (s *fakeSyncStorage) Repositories() pipelineservice.RepositoriesStorage {
	return &fakeSyncRepositories{parent: s}
}

func (s *fakeSyncStorage) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	snapshot := *s.repo
	s.inTx = true

	err := cb(ctx, s)

	s.inTx = false
	if err != nil {
		s.repo = &snapshot
	}

	return err
}

type fakeSyncRepositories struct {
	pipelineservice.RepositoriesStorage

	parent *fakeSyncStorage
}

func (r *fakeSyncRepositories) First(_ context.Context, _ *pipelineservice.RepositoryFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Repository, error) {
	repo := *r.parent.repo

	return &repo, nil
}

func (r *fakeSyncRepositories) Update(_ context.Context, repo *pipelineservice.Repository) (*pipelineservice.Repository, error) {
	stored := *repo
	r.parent.repo = &stored

	return repo, nil
}

func TestSyncRepository_FetchFailureIsPersistedOutsideTx(t *testing.T) {
	storage := &fakeSyncStorage{}

	var fetchedInTx bool

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		fetchedInTx = storage.inTx

		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	storage.repo = &pipelineservice.Repository{
		ID:             uuid.New(),
		WorkspaceID:    uuid.New(),
		URL:            server.URL,
		Status:         pipelineservice.RepositoryStatusSynced,
		ConnectorCount: 3,
	}

	uc := NewSyncRepository(
		storage,
		NewRegistryFetcher(pipelineservice.Config{RegistryFetchTimeout: 5 * time.Second}),
		nil,
		nil,
		pipelineservice.NoopAuditRecorder{},
	)

	_, err := uc.Execute(context.Background(), SyncRepositoryParams{
		RepositoryID: storage.repo.ID,
		WorkspaceID:  storage.repo.WorkspaceID,
	})

	require.ErrorContains(t, err, "fetching registry")
	assert.False(t, fetchedInTx, "registry must be fetched outside the transaction")
	assert.Equal(t, pipelineservice.RepositoryStatusFailed, storage.repo.Status)
	require.NotNil(t, storage.repo.LastError)
	assert.Contains(t, *storage.repo.LastError, "500")
	assert.Equal(t, 3, storage.repo.ConnectorCount)
}
