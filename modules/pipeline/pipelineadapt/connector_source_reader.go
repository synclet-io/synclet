package pipelineadapt

import (
	"context"
	"encoding/json"
	"io"

	"github.com/synclet-io/synclet/pkg/connector"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ConnectorSourceReader adapts connector.ConnectorClient to pipelineservice.SourceReader.
type ConnectorSourceReader struct {
	client *connector.ConnectorClient
}

// NewConnectorSourceReader creates a new ConnectorSourceReader.
func NewConnectorSourceReader(client *connector.ConnectorClient) *ConnectorSourceReader {
	return &ConnectorSourceReader{client: client}
}

// SetResourceLimits sets memory and CPU limits for subsequent Read calls.
func (a *ConnectorSourceReader) SetResourceLimits(memoryLimit int64, cpuLimit float64) {
	a.client.SetResourceLimits(memoryLimit, cpuLimit)
}

// Read adapts connector.ConnectorClient to pipelineservice.SourceReader.
// State is passed through as a raw JSON blob — connectors receive the full state.
func (a *ConnectorSourceReader) Read(ctx context.Context, image string, config json.RawMessage, catalog *protocol.ConfiguredAirbyteCatalog, state json.RawMessage, labels map[string]string) (io.ReadCloser, func(), error) {
	return a.client.Read(ctx, image, config, catalog, state, labels)
}
