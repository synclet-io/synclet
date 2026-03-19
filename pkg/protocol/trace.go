package protocol

// TraceType represents the type of trace message.
type TraceType string

const (
	TraceTypeError        TraceType = "ERROR"
	TraceTypeEstimate     TraceType = "ESTIMATE"
	TraceTypeStreamStatus TraceType = "STREAM_STATUS"
	TraceTypeAnalytics    TraceType = "ANALYTICS"
)

// FailureType represents the type of failure in an error trace.
type FailureType string

const (
	FailureTypeSystemError    FailureType = "system_error"
	FailureTypeConfigError    FailureType = "config_error"
	FailureTypeTransientError FailureType = "transient_error"
)

// EstimateType represents the type of estimate.
type EstimateType string

const (
	EstimateTypeStream EstimateType = "STREAM"
	EstimateTypeSync   EstimateType = "SYNC"
)

// StreamStatus represents the status of a stream.
type StreamStatus string

const (
	StreamStatusStarted    StreamStatus = "STARTED"
	StreamStatusRunning    StreamStatus = "RUNNING"
	StreamStatusComplete   StreamStatus = "COMPLETE"
	StreamStatusIncomplete StreamStatus = "INCOMPLETE"
)

// AirbyteTraceMessage represents a trace message from a connector.
type AirbyteTraceMessage struct {
	Type         TraceType                        `json:"type"`
	Error        *AirbyteErrorTraceMessage        `json:"error,omitempty"`
	Estimate     *AirbyteEstimateTraceMessage     `json:"estimate,omitempty"`
	StreamStatus *AirbyteStreamStatusTraceMessage `json:"stream_status,omitempty"`
	Analytics    *AirbyteAnalyticsTraceMessage    `json:"analytics,omitempty"`
	EmittedAt    float64                          `json:"emitted_at"`
}

// AirbyteErrorTraceMessage represents an error trace.
type AirbyteErrorTraceMessage struct {
	Message          string           `json:"message"`
	InternalMessage  string           `json:"internal_message,omitempty"`
	FailureType      FailureType      `json:"failure_type,omitempty"`
	StackTrace       string           `json:"stack_trace,omitempty"`
	StreamDescriptor *StreamDescriptor `json:"stream_descriptor,omitempty"`
}

// AirbyteEstimateTraceMessage represents a sync estimate.
type AirbyteEstimateTraceMessage struct {
	Name         string       `json:"name"`
	Type         EstimateType `json:"type"`
	Namespace    string       `json:"namespace,omitempty"`
	RowEstimate  int64        `json:"row_estimate,omitempty"`
	ByteEstimate int64        `json:"byte_estimate,omitempty"`
}

// AirbyteStreamStatusTraceMessage represents a stream status update.
type AirbyteStreamStatusTraceMessage struct {
	StreamDescriptor StreamDescriptor     `json:"stream_descriptor"`
	Status           StreamStatus         `json:"status"`
	Reasons          []StreamStatusReason `json:"reasons,omitempty"`
}

// StreamStatusReason provides additional context about why a stream has a particular status.
type StreamStatusReason struct {
	Type string `json:"type"`
}

// AirbyteAnalyticsTraceMessage represents an analytics event from a connector.
type AirbyteAnalyticsTraceMessage struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}
