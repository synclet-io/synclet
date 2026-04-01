package app

import (
	"context"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/healthcheck/pnphealthcheck"
	"github.com/go-pnp/go-pnp/healthcheck/pnphealthcheckgorm"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/logging/pnpzap"
	"github.com/go-pnp/go-pnp/pnpenv"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/go-pnp/sql/pnpgorm"
	"github.com/go-pnp/jobber"
	"github.com/google/uuid"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineadapt"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
	"github.com/synclet-io/synclet/pkg/connector"
)

// executorBackendConfig holds configuration for the RPC executor backend.
type executorBackendConfig struct {
	URL   string `env:"EXECUTOR_API_URL"`
	Token string `env:"EXECUTOR_API_TOKEN"`
}

// dockerSyncWorkerModule returns the FX module for the Docker sync worker.
// Provides DockerSyncWorker and registers it as a 1s interval jobber.
func dockerSyncWorkerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(
				cfg *pipelineConfig,
				backend pipelinesync.ExecutorBackend,
				executor *pipelinesync.SyncExecutor,
				metrics *pipelinemetrics.MetricsCollector,
				manager *pipelinesync.SyncWorkerManager,
				logger *logging.Logger,
			) *pipelinesync.DockerSyncWorker {
				return pipelinesync.NewDockerSyncWorker(pipelinesync.DockerSyncWorkerParams{
					Backend:           backend,
					Executor:          executor,
					Metrics:           metrics,
					Manager:           manager,
					MaxSyncDuration:   cfg.MaxSyncDuration,
					MaxConcurrentJobs: cfg.MaxConcurrentJobs,
					Logger:            logger,
				})
			},
		),
		pnpjobber.Module(func(cfg *pipelineConfig, worker *pipelinesync.DockerSyncWorker) jobber.Job {
			return jobber.NewIntervalJob(jobber.IntervalJobParams{
				Name:     "docker_sync_worker",
				Interval: cfg.WorkerInterval,
				Job: func(ctx context.Context) error {
					return worker.Execute(ctx)
				},
			})
		}),
	)
}

// k8sSyncWorkerModule returns the FX module for the Kubernetes sync worker.
// Provides K8sSyncWorker with K8sJobCreator adapter and registers it as a 5s interval jobber.
func k8sSyncWorkerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				pipelineadapt.NewK8sJobCreatorAdapter,
				fx.As(new(pipelinesync.K8sJobCreator)),
			),
			func(
				backend pipelinesync.ExecutorBackend,
				setK8sJobName *pipelinejobs.SetK8sJobName,
				k8sRunner pipelinesync.K8sJobCreator,
				k8sCfg *k8sConfig,
				logger *logging.Logger,
			) *pipelinesync.K8sSyncWorker {
				workerID := uuid.New().String()[:8]

				return pipelinesync.NewK8sSyncWorker(
					backend, setK8sJobName, k8sRunner,
					k8sCfg.ServerAddr, workerID, logger,
				)
			},
		),
		pnpjobber.Module(func(worker *pipelinesync.K8sSyncWorker) jobber.Job {
			return jobber.NewIntervalJob(jobber.IntervalJobParams{
				Name:     "k8s_sync_worker",
				Interval: 5 * time.Second,
				Job: func(ctx context.Context) error {
					return worker.Execute(ctx)
				},
			})
		}),
	)
}

// dockerConnectorTaskWorkerModule returns the FX module for the Docker connector task worker.
// Provides DockerConnectorTaskWorker and registers it as an interval jobber.
// Uses a separate goroutine pool from sync workers per D-16.
func dockerConnectorTaskWorkerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(
				cfg *pipelineConfig,
				backend pipelinesync.ExecutorBackend,
				client *connector.ConnectorClient,
				manager *pipelinesync.SyncWorkerManager,
				logger *logging.Logger,
			) *pipelinesync.DockerConnectorTaskWorker {
				return pipelinesync.NewDockerConnectorTaskWorker(pipelinesync.DockerConnectorTaskWorkerParams{
					Backend:       backend,
					Client:        client,
					Manager:       manager,
					MaxConcurrent: 5,
					Logger:        logger,
				})
			},
		),
		pnpjobber.Module(func(cfg *pipelineConfig, worker *pipelinesync.DockerConnectorTaskWorker) jobber.Job {
			return jobber.NewIntervalJob(jobber.IntervalJobParams{
				Name:     "docker_connector_task_worker",
				Interval: cfg.WorkerInterval,
				Job: func(ctx context.Context) error {
					return worker.Execute(ctx)
				},
			})
		}),
	)
}

