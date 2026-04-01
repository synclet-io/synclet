package app

import (
	"os"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"go.uber.org/fx"
)

// isRunningInK8s detects whether the process is running inside a Kubernetes pod
// by checking for KUBERNETES_SERVICE_HOST env var, which K8s injects into every pod.
func isRunningInK8s() bool {
	return os.Getenv("KUBERNETES_SERVICE_HOST") != ""
}

type RunOption = optionutil.Option[RunAppOptions]

type RunAppOptions struct {
	DotEnvFiles           []string
	RunModule             map[string]bool
	RunPublicHTTPServer   bool
	RunInternalHTTPServer bool
	RunJobs               bool
	DockerExecutor        bool
	K8sExecutor           bool
	Standalone            bool // When true, executors run in-process with use-case adapter
	fxOptions             []fx.Option
}

func (r *RunAppOptions) needToRunModule(module string) bool {
	if len(r.RunModule) == 0 {
		return true
	}

	return r.RunModule[module]
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

func WithRunJobs() RunOption {
	return func(options *RunAppOptions) {
		options.RunJobs = true
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

func WithStandalone() RunOption {
	return func(options *RunAppOptions) {
		options.Standalone = true
	}
}

func WithFxOptions(fxOptions ...fx.Option) RunOption {
	return func(options *RunAppOptions) {
		options.fxOptions = fxOptions
	}
}

func conditionalFxOption(condition bool, fn func() fx.Option) fx.Option {
	if !condition {
		return fx.Options()
	}

	return fn()
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
