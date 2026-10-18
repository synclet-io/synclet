package pipelineexec

import "github.com/synclet-io/synclet/pkg/protocol"

// sourceStatsCounter counts the RECORDs a source emits between STATE messages,
// per stream and in total, keyed by the source's own stream names.
type sourceStatsCounter struct {
	perStream map[protocol.StreamDescriptor]int64
	total     int64
}

func newSourceStatsCounter() *sourceStatsCounter {
	return &sourceStatsCounter{perStream: make(map[protocol.StreamDescriptor]int64)}
}

func (c *sourceStatsCounter) countRecord(record *protocol.AirbyteRecordMessage) {
	c.total++

	if record != nil {
		c.perStream[protocol.StreamDescriptor{Name: record.Stream, Namespace: record.Namespace}]++
	}
}

// fillState sets SourceStats on a STATE that carries none and resets the
// counts the STATE covers. A STREAM state covers its stream; GLOBAL and LEGACY
// cover every record since the previous STATE.
func (c *sourceStatsCounter) fillState(state *protocol.AirbyteStateMessage) {
	if state == nil {
		return
	}

	var count int64

	if protocol.NormalizeStateType(state.Type) == protocol.StateTypeStream && state.Stream != nil {
		key := state.Stream.StreamDescriptor
		count = c.perStream[key]
		c.total -= count
		delete(c.perStream, key)
	} else {
		count = c.total
		c.total = 0
		clear(c.perStream)
	}

	if state.SourceStats == nil {
		state.SourceStats = &protocol.StateStats{RecordCount: float64(count)}
	}
}
