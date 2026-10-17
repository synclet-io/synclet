package app

import (
	"strings"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineconnect"
)

func buildSourceHandlerConfig(t *testing.T, opts []RunOption) pipelineconnect.SourceHandlerConfig {
	t.Helper()

	options := optionutil.ApplyOptions(serverDefaultOptions(), opts...)

	var cfg pipelineconnect.SourceHandlerConfig

	app := fx.New(
		fx.NopLogger,
		fx.Supply(options),
		authConfigModule(),
		workspaceConfigModule(),
		notifyConfigModule(),
		pipelineConfigModule(),
		dockerExecutorConfigModule(),
		k8sExecutorConfigModule(options),
		fx.Populate(&cfg),
	)
	require.NoError(t, app.Err())

	return cfg
}

func TestSourceHandlerConfig_K8sExecutorWiring(t *testing.T) {
	t.Setenv("AUTH_JWT_SECRET", strings.Repeat("x", 32))
	t.Setenv("K8S_EXECUTOR_DEFAULT_CPU", "750m")
	t.Setenv("K8S_EXECUTOR_DEFAULT_MEMORY", "3Gi")
	unsetEnv(t, "KUBERNETES_SERVICE_HOST")

	for name, tc := range map[string]struct {
		opts        []RunOption
		executor    pipelineconnect.Executor
		cpu, memory string
	}{
		"docker executor": {
			opts:     ExecutorCommandOptions(WithDockerExecutor(), true),
			executor: pipelineconnect.ExecutorDocker,
		},
		"k8s executor": {
			opts:     ExecutorCommandOptions(WithK8sExecutor(), true),
			executor: pipelineconnect.ExecutorK8s,
			cpu:      "750m",
			memory:   "3Gi",
		},
		"distributed api server": {
			opts:     ServerCommandOptions(false),
			executor: pipelineconnect.ExecutorNone,
		},
	} {
		t.Run(name, func(t *testing.T) {
			defaults := buildSourceHandlerConfig(t, tc.opts).RuntimeDefaults

			assert.Equal(t, tc.executor, defaults.Executor)
			assert.Equal(t, tc.cpu, defaults.K8sDefaultCPU)
			assert.Equal(t, tc.memory, defaults.K8sDefaultMemory)
		})
	}
}
