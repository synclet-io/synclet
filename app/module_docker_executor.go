package app

import (
	"context"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexecdocker"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/pkg/container"
	"github.com/synclet-io/synclet/pkg/jobutil"
	"go.uber.org/fx"
)

// dockerExecutorJobsConfig holds intervals for Docker executor background jobs.
type dockerExecutorJobsConfig struct {
	CleanupInterval time.Duration `env:"RESOURCE_CLEANUP_INTERVAL" envDefault:"5m"`
	WorkerInterval  time.Duration `env:"WORKER_INTERVAL" envDefault:"1s"`
}

type dockerExecutorConfig struct {
	Jobs              dockerExecutorJobsConfig `envPrefix:"JOB_"`
	MaxSyncDuration   time.Duration            `env:"MAX_SYNC_DURATION" envDefault:"24h"`
	HeartbeatInterval time.Duration            `env:"HEARTBEAT_INTERVAL" envDefault:"5s"`
	MaxConcurrentJobs int                      `env:"MAX_CONCURRENT_JOBS" envDefault:"5"`
	// TempDirRoot is the directory under which per-task scratch directories
	// are created when ContainerRunner spawns connector containers. Leave
	// empty for host-native deployments. In Docker-in-Docker setups (synclet
	// running in a container with /var/run/docker.sock mounted) set this to
	// a path bind-mounted to the same location on the host — otherwise the
	// host Docker daemon cannot resolve the bind-mount source.
	TempDirRoot string `env:"TEMP_DIR_ROOT"`
}

// dockerExecutorConfigModule registers the Docker executor configuration and
// the always-on Docker container infrastructure. ContainerRunner (used by
// ImagePullerAdapter and any pipeline use case that talks to Docker) needs to
// resolve regardless of whether the background worker jobs are enabled.
func dockerExecutorConfigModule() fx.Option {
	return fx.Options(
		fx.Provide(
			configutil.NewPrefixedConfigProvider[dockerExecutorConfig]("DOCKER_EXECUTOR_"),
			configutil.NewPrefixedConfigInfoProvider[dockerExecutorConfig]("DOCKER_EXECUTOR_"),
			newContainerRunner,
			fx.Annotate(dockerContainerRunnerAdapter, fx.As(new(container.Runner))),
			pipelineexecdocker.NewConnectorClient,
		),
	)
}

func newContainerRunner(cfg *dockerExecutorConfig) (*pipelineexecdocker.ContainerRunner, error) {
	return pipelineexecdocker.NewContainerRunner(cfg.TempDirRoot)
}

func dockerContainerRunnerAdapter(r *pipelineexecdocker.ContainerRunner) container.Runner {
	return r
}

func dockerExecutorModule(options *RunAppOptions) fx.Option {
	if !options.DockerExecutor {
		return fx.Options()
	}

	return fx.Module(
		"docker-executor",
		logging.DecorateNamed("docker"),
		fx.Provide(
			pipelineexecdocker.NewOrphanCleaner,
			func(backend pipelineexec.ExecutorBackend) pipelineexecdocker.OrphanJobChecker {
				return backend
			},

			newDockerSyncWorker,
			newDockerTaskWorker,
		),
		fx.Invoke(invokeDockerCleanerStartup),
		pnpjobber.Module(newDockerCleanerJob),
		pnpjobber.Module(newDockerSyncWorkerJob),
		pnpjobber.Module(newDockerTaskWorkerJob),
	)
}

func invokeDockerCleanerStartup(lc fx.Lifecycle, cleaner *pipelineexecdocker.OrphanCleaner, logger *logging.Logger) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if err := cleaner.CleanupAll(ctx); err != nil {
				logger.WithError(err).Error(ctx, "startup: docker cleanup failed")
			}

			return nil
		},
	})
}

func newDockerCleanerJob(cfg *dockerExecutorConfig, cleaner *pipelineexecdocker.OrphanCleaner, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "docker_resource_cleaner",
		Interval:         cfg.Jobs.CleanupInterval,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := cleaner.Cleanup(ctx); err != nil {
				logger.WithError(err).Error(ctx, "docker cleanup failed")
			}

			return nil
		},
	})
}

func newDockerSyncWorker(
	cfg *dockerExecutorConfig,
	backend pipelineexec.ExecutorBackend,
	executor *pipelineexecdocker.DockerSyncExecutor,
	metrics *pipelinemetrics.MetricsCollector,
	manager *jobutil.WorkerManager,
	logger *logging.Logger,
) *pipelineexecdocker.DockerSyncWorker {
	return pipelineexecdocker.NewDockerSyncWorker(pipelineexecdocker.DockerSyncWorkerParams{
		Backend:           backend,
		Executor:          executor,
		Metrics:           metrics,
		Manager:           manager,
		MaxSyncDuration:   cfg.MaxSyncDuration,
		HeartbeatInterval: cfg.HeartbeatInterval,
		MaxConcurrentJobs: cfg.MaxConcurrentJobs,
		Logger:            logger,
	})
}

func newDockerSyncWorkerJob(cfg *dockerExecutorConfig, worker *pipelineexecdocker.DockerSyncWorker) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:     "docker_sync",
		Interval: cfg.Jobs.WorkerInterval,
		Job: func(ctx context.Context) error {
			return worker.Execute(ctx)
		},
	})
}

func newDockerTaskWorker(
	cfg *dockerExecutorConfig,
	backend pipelineexec.ExecutorBackend,
	client *pipelineexecdocker.ConnectorClient,
	manager *jobutil.WorkerManager,
	logger *logging.Logger,
) *pipelineexecdocker.DockerTaskWorker {
	return pipelineexecdocker.NewDockerTaskWorker(pipelineexecdocker.DockerTaskWorkerParams{
		Backend:       backend,
		Client:        client,
		Manager:       manager,
		MaxConcurrent: cfg.MaxConcurrentJobs,
		Logger:        logger,
	})
}

func newDockerTaskWorkerJob(cfg *dockerExecutorConfig, worker *pipelineexecdocker.DockerTaskWorker) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:     "docker_task",
		Interval: cfg.Jobs.WorkerInterval,
		Job: func(ctx context.Context) error {
			return worker.Execute(ctx)
		},
	})
}
