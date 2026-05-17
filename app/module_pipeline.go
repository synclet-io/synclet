package app

import (
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/fxutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/prometheus/pnpprometheus"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineadapt"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineconnect"
	_ "github.com/synclet-io/synclet/modules/pipeline/pipelinedbstate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineeventhandle"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinemetrics"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesettings"
	"github.com/synclet-io/synclet/modules/pipeline/pipelinestorage"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// pipelineJobsConfig holds intervals for all pipeline background jobs.
type pipelineJobsConfig struct {
	SchedulerInterval          time.Duration `env:"SCHEDULER_INTERVAL" envDefault:"30s"`
	WatchdogInterval           time.Duration `env:"WATCHDOG_INTERVAL" envDefault:"10s"`
	JobRetention               time.Duration `env:"RETENTION_INTERVAL" envDefault:"10m"`
	StatsRollup                time.Duration `env:"STATS_ROLLUP_INTERVAL" envDefault:"5m"`
	RepositorySync             time.Duration `env:"REPOSITORY_SYNC_INTERVAL" envDefault:"1h"`
	ConnectorTaskRetention     time.Duration `env:"CONNECTOR_TASK_RETENTION_INTERVAL" envDefault:"5m"`
	StaleConnectorTaskWatchdog time.Duration `env:"STALE_CONNECTOR_TASK_WATCHDOG_INTERVAL" envDefault:"5m"`
}

// pipelineRuntimeConfig holds K8s resource defaults for pipeline containers.
type pipelineRuntimeConfig struct {
	CPURequest         string `env:"CPU_REQUEST"`
	CPULimit           string `env:"CPU_LIMIT"`
	MemoryRequest      string `env:"MEMORY_REQUEST"`
	MemoryLimit        string `env:"MEMORY_LIMIT"`
	ServiceAccountName string `env:"SERVICE_ACCOUNT_NAME"`
}

// pipelineConfig holds pipeline module configuration loaded from environment variables.
type pipelineConfig struct {
	Jobs                        pipelineJobsConfig    `envPrefix:"JOB_"`
	Runtime                     pipelineRuntimeConfig `envPrefix:"RUNTIME_"`
	MaxConcurrentJobs           int                   `env:"MAX_CONCURRENT_JOBS" envDefault:"10"`
	IdleTimeout                 time.Duration         `env:"IDLE_TIMEOUT" envDefault:"10m"`
	ExecutorAPIToken            string                `env:"EXECUTOR_API_TOKEN"`
	ConnectorTaskRetention      time.Duration         `env:"CONNECTOR_TASK_RETENTION" envDefault:"24h"`
	ConnectorTaskRunningTimeout time.Duration         `env:"CONNECTOR_TASK_RUNNING_TIMEOUT" envDefault:"5m"`
	ConnectorTaskPendingTimeout time.Duration         `env:"CONNECTOR_TASK_PENDING_TIMEOUT" envDefault:"1m"`
	DefaultMaxAttempts          int                   `env:"DEFAULT_MAX_ATTEMPTS" envDefault:"3"`
	HeartbeatTimeout            time.Duration         `env:"HEARTBEAT_TIMEOUT" envDefault:"30s"`
	ConnectionCheckTimeout      time.Duration         `env:"CONNECTION_CHECK_TIMEOUT" envDefault:"2m"`
	ConnectorTaskPollInterval   time.Duration         `env:"CONNECTOR_TASK_POLL_INTERVAL" envDefault:"500ms"`
	EventEmitTimeout            time.Duration         `env:"EVENT_EMIT_TIMEOUT" envDefault:"30s"`
	StatsRollupHourlyLookback   time.Duration         `env:"STATS_ROLLUP_HOURLY_LOOKBACK" envDefault:"2h"`
	StatsRollupDailyLookback    time.Duration         `env:"STATS_ROLLUP_DAILY_LOOKBACK" envDefault:"48h"`
	RegistryFetchTimeout        time.Duration         `env:"REGISTRY_FETCH_TIMEOUT" envDefault:"60s"`
}

// runtimeDefaults converts pipelineRuntimeConfig to RuntimeDefaults.
func (c *pipelineConfig) runtimeDefaults() pipelineservice.RuntimeDefaults {
	return pipelineservice.RuntimeDefaults{
		CPURequest:         c.Runtime.CPURequest,
		CPULimit:           c.Runtime.CPULimit,
		MemoryRequest:      c.Runtime.MemoryRequest,
		MemoryLimit:        c.Runtime.MemoryLimit,
		ServiceAccountName: c.Runtime.ServiceAccountName,
	}
}

func pipelineModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		pipelineConfigModule(),
		pipelineMetricsModule(),
		pipelineDependenciesModule(),
		pipelineUseCasesModule(),
		pipelineEventHandlersModule(),
		pipelineJobsModule(options),
	)
}

func pipelineEventHandlersModule() fx.Option {
	return fx.Options(
		fx.Provide(
			pnpwatermill.HandlerProvider(pipelineeventhandle.NewWorkspaceCreatedHandler),
		),
	)
}

func pipelineConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[pipelineConfig]("PIPELINE_"),
		configutil.NewPrefixedConfigInfoProvider[pipelineConfig]("PIPELINE_"),
		newPipelineSourceHandlerConfig,
		newPipelineRuntimeDefaults,
		newPipelineServiceConfig,
	)
}

