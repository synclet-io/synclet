package app

import (
	"context"
	"os"
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
	// LocalRoot is the host directory mounted read-write at /local in every
	// connector container, where file-writing destinations leave their files.
	// It is shared by all jobs and never cleaned. Empty gives each job its own
	// /local under TempDirRoot, removed with the job. Like TempDirRoot, in a
	// Docker-in-Docker setup it must be bind-mounted at the same path on the
	// host.
	LocalRoot string `env:"LOCAL_ROOT"`
	// OwnerID is stamped as the synclet.io/owner label on every connector
	// container, and orphan cleanup removes only containers carrying it.
	// Empty uses the hostname; executors sharing one Docker daemon need
	// distinct values.
	OwnerID string `env:"OWNER_ID"`
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

func newContainerRunner(cfg *dockerExecutorConfig, logger *logging.Logger) (*pipelineexecdocker.ContainerRunner, error) {
	ownerID, err := pipelineexec.ResolveOwnerID(cfg.OwnerID, "DOCKER_EXECUTOR_OWNER_ID", os.Hostname)
	if err != nil {
		return nil, err
	}

	if cfg.LocalRoot != "" {
		if err := pipelineexecdocker.PrepareLocalRoot(context.Background(), cfg.LocalRoot, logger); err != nil {
			return nil, err
		}
	}

	return pipelineexecdocker.NewContainerRunner(cfg.TempDirRoot, cfg.LocalRoot, ownerID)
}

func dockerContainerRunnerAdapter(r *pipelineexecdocker.ContainerRunner) container.Runner {
	return r
}

// dockerExecutorModule runs the Docker executor's workers. It needs
// *dockerExecutorConfig, *executorRuntimeConfig and an ExecutorBackend.
func dockerExecutorModule() fx.Option {
	return fx.Module(
		"docker-executor",
		logging.DecorateNamed("docker"),
		fx.Provide(
			pipelineexecdocker.NewOrphanCleaner,
			func(backend pipelineexec.ExecutorBackend) pipelineexecdocker.OrphanJobChecker {
				return backend
			},

			newDockerSyncExecutor,
			newDockerSyncWorker,
			newDockerTaskWorker,
			newStartupCleanupFunc,
			fx.Private,
		),
		fx.Provide(newDockerExecutorInfo),
		executorRegistrationModule(),
		dockerStartupCleanupModule(),
		pnpjobber.Module(newDockerCleanerJob),
		pnpjobber.Module(newDockerSyncWorkerJob),
		pnpjobber.Module(newDockerTaskWorkerJob),
	)
}

// startupCleanupFunc removes the orphaned containers a previous run left behind.
type startupCleanupFunc func(ctx context.Context) error

func newStartupCleanupFunc(cleaner *pipelineexecdocker.OrphanCleaner) startupCleanupFunc {
	return cleaner.CleanupAll
}

// dockerStartupCleanupModule runs the startup cleanup once as a jobber job, so
// it starts after the app has started and is cancelled when the app stops.
func dockerStartupCleanupModule() fx.Option {
	return pnpjobber.Module(newDockerStartupCleanupJob)
}

func newDockerStartupCleanupJob(cleanup startupCleanupFunc, logger *logging.Logger) jobber.Job {
	return onceJob{
		name: "docker_startup_cleanup",
		job: func(ctx context.Context) error {
			if err := cleanup(ctx); err != nil {
				logger.WithError(err).Error(ctx, "startup: docker cleanup failed")
			}

			return nil
		},
	}
}

// onceJob is a jobber.Job that handles once, right after its runner starts.
type onceJob struct {
	name string
	job  func(ctx context.Context) error
}

func (j onceJob) Name() string { return j.name }

func (j onceJob) Init(context.Context) error { return nil }

func (j onceJob) Handle(ctx context.Context) error { return j.job(ctx) }

func (j onceJob) Timer() *time.Timer { return time.NewTimer(0) }

func (j onceJob) ResetTimer(*time.Timer) {}

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

func newDockerSyncExecutor(
	cfg *executorRuntimeConfig,
	dockerCfg *dockerExecutorConfig,
	backend pipelineexec.ExecutorBackend,
	client *pipelineexecdocker.ConnectorClient,
	logger *logging.Logger,
) *pipelineexecdocker.DockerSyncExecutor {
	return pipelineexecdocker.NewDockerSyncExecutor(pipelineexecdocker.DockerSyncExecutorParams{
		Client:          client,
		Reporter:        backend,
		RuntimeDefaults: cfg.Runtime.defaults(),
		IdleTimeout:     cfg.IdleTimeout,
		TempDirRoot:     dockerCfg.TempDirRoot,
		LocalRoot:       dockerCfg.LocalRoot,
		Logger:          logger,
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
	runtimeCfg *executorRuntimeConfig,
	backend pipelineexec.ExecutorBackend,
	client *pipelineexecdocker.ConnectorClient,
	manager *jobutil.WorkerManager,
	logger *logging.Logger,
) *pipelineexecdocker.DockerTaskWorker {
	return pipelineexecdocker.NewDockerTaskWorker(pipelineexecdocker.DockerTaskWorkerParams{
		Backend:         backend,
		Client:          client,
		RuntimeDefaults: runtimeCfg.Runtime.defaults(),
		Manager:         manager,
		MaxConcurrent:   cfg.MaxConcurrentJobs,
		Logger:          logger,
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
