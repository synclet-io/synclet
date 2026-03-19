package protocol

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMessageReader_Read(t *testing.T) {
	t.Run("reads record message", func(t *testing.T) {
		line := `{"type":"RECORD","record":{"stream":"users","data":{"id":1,"name":"Alice"},"emitted_at":1700000000000}}`
		reader := NewMessageReader(strings.NewReader(line + "\n"))

		msg, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeRecord, msg.Type)
		assert.NotNil(t, msg.Record)
		assert.Equal(t, "users", msg.Record.Stream)
	})

	t.Run("reads state message", func(t *testing.T) {
		line := `{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":100}}}}`
		reader := NewMessageReader(strings.NewReader(line + "\n"))

		msg, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeState, msg.Type)
		assert.NotNil(t, msg.State)
	})

	t.Run("reads log message", func(t *testing.T) {
		line := `{"type":"LOG","log":{"level":"INFO","message":"Starting sync"}}`
		reader := NewMessageReader(strings.NewReader(line + "\n"))

		msg, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeLog, msg.Type)
		assert.NotNil(t, msg.Log)
		assert.Equal(t, "Starting sync", msg.Log.Message)
	})

	t.Run("skips blank lines", func(t *testing.T) {
		input := "\n\n" + `{"type":"LOG","log":{"level":"INFO","message":"hello"}}` + "\n\n"
		reader := NewMessageReader(strings.NewReader(input))

		msg, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeLog, msg.Type)

		_, err = reader.Read()
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("skips unparseable lines", func(t *testing.T) {
		input := "not json at all\n" + `{"type":"LOG","log":{"level":"WARN","message":"ok"}}` + "\n"
		reader := NewMessageReader(strings.NewReader(input))

		msg, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeLog, msg.Type)
		assert.Equal(t, "ok", msg.Log.Message)
	})

	t.Run("drops unknown message types", func(t *testing.T) {
		input := `{"type":"UNKNOWN_NEW_TYPE","data":{}}` + "\n" + `{"type":"LOG","log":{"level":"INFO","message":"known"}}` + "\n"
		reader := NewMessageReader(strings.NewReader(input))

		msg, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, MessageTypeLog, msg.Type)
	})

	t.Run("returns EOF on empty input", func(t *testing.T) {
		reader := NewMessageReader(strings.NewReader(""))

		_, err := reader.Read()
		assert.ErrorIs(t, err, io.EOF)
	})

	t.Run("reads multiple messages", func(t *testing.T) {
		input := `{"type":"LOG","log":{"level":"INFO","message":"first"}}` + "\n" +
			`{"type":"LOG","log":{"level":"INFO","message":"second"}}` + "\n"
		reader := NewMessageReader(strings.NewReader(input))

		msg1, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, "first", msg1.Log.Message)

		msg2, err := reader.Read()
		require.NoError(t, err)
		assert.Equal(t, "second", msg2.Log.Message)

		_, err = reader.Read()
		assert.ErrorIs(t, err, io.EOF)
	})
}

func TestMessageReader_ReadAll(t *testing.T) {
	t.Run("reads all messages into channel", func(t *testing.T) {
		input := `{"type":"LOG","log":{"level":"INFO","message":"one"}}` + "\n" +
			`{"type":"LOG","log":{"level":"INFO","message":"two"}}` + "\n" +
			`{"type":"LOG","log":{"level":"INFO","message":"three"}}` + "\n"
		reader := NewMessageReader(strings.NewReader(input))

		ctx := context.Background()
		var messages []*AirbyteMessage
		for msg := range reader.ReadAll(ctx) {
			messages = append(messages, msg)
		}

		assert.Len(t, messages, 3)
		assert.Equal(t, "one", messages[0].Log.Message)
		assert.Equal(t, "two", messages[1].Log.Message)
		assert.Equal(t, "three", messages[2].Log.Message)
	})

	t.Run("respects context cancellation", func(t *testing.T) {
		// Create a reader with many messages.
		var lines []string
		for i := 0; i < 100; i++ {
			lines = append(lines, `{"type":"LOG","log":{"level":"INFO","message":"msg"}}`)
		}
		input := strings.Join(lines, "\n") + "\n"
		reader := NewMessageReader(strings.NewReader(input))

		ctx, cancel := context.WithCancel(context.Background())

		ch := reader.ReadAll(ctx)
		// Read one message then cancel.
		<-ch
		cancel()

		// Channel should eventually close without reading all 100 messages.
		count := 1
		for range ch {
			count++
		}
		assert.Less(t, count, 100)
	})

	t.Run("skips bad lines in channel mode", func(t *testing.T) {
		input := "garbage\n" + `{"type":"LOG","log":{"level":"INFO","message":"good"}}` + "\n"
		reader := NewMessageReader(strings.NewReader(input))

		var messages []*AirbyteMessage
		for msg := range reader.ReadAll(context.Background()) {
			messages = append(messages, msg)
		}

		assert.Len(t, messages, 1)
		assert.Equal(t, "good", messages[0].Log.Message)
	})
}

