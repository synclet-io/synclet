package app

import (
	"context"
	"database/sql"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/healthcheck/pnphealthcheck"
	"github.com/go-pnp/go-pnp/healthcheck/pnphealthcheckgorm"
	"github.com/go-pnp/go-pnp/logging/pnpzap"
	"github.com/go-pnp/go-pnp/pnpenv"
	"github.com/go-pnp/go-pnp/sql/pnpgorm"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineadapt"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexecrpc"
	"github.com/synclet-io/synclet/pkg/jobutil"
)

// executorBackendConfig points an executor process at the server's executor API.
type executorBackendConfig struct {
	URL   string `env:"EXECUTOR_API_URL"`
	Token string `env:"EXECUTOR_API_TOKEN"`
}

// standaloneBackendModule backs an executor embedded in the server with the
// use cases of the same process.
func standaloneBackendModule() fx.Option {
	return fx.Provide(
		fx.Annotate(
			pipelineadapt.NewUseCaseExecutorBackend,
			fx.As(new(pipelineexec.ExecutorBackend)),
		),
	)
}

// rpcBackendModule backs an executor process with the ConnectRPC client of
// the executor API.
func rpcBackendModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[executorBackendConfig](""),
		configutil.NewPrefixedConfigInfoProvider[executorBackendConfig](""),
		fx.Annotate(newRPCExecutorBackend, fx.As(new(pipelineexec.ExecutorBackend))),
	)
}

func newRPCExecutorBackend(cfg *executorBackendConfig) *pipelineexecrpc.Backend {
	return pipelineexecrpc.NewBackend(cfg.URL, cfg.Token)
}

// processModule is the infrastructure every synclet process has: environment,
// logging, health checkers, the worker manager and Prometheus metrics.
func processModule() fx.Option {
	return fx.Options(
		pnpenv.Module(),
		pnpzap.Module(),
		pnphealthcheck.Module(),
		fx.Provide(fx.Annotate(jobutil.NewWorkerManager, fx.OnStop(stopWorkerManager))),
		metricsModule(),
	)
}

// NewFxAppOptions returns the fx graph of the server and jobs commands.
func NewFxAppOptions(options *RunAppOptions) fx.Option {
	return fx.Options(
		fx.NopLogger,
		processModule(),
		pnpgorm.Module("postgres"),
		fx.Provide(func(db *gorm.DB) (*sql.DB, error) {
			return db.DB()
		}),
		pnphealthcheckgorm.Module(),
		fx.Supply(options),

		publicHTTPServerModule(options),
		internalHTTPServerModule(
			options.RunInternalHTTPServer || options.RunJobs,
			healthcheckModule(options.RunJobs, options.RunConsumers),
			conditionalFxOptions(options.RunInternalHTTPServer, pipelineInternalHTTPServerModule),
		),
		mcpConfigModule(options),
		conditionalFxOptions(options.RunMCPServer, func() fx.Option { return mcpHTTPServerModule(options) }),

		// Domain modules
		authModule(options),
		workspaceModule(options),
		secretModule(),
		pipelineModule(options),
		notifyModule(),
		auditModule(options),
		messagingModule(options),

		// Docker executor config is always-on: ContainerRunner serves image
		// pulls and connector tasks whether or not worker jobs run.
		dockerExecutorConfigModule(),

		// An executor flag embeds that executor, backed by the use cases.
		conditionalFxOptions(options.runsExecutor(), standaloneBackendModule, serverExecutorRuntimeModule, executorRegistrationConfigModule),
		conditionalFxOptions(options.DockerExecutor, dockerExecutorModule),
		conditionalFxOptions(options.K8sExecutor, k8sExecutorConfigModule, k8sExecutorModule, serverK8sPodAPITokenModule),

		fx.Options(options.fxOptions...),
	)
}

func stopWorkerManager(ctx context.Context, workerManager *jobutil.WorkerManager) error {
	return workerManager.Close(ctx)
}
