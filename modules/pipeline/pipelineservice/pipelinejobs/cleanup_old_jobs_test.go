package pipelinejobs

import (
	"context"
	"errors"
	"io"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// mockSettingsProvider implements SettingsProvider (backed by *pipelinesettings.GetWorkspaceSettings).
type mockSettingsProvider struct {
	mock.Mock
}

func (m *mockSettingsProvider) Execute(ctx context.Context, workspaceID uuid.UUID) (*pipelineservice.WorkspaceSettings, error) {
	args := m.Called(ctx, workspaceID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}

	return args.Get(0).(*pipelineservice.WorkspaceSettings), args.Error(1)
}

// Verify interface compliance.
var _ SettingsProvider = (*mockSettingsProvider)(nil)

// mockRetentionStorage implements pipelineservice.JobRetentionStorage.
type mockRetentionStorage struct {
	mock.Mock
}

func (m *mockRetentionStorage) ListExcessTerminalJobIDs(ctx context.Context, workspaceID uuid.UUID, keepCount, limit int) ([]uuid.UUID, error) {
	args := m.Called(ctx, workspaceID, keepCount, limit)

	ids, _ := args.Get(0).([]uuid.UUID)

	return ids, args.Error(1)
}

func (m *mockRetentionStorage) DeleteJobs(ctx context.Context, ids []uuid.UUID) (int64, error) {
	args := m.Called(ctx, ids)

	return args.Get(0).(int64), args.Error(1)
}

// Verify interface compliance.
var _ pipelineservice.JobRetentionStorage = (*mockRetentionStorage)(nil)

type retentionStore struct {
	pipelineservice.Storage
	segments *retentionSegments
}

func (s *retentionStore) JobLogSegments() pipelineservice.JobLogSegmentsStorage { return s.segments }

type retentionSegments struct {
	pipelineservice.JobLogSegmentsStorage
	records []*pipelineservice.JobLogSegment
}

func (s *retentionSegments) Find(_ context.Context, f *pipelineservice.JobLogSegmentFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.JobLogSegment, error) {
	ids := f.JobID.(*filter.InFilter[uuid.UUID]).Values

	var out []*pipelineservice.JobLogSegment

	for _, segment := range s.records {
		for _, id := range ids {
			if segment.JobID == id && segment.Status == pipelineservice.JobLogSegmentStatusArchived {
				out = append(out, segment)
			}
		}
	}

	return out, nil
}

type recordingArchive struct {
	deleted []string
	err     error
}

func (a *recordingArchive) Put(context.Context, string, io.Reader) error { return nil }

func (a *recordingArchive) Open(context.Context, string) (io.ReadCloser, error) { return nil, nil }

func (a *recordingArchive) Delete(_ context.Context, keys ...string) error {
	if a.err != nil {
		return a.err
	}

	a.deleted = append(a.deleted, keys...)

	return nil
}

func archivedSegment(jobID uuid.UUID, attempt int, key string) *pipelineservice.JobLogSegment {
	return &pipelineservice.JobLogSegment{
		JobID:      jobID,
		Attempt:    attempt,
		Status:     pipelineservice.JobLogSegmentStatusArchived,
		ArchiveKey: &key,
	}
}

func TestCleanupOldJobs_Unlimited(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	useCase := NewCleanupOldJobs(nil, storage, settings, &recordingArchive{}, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 0,
	}, nil)

	err := useCase.ExecuteForWorkspace(context.Background(), wsID)
	require.NoError(t, err)

	storage.AssertNotCalled(t, "ListExcessTerminalJobIDs")
	storage.AssertNotCalled(t, "DeleteJobs")
}

