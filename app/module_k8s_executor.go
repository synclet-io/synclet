package app

import (
	"context"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"github.com/google/uuid"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexeck8s"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"go.uber.org/fx"
)

// k8sExecutorJobsConfig holds intervals for K8s executor background jobs.
type k8sExecutorJobsConfig struct {
	WorkerInterval          time.Duration `env:"WORKER_INTERVAL" envDefault:"5s"`
	ResourceCleanupInterval time.Duration `env:"RESOURCE_CLEANUP_INTERVAL" envDefault:"5m"`
}

type k8sExecutorConfig struct {
	Namespace       string                `env:"NAMESPACE"`
	Kubeconfig      string                `env:"KUBECONFIG"`
	ImagePullSecret string                `env:"IMAGE_PULL_SECRET"`
	DefaultMemory   string                `env:"DEFAULT_MEMORY" envDefault:"2Gi"`
	DefaultCPU      string                `env:"DEFAULT_CPU" envDefault:"1"`
	SyncletImage    string                `env:"SYNCLET_IMAGE"`
	ServerAddr      string                `env:"SERVER_ADDR"`
	Jobs            k8sExecutorJobsConfig `envPrefix:"JOB_"`
}

func k8sExecutorModule(options *RunAppOptions) fx.Option {
	if !options.K8sExecutor {
		return fx.Options()
	}

	return fx.Module(
		"k8s-executor",
		logging.DecorateNamed("k8s"),
		fx.Provide(
			configutil.NewPrefixedConfigProvider[k8sExecutorConfig]("K8S_EXECUTOR_"),
			configutil.NewPrefixedConfigInfoProvider[k8sExecutorConfig]("K8S_EXECUTOR_"),
			newK8sSyncRunner,
			newK8sReconciler,

			fx.Annotate(pipelineexeck8s.NewK8sSyncLauncherAdapter, fx.As(new(pipelineexeck8s.K8sSyncLauncher))),
			newK8sSyncWorker,

			fx.Annotate(pipelineexeck8s.NewK8sTaskLauncherAdapter, fx.As(new(pipelineexeck8s.K8sTaskLauncher))),
			newK8sTaskWorker,

			func(backend pipelineexec.ExecutorBackend) pipelineexeck8s.ResourceChecker { return backend },
		),

		pnpjobber.Module(newK8sTaskWorkerJob),
		pnpjobber.Module(newK8sSyncWorkerJob),
		pnpjobber.Module(newK8sResourceCleanupJob),
	)
}

func newK8sSyncRunner(cfg *k8sExecutorConfig) (*pipelineexeck8s.SyncRunner, error) {
	return pipelineexeck8s.NewSyncRunner(pipelineexeck8s.Config{
		Namespace:       cfg.Namespace,
		Kubeconfig:      cfg.Kubeconfig,
		ImagePullSecret: cfg.ImagePullSecret,
		DefaultMemory:   cfg.DefaultMemory,
		DefaultCPU:      cfg.DefaultCPU,
		SyncletImage:    cfg.SyncletImage,
		ServerAddr:      cfg.ServerAddr,
	})
}

func newK8sReconciler(runner *pipelineexeck8s.SyncRunner, checker pipelineexeck8s.ResourceChecker, logger *logging.Logger) *pipelineexeck8s.ResourceCleaner {
	return pipelineexeck8s.NewReconciler(runner.Client(), runner.Namespace(), checker, logger)
}

func newK8sResourceCleanupJob(cfg *k8sExecutorConfig, reconciler *pipelineexeck8s.ResourceCleaner) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "k8s_resource_cleanup",
		Interval:         cfg.Jobs.ResourceCleanupInterval,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			reconciler.Reconcile(ctx, true)
			// Reconcile logs errors internally; always return nil to prevent
			// jobber from applying retry/backoff on partial failures.
			return nil
		},
	})
}

func newK8sSyncWorker(
	backend pipelineexec.ExecutorBackend,
	setK8sJobName *pipelinejobs.SetK8sJobName,
	k8sRunner pipelineexeck8s.K8sSyncLauncher,
	k8sCfg *k8sExecutorConfig,
	logger *logging.Logger,
) *pipelineexeck8s.K8sSyncWorker {
	workerID := uuid.New().String()[:8]

	return pipelineexeck8s.NewK8sSyncWorker(
		backend, setK8sJobName, k8sRunner,
		k8sCfg.ServerAddr, workerID, logger,
	)
}

func newK8sSyncWorkerJob(cfg *k8sExecutorConfig, worker *pipelineexeck8s.K8sSyncWorker) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:     "k8s_sync",
		Interval: cfg.Jobs.WorkerInterval,
		Job: func(ctx context.Context) error {
			return worker.Execute(ctx)
		},
	})
}

func newK8sTaskWorker(
	backend pipelineexec.ExecutorBackend,
	k8sRunner pipelineexeck8s.K8sTaskLauncher,
	k8sCfg *k8sExecutorConfig,
	logger *logging.Logger,
) *pipelineexeck8s.K8sTaskWorker {
	workerID := uuid.New().String()[:8]

	return pipelineexeck8s.NewK8sTaskWorker(
		backend, k8sRunner,
		workerID, logger,
	)
}

func newK8sTaskWorkerJob(cfg *k8sExecutorConfig, worker *pipelineexeck8s.K8sTaskWorker) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:     "k8s_task",
		Interval: cfg.Jobs.WorkerInterval,
		Job: func(ctx context.Context) error {
			return worker.Execute(ctx)
		},
	})
}
