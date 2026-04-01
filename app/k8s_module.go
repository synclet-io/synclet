package app

import (
	"context"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineadapt"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/k8s"
	"go.uber.org/fx"
	"go.uber.org/zap"
)

// k8sConfig holds K8s runner configuration loaded from environment variables with K8S_ prefix.
type k8sConfig struct {
	Namespace       string `env:"NAMESPACE"`
	Kubeconfig      string `env:"KUBECONFIG"`
	ImagePullSecret string `env:"IMAGE_PULL_SECRET"`
	DefaultMemory   string `env:"DEFAULT_MEMORY" envDefault:"2Gi"`
	DefaultCPU      string `env:"DEFAULT_CPU" envDefault:"1"`
	SyncletImage    string `env:"SYNCLET_IMAGE"`
	ServerAddr      string `env:"SERVER_ADDR"`
}

func k8sModule(k8sEnabled bool) fx.Option {
	if !k8sEnabled {
		return fx.Options()
	}

	return fx.Options(
		fx.Provide(
			configutil.NewPrefixedConfigProvider[k8sConfig]("K8S_"),
			func(cfg *k8sConfig) (*k8s.SyncRunner, error) {
				return k8s.NewSyncRunner(k8s.Config{
					Namespace:       cfg.Namespace,
					Kubeconfig:      cfg.Kubeconfig,
					ImagePullSecret: cfg.ImagePullSecret,
					DefaultMemory:   cfg.DefaultMemory,
					DefaultCPU:      cfg.DefaultCPU,
					SyncletImage:    cfg.SyncletImage,
					ServerAddr:      cfg.ServerAddr,
				})
			},
		),
		fx.Provide(func(runner *k8s.SyncRunner, provider k8s.StaleJobProvider, logger *logging.Logger) *k8s.Reconciler {
			return k8s.NewReconciler(runner.Client(), runner.Namespace(), provider, logger)
		}),
		fx.Provide(func(runner *k8s.SyncRunner, provider k8s.StaleJobProvider, logger *zap.Logger) *k8s.OrphanCleaner {
			return k8s.NewOrphanCleaner(runner.Client(), runner.Namespace(), provider, logger)
		}),
		fx.Provide(
			fx.Annotate(
				pipelineadapt.NewK8sSyncLauncherAdapter,
				fx.As(new(pipelineservice.K8sSyncLauncher)),
			),
			fx.Annotate(
				pipelineadapt.NewStaleJobProviderAdapter,
				fx.As(new(k8s.StaleJobProvider)),
			),
			fx.Annotate(
				func(runner *k8s.SyncRunner) pipelinejobs.K8sSyncStopper { return runner },
				fx.As(new(pipelinejobs.K8sSyncStopper)),
			),
		),

		// Start K8s orphan cleanup (startup + periodic).
		fx.Invoke(func(lc fx.Lifecycle, cleaner *k8s.OrphanCleaner, logger *zap.Logger) {
			runCtx, cancel := context.WithCancel(context.Background())

			lc.Append(fx.Hook{
				OnStart: func(ctx context.Context) error {
					if err := cleaner.CleanupAll(ctx); err != nil {
						logger.Error("k8s startup orphan cleanup failed", zap.Error(err))
					}

					go func() {
						ticker := time.NewTicker(5 * time.Minute)
						defer ticker.Stop()

						for {
							select {
							case <-runCtx.Done():
								return
							case <-ticker.C:
								if err := cleaner.Cleanup(runCtx); err != nil {
									logger.Error("k8s periodic orphan cleanup failed", zap.Error(err))
								}
							}
						}
					}()

					return nil
				},
				OnStop: func(_ context.Context) error {
					cancel()

					return nil
				},
			})
		}),

		// Start reconciler as lifecycle hook.
		fx.Invoke(func(lc fx.Lifecycle, reconciler *k8s.Reconciler) {
			runCtx, cancel := context.WithCancel(context.Background())

			lc.Append(fx.Hook{
				OnStart: func(_ context.Context) error {
					go reconciler.Run(runCtx)

					return nil
				},
				OnStop: func(_ context.Context) error {
					cancel()

					return nil
				},
			})
		}),
	)
}
