package protocol

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"

	"github.com/go-pnp/go-pnp/logging"
)

// MaxMessageSize is the longest message line a MessageReader accepts. Catalogs
// of wide sources and single records can reach tens of MiB.
const MaxMessageSize = 100 << 20

const initialBufferSize = 64 << 10

// ErrMessageTooLarge is returned by Read for a message line longer than the
// reader's limit. The reader cannot continue past such a line.
var ErrMessageTooLarge = errors.New("connector message too large")

// MessageReader reads JSON lines from an io.Reader and emits AirbyteMessage structs.
type MessageReader struct {
	scanner *bufio.Scanner
	maxSize int
	logger  *logging.Logger
}

// NewMessageReader creates a MessageReader that reads JSON lines from r,
// accepting lines up to MaxMessageSize. The line buffer grows on demand.
func NewMessageReader(r io.Reader) *MessageReader {
	return newMessageReader(r, MaxMessageSize)
}

func newMessageReader(r io.Reader, maxSize int) *MessageReader {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 0, min(initialBufferSize, maxSize)), maxSize)

	return &MessageReader{scanner: scanner, maxSize: maxSize, logger: nil}
}

// Read reads the next AirbyteMessage from the underlying reader.
// Blank lines are skipped. Unparseable lines are logged and skipped.
// Returns io.EOF when there are no more messages, and ErrMessageTooLarge,
// on this and every later call, once a line exceeds the limit.
func (mr *MessageReader) Read() (*AirbyteMessage, error) {
	for mr.scanner.Scan() {
		line := bytes.TrimSpace(mr.scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var msg AirbyteMessage
		if err := json.Unmarshal(line, &msg); err != nil {
			mr.logger.WithError(err).Warn(context.Background(), "skipping unparseable line", "line", truncateLine(string(line[:min(len(line), 256)]), 200))
			continue
		}

		// Drop messages with unknown or empty type (per Airbyte spec).
		if !isKnownMessageType(msg.Type) {
			continue
		}

		return &msg, nil
	}

	if err := mr.scanner.Err(); err != nil {
		if errors.Is(err, bufio.ErrTooLong) {
			return nil, fmt.Errorf("%w: a line exceeds the %s limit", ErrMessageTooLarge, formatSize(mr.maxSize))
		}

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

func formatSize(n int) string {
	if n >= 1<<20 && n%(1<<20) == 0 {
		return fmt.Sprintf("%d MiB", n>>20)
	}

	return fmt.Sprintf("%d bytes", n)
}
