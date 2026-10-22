package pipelineexecdocker

import "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"

// resolveResources resolves a connector's raw runtime_config against the
// executor's runtime defaults into the container's resources and network.
func resolveResources(defaults pipelineservice.RuntimeDefaults, runtimeConfig string) ResourceLimits {
	cfg := pipelineservice.ResolveRuntimeConfig(defaults, pipelineservice.ParseRuntimeConfig(&runtimeConfig))
	memoryLimit, cpuLimit, memoryRequest, cpuRequest := pipelineservice.ToContainerResources(cfg)

	return ResourceLimits{
		MemoryLimit:   memoryLimit,
		CPULimit:      cpuLimit,
		MemoryRequest: memoryRequest,
		CPURequest:    cpuRequest,
		NetworkMode:   cfg.DockerNetwork,
	}
}
