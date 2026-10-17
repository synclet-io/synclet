package pipelineexec

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

type stateRecorder struct {
	DefaultHandler

	mu      sync.Mutex
	cursors []int
}

func (h *stateRecorder) OnStateConfirmed(_ context.Context, msg *protocol.AirbyteStateMessage) error {
	var data struct {
		Cursor int `json:"cursor"`
	}

	if err := json.Unmarshal(msg.Data, &data); err != nil {
		return err
	}

	h.mu.Lock()
	defer h.mu.Unlock()

	h.cursors = append(h.cursors, data.Cursor)

	return nil
}

func (h *stateRecorder) stored() []int {
	h.mu.Lock()
	defer h.mu.Unlock()

	return append([]int(nil), h.cursors...)
}

// sourceStream emits RECORD id=1, STATE cursor=1, RECORD id=2, STATE cursor=2,
// RECORD id=3, STATE cursor=3: a STATE's cursor is the id of the last record before it.
func sourceStream() string {
	var b strings.Builder

	for id := 1; id <= 3; id++ {
		fmt.Fprintf(&b, `{"type":"RECORD","record":{"stream":"users","data":{"id":%d},"emitted_at":0}}`+"\n", id)
		fmt.Fprintf(&b, `{"type":"STATE","state":{"type":"LEGACY","data":{"cursor":%d}}}`+"\n", id)
	}

	return b.String()
}

var errDestinationKilled = errors.New("destination killed")

// fakeDestination commits the records it has read when it reads a STATE and
// then echoes that STATE. It is killed after reading killAfter messages, before
// handling the last of them, by breaking both of its pipes. It returns the
// highest record id it acknowledged.
func fakeDestination(stdin *io.PipeReader, stdout *io.PipeWriter, killAfter int) int {
	reader := protocol.NewMessageReader(stdin)
	writer := protocol.NewMessageWriter(stdout)

	pending, acked := 0, 0

	for read := 1; ; read++ {
		msg, err := reader.Read()
		if err != nil {
			_ = stdout.Close()

			return acked
		}

		if read == killAfter {
			_ = stdin.CloseWithError(errDestinationKilled)
			_ = stdout.CloseWithError(errDestinationKilled)

			return acked
		}

		switch msg.Type {
		case protocol.MessageTypeRecord:
			var data struct {
				ID int `json:"id"`
			}

			if err := json.Unmarshal(msg.Record.Data, &data); err != nil {
				panic(err)
			}

			pending = data.ID
		case protocol.MessageTypeState:
			acked = pending

			if err := writer.Write(msg); err != nil {
				return acked
			}
		}
	}
}

func TestRun_StoredStateNeverPassesUnacknowledgedRecords(t *testing.T) {
	tests := []struct {
		name      string
		killAfter int
		stored    []int
	}{
		{name: "killed on first record", killAfter: 1, stored: nil},
		{name: "killed between first record and its state", killAfter: 2, stored: nil},
		{name: "killed between second record and its state", killAfter: 4, stored: []int{1}},
		{name: "killed right after echoing second state", killAfter: 5, stored: []int{1, 2}},
		{name: "killed between third record and its state", killAfter: 6, stored: []int{1, 2}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			destStdinReader, destStdinWriter := io.Pipe()
			destStdoutReader, destStdoutWriter := io.Pipe()

			acked := make(chan int, 1)

			go func() { acked <- fakeDestination(destStdinReader, destStdoutWriter, tt.killAfter) }()

			handler := &stateRecorder{}
			result := make(chan error, 1)

			go func() {
				_, err := Run(context.Background(), strings.NewReader(sourceStream()), destStdinWriter, destStdoutReader, handler, RunConfig{}, nil)
				result <- err
			}()

			select {
			case err := <-result:
				require.ErrorIs(t, err, errDestinationKilled)
			case <-time.After(5 * time.Second):
				t.Fatal("router did not return after the destination was killed")
			}

			lastAcked := <-acked
			stored := handler.stored()

			assert.Equal(t, tt.stored, stored)

			for _, cursor := range stored {
				assert.LessOrEqual(t, cursor, lastAcked, "stored cursor passes records the destination did not acknowledge")
			}
		})
	}
}