func pipelineMetricsModule() fx.Option {
	return fx.Provide(
		pipelinemetrics.NewMetricsCollector,
		pnpprometheus.MetricsCollectorProvider(pipelineMetricsCollectorAdapter),
	)
}

func pipelineDependenciesModule() fx.Option {
	return fx.Options(
		// Adapters
		fx.Provide(
			fx.Annotate(pipelineadapt.NewConnectorDiscoverAdapter, fx.As(new(pipelineservice.ConnectorDiscoverer))),
			fx.Annotate(pipelineadapt.NewDBImageValidator, fx.As(new(pipelineservice.ConnectorImageValidator))),
			fx.Annotate(pipelineadapt.NewConnectorSpecFetcherAdapter, fx.As(new(pipelineservice.ConnectorSpecFetcher))),
			fx.Annotate(pipelineadapt.NewImagePullerAdapter, fx.As(new(pipelineservice.ImagePuller))),
			fx.Annotate(pipelineadapt.NewAuditRecorder, fx.As(new(pipelineservice.AuditRecorder))),
		),

		// Storage
		fx.Provide(
			fx.Annotate(newPipelineStorage, fx.As(new(pipelineservice.Storage))),
			fx.Annotate(pipelineadapt.NewEventEmitterAdapter, fx.As(new(pipelineservice.SyncEventEmitter))),
			fx.Annotate(pipelinestorage.NewJobRetentionStorage, fx.As(new(pipelineservice.JobRetentionStorage))),
			fx.Annotate(pipelinestorage.NewWorkspaceSettingsWriter, fx.As(new(pipelinesettings.WorkspaceSettingsWriter))),
		),

		// Secrets
		fx.Provide(
			fx.Annotate(pipelineadapt.NewDBSecretsProvider, fx.As(new(pipelineservice.SecretsProvider))),
		),

		// Stats
		fx.Provide(
			fx.Annotate(pipelinestorage.NewStatsStorage, fx.As(new(pipelineservice.StatsStorage))),
		),
	)
}

func pipelineMetricsCollectorAdapter(c *pipelinemetrics.MetricsCollector) prometheus.Collector {
	return c
}

func newPipelineSourceHandlerConfig(authCfg *authConfig, wsCfg *workspaceConfig) pipelineconnect.SourceHandlerConfig {
	return pipelineconnect.SourceHandlerConfig{
		RegistrationEnabled: authCfg.RegistrationEnabled,
		SingleWorkspaceMode: WorkspaceModeSingle == wsCfg.Mode,
	}
}

func newPipelineRuntimeDefaults(cfg *pipelineConfig) pipelineservice.RuntimeDefaults {
	return cfg.runtimeDefaults()
}

func newPipelineServiceConfig(cfg *pipelineConfig) pipelineservice.Config {
	return pipelineservice.Config{
		RuntimeDefaults:             cfg.runtimeDefaults(),
		IdleTimeout:                 cfg.IdleTimeout,
		MaxConcurrentJobs:           cfg.MaxConcurrentJobs,
		ConnectorTaskRetention:      cfg.ConnectorTaskRetention,
		ConnectorTaskRunningTimeout: cfg.ConnectorTaskRunningTimeout,
		ConnectorTaskPendingTimeout: cfg.ConnectorTaskPendingTimeout,
		DefaultMaxAttempts:          cfg.DefaultMaxAttempts,
		ConnectionCheckTimeout:      cfg.ConnectionCheckTimeout,
		ConnectorTaskPollInterval:   cfg.ConnectorTaskPollInterval,
		EventEmitTimeout:            cfg.EventEmitTimeout,
		StatsRollupHourlyLookback:   cfg.StatsRollupHourlyLookback,
		StatsRollupDailyLookback:    cfg.StatsRollupDailyLookback,
		RegistryFetchTimeout:        cfg.RegistryFetchTimeout,
		HeartbeatTimeout:            cfg.HeartbeatTimeout,
	}
}

func newPipelineStorage(db *gorm.DB, logger *logging.Logger) *pipelinestorage.Storages {
	return pipelinestorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
}

type newExecutorMuxRegistrarParams struct {
	fx.In

	Config  *pipelineConfig
	Handler executorv1connect.ExecutorServiceHandler
}

func newExecutorHandlerConstructorProvider() any {
	return fxutil.GroupProvider[pnpconnectrpchandling.ConnectHandlerConstructor](
		"pnpconnectrpchandling.handler_constructors",
		func(params newExecutorMuxRegistrarParams) pnpconnectrpchandling.ConnectHandlerConstructor {
			return func(opts ...connect.HandlerOption) (string, http.Handler) {
				allOpts := make([]connect.HandlerOption, 0, len(opts))
				allOpts = append(allOpts, opts...)
				allOpts = append(allOpts, connect.WithInterceptors(
					connectutil.NewInternalSecretInterceptor(params.Config.ExecutorAPIToken),
				))

				return executorv1connect.NewExecutorServiceHandler(params.Handler, allOpts...)
			}
		})
}

func pipelineInternalHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(pipelineconnect.NewExecutorHandler, fx.As(new(executorv1connect.ExecutorServiceHandler))),
			newExecutorHandlerConstructorProvider(),
			fx.Private,
		),
	)
}

func pipelineHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
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
