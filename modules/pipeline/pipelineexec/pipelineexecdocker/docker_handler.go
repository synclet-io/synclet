package pipelineexecdocker

import (
	"context"
	"sync"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/protocol"
)

const (
	defaultLogBatchSize     = 500
	defaultLogFlushInterval = time.Second
	logFlushTimeout         = 10 * time.Second
)

// DockerHandler implements Handler for Docker mode sync execution.
// It persists confirmed state via SaveSyncState and handles CONTROL config
// updates by delegating to the HandleConfigUpdate use case. Log lines are
// buffered and appended in batches, in the order they arrived.
type DockerHandler struct {
	pipelineexec.DefaultHandler
	connID             uuid.UUID
	sourceID           uuid.UUID
	destID             uuid.UUID
	jobID              uuid.UUID
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate
	saveSyncState      *pipelinestate.SaveSyncState
	appendJobLogs      logAppender
	logBatchSize       int
	logFlushInterval   time.Duration
	logger             *logging.Logger

	logMu  sync.Mutex
	logBuf []string
}

// DockerHandlerParams holds constructor parameters for DockerHandler.
type DockerHandlerParams struct {
	ConnectionID       uuid.UUID
	SourceID           uuid.UUID
	DestinationID      uuid.UUID
	JobID              uuid.UUID
	HandleConfigUpdate *pipelinejobs.HandleConfigUpdate
	SaveSyncState      *pipelinestate.SaveSyncState
	AppendJobLogs      logAppender
	LogBatchSize       int
	LogFlushInterval   time.Duration
	Logger             *logging.Logger
}

type logAppender interface {
	Execute(ctx context.Context, params pipelinelogs.BatchAppendJobLogsParams) error
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
		connID:             params.ConnectionID,
		sourceID:           params.SourceID,
		destID:             params.DestinationID,
		jobID:              params.JobID,
		handleConfigUpdate: params.HandleConfigUpdate,
		saveSyncState:      params.SaveSyncState,
		appendJobLogs:      params.AppendJobLogs,
		logBatchSize:       batchSize,
		logFlushInterval:   flushInterval,
		logger:             params.Logger.Named("docker-handler"),
	}
}

// OnStateConfirmed persists confirmed state via the SaveSyncState use case.
func (h *DockerHandler) OnStateConfirmed(ctx context.Context, msg *protocol.AirbyteStateMessage) error {
	return h.saveSyncState.Execute(ctx, pipelinestate.SaveSyncStateParams{
		ConnectionID: h.connID,
		StateMessage: msg,
	})
}

// OnSourceControl handles CONTROL config updates from the source connector.
// It delegates to the HandleConfigUpdate use case for secret encryption and persistence.
func (h *DockerHandler) OnSourceControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error {
	if msg.Type != protocol.ControlMessageTypeConnectorConfig || msg.ConnectorConfig == nil {
		return nil
	}

	if err := h.handleConfigUpdate.Execute(ctx, pipelinejobs.HandleConfigUpdateParams{
		ConnectorType: pipelineservice.ConnectorTypeSource,
		ConnectorID:   h.sourceID,
		Config:        msg.ConnectorConfig.Config,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "docker handler: failed to update source config from CONTROL message")
	}

	return nil // non-fatal per existing behavior
}

// OnDestControl handles CONTROL config updates from the destination connector.
// It delegates to the HandleConfigUpdate use case for secret encryption and persistence.
func (h *DockerHandler) OnDestControl(ctx context.Context, msg *protocol.AirbyteControlMessage) error {
	if msg.Type != protocol.ControlMessageTypeConnectorConfig || msg.ConnectorConfig == nil {
		return nil
	}

	if err := h.handleConfigUpdate.Execute(ctx, pipelinejobs.HandleConfigUpdateParams{
		ConnectorType: pipelineservice.ConnectorTypeDestination,
		ConnectorID:   h.destID,
		Config:        msg.ConnectorConfig.Config,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "docker handler: failed to update destination config from CONTROL message")
	}

	return nil // non-fatal per existing behavior
}

// OnSourceTrace logs error traces from the source connector.
func (h *DockerHandler) OnSourceTrace(ctx context.Context, msg *protocol.AirbyteTraceMessage) error {
	if msg.Error != nil {
		h.logger.WithFields(map[string]interface{}{"message": msg.Error.Message, "failure_type": msg.Error.FailureType}).Error(ctx, "source trace error")
	}

	return nil
}

// OnLog buffers a pre-formatted log line and appends the buffer once it
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

// StartLogFlush appends buffered log lines every flush interval until the
// returned stop function is called. Stop appends whatever is still buffered
// and returns once it is stored, so it must run when the attempt ends,
// whatever the outcome. Appends outlive ctx cancellation.
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

// flushLogsLocked appends the buffer as one batch. Holding logMu across the
// append keeps batches in arrival order.
func (h *DockerHandler) flushLogsLocked(ctx context.Context) {
	if len(h.logBuf) == 0 {
		return
	}

	lines := h.logBuf
	h.logBuf = nil

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), logFlushTimeout)
	defer cancel()

	if err := h.appendJobLogs.Execute(ctx, pipelinelogs.BatchAppendJobLogsParams{
		JobID:    h.jobID,
		LogLines: lines,
	}); err != nil {
		h.logger.WithError(err).WithField("lines", len(lines)).Error(ctx, "failed to append logs")
	}
}
