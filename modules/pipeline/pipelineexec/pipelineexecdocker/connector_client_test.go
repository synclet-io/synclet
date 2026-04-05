package pipelineexecdocker

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
)

// mockContainerRunner implements pkgcontainer.Runner for testing.
type mockContainerRunner struct {
	runCalls    int
	runResults  []*pkgcontainer.RunResult
	runErrors   []error
	lastRunOpts pkgcontainer.RunOptions
	pullCalled  bool
	pullImage   string
	pullError   error

	resolveDigestCalled bool
	resolveDigestImage  string
	resolveDigestResult string
	resolveDigestError  error
}

func (m *mockContainerRunner) Run(_ context.Context, opts pkgcontainer.RunOptions) (*pkgcontainer.RunResult, error) {
	m.lastRunOpts = opts
	idx := m.runCalls

	m.runCalls++
	if idx < len(m.runErrors) && m.runErrors[idx] != nil {
		return nil, m.runErrors[idx]
	}

	if idx < len(m.runResults) {
		return m.runResults[idx], nil
	}

	return &pkgcontainer.RunResult{}, nil
}

func (m *mockContainerRunner) Pull(_ context.Context, image string) error {
	m.pullCalled = true
	m.pullImage = image

	return m.pullError
}

func (m *mockContainerRunner) ResolveDigest(_ context.Context, image string) (string, error) {
	m.resolveDigestCalled = true
	m.resolveDigestImage = image

	return m.resolveDigestResult, m.resolveDigestError
}

func (m *mockContainerRunner) Stop(_ context.Context, _ string) error                   { return nil }
func (m *mockContainerRunner) StopWithTimeout(_ context.Context, _ string, _ int) error { return nil }
func (m *mockContainerRunner) Remove(_ context.Context, _ string) error                 { return nil }

func TestRunWithAutoPull(t *testing.T) {
	ctx := context.Background()
	opts := pkgcontainer.RunOptions{Image: "airbyte/source-postgres:latest"}

	t.Run("succeeds first try without pulling", func(t *testing.T) {
		expectedResult := &pkgcontainer.RunResult{ContainerID: "abc123"}
		runner := &mockContainerRunner{
			runResults: []*pkgcontainer.RunResult{expectedResult},
		}
		client := NewConnectorClient(runner)

		result, err := client.runWithAutoPull(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		assert.Equal(t, 1, runner.runCalls)
		assert.False(t, runner.pullCalled, "Pull should not be called when Run succeeds")
	})

	t.Run("pulls and retries on image not found with digest pinning", func(t *testing.T) {
		expectedResult := &pkgcontainer.RunResult{ContainerID: "def456"}
		runner := &mockContainerRunner{
			runErrors:           []error{errors.New("creating container: No such image: airbyte/source-postgres:latest")},
			runResults:          []*pkgcontainer.RunResult{nil, expectedResult},
			resolveDigestResult: "airbyte/source-postgres@sha256:abc123def456",
		}
		client := NewConnectorClient(runner)

		result, err := client.runWithAutoPull(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		assert.Equal(t, 2, runner.runCalls, "Run should be called twice (initial + retry)")
		assert.True(t, runner.pullCalled, "Pull should be called on image-not-found error")
		assert.Equal(t, "airbyte/source-postgres:latest", runner.pullImage)
		assert.True(t, runner.resolveDigestCalled, "ResolveDigest should be called after pull")
		assert.Equal(t, "airbyte/source-postgres@sha256:abc123def456", runner.lastRunOpts.Image,
			"retry Run should use digest-pinned image reference")
	})

	t.Run("returns error when pull fails", func(t *testing.T) {
		runner := &mockContainerRunner{
			runErrors: []error{errors.New("creating container: No such image: airbyte/source-postgres:latest")},
			pullError: errors.New("network timeout"),
		}
		client := NewConnectorClient(runner)

		result, err := client.runWithAutoPull(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "not found locally and pull failed")
		assert.Contains(t, err.Error(), "airbyte/source-postgres:latest")
		assert.Contains(t, err.Error(), "network timeout")
		assert.Equal(t, 1, runner.runCalls, "Run should only be called once (no retry after pull failure)")
	})

	t.Run("returns original error for non-image errors", func(t *testing.T) {
		runner := &mockContainerRunner{
			runErrors: []error{errors.New("permission denied")},
		}
		client := NewConnectorClient(runner)

		result, err := client.runWithAutoPull(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "permission denied", err.Error())
		assert.Equal(t, 1, runner.runCalls, "Run should only be called once")
		assert.False(t, runner.pullCalled, "Pull should not be called for non-image errors")
	})

	t.Run("returns retry error when pull succeeds but retry fails", func(t *testing.T) {
		runner := &mockContainerRunner{
			runErrors: []error{
				errors.New("creating container: No such image: airbyte/source-postgres:latest"),
				errors.New("container runtime error"),
			},
		}
		client := NewConnectorClient(runner)

		result, err := client.runWithAutoPull(ctx, opts)

		require.Error(t, err)
		assert.Nil(t, result)
		assert.Equal(t, "container runtime error", err.Error())
		assert.Equal(t, 2, runner.runCalls, "Run should be called twice")
		assert.True(t, runner.pullCalled, "Pull should be called")
	})

	t.Run("does not resolve digest when run succeeds first try", func(t *testing.T) {
		expectedResult := &pkgcontainer.RunResult{ContainerID: "abc123"}
		runner := &mockContainerRunner{
			runResults:          []*pkgcontainer.RunResult{expectedResult},
			resolveDigestResult: "airbyte/source-postgres@sha256:shouldnotbeused",
		}
		client := NewConnectorClient(runner)

		result, err := client.runWithAutoPull(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		assert.False(t, runner.resolveDigestCalled, "ResolveDigest should NOT be called when Run succeeds first try")
	})

	t.Run("falls back to tag when ResolveDigest fails", func(t *testing.T) {
		expectedResult := &pkgcontainer.RunResult{ContainerID: "ghi789"}
		runner := &mockContainerRunner{
			runErrors:          []error{errors.New("creating container: No such image: airbyte/source-postgres:latest")},
			runResults:         []*pkgcontainer.RunResult{nil, expectedResult},
			resolveDigestError: errors.New("no sha256 digest found"),
		}
		client := NewConnectorClient(runner)

		result, err := client.runWithAutoPull(ctx, opts)

		require.NoError(t, err)
		assert.Equal(t, expectedResult, result)
		assert.True(t, runner.resolveDigestCalled, "ResolveDigest should be called after pull")
		assert.Equal(t, "airbyte/source-postgres:latest", runner.lastRunOpts.Image,
			"retry Run should fall back to original tag-based image when digest resolution fails")
	})
}

func TestIsImageNotFoundError(t *testing.T) {
	tests := []struct {
		name     string
		err      error
		expected bool
	}{
		{"No such image error", errors.New("No such image: foo:latest"), true},
		{"not found error", errors.New("reference does not exist"), true},
		{"permission denied", errors.New("permission denied"), false},
		{"connection refused", errors.New("connection refused"), false},
		{"wrapped not found", errors.New("creating container: No such image: bar:v1"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, isImageNotFoundError(tt.err))
		})
	}
}
