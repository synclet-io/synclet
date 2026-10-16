package pipelineconnect

import (
	"context"

	"connectrpc.com/connect"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
)

// SourceHandlerConfig holds configuration values exposed via GetSystemInfo.
type SourceHandlerConfig struct {
	RegistrationEnabled bool
	SingleWorkspaceMode bool
	EmailEnabled        bool
}

func (h *SourceHandler) GetSystemInfo(_ context.Context, _ *connect.Request[pipelinev1.GetSystemInfoRequest]) (*connect.Response[pipelinev1.GetSystemInfoResponse], error) {
	workspacesMode := pipelinev1.WorkspacesMode_WORKSPACES_MODE_MULTI
	if h.config.SingleWorkspaceMode {
		workspacesMode = pipelinev1.WorkspacesMode_WORKSPACES_MODE_SINGLE
	}

	return connect.NewResponse(&pipelinev1.GetSystemInfoResponse{
		RegistrationEnabled: h.config.RegistrationEnabled,
		WorkspacesMode:      workspacesMode,
		EmailEnabled:        h.config.EmailEnabled,
	}), nil
}
