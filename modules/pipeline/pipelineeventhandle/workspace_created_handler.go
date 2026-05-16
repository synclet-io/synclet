// Package pipelineeventhandle contains Watermill handlers that react to
// inter-module events. Handlers here treat the producing module's
// {module}event package as a public contract; they never reach into the
// producer's storage or service packages directly.
package pipelineeventhandle

import (
	"context"
	"fmt"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinerepositories"
	"github.com/synclet-io/synclet/modules/workspace/workspaceevent"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
)

// defaultRepositoriesCreator is the narrow port the handler depends on. The
// real implementation is pipelinerepositories.CreateDefaultRepositories;
// tests substitute a stub.
type defaultRepositoriesCreator interface {
	Execute(ctx context.Context, workspaceID uuid.UUID) error
}

// WorkspaceCreatedHandler reacts to workspace.created events by seeding the
// new workspace with the default connector registries. The handler is
// idempotent: replays leave the workspace unchanged once the registries are
// already present.
type WorkspaceCreatedHandler struct {
	logger      *logging.Logger
	createRepos defaultRepositoriesCreator
}

// Compile-time assertions that the handler satisfies the pnpwatermill
// contract and that the real use case satisfies the local port.
var (
	_ pnpwatermill.Handler       = (*WorkspaceCreatedHandler)(nil)
	_ defaultRepositoriesCreator = (*pipelinerepositories.CreateDefaultRepositories)(nil)
)

// NewWorkspaceCreatedHandler creates the handler. Wired via
// pnpwatermill.HandlerProvider in the pipeline FX module.
func NewWorkspaceCreatedHandler(
	logger *logging.Logger,
	createRepos *pipelinerepositories.CreateDefaultRepositories,
) *WorkspaceCreatedHandler {
	return &WorkspaceCreatedHandler{
		logger:      logger.Named("workspace-created-handler"),
		createRepos: createRepos,
	}
}

// Name returns the handler identifier used by pnpwatermill for the
// subscriber. Must be unique per process.
func (h *WorkspaceCreatedHandler) Name() string {
	return "pipeline_workspace_created"
}

// Topic returns the topic this handler subscribes to. Topic constants live in
// the producing module's event package.
func (h *WorkspaceCreatedHandler) Topic() string {
	return workspaceevent.WorkspaceEventTopic
}

// Handle decodes the message and, for workspace.created events, seeds the
// default registries. Other event types on the same topic are intentionally
// no-ops so this handler can coexist with future subscribers.
//
// Decode failures and malformed payloads are logged and acked — replaying
// them would be useless. Storage failures inside the use case (e.g. database
// unreachable) are returned so Watermill retries the delivery; per-registry
// network failures are already swallowed inside the use case.
func (h *WorkspaceCreatedHandler) Handle(msg *message.Message) error {
	ctx := msg.Context()

	event, err := workspaceevent.DecodeWorkspaceEvent(msg.Metadata, msg.Payload)
	if err != nil {
		h.logger.WithError(err).Error(ctx, "cannot decode workspace event; acking and dropping")

		return nil
	}

	if event.EventType != workspaceservice.WorkspaceEventTypeCreated {
		return nil
	}

	data, ok := event.Data.(*workspaceservice.WorkspaceCreatedEventData)
	if !ok || data == nil || data.Workspace == nil {
		h.logger.Error(ctx, "workspace.created event missing workspace payload; acking and dropping")

		return nil
	}

	if err := h.createRepos.Execute(ctx, data.Workspace.ID); err != nil {
		h.logger.
			WithError(err).
			WithField("workspace_id", data.Workspace.ID.String()).
			Error(ctx, "failed to seed default registries")

		return fmt.Errorf("seeding default registries: %w", err)
	}

	return nil
}
