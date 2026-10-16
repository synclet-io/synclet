package notifyservice

import (
	"context"

	"github.com/google/uuid"
)

// ConnectionChecker is the port the notify module uses to confirm that a
// pipeline connection exists in a workspace. The implementation lives in
// modules/notify/notifyadapt.
type ConnectionChecker interface {
	ConnectionExists(ctx context.Context, workspaceID, connectionID uuid.UUID) (bool, error)
}
