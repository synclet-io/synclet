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
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

type staticRegistrationStatus bool

func (s staticRegistrationStatus) RegistrationOpen(context.Context) (bool, error) {
	return bool(s), nil
}

func TestSourceHandler_GetSystemInfo(t *testing.T) {
	tests := []struct {
		name             string
		registrationOpen bool
		config           SourceHandlerConfig
		want             *pipelinev1.GetSystemInfoResponse
	}{
		{
			name:             "multi workspace without email",
			registrationOpen: true,
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
				ServiceAccountName: "synclet-sync",
			}},
			want: &pipelinev1.GetSystemInfoResponse{
				WorkspacesMode: pipelinev1.WorkspacesMode_WORKSPACES_MODE_MULTI,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &SourceHandler{config: tt.config, registrationStatus: staticRegistrationStatus(tt.registrationOpen)}

			resp, err := h.GetSystemInfo(context.Background(), connect.NewRequest(&pipelinev1.GetSystemInfoRequest{}))
			require.NoError(t, err)
			assert.True(t, proto.Equal(tt.want, resp.Msg), "want %v, got %v", tt.want, resp.Msg)
		})
	}
}

// newRegistryHandler returns a SourceHandler whose executor registry holds rows.
func newRegistryHandler(config RuntimeDefaultsInfo, rows ...*pipelineservice.Executor) *SourceHandler {
	storage := &executorRegistryStorage{executors: &fakeExecutors{rows: rows}}

	return &SourceHandler{
		config:        SourceHandlerConfig{RuntimeDefaults: config},
		listExecutors: pipelinejobs.NewListExecutors(storage, pipelineservice.Config{ExecutorSilenceTimeout: time.Hour}),
	}
}