func TestCleanupOldJobs_DeletesArchivedLogsThenJobs(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)
	archive := &recordingArchive{}

	oldJob, otherJob, keptJob := uuid.New(), uuid.New(), uuid.New()
	store := &retentionStore{segments: &retentionSegments{records: []*pipelineservice.JobLogSegment{
		archivedSegment(oldJob, 1, "logs/old/1"),
		archivedSegment(oldJob, 2, "logs/old/2"),
		archivedSegment(keptJob, 1, "logs/kept/1"),
		{JobID: otherJob, Attempt: 1, Status: pipelineservice.JobLogSegmentStatusOpen},
	}}}

	useCase := NewCleanupOldJobs(store, storage, settings, archive, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 50,
	}, nil)

	ids := []uuid.UUID{oldJob, otherJob}
	storage.On("ListExcessTerminalJobIDs", mock.Anything, wsID, 50, retentionBatchSize).Return(ids, nil).Once()
	storage.On("DeleteJobs", mock.Anything, ids).Return(int64(2), nil).Run(func(mock.Arguments) {
		assert.ElementsMatch(t, []string{"logs/old/1", "logs/old/2"}, archive.deleted, "objects must be deleted before jobs")
	})

	require.NoError(t, useCase.ExecuteForWorkspace(context.Background(), wsID))
	storage.AssertExpectations(t)
}

func TestCleanupOldJobs_LoopsUntilBatchIsShort(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)
	store := &retentionStore{segments: &retentionSegments{}}

	useCase := NewCleanupOldJobs(store, storage, settings, &recordingArchive{}, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 5,
	}, nil)

	full := make([]uuid.UUID, retentionBatchSize)
	for i := range full {
		full[i] = uuid.New()
	}

	storage.On("ListExcessTerminalJobIDs", mock.Anything, wsID, 5, retentionBatchSize).Return(full, nil).Once()
	storage.On("ListExcessTerminalJobIDs", mock.Anything, wsID, 5, retentionBatchSize).Return([]uuid.UUID(nil), nil).Once()
	storage.On("DeleteJobs", mock.Anything, full).Return(int64(len(full)), nil).Once()

	require.NoError(t, useCase.ExecuteForWorkspace(context.Background(), wsID))
	storage.AssertExpectations(t)
}

func TestCleanupOldJobs_ArchiveDeleteFailureKeepsJobs(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	jobID := uuid.New()
	store := &retentionStore{segments: &retentionSegments{records: []*pipelineservice.JobLogSegment{
		archivedSegment(jobID, 1, "logs/job/1"),
	}}}

	useCase := NewCleanupOldJobs(store, storage, settings, &recordingArchive{err: errors.New("bucket unavailable")}, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 1,
	}, nil)
	storage.On("ListExcessTerminalJobIDs", mock.Anything, wsID, 1, retentionBatchSize).Return([]uuid.UUID{jobID}, nil)

	err := useCase.ExecuteForWorkspace(context.Background(), wsID)
	require.ErrorContains(t, err, "deleting archived job logs")
	storage.AssertNotCalled(t, "DeleteJobs", mock.Anything, mock.Anything)
}

func TestCleanupOldJobs_StorageError(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	jobID := uuid.New()
	store := &retentionStore{segments: &retentionSegments{}}
	useCase := NewCleanupOldJobs(store, storage, settings, &recordingArchive{}, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 10,
	}, nil)
	storage.On("ListExcessTerminalJobIDs", mock.Anything, wsID, 10, retentionBatchSize).Return([]uuid.UUID{jobID}, nil)
	storage.On("DeleteJobs", mock.Anything, []uuid.UUID{jobID}).Return(int64(0), errors.New("db error"))

	err := useCase.ExecuteForWorkspace(context.Background(), wsID)
	require.ErrorContains(t, err, "deleting old jobs")
}

func TestCleanupOldJobs_SettingsError(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	useCase := NewCleanupOldJobs(nil, storage, settings, &recordingArchive{}, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(nil, errors.New("workspace not found"))

	err := useCase.ExecuteForWorkspace(context.Background(), wsID)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "getting workspace settings")
}
