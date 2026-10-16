package pipelineconnect

import (
	"context"
	"math"
	"time"

	"connectrpc.com/connect"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
)

// SourceHandlerConfig holds configuration values exposed via GetSystemInfo.
type SourceHandlerConfig struct {
	RegistrationEnabled bool
	SingleWorkspaceMode bool
	EmailEnabled        bool
	RuntimeDefaults     RuntimeDefaultsInfo
}

// Executor names the sync executor running in this server process.
type Executor string

const (
	ExecutorNone   Executor = ""
	ExecutorDocker Executor = "docker"
	ExecutorK8s    Executor = "k8s"
)

// RuntimeDefaultsInfo holds the global runtime settings applied when a source
// or destination has no runtime config of its own. It carries no credentials.
// Empty resource values mean no default is applied.
type RuntimeDefaultsInfo struct {
	Executor           Executor
	CPURequest         string
	CPULimit           string
	MemoryRequest      string
	MemoryLimit        string
	ServiceAccountName string
	MaxConcurrentJobs  int
	IdleTimeout        time.Duration
	LogArchiveBackend  string
	// DockerMaxConcurrentJobs is the Docker executor's own sync limit, set
	// only when the executor is Docker.
	DockerMaxConcurrentJobs int
	// K8sDefaultCPU and K8sDefaultMemory are the Kubernetes executor's
	// request and limit for a sync container with no CPU or memory set.
	K8sDefaultCPU    string
	K8sDefaultMemory string
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

func (h *SourceHandler) GetRuntimeDefaults(_ context.Context, _ *connect.Request[pipelinev1.GetRuntimeDefaultsRequest]) (*connect.Response[pipelinev1.GetRuntimeDefaultsResponse], error) {
	return connect.NewResponse(&pipelinev1.GetRuntimeDefaultsResponse{
		RuntimeDefaults: runtimeDefaultsToProto(h.config.RuntimeDefaults),
	}), nil
}

// runtimeDefaultsToProto reports the resources a sync container gets: with
// the Kubernetes executor, a resource with neither request nor limit set takes
// the executor's default as both.
func runtimeDefaultsToProto(info RuntimeDefaultsInfo) *pipelinev1.RuntimeDefaults {
	cpuExecutorDefault := info.Executor == ExecutorK8s && info.CPURequest == "" && info.CPULimit == "" && info.K8sDefaultCPU != ""
	if cpuExecutorDefault {
		info.CPURequest, info.CPULimit = info.K8sDefaultCPU, info.K8sDefaultCPU
	}

	memoryExecutorDefault := info.Executor == ExecutorK8s && info.MemoryRequest == "" && info.MemoryLimit == "" && info.K8sDefaultMemory != ""
	if memoryExecutorDefault {
		info.MemoryRequest, info.MemoryLimit = info.K8sDefaultMemory, info.K8sDefaultMemory
	}

	return &pipelinev1.RuntimeDefaults{
		Executor:                executorToProto(info.Executor),
		CpuRequest:              info.CPURequest,
		CpuLimit:                info.CPULimit,
		MemoryRequest:           info.MemoryRequest,
		MemoryLimit:             info.MemoryLimit,
		ServiceAccountName:      info.ServiceAccountName,
		MaxConcurrentJobs:       clampInt32(info.MaxConcurrentJobs),
		IdleTimeoutSeconds:      int64(info.IdleTimeout / time.Second),
		LogArchiveBackend:       info.LogArchiveBackend,
		DockerMaxConcurrentJobs: clampInt32(info.DockerMaxConcurrentJobs),
		CpuExecutorDefault:      cpuExecutorDefault,
		MemoryExecutorDefault:   memoryExecutorDefault,
	}
}

func executorToProto(executor Executor) pipelinev1.ExecutorType {
	switch executor {
	case ExecutorDocker:
		return pipelinev1.ExecutorType_EXECUTOR_TYPE_DOCKER
	case ExecutorK8s:
		return pipelinev1.ExecutorType_EXECUTOR_TYPE_K8S
	default:
		return pipelinev1.ExecutorType_EXECUTOR_TYPE_UNSPECIFIED
	}
}

func clampInt32(v int) int32 {
	return int32(min(v, math.MaxInt32))
}
