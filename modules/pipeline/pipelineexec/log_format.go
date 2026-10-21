package pipelineexec

import (
	"fmt"
	"strings"

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

// MaxLogLineSize bounds a connector log line handed to a Handler.
const MaxLogLineSize = 64 << 10

func truncateLogLine(line string) string {
	if len(line) <= MaxLogLineSize {
		return line
	}

	kept := strings.ToValidUTF8(line[:MaxLogLineSize], "")

	return fmt.Sprintf("%s... (%d bytes truncated)", kept, len(line)-len(kept))
}
