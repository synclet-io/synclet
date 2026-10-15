package pipelinedestinations_test

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func configMap(t *testing.T, raw string) map[string]any {
	t.Helper()

	var parsed map[string]any

	require.NoError(t, json.Unmarshal([]byte(raw), &parsed))

	return parsed
}

func payloadMap(t *testing.T, value any) map[string]any {
	t.Helper()

	parsed, ok := value.(map[string]any)
	require.True(t, ok, "expected map[string]any payload, got %T", value)

	return parsed
}

func eventJSON(t *testing.T, event pipelineservice.AuditEvent) string {
	t.Helper()

	raw, err := json.Marshal(event)
	require.NoError(t, err)

	return string(raw)
}
