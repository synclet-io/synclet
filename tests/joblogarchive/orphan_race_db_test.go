package joblogarchive_test

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineworkspaces"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/pkg/logarchive"
)

// archiverHeadStart is how long a deleter waits inside its archive delete for
// a racing archiver before it carries on with deleting the jobs.
const archiverHeadStart = 500 * time.Millisecond

// hookArchive runs onDelete when a deleter deletes archive objects, which is
// after it has chosen the objects to delete and before it deletes the jobs.
type hookArchive struct {
	*logarchive.Filesystem
	onDelete func()
}

func (a *hookArchive) Delete(ctx context.Context, keys ...string) error {
	if a.onDelete != nil {
		a.onDelete()
		a.onDelete = nil
	}

	return a.Filesystem.Delete(ctx, keys...)
}

type keepJobs int

func (k keepJobs) Execute(_ context.Context, workspaceID uuid.UUID) (*pipelineservice.WorkspaceSettings, error) {
	return &pipelineservice.WorkspaceSettings{WorkspaceID: workspaceID, MaxJobsPerWorkspace: int(k)}, nil
}

type noSecrets struct {
	pipelineservice.SecretsProvider
}

func (noSecrets) DeleteByOwner(context.Context, string, uuid.UUID) error { return nil }

type noStats struct{ pipelineservice.StatsStorage }

func (noStats) DeleteWorkspaceRollups(context.Context, uuid.UUID) error { return nil }

func (f *fixture) archiver() *pipelinelogs.ArchiveJobLogs {
	return pipelinelogs.NewArchiveJobLogs(f.storage, f.archive, pipelineservice.Config{LogArchiveEnabled: true}, pipelinemetrics.NewMetricsCollector())
}

// raceArchiver runs deleteJobs with an archive whose delete starts archiving
// jobID and gives it archiverHeadStart to finish. It returns the archiver's
// error once both are done.
func (f *fixture) raceArchiver(t *testing.T, jobID uuid.UUID, deleteJobs func(pipelineservice.LogArchive) error, alongside ...func()) error {
	t.Helper()

	var (
		done     = make(chan error, 1)
		started  bool
		archive  = &hookArchive{Filesystem: f.archive}
		finished sync.WaitGroup
	)

	archive.onDelete = func() {
		started = true

		go func() { done <- f.archiver().Execute(context.Background(), jobID) }()

		for _, run := range alongside {
			finished.Go(run)
		}

		select {
		case err := <-done:
			done <- err
		case <-time.After(archiverHeadStart):
		}
	}

	require.NoError(t, deleteJobs(archive))
	require.True(t, started, "the deleter never deleted archive objects")

	finished.Wait()

	return <-done
}

func (f *fixture) objects(t *testing.T) []string {
	t.Helper()

	var keys []string

	require.NoError(t, filepath.WalkDir(f.archiveDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}

		rel, err := filepath.Rel(f.archiveDir, path)
		keys = append(keys, filepath.ToSlash(rel))

		return err
	}))

	return keys
}

// retention keeps the newest job of the workspace.
func (f *fixture) retention(archive pipelineservice.LogArchive) *pipelinejobs.CleanupOldJobs {
	return pipelinejobs.NewCleanupOldJobs(f.storage, pipelinestorage.NewJobRetentionStorage(f.db), keepJobs(1), archive, nil)
}

