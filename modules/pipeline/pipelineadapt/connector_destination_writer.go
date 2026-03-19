package pipelineadapt

import (
	"context"
	"encoding/json"
	"io"

	"github.com/synclet-io/synclet/pkg/connector"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ConnectorDestinationWriter adapts connector.ConnectorClient to pipelineservice.DestinationWriter.
type ConnectorDestinationWriter struct {
	client *connector.ConnectorClient
}

// NewConnectorDestinationWriter creates a new ConnectorDestinationWriter.
func NewConnectorDestinationWriter(client *connector.ConnectorClient) *ConnectorDestinationWriter {
	return &ConnectorDestinationWriter{client: client}
}

// SetResourceLimits sets memory and CPU limits for subsequent Write calls.
func (a *ConnectorDestinationWriter) SetResourceLimits(memoryLimit int64, cpuLimit float64) {
	a.client.SetResourceLimits(memoryLimit, cpuLimit)
}

func (a *ConnectorDestinationWriter) Write(ctx context.Context, image string, config json.RawMessage, catalog *protocol.ConfiguredAirbyteCatalog, stdin io.Reader, labels map[string]string) (io.ReadCloser, func(), error) {
	return a.client.Write(ctx, image, config, catalog, stdin, labels)
}
