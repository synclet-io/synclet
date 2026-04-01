package pipelineroute

import (
	"fmt"

	"github.com/synclet-io/synclet/pkg/protocol"
)

// formatLogLine formats an Airbyte LOG message into a pre-formatted text line.
func formatLogLine(prefix string, msg *protocol.AirbyteLogMessage) string {
	return fmt.Sprintf("%s %s: %s", prefix, msg.Level, msg.Message)
}

// formatTraceLine formats an Airbyte TRACE message into a pre-formatted text line.
func formatTraceLine(prefix string, msg *protocol.AirbyteTraceMessage) string {
	if msg.Type == protocol.TraceTypeError && msg.Error != nil {
		return fmt.Sprintf("%s ERROR: %s", prefix, msg.Error.Message)
	}

	if msg.Type == protocol.TraceTypeAnalytics && msg.Analytics != nil {
		return fmt.Sprintf("%s TRACE: %s = %s", prefix, msg.Analytics.Type, msg.Analytics.Value)
	}

	return fmt.Sprintf("%s TRACE: %s", prefix, msg.Type)
}
