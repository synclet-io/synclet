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

// TaskMessageMaxSize is the longest message line accepted from a connector
// running spec, check or discover. Catalogs of wide sources reach tens of MiB.
const TaskMessageMaxSize = 50 << 20

// SyncMessageMaxSize is the longest message line accepted from a source or
// destination during a sync.
const SyncMessageMaxSize = 16 << 20

const (
	initialBufferSize = 64 << 10
	// retainedLineSize bounds the line buffer a reader keeps between reads, so
	// one huge message does not pin its memory for the rest of the stream.
	retainedLineSize = 1 << 20
)

// ErrMessageTooLarge is returned by Read for a message line longer than the
// reader's limit. The reader cannot continue past such a line.
var ErrMessageTooLarge = errors.New("connector message too large")

// MessageReader reads JSON lines from an io.Reader and emits AirbyteMessage structs.
type MessageReader struct {
	reader  *bufio.Reader
	line    []byte
	maxSize int
	err     error
	logger  *logging.Logger
}

// NewMessageReader creates a MessageReader that reads JSON lines from r,
// accepting lines of up to maxSize bytes, line terminator excluded.
func NewMessageReader(r io.Reader, maxSize int) *MessageReader {
	return &MessageReader{reader: bufio.NewReaderSize(r, initialBufferSize), maxSize: maxSize, logger: nil}
}

// Read reads the next AirbyteMessage from the underlying reader.
// Blank lines are skipped. Unparseable lines are logged and skipped.
// Returns io.EOF when there are no more messages, and ErrMessageTooLarge,
// on this and every later call, once a line exceeds the limit.
func (mr *MessageReader) Read() (*AirbyteMessage, error) {
	for mr.err == nil {
		line, err := mr.readLine()
		if err != nil {
			mr.err = err
			break
		}

		msg, ok := mr.decode(bytes.TrimSpace(line))
		if cap(mr.line) > retainedLineSize {
			mr.line = nil
		}

		if ok {
			return msg, nil
		}
	}

	return nil, mr.err
}

func (mr *MessageReader) decode(line []byte) (*AirbyteMessage, bool) {
	if len(line) == 0 {
		return nil, false
	}

	var msg AirbyteMessage
	if err := json.Unmarshal(line, &msg); err != nil {
		// The line may echo connector config secrets, so only its size is logged.
		mr.logger.WithError(err).Warn(context.Background(), "skipping unparseable line", "bytes", len(line))
		return nil, false
	}

	// Drop messages with unknown or empty type (per Airbyte spec).
	if !isKnownMessageType(msg.Type) {
		return nil, false
	}

	return &msg, true
}

// readLine returns the next line without its terminator. A line that fits the
// bufio buffer is returned from it without copying; a longer one is assembled
// in mr.line.
func (mr *MessageReader) readLine() ([]byte, error) {
	mr.line = mr.line[:0]

	for {
		chunk, err := mr.reader.ReadSlice('\n')
		if err == nil && len(mr.line) == 0 {
			return mr.checkSize(trimEOL(chunk))
		}

		if len(mr.line)+len(chunk) > mr.maxSize+len("\r\n") {
			return nil, mr.tooLarge()
		}

		mr.line = append(mr.line, chunk...)

		switch {
		case err == nil:
			return mr.checkSize(trimEOL(mr.line))
		case errors.Is(err, bufio.ErrBufferFull):
			continue
		case errors.Is(err, io.EOF) && len(mr.line) > 0:
			return mr.checkSize(mr.line)
		default:
			return nil, err //nolint:wrapcheck // io.EOF and reader errors pass through unchanged
		}
	}
}

func (mr *MessageReader) checkSize(line []byte) ([]byte, error) {
	if len(line) > mr.maxSize {
		return nil, mr.tooLarge()
	}

	return line, nil
}

func (mr *MessageReader) tooLarge() error {
	return fmt.Errorf("%w: a line exceeds the %s limit", ErrMessageTooLarge, formatSize(mr.maxSize))
}

func trimEOL(line []byte) []byte {
	line = bytes.TrimSuffix(line, []byte("\n"))

	return bytes.TrimSuffix(line, []byte("\r"))
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

func formatSize(n int) string {
	if n >= 1<<20 && n%(1<<20) == 0 {
		return fmt.Sprintf("%d MiB", n>>20)
	}

	return fmt.Sprintf("%d bytes", n)
}
