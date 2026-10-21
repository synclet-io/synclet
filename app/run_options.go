package app

import (
	"os"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/samber/lo"
	"go.uber.org/fx"
)

// isRunningInK8s detects whether the process is running inside a Kubernetes pod
// by checking for KUBERNETES_SERVICE_HOST env var, which K8s injects into every pod.
func isRunningInK8s() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

type RunOption = optionutil.Option[RunAppOptions]

// RunAppOptions select what a server or jobs process runs. An executor flag
// embeds that executor in the process, backed by the use cases.
type RunAppOptions struct {
	DotEnvFiles           []string
	RunModule             map[string]bool
	RunConsumers          bool
	RunPublicHTTPServer   bool
	RunInternalHTTPServer bool
	RunMCPServer          bool
	RunJobs               bool
	DockerExecutor        bool
	K8sExecutor           bool
	fxOptions             []fx.Option
	// serverFxOptions are added inside the fx module of the HTTP server with
	// that module name, where its private values are visible.
	serverFxOptions map[string][]fx.Option
}

func (r *RunAppOptions) needToRunModule(module string) bool {
	if len(r.RunModule) == 0 {
		return true
	}

	return r.RunModule[module]
}

func (r *RunAppOptions) runsExecutor() bool {
	return r.DockerExecutor || r.K8sExecutor
}

func WithDotEnvFiles(dotEnvFiles ...string) RunOption {
	return func(options *RunAppOptions) {
		options.DotEnvFiles = dotEnvFiles
	}
}

func WithRunPublicHTTPServer() RunOption {
	return func(options *RunAppOptions) {
		options.RunPublicHTTPServer = true
	}
}

func WithRunInternalHTTPServer() RunOption {
	return func(options *RunAppOptions) {
		options.RunInternalHTTPServer = true
	}
}

func WithRunJobs() RunOption {
	return func(options *RunAppOptions) {
		options.RunJobs = true
	}
}

func WithRunConsumers() RunOption {
	return func(options *RunAppOptions) {
		options.RunConsumers = true
	}
}

func WithDockerExecutor() RunOption {
	return func(options *RunAppOptions) {
		options.DockerExecutor = true
	}
}

func WithK8sExecutor() RunOption {
	return func(options *RunAppOptions) {
		options.K8sExecutor = true
	}
}

func WithAutoExecutor() RunOption {
	return func(options *RunAppOptions) {
		if isRunningInK8s() {
			options.K8sExecutor = true
		} else {
			options.DockerExecutor = true
		}
	}
}

func WithFxOptions(fxOptions ...fx.Option) RunOption {
	return func(options *RunAppOptions) {
		options.fxOptions = fxOptions
	}
}

func conditionalFxOptions(condition bool, fns ...func() fx.Option) fx.Option {
	if !condition {
		return fx.Options()
	}

	return fx.Options(
		lo.Map(fns, func(f func() fx.Option, _ int) fx.Option { return f() })...,
	)
}

func WithRunModule(module string) RunOption {
	return func(options *RunAppOptions) {
		if module == "" {
			return
		}

		if options.RunModule == nil {
			options.RunModule = make(map[string]bool)
		}

		options.RunModule[module] = true
	}
}

// ServerCommandOptions are the run options of `synclet server`. Standalone
// adds background jobs, event consumers and an in-process executor.
func ServerCommandOptions(standalone bool) []RunOption {
	opts := []RunOption{WithRunPublicHTTPServer(), WithRunInternalHTTPServer()}
	if standalone {
		opts = append(opts, WithRunJobs(), WithRunConsumers(), WithAutoExecutor())
	}

	return opts
}

// JobsCommandOptions are the run options of `synclet jobs`: background jobs,
// the outbox publisher and event consumers, no public API.
func JobsCommandOptions() []RunOption {
	return []RunOption{WithRunJobs(), WithRunConsumers()}
}
