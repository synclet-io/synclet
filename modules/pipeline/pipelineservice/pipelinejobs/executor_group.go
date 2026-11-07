package pipelinejobs

import (
	"strings"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// clearNoLiveExecutorReason drops a connection's status reason when it is a
// NoLiveExecutorError message, and reports whether it did.
func clearNoLiveExecutorReason(conn *pipelineservice.Connection) bool {
	if conn.StatusReason == nil || !strings.HasPrefix(*conn.StatusReason, noLiveExecutorReasonPrefix) {
		return false
	}

	conn.StatusReason = nil

	return true
}

var noLiveExecutorReasonPrefix = strings.SplitN((&pipelineservice.NoLiveExecutorError{Group: ""}).Error(), `"`, 2)[0]
