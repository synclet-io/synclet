package pipelineadapt

import (
	"context"
	"encoding/json"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexecdocker"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ConnectorDiscoverAdapter adapts pipelineexecdocker.ConnectorClient to the Discover-only
// interface expected by the catalog service.
type ConnectorDiscoverAdapter struct {
	client *pipelineexecdocker.ConnectorClient
}

// NewConnectorDiscoverAdapter creates a new ConnectorDiscoverAdapter.
func NewConnectorDiscoverAdapter(client *pipelineexecdocker.ConnectorClient) *ConnectorDiscoverAdapter {
	return &ConnectorDiscoverAdapter{client: client}
}

func (a *ConnectorDiscoverAdapter) Discover(ctx context.Context, image string, config json.RawMessage) (*protocol.AirbyteCatalog, error) {
	return a.client.Discover(ctx, image, config)
}
