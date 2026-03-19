package protocol

// LogLevel represents the severity level of a log message.
type LogLevel string

const (
	LogLevelFatal LogLevel = "FATAL"
	LogLevelError LogLevel = "ERROR"
	LogLevelWarn  LogLevel = "WARN"
	LogLevelInfo  LogLevel = "INFO"
	LogLevelDebug LogLevel = "DEBUG"
	LogLevelTrace LogLevel = "TRACE"
)

// AirbyteLogMessage represents a log entry from a connector.
type AirbyteLogMessage struct {
	Level      LogLevel `json:"level"`
	Message    string   `json:"message"`
	StackTrace string   `json:"stack_trace,omitempty"`
}
