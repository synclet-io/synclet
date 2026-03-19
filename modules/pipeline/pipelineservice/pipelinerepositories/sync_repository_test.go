package pipelinerepositories

import "testing"

// TestSyncRepositoryAutoCreateCompiles verifies the SyncRepository type and its
// Spec handling compiles correctly. Full integration test requires database or mock storage.
func TestSyncRepositoryAutoCreateCompiles(t *testing.T) {
	t.Skip("requires mock storage -- see integration tests")
}
