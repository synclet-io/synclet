package pipelineadapt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/synclet-io/synclet/pkg/connector"
)

// ConnectorSpecFetcherAdapter adapts connector.ConnectorClient.Spec to return
// a JSON string instead of *protocol.ConnectorSpecification, keeping
// protocol types out of the pipeline service layer.
type ConnectorSpecFetcherAdapter struct {
	client *connector.ConnectorClient
}

// NewConnectorSpecFetcherAdapter creates a new ConnectorSpecFetcherAdapter.
func NewConnectorSpecFetcherAdapter(client *connector.ConnectorClient) *ConnectorSpecFetcherAdapter {
	return &ConnectorSpecFetcherAdapter{client: client}
}

func (a *ConnectorSpecFetcherAdapter) Spec(ctx context.Context, image string) (string, error) {
	spec, err := a.client.Spec(ctx, image)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(spec)
	if err != nil {
		return "", fmt.Errorf("marshaling connector spec: %w", err)
	}

	return string(data), nil
}
