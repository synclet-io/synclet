package pipelineexecdocker

import (
	"context"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

const (
	defaultLogBatchSize     = 500
	defaultLogFlushInterval = time.Second
	logFlushTimeout         = 10 * time.Second
)

// DockerHandler implements Handler for Docker mode sync execution. It reports
// confirmed state and CONTROL config updates through the SyncReporter. Log
// lines are buffered and reported in batches, in the order they arrived.
type DockerHandler struct {
	pipelineexec.DefaultHandler
	connID           uuid.UUID
	sourceID         uuid.UUID
	destID           uuid.UUID
	jobID            uuid.UUID
	reporter         pipelineexec.SyncReporter
	logBatchSize     int
	logFlushInterval time.Duration
	logger           *logging.Logger

	logMu  sync.Mutex
	logBuf []string
}

// DockerHandlerParams holds constructor parameters for DockerHandler.
type DockerHandlerParams struct {
	ConnectionID     uuid.UUID
	SourceID         uuid.UUID
	DestinationID    uuid.UUID
	JobID            uuid.UUID
	Reporter         pipelineexec.SyncReporter
	LogBatchSize     int
	LogFlushInterval time.Duration
	Logger           *logging.Logger
}

// NewDockerHandler creates a new DockerHandler.
func NewDockerHandler(params DockerHandlerParams) *DockerHandler {
	batchSize := params.LogBatchSize
	if batchSize <= 0 {
		batchSize = defaultLogBatchSize
	}

	flushInterval := params.LogFlushInterval
	if flushInterval <= 0 {
		flushInterval = defaultLogFlushInterval
	}

	return &DockerHandler{
		connID:           params.ConnectionID,
		sourceID:         params.SourceID,
		destID:           params.DestinationID,
		jobID:            params.JobID,
		reporter:         params.Reporter,
		logBatchSize:     batchSize,
		logFlushInterval: flushInterval,
		logger:           params.Logger.Named("docker-handler"),
	}
}

// OnStateConfirmed reports confirmed state to the control plane.
func (h *DockerHandler) OnStateConfirmed(ctx context.Context, msg *protocol.AirbyteStateMessage) error {
	return h.reporter.ReportState(ctx, h.connID, h.jobID, msg)
}

// OnSourceControl reports CONTROL config updates from the source connector;
// the control plane encrypts and persists them. A failed report is logged
// and the sync goes on.
func (h *DockerHandler) OnSourceControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error {
	if msg.Type != protocol.ControlMessageTypeConnectorConfig || msg.ConnectorConfig == nil {
		return nil
	}

	if err := h.reporter.ReportConfigUpdate(ctx, pipelineservice.ConnectorTypeSource, h.sourceID, msg.ConnectorConfig.Config); err != nil {
		h.logger.WithError(err).Error(ctx, "docker handler: failed to update source config from CONTROL message")
	}

	return nil
}

// OnDestControl reports CONTROL config updates from the destination connector;
// a failed report is logged and the sync goes on.
func (h *DockerHandler) OnDestControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error {
	if msg.Type != protocol.ControlMessageTypeConnectorConfig || msg.ConnectorConfig == nil {
		return nil
	}

	if err := h.reporter.ReportConfigUpdate(ctx, pipelineservice.ConnectorTypeDestination, h.destID, msg.ConnectorConfig.Config); err != nil {
		h.logger.WithError(err).Error(ctx, "docker handler: failed to update destination config from CONTROL message")
	}

	return nil
}

// OnSourceTrace logs error traces from the source connector.
func (h *DockerHandler) OnSourceTrace(ctx context.Context, msg *protocol.AirbyteTraceMessage) error {
	if msg.Error != nil {
		h.logger.WithFields(map[string]interface{}{"message": msg.Error.Message, "failure_type": msg.Error.FailureType}).Error(ctx, "source trace error")
	}

	return nil
}

// OnLog buffers a pre-formatted log line and reports the buffer once it
// holds a full batch. Errors are non-fatal to avoid disrupting sync execution.
func (h *DockerHandler) OnLog(ctx context.Context, line string) error {
	h.logMu.Lock()
	defer h.logMu.Unlock()

	h.logBuf = append(h.logBuf, line)
	if len(h.logBuf) >= h.logBatchSize {
		h.flushLogsLocked(ctx)
	}

	return nil
}

// StartLogFlush reports buffered log lines every flush interval until the
// returned stop function is called. Stop reports whatever is still buffered
// and returns once it is stored, so it must run when the attempt ends,
// whatever the outcome. Reports outlive ctx cancellation.
func (h *DockerHandler) StartLogFlush(ctx context.Context) (stop func()) {
	stopCh := make(chan struct{})
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(h.logFlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				h.flushLogs(ctx)
			case <-stopCh:
				return
			}
		}
	}()

	return func() {
		close(stopCh)
		<-done
		h.flushLogs(ctx)
	}
}

func (h *DockerHandler) flushLogs(ctx context.Context) {
	h.logMu.Lock()
	defer h.logMu.Unlock()

	h.flushLogsLocked(ctx)
}

// flushLogsLocked reports the buffer as one batch. Holding logMu across the
// report keeps batches in arrival order.
func (h *DockerHandler) flushLogsLocked(ctx context.Context) {
	if len(h.logBuf) == 0 {
		return
	}

	lines := h.logBuf
	h.logBuf = nil

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), logFlushTimeout)
	defer cancel()

	if err := h.reporter.ReportLog(ctx, h.jobID, lines); err != nil {
		h.logger.WithError(err).WithField("lines", len(lines)).Error(ctx, "failed to report logs")
	}
}
