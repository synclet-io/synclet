package coordinator

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestTaskConfigRedactor_MasksCheckStatusMessage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "task-config"), []byte(`{"password":"task-secret-9"}`), 0o600))

	redactor := taskConfigRedactor(dir)
	assert.NotContains(t, redactor.Redact("auth failed for task-secret-9"), "task-secret-9")

	result, err := extractTaskResult("Check", []*protocol.AirbyteMessage{{
		Type: protocol.MessageTypeConnectionStatus,
		ConnectionStatus: &protocol.AirbyteConnectionStatus{
			Status:  protocol.ConnectionStatusFailed,
			Message: "login rejected: task-secret-9",
		},
	}}, redactor)
	require.NoError(t, err)

	var decoded struct{ Message string }
	require.NoError(t, json.Unmarshal(result, &decoded))
	assert.NotContains(t, decoded.Message, "task-secret-9")
}

func TestTaskConfigRedactor_WithoutConfigRedactsNothing(t *testing.T) {
	assert.Nil(t, taskConfigRedactor(""))
	assert.Nil(t, taskConfigRedactor(t.TempDir()))
}
