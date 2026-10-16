package coordinator

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestReporter_QueueLogCountsDroppedLines(t *testing.T) {
	r := NewReporter("http://localhost", "job", "conn")

	for range cap(r.logCh) + 3 {
		r.QueueLog("line")
	}

	assert.Equal(t, []string{"a", "[synclet] 3 log lines dropped: log buffer full"}, r.withDroppedNotice([]string{"a"}))
	assert.Equal(t, []string{"a"}, r.withDroppedNotice([]string{"a"}))
}

func TestReporter_QueueStateBlocksUntilStopped(t *testing.T) {
	r := NewReporter("http://localhost", "job", "conn")
	msg := &protocol.AirbyteStateMessage{Type: protocol.StateTypeStream}

	for range cap(r.stateCh) {
		r.QueueState(msg)
	}

	queued := make(chan struct{})

	go func() {
		r.QueueState(msg)
		close(queued)
	}()

	select {
	case <-queued:
		t.Fatal("QueueState returned while the buffer was full")
	case <-time.After(50 * time.Millisecond):
	}

	close(r.done)

	select {
	case <-queued:
	case <-time.After(time.Second):
		require.Fail(t, "QueueState did not return after the reporter stopped")
	}

	assert.Len(t, r.stateCh, cap(r.stateCh))
}
