package pipelineexecdocker

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

func TestResolveResources(t *testing.T) {
	withRequests := pipelineservice.RuntimeDefaults{MemoryRequest: "1Gi", CPURequest: "1"}

	tests := []struct {
		name          string
		defaults      pipelineservice.RuntimeDefaults
		runtimeConfig string
		memoryRequest int64
		cpuRequest    float64
	}{
		{name: "nothing configured"},
		{
			name:          "runtime_config requests",
			runtimeConfig: `{"memory_request":"512Mi","cpu_request":"250m"}`,
			memoryRequest: 536870912,
			cpuRequest:    0.25,
		},
		{
			name:          "default requests",
			defaults:      withRequests,
			memoryRequest: 1073741824,
			cpuRequest:    1,
		},
		{
			name:          "runtime_config overrides one default",
			defaults:      withRequests,
			runtimeConfig: `{"cpu_request":"500m"}`,
			memoryRequest: 1073741824,
			cpuRequest:    0.5,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			limits := resolveResources(tt.defaults, tt.runtimeConfig)

			assert.Equal(t, tt.memoryRequest, limits.MemoryRequest)
			assert.InDelta(t, tt.cpuRequest, limits.CPURequest, 1e-9)
		})
	}
}

func TestResolveResourcesLimitsAndDockerOptions(t *testing.T) {
	limits := resolveResources(
		pipelineservice.RuntimeDefaults{MemoryLimit: "1Gi", CPULimit: "2"},
		`{"memory_limit":"512Mi","docker_network":"host","docker_volumes":[{"host_path":"/data","container_path":"/mnt"}]}`,
	)

	assert.Equal(t, int64(536870912), limits.MemoryLimit)
	assert.InDelta(t, 2.0, limits.CPULimit, 1e-9)
	assert.Equal(t, "host", limits.NetworkMode)
	assert.Len(t, limits.ExtraMounts, 1)
}
