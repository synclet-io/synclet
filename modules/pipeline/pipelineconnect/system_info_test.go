package pipelineconnect

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/pkg/connectutil"
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
		{
			name: "runtime defaults are not exposed",
			config: SourceHandlerConfig{RuntimeDefaults: RuntimeDefaultsInfo{
				Executor:           ExecutorK8s,
				ServiceAccountName: "synclet-sync",
			}},
			want: &pipelinev1.GetSystemInfoResponse{
				WorkspacesMode: pipelinev1.WorkspacesMode_WORKSPACES_MODE_MULTI,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SourceHandler{config: tt.config}

			resp, err := h.GetSystemInfo(context.Background(), connect.NewRequest(&pipelinev1.GetSystemInfoRequest{}))
			require.NoError(t, err)
			assert.True(t, proto.Equal(tt.want, resp.Msg), "want %v, got %v", tt.want, resp.Msg)
		})
	}
}

func TestSourceHandler_GetRuntimeDefaults(t *testing.T) {
	tests := []struct {
		name   string
		config RuntimeDefaultsInfo
		want   *pipelinev1.RuntimeDefaults
	}{
		{
			name:   "external executor",
			config: RuntimeDefaultsInfo{LogArchiveBackend: "none"},
			want:   &pipelinev1.RuntimeDefaults{LogArchiveBackend: "none"},
		},
		{
			name: "docker executor",
			config: RuntimeDefaultsInfo{
				Executor:                ExecutorDocker,
				CPULimit:                "2",
				MemoryLimit:             "2Gi",
				MaxConcurrentJobs:       10,
				IdleTimeout:             10 * time.Minute,
				LogArchiveBackend:       "s3",
				DockerMaxConcurrentJobs: 5,
			},
			want: &pipelinev1.RuntimeDefaults{
				Executor:                pipelinev1.ExecutorType_EXECUTOR_TYPE_DOCKER,
				CpuLimit:                "2",
				MemoryLimit:             "2Gi",
				MaxConcurrentJobs:       10,
				IdleTimeoutSeconds:      600,
				LogArchiveBackend:       "s3",
				DockerMaxConcurrentJobs: 5,
			},
		},
		{
			name: "k8s executor",
			config: RuntimeDefaultsInfo{
				Executor:           ExecutorK8s,
				CPURequest:         "500m",
				CPULimit:           "2",
				MemoryRequest:      "512Mi",
				MemoryLimit:        "2Gi",
				ServiceAccountName: "synclet-sync",
				LogArchiveBackend:  "gcs",
			},
			want: &pipelinev1.RuntimeDefaults{
				Executor:           pipelinev1.ExecutorType_EXECUTOR_TYPE_K8S,
				CpuRequest:         "500m",
				CpuLimit:           "2",
				MemoryRequest:      "512Mi",
				MemoryLimit:        "2Gi",
				ServiceAccountName: "synclet-sync",
				LogArchiveBackend:  "gcs",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SourceHandler{config: SourceHandlerConfig{RuntimeDefaults: tt.config}}

			resp, err := h.GetRuntimeDefaults(context.Background(), connect.NewRequest(&pipelinev1.GetRuntimeDefaultsRequest{}))
			require.NoError(t, err)
			assert.True(t, proto.Equal(tt.want, resp.Msg.GetRuntimeDefaults()), "want %v, got %v", tt.want, resp.Msg.GetRuntimeDefaults())
		})
	}
}

type fakeTokenValidator struct {
	userID uuid.UUID
}

func (v fakeTokenValidator) ValidateAccessToken(_ string) (userID, email string, err error) {
	return v.userID.String(), "user@example.com", nil
}

func (v fakeTokenValidator) ValidateAPIKey(_ context.Context, _ string) (userID, workspaceID string, err error) {
	return "", "", connectutil.ErrInvalidCredential
}

type fakeMembershipChecker struct {
	role string
}

func (c fakeMembershipChecker) IsMember(_ context.Context, _, _ uuid.UUID) (bool, error) {
	return c.role != "", nil
}

func (c fakeMembershipChecker) GetMemberRole(_ context.Context, _, _ uuid.UUID) (string, error) {
	if c.role == "" {
		return "", connectutil.ErrNotMember
	}

	return c.role, nil
}

func TestGetRuntimeDefaults_RequiresWorkspaceMember(t *testing.T) {
	tests := []struct {
		name     string
		token    string
		role     string
		wantCode connect.Code
	}{
		{name: "unauthenticated", wantCode: connect.CodeUnauthenticated},
		{name: "not a workspace member", token: "token", wantCode: connect.CodePermissionDenied},
		{name: "viewer", token: "token", role: "Viewer"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SourceHandler{config: SourceHandlerConfig{RuntimeDefaults: RuntimeDefaultsInfo{Executor: ExecutorDocker}}}
			path, handler := pipelinev1connect.NewSourceServiceHandler(h, connect.WithInterceptors(
				connectutil.NewAuthInterceptor(fakeTokenValidator{userID: uuid.New()}),
				connectutil.NewRoleInterceptor(fakeMembershipChecker{role: tt.role}),
			))
			mux := http.NewServeMux()
			mux.Handle(path, handler)
			srv := httptest.NewServer(mux)
			t.Cleanup(srv.Close)

			req := connect.NewRequest(&pipelinev1.GetRuntimeDefaultsRequest{})
			req.Header().Set("Workspace-Id", uuid.NewString())

			if tt.token != "" {
				req.Header().Set("Authorization", "Bearer "+tt.token)
			}

			resp, err := pipelinev1connect.NewSourceServiceClient(srv.Client(), srv.URL).GetRuntimeDefaults(context.Background(), req)
			if tt.wantCode != 0 {
				require.Error(t, err)
				assert.Equal(t, tt.wantCode, connect.CodeOf(err))

				return
			}

			require.NoError(t, err)
			assert.Equal(t, pipelinev1.ExecutorType_EXECUTOR_TYPE_DOCKER, resp.Msg.GetRuntimeDefaults().GetExecutor())
		})
	}
}