func TestSourceHandler_GetRuntimeDefaults(t *testing.T) {
	now := time.Now()
	config := RuntimeDefaultsInfo{
		CPULimit:          "2",
		MemoryLimit:       "2Gi",
		MaxConcurrentJobs: 10,
		IdleTimeout:       10 * time.Minute,
		LogArchiveBackend: "s3",
	}
	global := &pipelinev1.RuntimeDefaults{
		CpuLimit:           "2",
		MemoryLimit:        "2Gi",
		MaxConcurrentJobs:  10,
		IdleTimeoutSeconds: 600,
		LogArchiveBackend:  "s3",
	}
	docker := &pipelineservice.Executor{
		InstanceID:       "docker-0",
		Type:             pipelineservice.ExecutorTypeDocker,
		Capacity:         5,
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{CPULimit: "2", MemoryLimit: "2Gi"},
		LastSeenAt:       now,
	}
	k8sOld := &pipelineservice.Executor{
		InstanceID:       "k8s-0",
		Type:             pipelineservice.ExecutorTypeK8s,
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{CPURequest: "1", CPULimit: "1", CPUExecutorDefault: true},
		LastSeenAt:       now.Add(-30 * time.Second),
	}
	k8sNew := &pipelineservice.Executor{
		InstanceID: "k8s-1",
		Type:       pipelineservice.ExecutorTypeK8s,
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{
			CPURequest: "750m", CPULimit: "750m", CPUExecutorDefault: true,
			MemoryRequest: "3Gi", MemoryLimit: "3Gi", MemoryExecutorDefault: true,
		},
		LastSeenAt: now,
	}

	tests := []struct {
		name      string
		rows      []*pipelineservice.Executor
		executors []*pipelinev1.ExecutorTypeDefaults
	}{
		{name: "no executors", rows: nil, executors: nil},
		{
			name: "docker and k8s executors",
			rows: []*pipelineservice.Executor{k8sNew, docker},
			executors: []*pipelinev1.ExecutorTypeDefaults{
				{
					Type:             pipelinev1.ExecutorType_EXECUTOR_TYPE_DOCKER,
					Capacity:         5,
					ResourceDefaults: &pipelinev1.ExecutorResourceDefaults{CpuLimit: "2", MemoryLimit: "2Gi"},
				},
				{
					Type: pipelinev1.ExecutorType_EXECUTOR_TYPE_K8S,
					ResourceDefaults: &pipelinev1.ExecutorResourceDefaults{
						CpuRequest: "750m", CpuLimit: "750m", CpuExecutorDefault: true,
						MemoryRequest: "3Gi", MemoryLimit: "3Gi", MemoryExecutorDefault: true,
					},
				},
			},
		},
		{
			name: "the most recently seen k8s executor wins",
			rows: []*pipelineservice.Executor{k8sNew, k8sOld},
			executors: []*pipelinev1.ExecutorTypeDefaults{
				{
					Type: pipelinev1.ExecutorType_EXECUTOR_TYPE_K8S,
					ResourceDefaults: &pipelinev1.ExecutorResourceDefaults{
						CpuRequest: "750m", CpuLimit: "750m", CpuExecutorDefault: true,
						MemoryRequest: "3Gi", MemoryLimit: "3Gi", MemoryExecutorDefault: true,
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := newRegistryHandler(config, tt.rows...)

			resp, err := h.GetRuntimeDefaults(context.Background(), connect.NewRequest(&pipelinev1.GetRuntimeDefaultsRequest{}))
			require.NoError(t, err)

			want := proto.Clone(global).(*pipelinev1.RuntimeDefaults)
			want.Executors = tt.executors
			assert.True(t, proto.Equal(want, resp.Msg.GetRuntimeDefaults()), "want %v, got %v", want, resp.Msg.GetRuntimeDefaults())
		})
	}
}

func TestSourceHandler_ListExecutors(t *testing.T) {
	seen := time.Date(2026, 10, 18, 12, 0, 0, 0, time.UTC)
	now := time.Now()
	h := newRegistryHandler(RuntimeDefaultsInfo{},
		&pipelineservice.Executor{
			InstanceID:       "synclet-executor-0",
			Type:             pipelineservice.ExecutorTypeK8s,
			Version:          "v0.5.0",
			Capabilities:     pipelineservice.ExecutorCapabilities{NativeDestinations: []string{"postgres"}},
			ResourceDefaults: pipelineservice.ExecutorResourceDefaults{CPURequest: "1"},
			FirstSeenAt:      seen,
			LastSeenAt:       now,
		},
		&pipelineservice.Executor{InstanceID: "silent", Type: pipelineservice.ExecutorTypeDocker, LastSeenAt: now.Add(-2 * time.Hour)},
	)

	resp, err := h.ListExecutors(context.Background(), connect.NewRequest(&pipelinev1.ListExecutorsRequest{}))
	require.NoError(t, err)

	want := &pipelinev1.ListExecutorsResponse{Executors: []*pipelinev1.ExecutorInstance{{
		Id:                 "synclet-executor-0",
		Type:               pipelinev1.ExecutorType_EXECUTOR_TYPE_K8S,
		Version:            "v0.5.0",
		NativeDestinations: []string{"postgres"},
		ResourceDefaults:   &pipelinev1.ExecutorResourceDefaults{CpuRequest: "1"},
		FirstSeenAt:        timestamppb.New(seen),
		LastSeenAt:         timestamppb.New(now),
	}}}
	assert.True(t, proto.Equal(want, resp.Msg), "want %v, got %v", want, resp.Msg)
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
			h := newRegistryHandler(RuntimeDefaultsInfo{}, &pipelineservice.Executor{
				InstanceID: "docker-0",
				Type:       pipelineservice.ExecutorTypeDocker,
				LastSeenAt: time.Now(),
			})
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
			require.Len(t, resp.Msg.GetRuntimeDefaults().GetExecutors(), 1)
			assert.Equal(t, pipelinev1.ExecutorType_EXECUTOR_TYPE_DOCKER, resp.Msg.GetRuntimeDefaults().GetExecutors()[0].GetType())
		})
	}
}
