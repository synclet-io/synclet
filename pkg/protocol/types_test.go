package protocol

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRecordMessageWithMeta(t *testing.T) {
	line := `{"type":"RECORD","record":{"stream":"users","data":{"id":1},"emitted_at":1700000000000,"meta":{"changes":[{"field":"email","change":"INSERT","reason":"schema_change"}]}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, MessageTypeRecord, msg.Type)
	require.NotNil(t, msg.Record.Meta)
	require.Len(t, msg.Record.Meta.Changes, 1)
	assert.Equal(t, "email", msg.Record.Meta.Changes[0].Field)
	assert.Equal(t, ChangeInsert, msg.Record.Meta.Changes[0].Change)
	assert.Equal(t, "schema_change", msg.Record.Meta.Changes[0].Reason)
}

func TestRecordMessageWithFileReference(t *testing.T) {
	line := `{"type":"RECORD","record":{"stream":"files","data":{},"emitted_at":1700000000000,"file_reference":{"staging_file_path":"/tmp/staging/file.csv","source_file_url":"s3://bucket/file.csv","file_size":1024}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, msg.Record.FileReference)
	assert.Equal(t, "/tmp/staging/file.csv", msg.Record.FileReference.StagingFilePath)
	assert.Equal(t, "s3://bucket/file.csv", msg.Record.FileReference.SourceFileURL)
	assert.Equal(t, int64(1024), msg.Record.FileReference.FileSize)
}

func TestStateMessageWithStats(t *testing.T) {
	line := `{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":100}},"sourceStats":{"recordCount":42},"destinationStats":{"recordCount":40}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, MessageTypeState, msg.Type)
	require.NotNil(t, msg.State.SourceStats)
	assert.Equal(t, float64(42), msg.State.SourceStats.RecordCount)
	require.NotNil(t, msg.State.DestinationStats)
	assert.Equal(t, float64(40), msg.State.DestinationStats.RecordCount)
}

func TestLogMessageWithStackTrace(t *testing.T) {
	line := `{"type":"LOG","log":{"level":"ERROR","message":"failed","stack_trace":"at main.go:42\nat handler.go:10"}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, "at main.go:42\nat handler.go:10", msg.Log.StackTrace)
}

func TestErrorTraceWithTransientError(t *testing.T) {
	line := `{"type":"TRACE","trace":{"type":"ERROR","emitted_at":1700000000000,"error":{"message":"timeout","failure_type":"transient_error","stream_descriptor":{"name":"users","namespace":"public"}}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, msg.Trace.Error)
	assert.Equal(t, FailureTypeTransientError, msg.Trace.Error.FailureType)
	require.NotNil(t, msg.Trace.Error.StreamDescriptor)
	assert.Equal(t, "users", msg.Trace.Error.StreamDescriptor.Name)
	assert.Equal(t, "public", msg.Trace.Error.StreamDescriptor.Namespace)
}

func TestAnalyticsTraceMessage(t *testing.T) {
	line := `{"type":"TRACE","trace":{"type":"ANALYTICS","emitted_at":1700000000000,"analytics":{"type":"connector_startup_time","value":"1500"}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, TraceTypeAnalytics, msg.Trace.Type)
	require.NotNil(t, msg.Trace.Analytics)
	assert.Equal(t, "connector_startup_time", msg.Trace.Analytics.Type)
	assert.Equal(t, "1500", msg.Trace.Analytics.Value)
}

func TestStreamStatusWithReasons(t *testing.T) {
	line := `{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1700000000000,"stream_status":{"stream_descriptor":{"name":"users"},"status":"RUNNING","reasons":[{"type":"RATE_LIMITED"}]}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	require.NotNil(t, msg.Trace.StreamStatus)
	require.Len(t, msg.Trace.StreamStatus.Reasons, 1)
	assert.Equal(t, "RATE_LIMITED", msg.Trace.StreamStatus.Reasons[0].Type)
}

func TestControlMessageConnectorConfig(t *testing.T) {
	line := `{"type":"CONTROL","control":{"type":"CONNECTOR_CONFIG","emitted_at":1700000000000,"connectorConfig":{"config":{"api_key":"new_key","host":"example.com"}}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, MessageTypeControl, msg.Type)
	require.NotNil(t, msg.Control)
	assert.Equal(t, ControlMessageTypeConnectorConfig, msg.Control.Type)
	require.NotNil(t, msg.Control.ConnectorConfig)

	var config map[string]string
	err = json.Unmarshal(msg.Control.ConnectorConfig.Config, &config)
	require.NoError(t, err)
	assert.Equal(t, "new_key", config["api_key"])
}

func TestDestinationCatalogMessage(t *testing.T) {
	line := `{"type":"DESTINATION_CATALOG","destinationCatalog":{"catalog":{"streams":[{"name":"files","json_schema":{"type":"object"}}]}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, MessageTypeDestinationCatalog, msg.Type)
	require.NotNil(t, msg.DestinationCatalog)
	require.NotNil(t, msg.DestinationCatalog.Catalog)
	require.Len(t, msg.DestinationCatalog.Catalog.Streams, 1)
	assert.Equal(t, "files", msg.DestinationCatalog.Catalog.Streams[0].Name)
}

func TestSpecWithAdvancedAuth(t *testing.T) {
	line := `{"type":"SPEC","spec":{"documentationUrl":"https://example.com","connectionSpecification":{},"protocol_version":"0.3.2","changelogUrl":"https://example.com/changelog","advancedAuth":{"auth_flow_type":"oauth2.0","predicate_key":["credentials","auth_type"],"predicate_value":"oauth"}}}`
	reader := NewMessageReader(strings.NewReader(line + "\n"))

	msg, err := reader.Read()
	require.NoError(t, err)
	assert.Equal(t, MessageTypeSpec, msg.Type)
	require.NotNil(t, msg.Spec)
	assert.Equal(t, "0.3.2", msg.Spec.ProtocolVersion)
	assert.Equal(t, "https://example.com/changelog", msg.Spec.ChangelogURL)
	require.NotNil(t, msg.Spec.AdvancedAuth)
	assert.Equal(t, "oauth2.0", msg.Spec.AdvancedAuth.AuthFlowType)
}

func TestConfiguredStreamExtendedFields(t *testing.T) {
	input := `{"streams":[{"stream":{"name":"users","json_schema":{}},"sync_mode":"incremental","destination_sync_mode":"append_dedup","generation_id":5,"minimum_generation_id":3,"sync_id":42,"destination_object_name":"public.users"}]}`

	var catalog ConfiguredAirbyteCatalog
	err := json.Unmarshal([]byte(input), &catalog)
	require.NoError(t, err)
	require.Len(t, catalog.Streams, 1)

	s := catalog.Streams[0]
	assert.Equal(t, int64(5), s.GenerationID)
	assert.Equal(t, int64(3), s.MinimumGenerationID)
	assert.Equal(t, int64(42), s.SyncID)
	assert.Equal(t, "public.users", s.DestinationObjectName)
}

func TestStreamWithResumable(t *testing.T) {
	input := `{"name":"users","json_schema":{},"is_resumable":true,"is_file_based":true}`

	var stream AirbyteStream
	err := json.Unmarshal([]byte(input), &stream)
	require.NoError(t, err)
	assert.True(t, stream.IsResumable)
	assert.True(t, stream.IsFileBased)
}

func TestNewDestinationSyncModes(t *testing.T) {
	assert.Equal(t, DestinationSyncMode("update"), DestinationSyncModeUpdate)
	assert.Equal(t, DestinationSyncMode("soft_delete"), DestinationSyncModeSoftDelete)
}

func TestRealConnectorOutputWithNewTypes(t *testing.T) {
	// Simulates a modern connector producing all new v0.3.2 message types.
	lines := []string{
		`{"type":"LOG","log":{"level":"INFO","message":"Starting connector..."}}`,
		`{"type":"SPEC","spec":{"documentationUrl":"https://example.com","connectionSpecification":{},"protocol_version":"0.3.2"}}`,
		`{"type":"TRACE","trace":{"type":"ANALYTICS","emitted_at":1700000000000,"analytics":{"type":"connector_startup_time","value":"800"}}}`,
		`{"type":"RECORD","record":{"stream":"users","data":{"id":1},"emitted_at":1700000000000,"meta":{"changes":[{"field":"age","change":"INSERT"}]}}}`,
		`{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":1}},"sourceStats":{"recordCount":1}}}`,
		`{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1700000000001,"stream_status":{"stream_descriptor":{"name":"users"},"status":"RUNNING","reasons":[{"type":"RATE_LIMITED"}]}}}`,
		`{"type":"CONTROL","control":{"type":"CONNECTOR_CONFIG","emitted_at":1700000000002,"connectorConfig":{"config":{"token":"refreshed"}}}}`,
		`{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1700000000003,"stream_status":{"stream_descriptor":{"name":"users"},"status":"COMPLETE"}}}`,
	}
	input := strings.Join(lines, "\n") + "\n"
	reader := NewMessageReader(strings.NewReader(input))

	var messages []*AirbyteMessage
	for msg := range reader.ReadAll(context.Background()) {
		messages = append(messages, msg)
	}

	require.Len(t, messages, 8)
	assert.Equal(t, MessageTypeLog, messages[0].Type)
	assert.Equal(t, MessageTypeSpec, messages[1].Type)
	assert.Equal(t, "0.3.2", messages[1].Spec.ProtocolVersion)
	assert.Equal(t, TraceTypeAnalytics, messages[2].Trace.Type)
	assert.Equal(t, MessageTypeRecord, messages[3].Type)
	require.NotNil(t, messages[3].Record.Meta)
	assert.Equal(t, MessageTypeState, messages[4].Type)
	require.NotNil(t, messages[4].State.SourceStats)
	assert.Equal(t, StreamStatusRunning, messages[5].Trace.StreamStatus.Status)
	assert.Equal(t, MessageTypeControl, messages[6].Type)
	assert.Equal(t, StreamStatusComplete, messages[7].Trace.StreamStatus.Status)
}
