package app

import (
	"testing"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/buildinfo"
)

// buildExecutorInfo resolves the ExecutorInfo of newInfo from the
// environment, with configs providing the executor's own configuration.
func buildExecutorInfo(t *testing.T, configs fx.Option, newInfo any) pipelineexec.ExecutorInfo {
	t.Helper()

	var info pipelineexec.ExecutorInfo

	app := fx.New(
		fx.NopLogger,
		configs,
		fx.Provide(configutil.NewPrefixedConfigProvider[executorRuntimeConfig]("PIPELINE_")),
		fx.Provide(newInfo),
		fx.Populate(&info),
	)
	require.NoError(t, app.Err())

	return info
}

func TestDockerExecutorInfo(t *testing.T) {
	t.Setenv("DOCKER_EXECUTOR_OWNER_ID", "docker-0")
	t.Setenv("PIPELINE_RUNTIME_CPU_LIMIT", "2")
	unsetEnv(t, "DOCKER_EXECUTOR_MAX_CONCURRENT_JOBS", "PIPELINE_RUNTIME_CPU_REQUEST", "PIPELINE_RUNTIME_MEMORY_REQUEST", "PIPELINE_RUNTIME_MEMORY_LIMIT")

	info := buildExecutorInfo(t, fx.Provide(configutil.NewPrefixedConfigProvider[dockerExecutorConfig]("DOCKER_EXECUTOR_")), newDockerExecutorInfo)

	assert.Equal(t, pipelineexec.ExecutorInfo{
		ID:               "docker-0",
		Type:             pipelineservice.ExecutorTypeDocker,
		Version:          buildinfo.Version(),
		Capacity:         5,
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{CPULimit: "2"},
	}, info)
}

func TestK8sExecutorInfo(t *testing.T) {
	t.Setenv("K8S_EXECUTOR_OWNER_ID", "synclet-executor-0")
	t.Setenv("K8S_EXECUTOR_DEFAULT_CPU", "750m")
	t.Setenv("K8S_EXECUTOR_DEFAULT_MEMORY", "3Gi")
	t.Setenv("PIPELINE_RUNTIME_MEMORY_LIMIT", "1Gi")
	unsetEnv(t, "KUBERNETES_SERVICE_HOST", "PIPELINE_RUNTIME_CPU_REQUEST", "PIPELINE_RUNTIME_CPU_LIMIT", "PIPELINE_RUNTIME_MEMORY_REQUEST")

	info := buildExecutorInfo(t, k8sExecutorConfigModule(), newK8sExecutorInfo)

	assert.Equal(t, pipelineexec.ExecutorInfo{
		ID:      "synclet-executor-0",
		Type:    pipelineservice.ExecutorTypeK8s,
		Version: buildinfo.Version(),
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{
			CPURequest:         "750m",
			CPULimit:           "750m",
			CPUExecutorDefault: true,
			MemoryLimit:        "1Gi",
		},
	}, info)
}
