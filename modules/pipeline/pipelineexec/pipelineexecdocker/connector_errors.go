package pipelineexecdocker

import (
	"fmt"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// connectorError represents an error reported by a connector via a TRACE error message.
type connectorError struct {
	// Message is the user-facing error message from the connector.
	Message string
	// FailureType indicates whether the error is a config error or a system error.
	FailureType protocol.FailureType
}

func (e *connectorError) Error() string {
	if e.FailureType != "" {
		return fmt.Sprintf("connector error (%s): %s", e.FailureType, e.Message)
	}

	return "connector error: " + e.Message
}
