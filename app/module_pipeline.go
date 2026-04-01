package app

import (
	"context"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/go-pnp/prometheus/pnpprometheus"
	"github.com/go-pnp/jobber"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineadapt"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineconnect"
	_ "github.com/synclet-io/synclet/modules/pipeline/pipelinedbstate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconfig"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinerepositories"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesettings"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/pkg/connector"
	"github.com/synclet-io/synclet/pkg/container"
	"github.com/synclet-io/synclet/pkg/docker"
)

// pipelineConfig holds pipeline module configuration loaded from environment variables.
type pipelineConfig struct {
	WorkerInterval            time.Duration `env:"WORKER_INTERVAL" envDefault:"1s"`
	SchedulerInterval         time.Duration `env:"SCHEDULER_INTERVAL" envDefault:"30s"`
	WatchdogInterval          time.Duration `env:"WATCHDOG_INTERVAL" envDefault:"10s"`
	OrphanCleanupInterval     time.Duration `env:"ORPHAN_CLEANUP_INTERVAL" envDefault:"5m"`
	MaxSyncDuration           time.Duration `env:"MAX_SYNC_DURATION" envDefault:"24h"`
	MaxConcurrentJobs         int           `env:"MAX_CONCURRENT_JOBS" envDefault:"10"`
	IdleTimeout               time.Duration `env:"IDLE_TIMEOUT" envDefault:"10m"`
	DefaultCPURequest         string        `env:"DEFAULT_CPU_REQUEST" envDefault:""`
	DefaultCPULimit           string        `env:"DEFAULT_CPU_LIMIT" envDefault:""`
	DefaultMemoryRequest      string        `env:"DEFAULT_MEMORY_REQUEST" envDefault:""`
	DefaultMemoryLimit        string        `env:"DEFAULT_MEMORY_LIMIT" envDefault:""`
	DefaultServiceAccountName string        `env:"DEFAULT_SERVICE_ACCOUNT_NAME" envDefault:""`
	ConnectorTaskRetention    time.Duration `env:"CONNECTOR_TASK_RETENTION" envDefault:"24h"`
	ConnectorTaskTimeout      time.Duration `env:"CONNECTOR_TASK_TIMEOUT" envDefault:"5m"`
}

// runtimeDefaults converts pipelineConfig env vars to RuntimeDefaults.
func (c *pipelineConfig) runtimeDefaults() pipelineservice.RuntimeDefaults {
	return pipelineservice.RuntimeDefaults{
		CPURequest:         c.DefaultCPURequest,
		CPULimit:           c.DefaultCPULimit,
		MemoryRequest:      c.DefaultMemoryRequest,
		MemoryLimit:        c.DefaultMemoryLimit,
		ServiceAccountName: c.DefaultServiceAccountName,
	}
}

// k8sSyncStopperOptional wraps the optional K8sSyncStopper dependency.
// In Docker mode, Stopper is nil. In K8s mode, it's provided by k8sModule.
type k8sSyncStopperOptional struct {
	fx.In
	Stopper pipelinejobs.K8sSyncStopper `optional:"true"`
}

func newStaleJobWatchdogJob(cfg *pipelineConfig, recoverStale *pipelinejobs.RecoverStaleJobs,
	reportCompletion *pipelinejobs.ReportCompletion, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "stale_job_watchdog",
		Interval:         cfg.WatchdogInterval,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			recovered, err := recoverStale.Execute(ctx, pipelinejobs.RecoverStaleJobsParams{
				HeartbeatTimeout: cfg.WatchdogInterval,
			})
			if err != nil {
				return err
			}

			if recovered > 0 {
				logger.WithField("count", recovered).Info(ctx, "watchdog: recovered stale jobs")
			}

			return nil
		},
	})
}

func newOrphanCleanupJob(cfg *pipelineConfig, cleaner *docker.OrphanCleaner, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "orphan_cleanup",
		Interval:         cfg.OrphanCleanupInterval,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			if err := cleaner.Cleanup(ctx); err != nil {
				logger.WithError(err).Error(ctx, "orphan cleanup failed")
			}
			// Always return nil so jobber doesn't mark the job as failed.
			return nil
		},
	})
}

func newSyncSchedulerJob(cfg *pipelineConfig, syncScheduler *pipelinesync.SyncScheduler) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "sync_scheduler",
		Interval:         cfg.SchedulerInterval,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			return syncScheduler.Execute(ctx)
		},
	})
}

