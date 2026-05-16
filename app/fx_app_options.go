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
	"github.com/synclet-io/synclet/pkg/jobutil"
)

// executorBackendConfig holds configuration for the RPC executor backend.
type executorBackendConfig struct {
	URL   string `env:"EXECUTOR_API_URL"`
	Token string `env:"EXECUTOR_API_TOKEN"`
}

// standaloneBackendModule provides UseCaseExecutorBackend (in-process, per D-15).
func standaloneBackendModule() fx.Option {
	return fx.Provide(
		fx.Annotate(
			pipelineadapt.NewUseCaseExecutorBackend,
			fx.As(new(pipelineexec.ExecutorBackend)),
		),
	)
}

// rpcBackendModule provides RPCExecutorBackend (distributed, per D-15).
func rpcBackendModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[executorBackendConfig](""),
		configutil.NewPrefixedConfigInfoProvider[executorBackendConfig](""),
		fx.Annotate(newRPCExecutorBackend, fx.As(new(pipelineexec.ExecutorBackend))),
	)
}

// NewFxAppOptions returns common fx options for all binaries.
func NewFxAppOptions(options *RunAppOptions) fx.Option {
	return fx.Options(
		fx.NopLogger,
		pnpgorm.Module("postgres"),
		fx.Provide(func(db *gorm.DB) (*sql.DB, error) {
			return db.DB()
		}),
		pnpenv.Module(),
		pnpzap.Module(),
		pnphealthcheck.Module(),
		pnphealthcheckgorm.Module(),
		fx.Supply(options),

		fx.Provide(fx.Annotate(jobutil.NewWorkerManager, fx.OnStop(stopWorkerManager))),

		publicHTTPServerModule(options),
		internalHTTPServerModule(options),

		// Domain modules
		authModule(options),
		workspaceModule(options),
		secretModule(),
		pipelineModule(options),
		notifyModule(),
		messagingModule(options),

		// K8s executor
		k8sExecutorModule(options),
		// Docker executor — config is always-on (ContainerRunner needs it
		// regardless of whether worker jobs run); worker jobs are conditional.
		dockerExecutorConfigModule(),
		dockerExecutorModule(options),

		// Metrics
		metricsModule(),

		// ExecutorBackend: standalone mode (use-case adapter) vs distributed mode (RPC adapter) per D-16
		conditionalFxOptions(options.Standalone && (options.DockerExecutor || options.K8sExecutor), standaloneBackendModule),
		conditionalFxOptions(!options.Standalone && (options.DockerExecutor || options.K8sExecutor), rpcBackendModule),

		fx.Options(options.fxOptions...),
	)
}

func stopWorkerManager(ctx context.Context, workerManager *jobutil.WorkerManager) error {
	return workerManager.Close(ctx)
}

func newRPCExecutorBackend(cfg *executorBackendConfig) pipelineexec.ExecutorBackend {
	return pipelineadapt.NewRPCExecutorBackend(cfg.URL, cfg.Token)
}