func TestJobLogArchive_RetentionRacingArchiverLeavesNoObject(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	old := f.newJob(t, "completed")
	f.appendLines(t, 0, old, "x")
	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET created_at = NOW() - INTERVAL '1 hour' WHERE id = ?`, old).Error)
	f.newJob(t, "completed")

	_ = f.raceArchiver(t, old, func(archive pipelineservice.LogArchive) error {
		return f.retention(archive).ExecuteForWorkspace(ctx, f.workspaceID)
	})

	assert.Empty(t, f.objects(t), "no archive object outlives its job")
}

func TestJobLogArchive_PurgeRacingArchiverLeavesNoObject(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	jobID := f.newJob(t, "completed")
	f.appendLines(t, 0, jobID, "x")

	_ = f.raceArchiver(t, jobID, func(archive pipelineservice.LogArchive) error {
		return pipelineworkspaces.NewPurgeWorkspace(f.storage, noStats{}, noSecrets{}, archive).Execute(ctx, f.workspaceID)
	})

	assert.Empty(t, f.objects(t), "no archive object outlives its job")
}

func TestJobLogArchive_OrphanCleanupIsIdempotentAndSparesLiveJobs(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	old := f.newJob(t, "completed")
	f.appendLines(t, 0, old, "old")
	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET created_at = NOW() - INTERVAL '1 hour' WHERE id = ?`, old).Error)

	live := f.newJob(t, "completed")
	f.appendLines(t, 0, live, "live")

	archiveLive := func() { assert.NoError(t, f.archiver().Execute(ctx, live)) }

	_ = f.raceArchiver(t, old, func(archive pipelineservice.LogArchive) error {
		return f.retention(archive).ExecuteForWorkspace(ctx, f.workspaceID)
	}, archiveLive, archiveLive)

	for range 2 {
		require.NoError(t, f.archiver().Execute(ctx, old), "a redelivered archive event of a deleted job")
		archiveLive()
	}

	liveKey := pipelineservice.LogArchiveKey(f.workspaceID, f.connID, live, 1)
	assert.Equal(t, []string{liveKey}, f.objects(t))
	assert.Zero(t, f.countRows(t, live))
	assert.Equal(t, []string{"live"}, f.readAll(t, live, 10))
}

var errStoreDown = errors.New("object store down")

type failingDeletes struct{ *logarchive.Filesystem }

func (failingDeletes) Delete(context.Context, ...string) error { return errStoreDown }

// putHook runs afterPut once, after the first object is written.
type putHook struct {
	pipelineservice.LogArchive
	afterPut func()
}

func (a *putHook) Put(ctx context.Context, key string, body io.Reader) error {
	if err := a.LogArchive.Put(ctx, key, body); err != nil {
		return err
	}

	if a.afterPut != nil {
		a.afterPut()
		a.afterPut = nil
	}

	return nil
}

func TestJobLogArchive_JobDeletedAfterPutLeavesNoObjectWhenDeletesFail(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	old := f.newJob(t, "completed")
	f.appendLines(t, 0, old, "x")
	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET created_at = NOW() - INTERVAL '1 hour' WHERE id = ?`, old).Error)
	f.newJob(t, "completed")

	retentionDone := make(chan error, 1)
	archive := &putHook{LogArchive: failingDeletes{f.archive}, afterPut: func() {
		go func() { retentionDone <- f.retention(f.archive).ExecuteForWorkspace(ctx, f.workspaceID) }()

		select {
		case err := <-retentionDone:
			retentionDone <- err
		case <-time.After(archiverHeadStart):
		}
	}}
	archiver := pipelinelogs.NewArchiveJobLogs(f.storage, archive, pipelineservice.Config{LogArchiveEnabled: true}, pipelinemetrics.NewMetricsCollector())

	_ = archiver.Execute(ctx, old)

	require.NoError(t, <-retentionDone)
	require.NoError(t, archiver.Execute(ctx, old), "a redelivered archive event of the deleted job")

	assert.Empty(t, f.objects(t), "no archive object outlives its job")
}

func TestJobLogArchive_RetentionDeletesObjectOfUnmarkedAttempt(t *testing.T) {
	f := setup(t)
	ctx := context.Background()

	old := f.newJob(t, "completed")
	f.appendLines(t, 0, old, "x")
	require.NoError(t, f.db.Exec(`UPDATE pipeline.jobs SET created_at = NOW() - INTERVAL '1 hour' WHERE id = ?`, old).Error)
	f.newJob(t, "completed")

	require.NoError(t, f.archive.Put(ctx, pipelineservice.LogArchiveKey(f.workspaceID, f.connID, old, 1), strings.NewReader("written, never marked")))

	require.NoError(t, f.retention(f.archive).ExecuteForWorkspace(ctx, f.workspaceID))

	assert.Empty(t, f.objects(t), "no archive object outlives its job")
}
