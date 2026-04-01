package pipelineroute

import (
	"context"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// DockerHandler implements Handler for Docker mode sync execution.
// It persists confirmed state via SaveSyncState and handles CONTROL config
// updates by delegating to the HandleConfigUpdate use case.
type DockerHandler struct {
	DefaultHandler
	connID             uuid.UUID
	sourceID           uuid.UUID
	destID             uuid.UUID
	jobID              uuid.UUID
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate
	saveSyncState      *pipelinestate.SaveSyncState
	appendJobLog       *pipelinelogs.AppendJobLog
	logger             *logging.Logger
}

// DockerHandlerParams holds constructor parameters for DockerHandler.
type DockerHandlerParams struct {
	ConnectionID       uuid.UUID
	SourceID           uuid.UUID
	DestinationID      uuid.UUID
	JobID              uuid.UUID
	HandleConfigUpdate *pipelinejobs.HandleConfigUpdate
	SaveSyncState      *pipelinestate.SaveSyncState
	AppendJobLog       *pipelinelogs.AppendJobLog
	Logger             *logging.Logger
}

// NewDockerHandler creates a new DockerHandler.
func NewDockerHandler(params DockerHandlerParams) *DockerHandler {
	return &DockerHandler{
		connID:             params.ConnectionID,
		sourceID:           params.SourceID,
		destID:             params.DestinationID,
		jobID:              params.JobID,
		handleConfigUpdate: params.HandleConfigUpdate,
		saveSyncState:      params.SaveSyncState,
		appendJobLog:       params.AppendJobLog,
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

// OnLog appends a pre-formatted log line to the job's log storage.
// Errors are non-fatal to avoid disrupting sync execution.
func (h *DockerHandler) OnLog(ctx context.Context, line string) error {
	if err := h.appendJobLog.Execute(ctx, pipelinelogs.AppendJobLogParams{
		JobID:   h.jobID,
		LogLine: line,
	}); err != nil {
		h.logger.WithError(err).Error(ctx, "failed to append log")
	}

	return nil // non-fatal
}
