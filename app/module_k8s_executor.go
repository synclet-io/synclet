package app

import (
	"context"
	"os"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"github.com/google/uuid"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexeck8s"
	"go.uber.org/fx"
)

// k8sExecutorJobsConfig holds intervals for K8s executor background jobs.
type k8sExecutorJobsConfig struct {
	WorkerInterval          time.Duration `env:"WORKER_INTERVAL" envDefault:"5s"`
	ResourceCleanupInterval time.Duration `env:"RESOURCE_CLEANUP_INTERVAL" envDefault:"5m"`
}

type k8sExecutorConfig struct {
	Namespace       string `env:"NAMESPACE"`
	Kubeconfig      string `env:"KUBECONFIG"`
	ImagePullSecret string `env:"IMAGE_PULL_SECRET"`
	DefaultMemory   string `env:"DEFAULT_MEMORY" envDefault:"2Gi"`
	DefaultCPU      string `env:"DEFAULT_CPU" envDefault:"1"`
	SyncletImage    string `env:"SYNCLET_IMAGE"`
	ServerAddr      string `env:"SERVER_ADDR"`
	// OwnerID is stamped as the synclet.io/owner label on every Job and
	// Secret the executor creates, and the reconciler handles only those
	// carrying it. Empty uses the hostname, which a StatefulSet keeps stable
	// per replica; executors sharing one namespace need distinct values.
	OwnerID string `env:"OWNER_ID"`
	// LocalVolumeClaim names an existing PersistentVolumeClaim mounted at
	// /local in the connector containers, where file-writing destinations
	// leave their files. Empty mounts a per-pod emptyDir.
	LocalVolumeClaim string                `env:"LOCAL_VOLUME_CLAIM"`
	Jobs             k8sExecutorJobsConfig `envPrefix:"JOB_"`
}

// k8sExecutorConfigModule provides *k8sExecutorConfig to a process that runs
// the Kubernetes executor.
func k8sExecutorConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[k8sExecutorConfig]("K8S_EXECUTOR_"),
		configutil.NewPrefixedConfigInfoProvider[k8sExecutorConfig]("K8S_EXECUTOR_"),
	)
}

// k8sPodAPITokenTag names the token the Kubernetes executor hands to the
// coordinator pods it launches, so they can call the executor API.
const k8sPodAPITokenTag = `name:"k8sPodAPIToken"` //nolint:gosec // fx name tag, not a credential.

// executorK8sPodAPITokenModule passes on the token the executor process
// itself authenticates with.
func executorK8sPodAPITokenModule() fx.Option {
	return fx.Provide(fx.Annotate(
		func(cfg *executorBackendConfig) string { return cfg.Token },
		fx.ResultTags(k8sPodAPITokenTag),
	))
}

// serverK8sPodAPITokenModule passes on the token the server's executor API
// accepts.
func serverK8sPodAPITokenModule() fx.Option {
	return fx.Provide(fx.Annotate(
		func(cfg *pipelineConfig) string { return cfg.ExecutorAPIToken },
		fx.ResultTags(k8sPodAPITokenTag),
	))
}

// k8sExecutorModule runs the Kubernetes executor's workers. It needs
// *k8sExecutorConfig, the named k8sPodAPIToken and an ExecutorBackend.
func k8sExecutorModule() fx.Option {
	return fx.Module(
		"k8s-executor",
		logging.DecorateNamed("k8s"),
		fx.Provide(
			fx.Annotate(newK8sSyncRunner, fx.ParamTags("", k8sPodAPITokenTag)),
			newK8sReconciler,

			fx.Annotate(pipelineexeck8s.NewK8sSyncLauncherAdapter, fx.As(new(pipelineexeck8s.K8sSyncLauncher))),
			newK8sSyncWorker,

			fx.Annotate(pipelineexeck8s.NewK8sTaskLauncherAdapter, fx.As(new(pipelineexeck8s.K8sTaskLauncher))),
			newK8sTaskWorker,

			func(backend pipelineexec.ExecutorBackend) pipelineexeck8s.ResourceChecker { return backend },
			fx.Private,
		),
		fx.Provide(newK8sExecutorInfo),
		executorRegistrationModule(),

		pnpjobber.Module(newK8sTaskWorkerJob),
		pnpjobber.Module(newK8sSyncWorkerJob),
		pnpjobber.Module(newK8sResourceCleanupJob),
	)
}

// newK8sSyncRunner gives the pods it launches the executor API token of the
// process: the executor's EXECUTOR_API_TOKEN, or the server's own
// PIPELINE_EXECUTOR_API_TOKEN inside `server --standalone`.
func newK8sSyncRunner(cfg *k8sExecutorConfig, apiToken string) (*pipelineexeck8s.SyncRunner, error) {
	ownerID, err := pipelineexec.ResolveOwnerID(cfg.OwnerID, "K8S_EXECUTOR_OWNER_ID", os.Hostname)
	if err != nil {
		return nil, err
	}

	return pipelineexeck8s.NewSyncRunner(pipelineexeck8s.Config{
		Namespace:        cfg.Namespace,
		Kubeconfig:       cfg.Kubeconfig,
		ImagePullSecret:  cfg.ImagePullSecret,
		DefaultMemory:    cfg.DefaultMemory,
		DefaultCPU:       cfg.DefaultCPU,
		SyncletImage:     cfg.SyncletImage,
		ServerAddr:       cfg.ServerAddr,
		APIToken:         apiToken,
		OwnerID:          ownerID,
		LocalVolumeClaim: cfg.LocalVolumeClaim,
	})
}

func newK8sReconciler(runner *pipelineexeck8s.SyncRunner, checker pipelineexeck8s.ResourceChecker, logger *logging.Logger) *pipelineexeck8s.ResourceCleaner {
	return pipelineexeck8s.NewReconciler(runner.Client(), runner.Namespace(), runner.OwnerID(), checker, logger)
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
	k8sRunner pipelineexeck8s.K8sSyncLauncher,
	k8sCfg *k8sExecutorConfig,
	logger *logging.Logger,
) *pipelineexeck8s.K8sSyncWorker {
	workerID := uuid.New().String()[:8]

	return pipelineexeck8s.NewK8sSyncWorker(
		backend, k8sRunner,
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
