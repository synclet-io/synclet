package pipelineconnect

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
)

func TestSourceHandler_GetSystemInfo(t *testing.T) {
	tests := []struct {
		name   string
		config SourceHandlerConfig
		want   *pipelinev1.GetSystemInfoResponse
	}{
		{
			name:   "multi workspace without email",
			config: SourceHandlerConfig{RegistrationEnabled: true},
			want: &pipelinev1.GetSystemInfoResponse{
				RegistrationEnabled: true,
				WorkspacesMode:      pipelinev1.WorkspacesMode_WORKSPACES_MODE_MULTI,
			},
		},
		{
			name:   "single workspace with email",
			config: SourceHandlerConfig{SingleWorkspaceMode: true, EmailEnabled: true},
			want: &pipelinev1.GetSystemInfoResponse{
				WorkspacesMode: pipelinev1.WorkspacesMode_WORKSPACES_MODE_SINGLE,
				EmailEnabled:   true,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SourceHandler{config: tt.config}

			resp, err := h.GetSystemInfo(context.Background(), connect.NewRequest(&pipelinev1.GetSystemInfoRequest{}))
			require.NoError(t, err)
			assert.Equal(t, tt.want.GetRegistrationEnabled(), resp.Msg.GetRegistrationEnabled())
			assert.Equal(t, tt.want.GetWorkspacesMode(), resp.Msg.GetWorkspacesMode())
			assert.Equal(t, tt.want.GetEmailEnabled(), resp.Msg.GetEmailEnabled())
		})
	}
}
