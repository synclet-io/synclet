package app

import (
	"fmt"
	"log/slog"
	"os"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/fx/fxevent"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
)

// ExecutorOptions are the run options of the executor commands.
type ExecutorOptions struct {
	DotEnvFiles []string
}

// RunDockerExecutor runs the Docker executor: workers that claim sync jobs
// and connector tasks from the executor API and run them as containers of
// the local Docker daemon.
func RunDockerExecutor(options ExecutorOptions) error {
	return runExecutor(options, dockerExecutorGraph())
}

// RunK8sExecutor runs the Kubernetes executor: workers that claim sync jobs
// and connector tasks from the executor API and launch them as Kubernetes
// Jobs.
func RunK8sExecutor(options ExecutorOptions) error {
	return runExecutor(options, k8sExecutorGraph())
}

// runExecutor loads the dotenv files, drops the control-plane variables from
// the environment and runs the graph. An executor holds no secrets of its
// own, so none are generated or validated.
func runExecutor(options ExecutorOptions, graph executorGraph) error {
	bootLogger := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	if len(options.DotEnvFiles) > 0 {
		if err := godotenv.Load(options.DotEnvFiles...); err != nil {
			return fmt.Errorf("loading dotenv files: %w", err)
		}
	}

	dropControlPlaneEnv(bootLogger)
	applyExecutorEnvDefaults()

	fx.New(
		graph.options(),
		fx.WithLogger(func() fxevent.Logger {
			return FxLogger{logger: bootLogger}
		}),
		fx.StopTimeout(shutdownTimeout),
	).Run()

	return nil
}

// executorGraph is the fx graph of one executor command. The configuration
// is kept apart from the workers so configInventory can read it: the worker
// modules register jobs whose constructors reach Docker or Kubernetes.
type executorGraph struct {
	configs fx.Option
	workers fx.Option
}

// options is the whole graph: process infrastructure, the internal HTTP
// server with the probes, the executor API client, the executor's
// configuration and its workers. It provides no database connection,
// storage, use cases or outbox; TestExecutorGraph guards that.
func (g executorGraph) options() fx.Option {
	return fx.Options(executorBaseModule(), g.configs, g.workers)
}

// configOptions is the graph without its workers, for configInventory.
func (g executorGraph) configOptions() fx.Option {
	return fx.Options(executorBaseModule(), g.configs)
}

func dockerExecutorGraph() executorGraph {
	return executorGraph{
		configs: dockerExecutorConfigModule(),
		workers: dockerExecutorModule(),
	}
}

func k8sExecutorGraph() executorGraph {
	return executorGraph{
		configs: k8sExecutorConfigModule(),
		workers: fx.Options(k8sExecutorModule(), executorK8sPodAPITokenModule()),
	}
}

// executorBaseModule is what both executors share: the process
// infrastructure, the internal HTTP server with the probes, the executor
// API client, pipeline metrics and the runtime configuration.
func executorBaseModule() fx.Option {
	return fx.Options(
		processModule(),
		internalHTTPServerModule(true, healthcheckModule(false, false)),
		rpcBackendModule(),
		pipelineMetricsModule(pipelinemetrics.NewMetricsCollector),
		fx.Provide(
			configutil.NewPrefixedConfigProvider[executorRuntimeConfig]("PIPELINE_"),
			configutil.NewPrefixedConfigInfoProvider[executorRuntimeConfig]("PIPELINE_"),
			func(cfg *executorRuntimeConfig) pipelineservice.RuntimeDefaults { return cfg.Runtime.defaults() },
		),
	)
}

// executorRuntimeConfig is the part of the pipeline configuration an executor
// reads: the resource defaults of connector containers and the idle timeout
// of a sync. The server reads the same keys through pipelineConfig.
type executorRuntimeConfig struct {
	Runtime     pipelineRuntimeConfig `envPrefix:"RUNTIME_"`
	IdleTimeout time.Duration         `env:"IDLE_TIMEOUT" envDefault:"10m"`
}

// serverExecutorRuntimeModule derives the runtime configuration of an
// executor embedded in the server from the server's pipeline configuration.
func serverExecutorRuntimeModule() fx.Option {
	return fx.Provide(func(cfg *pipelineConfig) *executorRuntimeConfig {
		return &executorRuntimeConfig{Runtime: cfg.Runtime, IdleTimeout: cfg.IdleTimeout}
	})
}
