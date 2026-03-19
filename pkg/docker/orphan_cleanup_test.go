package docker

import (
	"context"
	"testing"
	"time"

	"github.com/docker/docker/api/types/container"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockRunner implements orphanCleanerRunner for testing.
type mockRunner struct {
	containers []container.Summary
	stopped    []string
	removed    []string
}

func (m *mockRunner) ListByLabel(_ context.Context, _ string) ([]container.Summary, error) {
	return m.containers, nil
}

func (m *mockRunner) Stop(_ context.Context, containerID string) error {
	m.stopped = append(m.stopped, containerID)
	return nil
}

func (m *mockRunner) Remove(_ context.Context, containerID string) error {
	m.removed = append(m.removed, containerID)
	return nil
}

// mockJobChecker implements OrphanJobChecker for testing.
type mockJobChecker struct {
	activeJobs map[string]bool
}

func (m *mockJobChecker) IsJobActive(_ context.Context, jobID string) (bool, error) {
	return m.activeJobs[jobID], nil
}

// newTestOrphanCleaner creates an OrphanCleaner with mock dependencies for testing.
// Uses nil logger which is nil-safe in go-pnp logging package.
func newTestOrphanCleaner(runner *mockRunner, checker OrphanJobChecker) *OrphanCleaner {
	var logger *logging.Logger // nil is safe -- all methods are nil-receiver safe
	return &OrphanCleaner{
		runner:      runner,
		checker:     checker,
		gracePeriod: 15 * time.Minute,
		logger:      logger.Named("test"),
	}
}

func TestCleanupAll_RemovesAllOrphans(t *testing.T) {
	runner := &mockRunner{
		containers: []container.Summary{
			{
				ID:      "container1-abcdef",
				Created: time.Now().Add(-1 * time.Minute).Unix(),
				Labels:  map[string]string{"synclet.io/managed": "true", "synclet.io/job-id": "job-1"},
			},
			{
				ID:      "container2-ghijkl",
				Created: time.Now().Add(-30 * time.Minute).Unix(),
				Labels:  map[string]string{"synclet.io/managed": "true", "synclet.io/job-id": "job-2"},
			},
		},
	}
	checker := &mockJobChecker{activeJobs: map[string]bool{"job-1": false, "job-2": false}}
	cleaner := newTestOrphanCleaner(runner, checker)

	err := cleaner.CleanupAll(context.Background())
	require.NoError(t, err)

	// Both containers should be stopped and removed regardless of age.
	assert.Len(t, runner.stopped, 2)
	assert.Len(t, runner.removed, 2)
	assert.Contains(t, runner.removed, "container1-abcdef")
	assert.Contains(t, runner.removed, "container2-ghijkl")
}

func TestCleanupAll_SkipsActiveJobs(t *testing.T) {
	runner := &mockRunner{
		containers: []container.Summary{
			{
				ID:      "container1-abcdef",
				Created: time.Now().Add(-5 * time.Minute).Unix(),
				Labels:  map[string]string{"synclet.io/managed": "true", "synclet.io/job-id": "job-active"},
			},
			{
				ID:      "container2-ghijkl",
				Created: time.Now().Add(-5 * time.Minute).Unix(),
				Labels:  map[string]string{"synclet.io/managed": "true", "synclet.io/job-id": "job-inactive"},
			},
		},
	}
	checker := &mockJobChecker{activeJobs: map[string]bool{"job-active": true, "job-inactive": false}}
	cleaner := newTestOrphanCleaner(runner, checker)

	err := cleaner.CleanupAll(context.Background())
	require.NoError(t, err)

	// Only the inactive job's container should be removed.
	assert.Len(t, runner.stopped, 1)
	assert.Len(t, runner.removed, 1)
	assert.Equal(t, "container2-ghijkl", runner.removed[0])
}

func TestCleanupAll_NoContainers(t *testing.T) {
	runner := &mockRunner{containers: []container.Summary{}}
	checker := &mockJobChecker{activeJobs: map[string]bool{}}
	cleaner := newTestOrphanCleaner(runner, checker)

	err := cleaner.CleanupAll(context.Background())
	require.NoError(t, err)

	assert.Empty(t, runner.stopped)
	assert.Empty(t, runner.removed)
}

func TestCleanup_StillRespectsGracePeriod(t *testing.T) {
	// A container created 1 minute ago should be skipped by Cleanup (grace period is 15 min).
	runner := &mockRunner{
		containers: []container.Summary{
			{
				ID:      "young-container",
				Created: time.Now().Add(-1 * time.Minute).Unix(),
				Labels:  map[string]string{"synclet.io/managed": "true", "synclet.io/job-id": "job-1"},
			},
		},
	}
	checker := &mockJobChecker{activeJobs: map[string]bool{"job-1": false}}
	cleaner := newTestOrphanCleaner(runner, checker)

	err := cleaner.Cleanup(context.Background())
	require.NoError(t, err)

	// Container is younger than grace period -- should NOT be removed.
	assert.Empty(t, runner.stopped)
	assert.Empty(t, runner.removed)
}
