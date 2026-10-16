package notifyadapt

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
)

// ConnectionChecker bridges the notify module's ConnectionChecker port to the
// pipeline GetConnection use case.
type ConnectionChecker struct {
	getConnection *pipelineconnections.GetConnection
}

// NewConnectionChecker creates the adapter.
func NewConnectionChecker(getConnection *pipelineconnections.GetConnection) *ConnectionChecker {
	return &ConnectionChecker{getConnection: getConnection}
}

var _ notifyservice.ConnectionChecker = (*ConnectionChecker)(nil)

// ConnectionExists reports whether the connection exists in the workspace.
func (c *ConnectionChecker) ConnectionExists(ctx context.Context, workspaceID, connectionID uuid.UUID) (bool, error) {
	_, err := c.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          connectionID,
		WorkspaceID: workspaceID,
	})

	var notFound pipelineservice.NotFoundError
	if errors.As(err, &notFound) {
		return false, nil
	}

	if err != nil {
		return false, fmt.Errorf("getting connection: %w", err)
	}

	return true, nil
}
