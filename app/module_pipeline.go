package app

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
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
	"github.com/synclet-io/synclet/pkg/logarchive"
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

// pipelineLogArchiveS3Config configures the s3 log archive backend.
type pipelineLogArchiveS3Config struct {
	Endpoint        string `env:"ENDPOINT"`
	Region          string `env:"REGION"`
	Bucket          string `env:"BUCKET"`
	Prefix          string `env:"PREFIX"`
	AccessKeyID     string `env:"ACCESS_KEY_ID"`
	SecretAccessKey string `env:"SECRET_ACCESS_KEY"`
	UsePathStyle    bool   `env:"USE_PATH_STYLE" envDefault:"false"`
}

// pipelineLogArchiveGCSConfig configures the gcs log archive backend.
type pipelineLogArchiveGCSConfig struct {
	Bucket          string `env:"BUCKET"`
	Prefix          string `env:"PREFIX"`
	CredentialsFile string `env:"CREDENTIALS_FILE"`
}

// pipelineLogArchiveConfig selects where the logs of finished jobs are stored.
type pipelineLogArchiveConfig struct {
	Backend string                      `env:"BACKEND" envDefault:"none"`
	Dir     string                      `env:"DIR"`
	S3      pipelineLogArchiveS3Config  `envPrefix:"S3_"`
	GCS     pipelineLogArchiveGCSConfig `envPrefix:"GCS_"`
}

const (
	logArchiveBackendNone       = "none"
	logArchiveBackendFilesystem = "filesystem"
	logArchiveBackendS3         = "s3"
	logArchiveBackendGCS        = "gcs"
)

// pipelineConfig holds pipeline module configuration loaded from environment variables.
type pipelineConfig struct {
	Jobs                        pipelineJobsConfig       `envPrefix:"JOB_"`
	Runtime                     pipelineRuntimeConfig    `envPrefix:"RUNTIME_"`
	MaxConcurrentJobs           int                      `env:"MAX_CONCURRENT_JOBS" envDefault:"10"`
	IdleTimeout                 time.Duration            `env:"IDLE_TIMEOUT" envDefault:"10m"`
	ExecutorAPIToken            string                   `env:"EXECUTOR_API_TOKEN"`
	ConnectorTaskRetention      time.Duration            `env:"CONNECTOR_TASK_RETENTION" envDefault:"24h"`
	ConnectorTaskRunningTimeout time.Duration            `env:"CONNECTOR_TASK_RUNNING_TIMEOUT" envDefault:"5m"`
	ConnectorTaskPendingTimeout time.Duration            `env:"CONNECTOR_TASK_PENDING_TIMEOUT" envDefault:"1m"`
	DefaultMaxAttempts          int                      `env:"DEFAULT_MAX_ATTEMPTS" envDefault:"3"`
	HeartbeatTimeout            time.Duration            `env:"HEARTBEAT_TIMEOUT" envDefault:"30s"`
	ConnectionCheckTimeout      time.Duration            `env:"CONNECTION_CHECK_TIMEOUT" envDefault:"2m"`
	ConnectorTaskPollInterval   time.Duration            `env:"CONNECTOR_TASK_POLL_INTERVAL" envDefault:"500ms"`
	EventEmitTimeout            time.Duration            `env:"EVENT_EMIT_TIMEOUT" envDefault:"30s"`
	StatsRollupHourlyLookback   time.Duration            `env:"STATS_ROLLUP_HOURLY_LOOKBACK" envDefault:"2h"`
	StatsRollupDailyLookback    time.Duration            `env:"STATS_ROLLUP_DAILY_LOOKBACK" envDefault:"48h"`
	RegistryFetchTimeout        time.Duration            `env:"REGISTRY_FETCH_TIMEOUT" envDefault:"60s"`
	LogArchive                  pipelineLogArchiveConfig `envPrefix:"LOG_ARCHIVE_"`
	MaxLogLinesPerAttempt       int                      `env:"LOG_MAX_LINES_PER_ATTEMPT" envDefault:"100000"`
}

// defaults converts pipelineRuntimeConfig to RuntimeDefaults.
func (c pipelineRuntimeConfig) defaults() pipelineservice.RuntimeDefaults {
	return pipelineservice.RuntimeDefaults{
		CPURequest:         c.CPURequest,
		CPULimit:           c.CPULimit,
		MemoryRequest:      c.MemoryRequest,
		MemoryLimit:        c.MemoryLimit,
		ServiceAccountName: c.ServiceAccountName,
	}
}

func pipelineModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		pipelineConfigModule(),
		pipelineMetricsModule(newPipelineMetricsCollector),
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
			pnpwatermill.HandlerProvider(pipelineeventhandle.NewWorkspaceDeletedHandler),
			pnpwatermill.HandlerProvider(pipelineeventhandle.NewJobLogArchiveHandler),
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

// pipelineMetricsModule provides the pipeline metrics collector built by
// newCollector and registers it with Prometheus.
func pipelineMetricsModule(newCollector any) fx.Option {
	return fx.Provide(
		newCollector,
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
			newPipelineLogArchive,
		),

		// Storage
		fx.Provide(
			fx.Annotate(newPipelineStorage, fx.As(new(pipelineservice.Storage))),
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

func newPipelineMetricsCollector(cfg *pipelineConfig) *pipelinemetrics.MetricsCollector {
	metrics := pipelinemetrics.NewMetricsCollector()
	if cfg.LogArchive.Backend != logArchiveBackendNone {
		metrics.InitLogArchiveFailures(cfg.LogArchive.Backend)
	}

	return metrics
}

func pipelineMetricsCollectorAdapter(c *pipelinemetrics.MetricsCollector) prometheus.Collector {
	return c
}

type pipelineSourceHandlerConfigParams struct {
	fx.In

	Options   *RunAppOptions
	AuthCfg   *authConfig
	WsCfg     *workspaceConfig
	NotifyCfg *notifyConfig
	Cfg       *pipelineConfig
	DockerCfg *dockerExecutorConfig
	K8sCfg    *k8sExecutorConfig `optional:"true"`
}

func newPipelineSourceHandlerConfig(params pipelineSourceHandlerConfigParams) pipelineconnect.SourceHandlerConfig {
	executor := pipelineconnect.ExecutorNone
	dockerMaxConcurrentJobs := 0
	k8sDefaultCPU, k8sDefaultMemory := "", ""

	switch {
	case params.Options.DockerExecutor:
		executor = pipelineconnect.ExecutorDocker
		dockerMaxConcurrentJobs = params.DockerCfg.MaxConcurrentJobs
	case params.Options.K8sExecutor:
		executor = pipelineconnect.ExecutorK8s

		if params.K8sCfg != nil {
			k8sDefaultCPU, k8sDefaultMemory = params.K8sCfg.DefaultCPU, params.K8sCfg.DefaultMemory
		}
	}

	return pipelineconnect.SourceHandlerConfig{
		RegistrationEnabled: params.AuthCfg.RegistrationEnabled,
		SingleWorkspaceMode: WorkspaceModeSingle == params.WsCfg.Mode,
		EmailEnabled:        params.NotifyCfg.SMTP.Host != "",
		RuntimeDefaults: pipelineconnect.RuntimeDefaultsInfo{
			Executor:                executor,
			CPURequest:              params.Cfg.Runtime.CPURequest,
			CPULimit:                params.Cfg.Runtime.CPULimit,
			MemoryRequest:           params.Cfg.Runtime.MemoryRequest,
			MemoryLimit:             params.Cfg.Runtime.MemoryLimit,
			ServiceAccountName:      params.Cfg.Runtime.ServiceAccountName,
			MaxConcurrentJobs:       params.Cfg.MaxConcurrentJobs,
			IdleTimeout:             params.Cfg.IdleTimeout,
			LogArchiveBackend:       params.Cfg.LogArchive.Backend,
			DockerMaxConcurrentJobs: dockerMaxConcurrentJobs,
			K8sDefaultCPU:           k8sDefaultCPU,
			K8sDefaultMemory:        k8sDefaultMemory,
		},
	}
}

func newPipelineRuntimeDefaults(cfg *pipelineConfig) pipelineservice.RuntimeDefaults {
	return cfg.Runtime.defaults()
}

func newPipelineServiceConfig(cfg *pipelineConfig) pipelineservice.Config {
	return pipelineservice.Config{
		RuntimeDefaults:             cfg.Runtime.defaults(),
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
		LogArchiveEnabled:           cfg.LogArchive.Backend != logArchiveBackendNone,
		LogArchiveBackend:           cfg.LogArchive.Backend,
		MaxLogLinesPerAttempt:       cfg.MaxLogLinesPerAttempt,
	}
}

func newPipelineLogArchive(lc fx.Lifecycle, cfg *pipelineConfig, logger *logging.Logger) (pipelineservice.LogArchive, error) {
	archiveCfg := cfg.LogArchive

	switch archiveCfg.Backend {
	case logArchiveBackendNone:
		return logarchive.None{}, nil
	case logArchiveBackendFilesystem:
		if archiveCfg.Dir == "" {
			return nil, fmt.Errorf("PIPELINE_LOG_ARCHIVE_DIR is required when PIPELINE_LOG_ARCHIVE_BACKEND=%s", logArchiveBackendFilesystem)
		}

		archive, err := logarchive.NewFilesystem(archiveCfg.Dir)
		if err != nil {
			return nil, fmt.Errorf("PIPELINE_LOG_ARCHIVE_DIR: %w", err)
		}

		return archive, nil
	case logArchiveBackendS3:
		s3Cfg := archiveCfg.S3
		if s3Cfg.Bucket == "" {
			return nil, fmt.Errorf("PIPELINE_LOG_ARCHIVE_S3_BUCKET is required when PIPELINE_LOG_ARCHIVE_BACKEND=%s", logArchiveBackendS3)
		}

		if (s3Cfg.AccessKeyID == "") != (s3Cfg.SecretAccessKey == "") {
			return nil, errors.New("PIPELINE_LOG_ARCHIVE_S3_ACCESS_KEY_ID and PIPELINE_LOG_ARCHIVE_S3_SECRET_ACCESS_KEY must be set together")
		}

		archive, err := logarchive.NewS3(context.Background(), logarchive.S3Config{
			Endpoint:        s3Cfg.Endpoint,
			Region:          s3Cfg.Region,
			Bucket:          s3Cfg.Bucket,
			Prefix:          s3Cfg.Prefix,
			AccessKeyID:     s3Cfg.AccessKeyID,
			SecretAccessKey: s3Cfg.SecretAccessKey,
			UsePathStyle:    s3Cfg.UsePathStyle,
		})
		if errors.Is(err, logarchive.ErrNoRegion) {
			return nil, fmt.Errorf("PIPELINE_LOG_ARCHIVE_S3_REGION is required when PIPELINE_LOG_ARCHIVE_BACKEND=%s and AWS_REGION is unset", logArchiveBackendS3)
		}

		if err != nil {
			return nil, fmt.Errorf("configuring PIPELINE_LOG_ARCHIVE_S3_*: %w", err)
		}

		return archive, nil
	case logArchiveBackendGCS:
		return newGCSLogArchive(lc, archiveCfg.GCS, logger)
	default:
		return nil, fmt.Errorf("PIPELINE_LOG_ARCHIVE_BACKEND=%q is not one of %s, %s, %s, %s",
			archiveCfg.Backend, logArchiveBackendNone, logArchiveBackendFilesystem, logArchiveBackendS3, logArchiveBackendGCS)
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

func newGCSLogArchive(lc fx.Lifecycle, cfg pipelineLogArchiveGCSConfig, logger *logging.Logger) (pipelineservice.LogArchive, error) {
	if cfg.Bucket == "" {
		return nil, fmt.Errorf("PIPELINE_LOG_ARCHIVE_GCS_BUCKET is required when PIPELINE_LOG_ARCHIVE_BACKEND=%s", logArchiveBackendGCS)
	}

	if cfg.CredentialsFile != "" {
		if _, err := os.Stat(cfg.CredentialsFile); err != nil {
			return nil, fmt.Errorf("PIPELINE_LOG_ARCHIVE_GCS_CREDENTIALS_FILE: %w", err)
		}
	}

	archive, err := logarchive.NewGCS(context.Background(), logarchive.GCSConfig{
		Bucket:          cfg.Bucket,
		Prefix:          cfg.Prefix,
		CredentialsFile: cfg.CredentialsFile,
	})
	if err != nil {
		return nil, fmt.Errorf("configuring PIPELINE_LOG_ARCHIVE_GCS_*: %w", err)
	}

	logger.WithFields(map[string]interface{}{
		"bucket":            cfg.Bucket,
		"credential_source": archive.CredentialSource(),
	}).Info(context.Background(), "job log archive: using GCS")

	lc.Append(fx.StopHook(archive.Close))

	return archive, nil
}
