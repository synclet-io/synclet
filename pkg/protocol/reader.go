package protocol

import (
	"bufio"
	"context"
	"encoding/json"
	"io"
	"strings"

	"github.com/go-pnp/go-pnp/logging"
)

const defaultScannerBufferSize = 1024 * 1024 // 1MB — connector records can be large.

// MessageReader reads JSON lines from an io.Reader and emits AirbyteMessage structs.
type MessageReader struct {
	scanner *bufio.Scanner
	logger  *logging.Logger
}

// NewMessageReader creates a MessageReader that reads JSON lines from r.
// The internal scanner buffer is set to 1MB to handle large connector records.
func NewMessageReader(r io.Reader) *MessageReader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, defaultScannerBufferSize), defaultScannerBufferSize)

	return &MessageReader{scanner: scanner, logger: nil}
}

// Read reads the next AirbyteMessage from the underlying reader.
// Blank lines are skipped. Unparseable lines are logged and skipped.
// Returns io.EOF when there are no more messages.
func (mr *MessageReader) Read() (*AirbyteMessage, error) {
	for mr.scanner.Scan() {
		line := strings.TrimSpace(mr.scanner.Text())
		if line == "" {
			continue
		}

		var msg AirbyteMessage
		if err := json.Unmarshal([]byte(line), &msg); err != nil {
			mr.logger.WithError(err).Warn(context.Background(), "skipping unparseable line", "line", truncateLine(line, 200))
			continue
		}

		// Drop messages with unknown or empty type (per Airbyte spec).
		if !isKnownMessageType(msg.Type) {
			continue
		}

		return &msg, nil
	}

	if err := mr.scanner.Err(); err != nil {
		return nil, err
	}

	return nil, io.EOF
}

// ReadAll reads all messages into a channel. Unparseable lines are logged and skipped.
// The channel is closed when the reader reaches EOF or the context is cancelled.
func (mr *MessageReader) ReadAll(ctx context.Context) <-chan *AirbyteMessage {
	ch := make(chan *AirbyteMessage)

	go func() {
		defer close(ch)

		for {
			msg, err := mr.Read()
			if err != nil {
				if err != io.EOF {
					mr.logger.WithError(err).Error(ctx, "message reader error")
				}
				return
			}

			select {
			case <-ctx.Done():
				return
			case ch <- msg:
			}
		}
	}()

	return ch
}

func isKnownMessageType(t MessageType) bool {
	switch t {
	case MessageTypeRecord,
		MessageTypeState,
		MessageTypeLog,
		MessageTypeSpec,
		MessageTypeCatalog,
		MessageTypeConnectionStatus,
		MessageTypeTrace,
		MessageTypeControl,
		MessageTypeDestinationCatalog:
		return true
	default:
		return false
	}
}

func truncateLine(line string, maxLen int) string {
	if len(line) <= maxLen {
		return line
	}
	return line[:maxLen] + "..."
}
