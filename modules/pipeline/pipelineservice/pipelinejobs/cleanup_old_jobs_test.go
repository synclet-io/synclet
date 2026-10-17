package pipelinejobs

import (
	"context"
	"errors"
	"io"
	"slices"
	"testing"

	"github.com/google/uuid"
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

// Verify interface compliance.
var _ pipelineservice.JobRetentionStorage = (*mockRetentionStorage)(nil)

type retentionStore struct {
	pipelineservice.Storage
	segments *retentionSegments

	deletedJobs   [][]uuid.UUID
	deleteJobsErr error
}

func (s *retentionStore) JobLogSegments() pipelineservice.JobLogSegmentsStorage { return s.segments }

func (s *retentionStore) Jobs() pipelineservice.JobsStorage { return &retentionJobs{store: s} }

func (s *retentionStore) ExecuteInTransaction(ctx context.Context, cb func(ctx context.Context, tx pipelineservice.Storage) error) error {
	return cb(ctx, s)
}

type retentionJobs struct {
	pipelineservice.JobsStorage
	store *retentionStore
}

func (j *retentionJobs) Delete(_ context.Context, f *pipelineservice.JobFilter) error {
	if j.store.deleteJobsErr != nil {
		return j.store.deleteJobsErr
	}

	j.store.deletedJobs = append(j.store.deletedJobs, f.ID.(*filter.InFilter[uuid.UUID]).Values)

	return nil
}

type retentionSegments struct {
	pipelineservice.JobLogSegmentsStorage
	records []*pipelineservice.JobLogSegment
	locked  [][]uuid.UUID
}

func (s *retentionSegments) LockArchiveKeys(_ context.Context, ids []uuid.UUID) ([]string, error) {
	s.locked = append(s.locked, ids)

	var keys []string

	for _, segment := range s.records {
		if slices.Contains(ids, segment.JobID) && segment.ArchiveKey != nil {
			keys = append(keys, *segment.ArchiveKey)
		}
	}

	return keys, nil
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
}

func TestCleanupOldJobs_LocksJobsDeletesArchivedLogsThenJobs(t *testing.T) {
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

	require.NoError(t, useCase.ExecuteForWorkspace(context.Background(), wsID))
	storage.AssertExpectations(t)

	assert.Equal(t, [][]uuid.UUID{ids}, store.segments.locked)
	assert.ElementsMatch(t, []string{"logs/old/1", "logs/old/2"}, archive.deleted)
	assert.Equal(t, [][]uuid.UUID{ids}, store.deletedJobs)
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

	require.NoError(t, useCase.ExecuteForWorkspace(context.Background(), wsID))
	storage.AssertExpectations(t)
	assert.Equal(t, [][]uuid.UUID{full}, store.deletedJobs)
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
	assert.Empty(t, store.deletedJobs)
}

func TestCleanupOldJobs_StorageError(t *testing.T) {
	settings := new(mockSettingsProvider)
	storage := new(mockRetentionStorage)

	jobID := uuid.New()
	store := &retentionStore{segments: &retentionSegments{}, deleteJobsErr: errors.New("db error")}
	useCase := NewCleanupOldJobs(store, storage, settings, &recordingArchive{}, nil)

	wsID := uuid.New()
	settings.On("Execute", mock.Anything, wsID).Return(&pipelineservice.WorkspaceSettings{
		WorkspaceID: wsID, MaxJobsPerWorkspace: 10,
	}, nil)
	storage.On("ListExcessTerminalJobIDs", mock.Anything, wsID, 10, retentionBatchSize).Return([]uuid.UUID{jobID}, nil)

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
