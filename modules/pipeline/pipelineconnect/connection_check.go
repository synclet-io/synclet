package pipelineconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

const connectionCheckTimeout = 2 * time.Minute

// runConnectionCheck creates a check task with raw config and blocks until completion.
// Returns nil on success, or a ConnectRPC error on failure/timeout.
func runConnectionCheck(
	ctx context.Context,
	createCheckTask *pipelinetasks.CreateCheckTask,
	waitForResult *pipelinetasks.WaitForTaskResult,
	workspaceID uuid.UUID,
	managedConnectorID uuid.UUID,
	config json.RawMessage,
) error {
	checkCtx, cancel := context.WithTimeout(ctx, connectionCheckTimeout)
	defer cancel()

	mcID := managedConnectorID
	taskResult, err := createCheckTask.Execute(checkCtx, pipelinetasks.CreateCheckTaskParams{
		WorkspaceID:        workspaceID,
		ManagedConnectorID: &mcID,
		Config:             config,
	})
	if err != nil {
		return connect.NewError(connect.CodeInternal,
			fmt.Errorf("connection check setup failed: %w", err))
	}

	result, err := waitForResult.Execute(checkCtx, pipelinetasks.WaitForTaskResultParams{
		TaskID:      taskResult.TaskID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		if checkCtx.Err() != nil {
			return connect.NewError(connect.CodeDeadlineExceeded,
				fmt.Errorf("connection check timed out"))
		}
		return connect.NewError(connect.CodeInternal,
			fmt.Errorf("connection check failed: %w", err))
	}

	if result.Status == pipelineservice.ConnectorTaskStatusFailed {
		msg := result.ErrorMessage
		if msg == "" {
			msg = "connection check failed"
		}
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%s", msg))
	}

	// Extract typed check result for success/failure details.
	if cr, ok := result.Result.(*pipelineservice.CheckResult); ok && !cr.Success {
		msg := cr.Message
		if msg == "" {
			msg = "connection check failed"
		}
		return connect.NewError(connect.CodeFailedPrecondition, fmt.Errorf("%s", msg))
	}

	return nil
}
