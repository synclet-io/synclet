package app

import (
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/stretchr/testify/assert"
	"go.uber.org/fx"
)

type runMatrix struct {
	publicAPI   bool
	internalAPI bool
	jobs        bool
	consumers   bool
	standalone  bool
	executor    string
}

func executorKind(options *RunAppOptions) string {
	switch {
	case options.DockerExecutor:
		return "docker"
	case options.K8sExecutor:
		return "k8s"
	default:
		return ""
	}
}

// TestFxGraph resolves the run options cmd/ passes for every CLI command on
// top of the defaults of RunServer or RunJobs, asserts which HTTP servers,
// background jobs and consumers each starts, and validates its dependency
// graph without starting it.
func TestFxGraph(t *testing.T) {
	cases := []struct {
		name     string
		inK8s    bool
		defaults func() *RunAppOptions
		opts     []RunOption
		want     runMatrix
	}{
		{
			name:     "server",
			defaults: serverDefaultOptions,
			opts:     ServerCommandOptions(false),
			want:     runMatrix{publicAPI: true, internalAPI: true},
		},
		{
			name:     "server --standalone outside k8s",
			defaults: serverDefaultOptions,
			opts:     ServerCommandOptions(true),
			want: runMatrix{
				publicAPI: true, internalAPI: true, jobs: true, consumers: true, standalone: true, executor: "docker",
			},
		},
		{
			name:     "server --standalone in k8s",
			inK8s:    true,
			defaults: serverDefaultOptions,
			opts:     ServerCommandOptions(true),
			want: runMatrix{
				publicAPI: true, internalAPI: true, jobs: true, consumers: true, standalone: true, executor: "k8s",
			},
		},
		{
			name:     "jobs",
			defaults: func() *RunAppOptions { return &RunAppOptions{} },
			opts:     JobsCommandOptions(),
			want:     runMatrix{jobs: true, consumers: true},
		},
		{
			name:     "docker-executor",
			defaults: serverDefaultOptions,
			opts:     ExecutorCommandOptions(WithDockerExecutor(), false),
			want:     runMatrix{publicAPI: true, internalAPI: true, executor: "docker"},
		},
		{
			name:     "docker-executor --standalone",
			defaults: serverDefaultOptions,
			opts:     ExecutorCommandOptions(WithDockerExecutor(), true),
			want:     runMatrix{publicAPI: true, internalAPI: true, standalone: true, executor: "docker"},
		},
		{
			name:     "k8s-executor",
			defaults: serverDefaultOptions,
			opts:     ExecutorCommandOptions(WithK8sExecutor(), false),
			want:     runMatrix{publicAPI: true, internalAPI: true, executor: "k8s"},
		},
		{
			name:     "k8s-executor --standalone",
			defaults: serverDefaultOptions,
			opts:     ExecutorCommandOptions(WithK8sExecutor(), true),
			want:     runMatrix{publicAPI: true, internalAPI: true, standalone: true, executor: "k8s"},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.inK8s {
				t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
			} else {
				unsetEnv(t, "KUBERNETES_SERVICE_HOST")
			}

			options := optionutil.ApplyOptions(tc.defaults(), tc.opts...)

			assert.Equal(t, tc.want, runMatrix{
				publicAPI:   options.RunPublicHTTPServer,
				internalAPI: options.RunInternalHTTPServer,
				jobs:        options.RunJobs,
				consumers:   options.RunConsumers,
				standalone:  options.Standalone,
				executor:    executorKind(options),
			})

			if err := fx.ValidateApp(NewFxAppOptions(options)); err != nil {
				t.Fatalf("fx graph invalid: %v", err)
			}
		})
	}
}
