package notifyeventhandle

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceevent"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

type workspacePurger interface {
	Execute(ctx context.Context, workspaceID uuid.UUID) error
}

// WorkspaceDeletedHandler reacts to workspace.deleted events by deleting the
// workspace's notification channels, rules and webhooks. The purge is
// idempotent, so redelivered events are harmless.
type WorkspaceDeletedHandler struct {
	logger *logging.Logger
	purge  workspacePurger
}

var (
	_ pnpwatermill.Handler = (*WorkspaceDeletedHandler)(nil)
	_ workspacePurger      = (*notifyservice.PurgeWorkspace)(nil)
)

// NewWorkspaceDeletedHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the notify FX module.
func NewWorkspaceDeletedHandler(logger *logging.Logger, purge *notifyservice.PurgeWorkspace) *WorkspaceDeletedHandler {
	return &WorkspaceDeletedHandler{
		logger: logger.Named("workspace-deleted-handler"),
		purge:  purge,
	}
}

func (h *WorkspaceDeletedHandler) Name() string {
	return "notify_workspace_deleted"
}

func (h *WorkspaceDeletedHandler) Topic() string {
	return workspaceevent.WorkspaceEventTopic
}

// Handle purges the deleted workspace's notification data. Undecodable
// messages are acked; purge failures are returned so the bus retries the
// delivery.
func (h *WorkspaceDeletedHandler) Handle(msg *message.Message) error {
	ctx := msg.Context()

	event, err := workspaceevent.DecodeWorkspaceEvent(msg.Metadata, msg.Payload)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "cannot decode workspace event; acking and dropping")

		return nil
	}

	if event.EventType != workspaceservice.WorkspaceEventTypeDeleted {
		return nil
	}

	data, ok := event.Data.(*workspaceservice.WorkspaceDeletedEventData)
	if !ok || data == nil || data.Workspace == nil {
		h.logger.Error(ctx, "workspace.deleted event missing workspace payload; acking and dropping")

		return nil
	}

	if err := h.purge.Execute(ctx, data.Workspace.ID); err != nil {
		return fmt.Errorf("purging notification data of workspace %s: %w", data.Workspace.ID, err)
	}

	return nil
}