func newJobRetentionJob(cleanup *pipelinejobs.CleanupOldJobs, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "job_retention",
		Interval:         10 * time.Minute,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := cleanup.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "job retention cleanup failed")
			}
			// Always return nil so jobber doesn't mark the job as failed.
			return nil
		},
	})
}

func newStatsRollupJob(computeRollup *pipelinestats.ComputeStatsRollup, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "stats_rollup",
		Interval:         5 * time.Minute,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			if err := computeRollup.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "stats rollup failed")
			}
			// Always return nil so jobber doesn't mark the job as failed.
			return nil
		},
	})
}

func newConnectorTaskCleanupJob(cleanup *pipelinetasks.CleanupTasks, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "connector_task_cleanup",
		Interval:         5 * time.Minute,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := cleanup.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "connector task cleanup failed")
			}
			// Always return nil so jobber doesn't mark the job as failed.
			return nil
		},
	})
}

// executorBackendOrphanChecker adapts ExecutorBackend.IsJobActive for docker.OrphanJobChecker.
// Works for both standalone and distributed modes since ExecutorBackend already has IsJobActive.
type executorBackendOrphanChecker struct {
	backend pipelinesync.ExecutorBackend
}

func (c *executorBackendOrphanChecker) IsJobActive(ctx context.Context, jobID string) (bool, error) {
	return c.backend.IsJobActive(ctx, jobID)
}

func pipelineModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		// Metrics collector for pipeline module
		fx.Provide(
			pipelinemetrics.NewMetricsCollector,
			pnpprometheus.MetricsCollectorProvider(func(c *pipelinemetrics.MetricsCollector) prometheus.Collector { return c }),
		),

		// Connector infrastructure (container runner, connector client)
		fx.Provide(
			configutil.NewPrefixedConfigProvider[pipelineConfig]("PIPELINE_"),
			configutil.NewPrefixedConfigInfoProvider[pipelineConfig]("PIPELINE_"),

			docker.NewContainerRunner,
			fx.Annotate(func(r *docker.ContainerRunner) container.Runner { return r }, fx.As(new(container.Runner))),
			connector.NewConnectorClient,
			fx.Annotate(pipelineadapt.NewConnectorCheckAdapter, fx.As(new(pipelineservice.ConnectorClient))),
			fx.Annotate(pipelineadapt.NewConnectorDiscoverAdapter, fx.As(new(pipelineservice.ConnectorDiscoverer))),
			fx.Annotate(pipelineadapt.NewConnectorSourceReader, fx.As(new(pipelineservice.SourceReader))),
			fx.Annotate(pipelineadapt.NewConnectorDestinationWriter, fx.As(new(pipelineservice.DestinationWriter))),
			fx.Annotate(pipelineadapt.NewDBImageValidator, fx.As(new(pipelineservice.ConnectorImageValidator))),
			fx.Annotate(pipelineadapt.NewConnectorSpecFetcherAdapter, fx.As(new(pipelineservice.ConnectorSpecFetcher))),
			fx.Annotate(pipelineadapt.NewImagePullerAdapter, fx.As(new(pipelineservice.ImagePuller))),
			func(re registrationEnabled) pipelineconnect.RegistrationEnabled {
				return pipelineconnect.RegistrationEnabled(re)
			},
			func(cfg *pipelineConfig) pipelineservice.RuntimeDefaults {
				return cfg.runtimeDefaults()
			},

			fx.Annotate(
				func(db *gorm.DB, logger *logging.Logger) *pipelinestorage.Storages {
					return pipelinestorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
				},
				fx.As(new(pipelineservice.Storage)),
			),
			fx.Annotate(pipelineadapt.NewEventEmitterAdapter, fx.As(new(pipelineservice.SyncEventEmitter))),
			fx.Annotate(pipelinestorage.NewJobRetentionStorage, fx.As(new(pipelineservice.JobRetentionStorage))),
			fx.Annotate(
				pipelinestorage.NewWorkspaceSettingsWriter,
				fx.As(new(pipelinesettings.WorkspaceSettingsWriter)),
			),
		),

		// Secrets provider adapter (use cases provided by secretModule)
		fx.Provide(
			fx.Annotate(
				pipelineadapt.NewDBSecretsProvider,
				fx.As(new(pipelineservice.SecretsProvider)),
			),
		),

		// Connector use cases
		fx.Provide(
			pipelineconnectors.NewAddConnector,
			pipelineconnectors.NewGetConnector,
			pipelineconnectors.NewGetConnectorSpec,
			pipelineconnectors.NewListConnectors,
			pipelineconnectors.NewDeleteConnector,
			pipelineconnectors.NewUpdateManagedConnector,
			pipelineconnectors.NewBatchUpdateConnectors,
			pipelineconnectors.NewListConnectorsWithUpdateInfo,
			pipelineconnectors.NewGetConnectorWithUpdateInfo,
		),

		// Repository use cases
		fx.Provide(
			pipelinerepositories.NewValidateImage,
			pipelinerepositories.NewRegistryFetcher,
			pipelinerepositories.NewSyncRepository,
			pipelinerepositories.NewAddRepository,
			pipelinerepositories.NewListRepositories,
			pipelinerepositories.NewDeleteRepository,
			pipelinerepositories.NewListRepositoryConnectors,
			pipelinerepositories.NewGetConnectorVersions,
			pipelinerepositories.NewRepositorySyncer,
		),

		// Source use cases
		fx.Provide(
			pipelinesources.NewCreateSource,
			pipelinesources.NewUpdateSource,
			pipelinesources.NewDeleteSource,
			pipelinesources.NewGetSource,
			pipelinesources.NewListSources,
			pipelinesources.NewTestSourceConnection,
			pipelinesources.NewUpdateSourceInternal,
		),

		// Destination use cases
		fx.Provide(
			pipelinedestinations.NewCreateDestination,
			pipelinedestinations.NewUpdateDestination,
			pipelinedestinations.NewDeleteDestination,
			pipelinedestinations.NewGetDestination,
			pipelinedestinations.NewListDestinations,
			pipelinedestinations.NewTestDestinationConnection,
			pipelinedestinations.NewUpdateDestinationInternal,
		),

		// Connection use cases
		fx.Provide(
			pipelineconnections.NewCreateConnection,
			pipelineconnections.NewUpdateConnection,
			pipelineconnections.NewDeleteConnection,
			pipelineconnections.NewGetConnection,
			pipelineconnections.NewListConnections,
			pipelineconnections.NewUpdateConnectionStatus,
			pipelineconnections.NewEnableConnection,
			pipelineconnections.NewDisableConnection,
		),

		// Config import/export use cases
		fx.Provide(
			pipelineconfig.NewExportConfig,
			pipelineconfig.NewImportConfig,
		),

		// Catalog use cases
		fx.Provide(
			pipelinecatalog.NewDiscoverCatalog,
			pipelinecatalog.NewGetConfiguredCatalog,
			pipelinecatalog.NewConfigureStreams,
			pipelinecatalog.NewDetectSchemaChanges,
			pipelinecatalog.NewGetDiscoveredCatalogForConnection,
			pipelinecatalog.NewGetSourceCatalog,
			pipelinecatalog.NewPopulateGenerationIDs,
		),

		// Job use cases
		fx.Provide(
			pipelinejobs.NewFindStaleJobs,
			pipelinejobs.NewIsJobActive,
			pipelinejobs.NewIsTaskActive,
			pipelinejobs.NewGetLaunchBundle,
			pipelinejobs.NewCountConnectionJobs,
			pipelinejobs.NewQueueJob,
			pipelinejobs.NewGetJob,
			pipelinejobs.NewListJobs,
			pipelinejobs.NewListJobAttempts,
			func(storage pipelineservice.Storage, opts k8sSyncStopperOptional, logger *logging.Logger) *pipelinejobs.CancelJob {
				return pipelinejobs.NewCancelJob(storage, opts.Stopper, logger)
			},
			pipelinejobs.NewClaimJob,
			pipelinejobs.NewUpdateJobStatus,
			pipelinejobs.NewUpdateHeartbeat,
			pipelinejobs.NewRecoverStaleJobs,
			pipelinejobs.NewSetK8sJobName,
			pipelinejobs.NewTriggerSync,
			pipelinejobs.NewCancelJobForWorkspace,
			pipelinejobs.NewGetJobWithAttempts,
			pipelinejobs.NewListJobsWithAttempts,
			pipelinejobs.NewReportCompletion,
			pipelinejobs.NewHandleConfigUpdate,
			fx.Annotate(
				pipelinesettings.NewGetWorkspaceSettings,
				fx.As(new(pipelinejobs.SettingsProvider)),
				fx.As(fx.Self()),
			),
			pipelinesettings.NewUpdateWorkspaceSettings,
			pipelinejobs.NewCleanupOldJobs,
			pipelinejobs.NewClaimJobBundle,
			pipelinejobs.NewCheckJobCancelled,
		),

		// Connector task use cases
		fx.Provide(
			pipelinetasks.NewCreateCheckTask,
			pipelinetasks.NewCreateSpecTask,
			pipelinetasks.NewCreateDiscoverTask,
			pipelinetasks.NewGetTaskResult,
			pipelinetasks.NewWaitForTaskResult,
			pipelinetasks.NewClaimTask,
			pipelinetasks.NewReportTaskResult,
			func(cfg *pipelineConfig, storage pipelineservice.Storage) (*pipelinetasks.CleanupTasks, error) {
				return pipelinetasks.NewCleanupTasks(storage, pipelinetasks.CleanupTasksConfig{
					RetentionPeriod: cfg.ConnectorTaskRetention,
					PendingTimeout:  cfg.ConnectorTaskTimeout,
					RunningTimeout:  10 * time.Minute,
				})
			},
		),

		// State use cases
		fx.Provide(
			pipelinestate.NewGetSyncState,
			pipelinestate.NewSaveSyncState,
			pipelinestate.NewClearSyncState,
			pipelinestate.NewListSyncStates,
			pipelinestate.NewResetStreamState,
			pipelinestate.NewResetConnectionState,
			pipelinestate.NewUpdateStreamState,
			pipelinestate.NewListStreamStates,
		),

		// Log use cases
		fx.Provide(
			pipelinelogs.NewAppendJobLog,
			pipelinelogs.NewGetJobLog,
			pipelinelogs.NewBatchAppendJobLogs,
		),

		// Sync worker manager (tracks active sync goroutines for orderly shutdown)
		fx.Provide(
			func(logger *logging.Logger) *pipelinesync.SyncWorkerManager {
				return pipelinesync.NewSyncWorkerManager(context.Background(), logger)
			},
		),

		// Sync use cases
		fx.Provide(
			func(
				cfg *pipelineConfig,
				sourceReader pipelineservice.SourceReader,
				destWriter pipelineservice.DestinationWriter,
				saveSyncState *pipelinestate.SaveSyncState,
				appendJobLog *pipelinelogs.AppendJobLog,
				handleConfigUpdate *pipelinejobs.HandleConfigUpdate,
				logger *logging.Logger,
			) *pipelinesync.SyncExecutor {
				return pipelinesync.NewSyncExecutor(pipelinesync.SyncExecutorParams{
					SourceReader:       sourceReader,
					DestWriter:         destWriter,
					SaveSyncState:      saveSyncState,
					AppendJobLog:       appendJobLog,
					HandleConfigUpdate: handleConfigUpdate,
					RuntimeDefaults:    cfg.runtimeDefaults(),
					IdleTimeout:        cfg.IdleTimeout,
					Logger:             logger,
				})
			},
			func(cfg *pipelineConfig, storage pipelineservice.Storage, logger *logging.Logger) *pipelinesync.SyncScheduler {
				return pipelinesync.NewSyncScheduler(storage, cfg.MaxConcurrentJobs, logger)
			},
		),

		// Stats storage + use cases
		fx.Provide(
			fx.Annotate(
				pipelinestorage.NewStatsStorage,
				fx.As(new(pipelineservice.StatsStorage)),
			),
			pipelinestats.NewGetWorkspaceStats,
			pipelinestats.NewGetConnectionStats,
			pipelinestats.NewGetSyncTimeline,
			pipelinestats.NewComputeStatsRollup,
		),

		// Docker orphan cleanup module (only when DockerExecutor is true per D-08).
		// Server-only builds must NOT include orphan cleanup (they have no Docker access).
		conditionalFxOption(options.DockerExecutor, func() fx.Option {
			return fx.Options(
				fx.Provide(
					docker.NewOrphanCleaner,
					fx.Annotate(
						func(backend pipelinesync.ExecutorBackend) *executorBackendOrphanChecker {
							return &executorBackendOrphanChecker{backend: backend}
						},
						fx.As(new(docker.OrphanJobChecker)),
					),
				),
			)
		}),

		// Background jobs (only when RunJobs is enabled)
		conditionalFxOption(options.RunJobs, func() fx.Option {
			return fx.Options(
				// FIRST: Startup recovery + shutdown drain lifecycle hook.
				// Must be registered BEFORE pnpjobber.Module calls so that OnStart
				// runs before workers begin polling.
				fx.Invoke(func(
					lc fx.Lifecycle,
					manager *pipelinesync.SyncWorkerManager,
					recoverStale *pipelinejobs.RecoverStaleJobs,
					reportCompletion *pipelinejobs.ReportCompletion,
					logger *logging.Logger,
				) {
					lc.Append(fx.Hook{
						OnStart: func(ctx context.Context) error {
							// Mark ALL running jobs from previous process as failed.
							// HeartbeatTimeout=0 means "heartbeat before now" = all running jobs are stale.
							recovered, err := recoverStale.Execute(ctx, pipelinejobs.RecoverStaleJobsParams{
								HeartbeatTimeout: 0,
							})
							if err != nil {
								logger.WithError(err).Error(ctx, "startup: failed to recover stale jobs")
							} else if recovered > 0 {
								logger.WithField("count", recovered).Info(ctx, "startup: recovered stale jobs")
							}

							return nil
						},
						OnStop: func(ctx context.Context) error {
							if err := manager.Shutdown(45 * time.Second); err != nil {
								logger.Warn(ctx, "shutdown: "+err.Error())
							}
							// Drain background event emissions and retention cleanup goroutines.
							reportCompletion.Wait()

							return nil
						},
					})
				}),
				// Docker orphan startup cleanup + periodic jobber (only when DockerExecutor is true per D-08).
				conditionalFxOption(options.DockerExecutor, func() fx.Option {
					return fx.Options(
						fx.Invoke(func(lc fx.Lifecycle, cleaner *docker.OrphanCleaner, logger *logging.Logger) {
							lc.Append(fx.Hook{
								OnStart: func(ctx context.Context) error {
									if err := cleaner.CleanupAll(ctx); err != nil {
										logger.WithError(err).Error(ctx, "startup: orphan cleanup failed")
									}

									return nil
								},
							})
						}),
						pnpjobber.Module(newOrphanCleanupJob),
					)
				}),
				// Worker modules registered AFTER recovery hook so they start polling
				// only after stale jobs and orphan containers have been cleaned up.
				pnpjobber.Module(newSyncSchedulerJob),
				pnpjobber.Module(newStaleJobWatchdogJob),
				pnpjobber.Module(newStatsRollupJob),
				pnpjobber.Module(newJobRetentionJob),
				pnpjobber.Module(newConnectorTaskCleanupJob),
				fx.Invoke(func(lc fx.Lifecycle, repoSyncer *pipelinerepositories.RepositorySyncer) {
					runCtx, cancel := context.WithCancel(context.Background())

					lc.Append(fx.Hook{
						OnStart: func(_ context.Context) error {
							go repoSyncer.Run(runCtx)

							return nil
						},
						OnStop: func(_ context.Context) error {
							cancel()

							return nil
						},
					})
				}),
			)
		}),
	)
}

func pipelineHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			// 4 separate handlers for the 4 pipeline services
			fx.Annotate(pipelineconnect.NewSourceHandler, fx.As(new(pipelinev1connect.SourceServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(pipelinev1connect.NewSourceServiceHandler),

			fx.Annotate(pipelineconnect.NewDestinationHandler, fx.As(new(pipelinev1connect.DestinationServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(pipelinev1connect.NewDestinationServiceHandler),

			fx.Annotate(pipelineconnect.NewConnectionHandler, fx.As(new(pipelinev1connect.ConnectionServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(pipelinev1connect.NewConnectionServiceHandler),

			fx.Annotate(pipelineconnect.NewJobHandler, fx.As(new(pipelinev1connect.JobServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(pipelinev1connect.NewJobServiceHandler),

			// Stats handler
			fx.Annotate(pipelineconnect.NewStatsHandler, fx.As(new(statsv1connect.StatsServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(statsv1connect.NewStatsServiceHandler),

			// Connector registry handler (absorbed from connector module)
			fx.Annotate(pipelineconnect.NewRegistryHandler, fx.As(new(registryv1connect.ConnectorRegistryServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(registryv1connect.NewConnectorRegistryServiceHandler),

			// Connector task handler (async task result polling)
			fx.Annotate(pipelineconnect.NewConnectorTaskHandler, fx.As(new(pipelinev1connect.ConnectorTaskServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(pipelinev1connect.NewConnectorTaskServiceHandler),

			fx.Private,
		),
	)
}
