package pipelineadapt

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/synclet-io/synclet/pkg/connector"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ConnectorCheckAdapter adapts connector.ConnectorClient to the Check-only
// interface expected by source and destination services.
type ConnectorCheckAdapter struct {
	client *connector.ConnectorClient
}

// NewConnectorCheckAdapter creates a new ConnectorCheckAdapter.
func NewConnectorCheckAdapter(client *connector.ConnectorClient) *ConnectorCheckAdapter {
	return &ConnectorCheckAdapter{client: client}
}

func (a *ConnectorCheckAdapter) Check(ctx context.Context, image string, config json.RawMessage) error {
	status, err := a.client.Check(ctx, image, config)
	if err != nil {
		return err
	}
	if status.Status != protocol.ConnectionStatusSucceeded {
		return fmt.Errorf("connector check failed: %s", status.Message)
	}
	return nil
}
