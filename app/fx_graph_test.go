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
	mcp         bool
	jobs        bool
	consumers   bool
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

// TestFxGraph resolves the run options cmd/ passes for the server and jobs
// commands on top of the defaults of RunServer or RunJobs, asserts which HTTP
// servers, background jobs, consumers and executor each starts, and validates
// its dependency graph without starting it. TestExecutorGraph covers the
// executor commands.
func TestFxGraph(t *testing.T) {
	cases := []struct {
		name     string
		inK8s    bool
		mcpEnv   string
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
				publicAPI: true, internalAPI: true, jobs: true, consumers: true, executor: "docker",
			},
		},
		{
			name:     "server --standalone in k8s",
			inK8s:    true,
			defaults: serverDefaultOptions,
			opts:     ServerCommandOptions(true),
			want: runMatrix{
				publicAPI: true, internalAPI: true, jobs: true, consumers: true, executor: "k8s",
			},
		},
		{
			name:     "jobs",
			defaults: func() *RunAppOptions { return &RunAppOptions{} },
			opts:     JobsCommandOptions(),
			want:     runMatrix{jobs: true, consumers: true},
		},
		{
			name:     "server with MCP",
			mcpEnv:   "true",
			defaults: serverDefaultOptions,
			opts:     ServerCommandOptions(false),
			want:     runMatrix{publicAPI: true, internalAPI: true, mcp: true},
		},
		{
			name:     "server with MCP disabled",
			mcpEnv:   "false",
			defaults: serverDefaultOptions,
			opts:     ServerCommandOptions(false),
			want:     runMatrix{publicAPI: true, internalAPI: true},
		},
		{
			name:     "jobs with MCP_ENABLED",
			mcpEnv:   "true",
			defaults: func() *RunAppOptions { return &RunAppOptions{} },
			opts:     JobsCommandOptions(),
			want:     runMatrix{jobs: true, consumers: true},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if tc.inK8s {
				t.Setenv("KUBERNETES_SERVICE_HOST", "10.0.0.1")
			} else {
				unsetEnv(t, "KUBERNETES_SERVICE_HOST")
			}

			if tc.mcpEnv != "" {
				t.Setenv("MCP_ENABLED", tc.mcpEnv)
			} else {
				unsetEnv(t, "MCP_ENABLED")
			}

			options := optionutil.ApplyOptions(tc.defaults(), tc.opts...)
			options.RunMCPServer = runsMCPServer(options)

			assert.Equal(t, tc.want, runMatrix{
				publicAPI:   options.RunPublicHTTPServer,
				internalAPI: options.RunInternalHTTPServer,
				mcp:         options.RunMCPServer,
				jobs:        options.RunJobs,
				consumers:   options.RunConsumers,
				executor:    executorKind(options),
			})

			if err := fx.ValidateApp(NewFxAppOptions(options)); err != nil {
				t.Fatalf("fx graph invalid: %v", err)
			}
		})
	}
}