func TestMessageReader_RealConnectorOutput(t *testing.T) {
	// Simulates real Airbyte connector output with mixed message types.
	lines := []string{
		`{"type":"LOG","log":{"level":"INFO","message":"Starting connector..."}}`,
		`{"type":"SPEC","spec":{"documentationUrl":"https://example.com","connectionSpecification":{"type":"object","properties":{}}}}`,
		`{"type":"CONNECTION_STATUS","connectionStatus":{"status":"SUCCEEDED","message":"Connected"}}`,
		`{"type":"CATALOG","catalog":{"streams":[{"stream":{"name":"users","json_schema":{"type":"object"},"supported_sync_modes":["full_refresh"]},"sync_mode":"full_refresh","destination_sync_mode":"overwrite"}]}}`,
		`{"type":"RECORD","record":{"stream":"users","data":{"id":1,"name":"Alice","email":"alice@example.com"},"emitted_at":1700000000000}}`,
		`{"type":"RECORD","record":{"stream":"users","data":{"id":2,"name":"Bob","email":"bob@example.com"},"emitted_at":1700000000001}}`,
		`{"type":"STATE","state":{"type":"STREAM","stream":{"stream_descriptor":{"name":"users"},"stream_state":{"cursor":2}}}}`,
		`{"type":"TRACE","trace":{"type":"STREAM_STATUS","emitted_at":1700000000002,"stream_status":{"stream_descriptor":{"name":"users"},"status":"COMPLETE"}}}`,
		`{"type":"LOG","log":{"level":"INFO","message":"Sync complete"}}`,
	}
	input := strings.Join(lines, "\n") + "\n"
	reader := NewMessageReader(strings.NewReader(input))

	var messages []*AirbyteMessage
	for msg := range reader.ReadAll(context.Background()) {
		messages = append(messages, msg)
	}

	require.Len(t, messages, 9)
	assert.Equal(t, MessageTypeLog, messages[0].Type)
	assert.Equal(t, MessageTypeSpec, messages[1].Type)
	assert.Equal(t, MessageTypeConnectionStatus, messages[2].Type)
	assert.Equal(t, MessageTypeCatalog, messages[3].Type)
	assert.Equal(t, MessageTypeRecord, messages[4].Type)
	assert.Equal(t, MessageTypeRecord, messages[5].Type)
	assert.Equal(t, MessageTypeState, messages[6].Type)
	assert.Equal(t, MessageTypeTrace, messages[7].Type)
	assert.Equal(t, MessageTypeLog, messages[8].Type)
}

func TestIsKnownMessageType(t *testing.T) {
	known := []MessageType{
		MessageTypeRecord, MessageTypeState, MessageTypeLog,
		MessageTypeSpec, MessageTypeCatalog, MessageTypeConnectionStatus,
		MessageTypeTrace, MessageTypeControl, MessageTypeDestinationCatalog,
	}
	for _, mt := range known {
		assert.True(t, isKnownMessageType(mt), "expected %s to be known", mt)
	}

	assert.False(t, isKnownMessageType("UNKNOWN"))
	assert.False(t, isKnownMessageType(""))
}