// k8sConnectorTaskWorkerModule returns the FX module for the Kubernetes connector task worker.
// Provides K8sConnectorTaskWorker and registers it as a 5s interval jobber.
func k8sConnectorTaskWorkerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(
				pipelineadapt.NewK8sConnectorTaskLauncherAdapter,
				fx.As(new(pipelinesync.K8sConnectorTaskLauncher)),
			),
			func(
				backend pipelinesync.ExecutorBackend,
				k8sRunner pipelinesync.K8sConnectorTaskLauncher,
				k8sCfg *k8sConfig,
				logger *logging.Logger,
			) *pipelinesync.K8sConnectorTaskWorker {
				workerID := uuid.New().String()[:8]

				return pipelinesync.NewK8sConnectorTaskWorker(
					backend, k8sRunner,
					k8sCfg.ServerAddr, workerID, logger,
				)
			},
		),
		pnpjobber.Module(func(worker *pipelinesync.K8sConnectorTaskWorker) jobber.Job {
			return jobber.NewIntervalJob(jobber.IntervalJobParams{
				Name:     "k8s_connector_task_worker",
				Interval: 5 * time.Second,
				Job: func(ctx context.Context) error {
					return worker.Execute(ctx)
				},
			})
		}),
	)
}

// standaloneBackendModule provides UseCaseExecutorBackend (in-process, per D-15).
func standaloneBackendModule() fx.Option {
	return fx.Provide(
		fx.Annotate(
			pipelineadapt.NewUseCaseExecutorBackend,
			fx.As(new(pipelinesync.ExecutorBackend)),
		),
	)
}

// rpcBackendModule provides RPCExecutorBackend (distributed, per D-15).
func rpcBackendModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[executorBackendConfig](""),
		configutil.NewPrefixedConfigInfoProvider[executorBackendConfig](""),
		fx.Annotate(
			func(cfg *executorBackendConfig) pipelinesync.ExecutorBackend {
				return pipelineadapt.NewRPCExecutorBackend(cfg.URL, cfg.Token)
			},
			fx.As(new(pipelinesync.ExecutorBackend)),
		),
	)
}

// NewFxAppOptions returns common fx options for all binaries.
func NewFxAppOptions(options *RunAppOptions) fx.Option {
	return fx.Options(
		fx.NopLogger,
		pnpgorm.Module("postgres"),
		pnpenv.Module(),
		pnpzap.Module(),
		pnphealthcheck.Module(),
		pnphealthcheckgorm.Module(),
		fx.Supply(options),

		publicHTTPServerModule(options),
		internalHTTPServerModule(options),

		// Domain modules
		authModule(),
		workspaceModule(options),
		secretModule(),
		pipelineModule(options),
		notifyModule(),

		// K8s orchestration (auto-detected or opt-in via WithK8sExecutor)
		k8sModule(options.K8sExecutor),

		// Auth cleanup job (expired tokens + OIDC states)
		conditionalFxOption(options.RunJobs, func() fx.Option {
			return pnpjobber.Module(func(cleanup *authservice.CleanupExpiredTokens, logger *logging.Logger) jobber.Job {
				return jobber.NewIntervalJob(jobber.IntervalJobParams{
					Name:             "auth_token_cleanup",
					Interval:         1 * time.Hour,
					StartImmediately: false,
					Job: func(ctx context.Context) error {
						if err := cleanup.Execute(ctx); err != nil {
							logger.WithError(err).Error(ctx, "auth token cleanup failed")
						}

						return nil
					},
				})
			})
		}),

		// Metrics
		metricsModule(),

		// ExecutorBackend: standalone mode (use-case adapter) vs distributed mode (RPC adapter) per D-16
		conditionalFxOption(options.Standalone && (options.DockerExecutor || options.K8sExecutor), standaloneBackendModule),
		conditionalFxOption(!options.Standalone && (options.DockerExecutor || options.K8sExecutor), rpcBackendModule),

		// Executor modules (opt-in via CLI flags)
		conditionalFxOption(options.DockerExecutor, dockerSyncWorkerModule),
		conditionalFxOption(options.DockerExecutor, dockerConnectorTaskWorkerModule),
		conditionalFxOption(options.K8sExecutor, k8sSyncWorkerModule),
		conditionalFxOption(options.K8sExecutor, k8sConnectorTaskWorkerModule),

		fx.Options(options.fxOptions...),
	)
}
