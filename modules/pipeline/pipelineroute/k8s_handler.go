package pipelineroute

import (
	"context"
	"log/slog"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// K8sReporter abstracts the reporter methods needed by K8sHandler.
// This avoids importing pkg/orchestrator from modules/pipeline.
type K8sReporter interface {
	QueueState(msg *protocol.AirbyteStateMessage)
	QueueConfigUpdate(connectorType executorv1.ConnectorType, connectorID string, config []byte)
	QueueLog(line string)
}

// K8sHandler implements Handler for K8s mode sync execution.
// It forwards state confirmations and config updates to the server via the reporter.
type K8sHandler struct {
	DefaultHandler // embed for OnSourceTrace no-op
	logger         *slog.Logger
	reporter       K8sReporter
	sourceID       string
	destID         string
}

// K8sHandlerParams holds constructor parameters for K8sHandler.
type K8sHandlerParams struct {
	Logger        *slog.Logger
	Reporter      K8sReporter
	SourceID      string // UUID string of source
	DestinationID string // UUID string of destination
}

// NewK8sHandler creates a new K8sHandler.
func NewK8sHandler(params K8sHandlerParams) *K8sHandler {
	return &K8sHandler{
		logger:   params.Logger,
		reporter: params.Reporter,
		sourceID: params.SourceID,
		destID:   params.DestinationID,
	}
}

// OnStateConfirmed forwards confirmed state to the server via the reporter.
func (h *K8sHandler) OnStateConfirmed(_ context.Context, msg *protocol.AirbyteStateMessage) error {
	h.reporter.QueueState(msg)
	return nil
}

// OnSourceControl handles CONTROL config updates from the source connector.
func (h *K8sHandler) OnSourceControl(_ context.Context, msg *protocol.AirbyteControlMessage) error {
	if msg.Type != protocol.ControlMessageTypeConnectorConfig || msg.ConnectorConfig == nil {
		return nil
	}
	h.reporter.QueueConfigUpdate(executorv1.ConnectorType_CONNECTOR_TYPE_SOURCE, h.sourceID, msg.ConnectorConfig.Config)
	return nil
}

// OnDestControl handles CONTROL config updates from the destination connector.
func (h *K8sHandler) OnDestControl(_ context.Context, msg *protocol.AirbyteControlMessage) error {
	if msg.Type != protocol.ControlMessageTypeConnectorConfig || msg.ConnectorConfig == nil {
		return nil
	}
	h.reporter.QueueConfigUpdate(executorv1.ConnectorType_CONNECTOR_TYPE_DESTINATION, h.destID, msg.ConnectorConfig.Config)
	return nil
}

// OnLog forwards a pre-formatted log line to the K8s reporter for async delivery.
func (h *K8sHandler) OnLog(_ context.Context, line string) error {
	h.reporter.QueueLog(line)
	return nil
}
