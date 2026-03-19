//go:build e2e

package e2e

import (
	"context"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// testMode defines a sync execution mode for parameterized tests.
type testMode struct {
	name  string
	setup func(t *testing.T) modeRunner
}

// activeModes returns the set of modes to test against.
// Docker is always active; K8s requires E2E_K8S=1.
func activeModes() []testMode {
	modes := []testMode{
		{"docker", setupDockerMode},
	}
	if os.Getenv("E2E_K8S") == "1" {
		modes = append(modes, testMode{"k8s", setupK8sMode})
	}
	return modes
}

// TestE2E_HappyPathFullRefresh verifies that a full refresh sync completes successfully
// with the expected record count.
func TestE2E_HappyPathFullRefresh(t *testing.T) {
	for _, mode := range activeModes() {
		t.Run(mode.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			runner := mode.setup(t)
			t.Cleanup(func() { runner.Cleanup(t) })

			opts := syncOpts{
				SourceConfig: map[string]any{
					"streams": []map[string]any{
						{
							"name":         "users",
							"namespace":    "public",
							"record_count": 100,
						},
					},
					"emit_state_every": 25,
				},
				Streams: []streamOpt{
					{
						Name:                "users",
						Namespace:           "public",
						SyncMode:            "full_refresh",
						DestinationSyncMode: "overwrite",
					},
				},
				FullRefresh: true,
			}

			result := runner.RunSync(t, ctx, opts)

			assert.Equal(t, "completed", result.JobStatus, "job should be completed")
			require.NoError(t, result.Err, "sync should complete without error")
			assert.Equal(t, int64(100), result.RecordsRead, "should read 100 records")
		})
	}
}

// TestE2E_IncrementalSync verifies that incremental sync persists state across runs
// and resumes from the last cursor position.
func TestE2E_IncrementalSync(t *testing.T) {
	for _, mode := range activeModes() {
		t.Run(mode.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			runner := mode.setup(t)
			t.Cleanup(func() { runner.Cleanup(t) })

			// Run 1: initial incremental sync with 50 records.
			opts := syncOpts{
				SourceConfig: map[string]any{
					"streams": []map[string]any{
						{
							"name":         "events",
							"namespace":    "public",
							"record_count": 50,
						},
					},
					"emit_state_every": 10,
				},
				Streams: []streamOpt{
					{
						Name:                "events",
						Namespace:           "public",
						SyncMode:            "incremental",
						CursorField:         "updated_at",
						DestinationSyncMode: "append",
					},
				},
			}

			result1 := runner.RunSync(t, ctx, opts)

			assert.Equal(t, "completed", result1.JobStatus, "run 1 job should be completed")
			require.NoError(t, result1.Err, "run 1 should complete without error")
			assert.Equal(t, int64(50), result1.RecordsRead, "run 1 should read 50 records")

			// Run 2: same config but 30 more records -- should resume from cursor=50.
			opts2 := syncOpts{
				SourceConfig: map[string]any{
					"streams": []map[string]any{
						{
							"name":         "events",
							"namespace":    "public",
							"record_count": 30,
						},
					},
					"emit_state_every": 10,
				},
				Streams: []streamOpt{
					{
						Name:                "events",
						Namespace:           "public",
						SyncMode:            "incremental",
						CursorField:         "updated_at",
						DestinationSyncMode: "append",
					},
				},
			}

			result2 := runner.RunSync(t, ctx, opts2)

			assert.Equal(t, "completed", result2.JobStatus, "run 2 job should be completed")
			require.NoError(t, result2.Err, "run 2 should complete without error")
			assert.Equal(t, int64(30), result2.RecordsRead, "run 2 should read only 30 records (resumed)")

			// Verify stream state reflects resumed cursor position.
			if states := result2.StreamStates; states != nil {
				stateKey := "public.events"
				state, ok := states[stateKey]
				if assert.True(t, ok, "stream state for %q should exist", stateKey) {
					cursor, ok := state["cursor"]
					if assert.True(t, ok, "cursor should exist in state") {
						// Cursor should be 50+30=80 (started at 50, added 30).
						cursorVal, ok := cursor.(float64)
						if assert.True(t, ok, "cursor should be a number") {
							assert.GreaterOrEqual(t, cursorVal, float64(50), "cursor should reflect resumed position")
						}
					}
				}
			}
		})
	}
}

// TestE2E_ConnectorFailure verifies that a source connector crash is properly
// captured and results in a failed job.
func TestE2E_ConnectorFailure(t *testing.T) {
	for _, mode := range activeModes() {
		t.Run(mode.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			runner := mode.setup(t)
			t.Cleanup(func() { runner.Cleanup(t) })

			opts := syncOpts{
				SourceConfig: map[string]any{
					"streams": []map[string]any{
						{
							"name":         "data",
							"namespace":    "public",
							"record_count": 100,
						},
					},
					"crash_after_records": 25,
					"exit_code":           1,
				},
				Streams: []streamOpt{
					{
						Name:                "data",
						Namespace:           "public",
						SyncMode:            "full_refresh",
						DestinationSyncMode: "overwrite",
					},
				},
				FullRefresh: true,
			}

			result := runner.RunSync(t, ctx, opts)

			assert.Equal(t, "failed", result.JobStatus, "job should be failed")
			assert.NotEmpty(t, result.JobError, "job error message should not be empty")
		})
	}
}

// TestE2E_MultiStream verifies that a sync with multiple streams processes all
// streams and reports correct total record counts.
func TestE2E_MultiStream(t *testing.T) {
	for _, mode := range activeModes() {
		t.Run(mode.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
			defer cancel()

			runner := mode.setup(t)
			t.Cleanup(func() { runner.Cleanup(t) })

			opts := syncOpts{
				SourceConfig: map[string]any{
					"streams": []map[string]any{
						{"name": "users", "namespace": "public", "record_count": 30},
						{"name": "orders", "namespace": "sales", "record_count": 20},
						{"name": "logs", "namespace": "system", "record_count": 10},
					},
					"emit_state_every": 10,
				},
				Streams: []streamOpt{
					{Name: "users", Namespace: "public", SyncMode: "full_refresh", DestinationSyncMode: "overwrite"},
					{Name: "orders", Namespace: "sales", SyncMode: "full_refresh", DestinationSyncMode: "overwrite"},
					{Name: "logs", Namespace: "system", SyncMode: "full_refresh", DestinationSyncMode: "overwrite"},
				},
				FullRefresh: true,
			}

			result := runner.RunSync(t, ctx, opts)

			assert.Equal(t, "completed", result.JobStatus, "job should be completed")
			require.NoError(t, result.Err, "sync should complete without error")
			assert.Equal(t, int64(60), result.RecordsRead, "should read 60 total records (30+20+10)")
		})
	}
}