func TestTruncateLine(t *testing.T) {
	assert.Equal(t, "short", truncateLine("short", 10))
	assert.Equal(t, "0123456789...", truncateLine("0123456789extra", 10))
	assert.Equal(t, "", truncateLine("", 10))
}

func TestMessageWriter_Write(t *testing.T) {
	t.Run("writes JSON line", func(t *testing.T) {
		var buf strings.Builder
		writer := NewMessageWriter(&buf)

		msg := &AirbyteMessage{
			Type: MessageTypeLog,
			Log: &AirbyteLogMessage{
				Level:   LogLevelInfo,
				Message: "hello",
			},
		}

		err := writer.Write(msg)
		require.NoError(t, err)

		output := buf.String()
		assert.True(t, strings.HasSuffix(output, "\n"), "output should end with newline")

		// Verify it can be parsed back.
		var parsed AirbyteMessage
		err = json.Unmarshal([]byte(strings.TrimSpace(output)), &parsed)
		require.NoError(t, err)
		assert.Equal(t, MessageTypeLog, parsed.Type)
		assert.Equal(t, "hello", parsed.Log.Message)
	})

	t.Run("does not escape HTML", func(t *testing.T) {
		var buf strings.Builder
		writer := NewMessageWriter(&buf)

		msg := &AirbyteMessage{
			Type: MessageTypeLog,
			Log: &AirbyteLogMessage{
				Level:   LogLevelInfo,
				Message: "a < b & c > d",
			},
		}

		err := writer.Write(msg)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "a < b & c > d")
		assert.NotContains(t, output, `\u003c`)
		assert.NotContains(t, output, `\u0026`)
	})

	t.Run("writes multiple messages", func(t *testing.T) {
		var buf strings.Builder
		writer := NewMessageWriter(&buf)

		for i := 0; i < 3; i++ {
			err := writer.Write(&AirbyteMessage{
				Type: MessageTypeLog,
				Log:  &AirbyteLogMessage{Level: LogLevelInfo, Message: "msg"},
			})
			require.NoError(t, err)
		}

		lines := strings.Split(strings.TrimSpace(buf.String()), "\n")
		assert.Len(t, lines, 3)
	})

	t.Run("omits empty fields", func(t *testing.T) {
		var buf strings.Builder
		writer := NewMessageWriter(&buf)

		err := writer.Write(&AirbyteMessage{
			Type: MessageTypeLog,
			Log:  &AirbyteLogMessage{Level: LogLevelInfo, Message: "test"},
		})
		require.NoError(t, err)

		output := buf.String()
		assert.NotContains(t, output, "record")
		assert.NotContains(t, output, "state")
		assert.NotContains(t, output, "catalog")
	})
}

func TestRoundTrip(t *testing.T) {
	// Write messages, then read them back.
	original := []*AirbyteMessage{
		{Type: MessageTypeLog, Log: &AirbyteLogMessage{Level: LogLevelInfo, Message: "start"}},
		{Type: MessageTypeRecord, Record: &AirbyteRecordMessage{Stream: "users", Data: json.RawMessage(`{"id":1}`), EmittedAt: 1700000000000}},
		{Type: MessageTypeState, State: &AirbyteStateMessage{Type: StateTypeStream}},
		{Type: MessageTypeLog, Log: &AirbyteLogMessage{Level: LogLevelInfo, Message: "done"}},
	}

	var buf strings.Builder
	writer := NewMessageWriter(&buf)
	for _, msg := range original {
		require.NoError(t, writer.Write(msg))
	}

	reader := NewMessageReader(strings.NewReader(buf.String()))
	var roundTripped []*AirbyteMessage
	for msg := range reader.ReadAll(context.Background()) {
		roundTripped = append(roundTripped, msg)
	}

	require.Len(t, roundTripped, len(original))
	for i, msg := range roundTripped {
		assert.Equal(t, original[i].Type, msg.Type)
	}
}
