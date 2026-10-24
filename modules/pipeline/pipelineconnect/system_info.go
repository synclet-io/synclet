package pipelineconnect

import (
	"context"
	"math"
	"time"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
)

// RegistrationStatus reports whether new accounts may sign up.
type RegistrationStatus interface {
	RegistrationOpen(ctx context.Context) (bool, error)
}

// SourceHandlerConfig holds configuration values exposed via GetSystemInfo.
type SourceHandlerConfig struct {
	SingleWorkspaceMode bool
	EmailEnabled        bool
	RuntimeDefaults     RuntimeDefaultsInfo
}

// RuntimeDefaultsInfo holds the global runtime settings applied when a source
// or destination has no runtime config of its own. It carries no credentials.
// Empty resource values mean no default is applied.
type RuntimeDefaultsInfo struct {
	CPURequest         string
	CPULimit           string
	MemoryRequest      string
	MemoryLimit        string
	ServiceAccountName string
	MaxConcurrentJobs  int
	IdleTimeout        time.Duration
	LogArchiveBackend  string
}

func (h *SourceHandler) GetSystemInfo(ctx context.Context, _ *connect.Request[pipelinev1.GetSystemInfoRequest]) (*connect.Response[pipelinev1.GetSystemInfoResponse], error) {
	registrationOpen, err := h.registrationStatus.RegistrationOpen(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	workspacesMode := pipelinev1.WorkspacesMode_WORKSPACES_MODE_MULTI
	if h.config.SingleWorkspaceMode {
		workspacesMode = pipelinev1.WorkspacesMode_WORKSPACES_MODE_SINGLE
	}

	return connect.NewResponse(&pipelinev1.GetSystemInfoResponse{
		RegistrationEnabled: registrationOpen,
		WorkspacesMode:      workspacesMode,
		EmailEnabled:        h.config.EmailEnabled,
	}), nil
}

func (h *SourceHandler) GetRuntimeDefaults(ctx context.Context, _ *connect.Request[pipelinev1.GetRuntimeDefaultsRequest]) (*connect.Response[pipelinev1.GetRuntimeDefaultsResponse], error) {
	executors, err := h.listExecutors.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	defaults := runtimeDefaultsToProto(h.config.RuntimeDefaults)
	for _, executor := range pipelinejobs.LatestExecutorPerType(executors) {
		defaults.Executors = append(defaults.Executors, &pipelinev1.ExecutorTypeDefaults{
			Type:             domainExecutorTypeToProto(executor.Type),
			Capacity:         clampInt32(executor.Capacity),
			ResourceDefaults: domainExecutorResourceDefaultsToProto(executor.ResourceDefaults),
		})
	}

	return connect.NewResponse(&pipelinev1.GetRuntimeDefaultsResponse{RuntimeDefaults: defaults}), nil
}

// ListExecutors returns the executors known to the control plane.
func (h *SourceHandler) ListExecutors(ctx context.Context, _ *connect.Request[pipelinev1.ListExecutorsRequest]) (*connect.Response[pipelinev1.ListExecutorsResponse], error) {
	executors, err := h.listExecutors.Execute(ctx)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &pipelinev1.ListExecutorsResponse{Executors: make([]*pipelinev1.ExecutorInstance, 0, len(executors))}
	for _, executor := range executors {
		resp.Executors = append(resp.Executors, &pipelinev1.ExecutorInstance{
			Id:                 executor.InstanceID,
			Type:               domainExecutorTypeToProto(executor.Type),
			Version:            executor.Version,
			Capacity:           clampInt32(executor.Capacity),
			NativeDestinations: executor.Capabilities.NativeDestinations,
			ResourceDefaults:   domainExecutorResourceDefaultsToProto(executor.ResourceDefaults),
			FirstSeenAt:        timestamppb.New(executor.FirstSeenAt),
			LastSeenAt:         timestamppb.New(executor.LastSeenAt),
		})
	}

	return connect.NewResponse(resp), nil
}

func runtimeDefaultsToProto(info RuntimeDefaultsInfo) *pipelinev1.RuntimeDefaults {
	return &pipelinev1.RuntimeDefaults{
		CpuRequest:         info.CPURequest,
		CpuLimit:           info.CPULimit,
		MemoryRequest:      info.MemoryRequest,
		MemoryLimit:        info.MemoryLimit,
		ServiceAccountName: info.ServiceAccountName,
		MaxConcurrentJobs:  clampInt32(info.MaxConcurrentJobs),
		IdleTimeoutSeconds: int64(info.IdleTimeout / time.Second),
		LogArchiveBackend:  info.LogArchiveBackend,
	}
}

func clampInt32(v int) int32 {
	return int32(min(v, math.MaxInt32))
}
