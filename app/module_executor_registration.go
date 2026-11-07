package app

import (
	"fmt"
	"os"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/buildinfo"
)

// executorRegistrationConfig sets how often an executor reports itself to the
// executor registry and the executor group it declares.
type executorRegistrationConfig struct {
	Interval time.Duration `env:"REGISTRATION_INTERVAL" envDefault:"15s"`
	Group    string        `env:"GROUP" envDefault:"default"`
}

// executorRegistrationConfigModule provides the registration interval and the
// executor group to a process that runs an executor.
func executorRegistrationConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[executorRegistrationConfig]("EXECUTOR_"),
		configutil.NewPrefixedConfigInfoProvider[executorRegistrationConfig]("EXECUTOR_"),
		newExecutorGroup,
	)
}

// newExecutorGroup normalizes EXECUTOR_GROUP; an invalid name stops the process.
func newExecutorGroup(cfg *executorRegistrationConfig) (pipelineexec.ExecutorGroup, error) {
	group, err := pipelineservice.NormalizeExecutorGroup(cfg.Group)
	if err != nil {
		return "", fmt.Errorf("EXECUTOR_GROUP: %w", err)
	}

	return pipelineexec.ExecutorGroup(group), nil
}

// executorRegistrationModule sends the executor's registration heartbeat at
// start and then every interval. It needs the executor's
// pipelineexec.ExecutorInfo and an ExecutorBackend.
func executorRegistrationModule() fx.Option {
	return pnpjobber.Module(newExecutorRegistrationJob)
}

func newExecutorRegistrationJob(
	cfg *executorRegistrationConfig,
	backend pipelineexec.ExecutorBackend,
	info pipelineexec.ExecutorInfo,
	logger *logging.Logger,
) jobber.Job {
	registrar := pipelineexec.NewRegistrar(backend, info, logger)

	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "executor_registration",
		Interval:         cfg.Interval,
		StartImmediately: true,
		Job:              registrar.Execute,
	})
}

// newDockerExecutorInfo describes the Docker executor: its capacity is its
// concurrent sync limit and its resource defaults are the runtime defaults.
func newDockerExecutorInfo(cfg *dockerExecutorConfig, runtime *executorRuntimeConfig, group pipelineexec.ExecutorGroup) (pipelineexec.ExecutorInfo, error) {
	ownerID, err := pipelineexec.ResolveOwnerID(cfg.OwnerID, "", "DOCKER_EXECUTOR_OWNER_ID", "", "Docker daemon", os.Hostname)
	if err != nil {
		return pipelineexec.ExecutorInfo{}, err
	}

	return pipelineexec.ExecutorInfo{
		ID:       ownerID,
		Type:     pipelineservice.ExecutorTypeDocker,
		Version:  buildinfo.Version(),
		Capacity: cfg.MaxConcurrentJobs,
		Group:    string(group),
		ResourceDefaults: pipelineservice.ExecutorResourceDefaults{
			CPURequest:    runtime.Runtime.CPURequest,
			CPULimit:      runtime.Runtime.CPULimit,
			MemoryRequest: runtime.Runtime.MemoryRequest,
			MemoryLimit:   runtime.Runtime.MemoryLimit,
		},
	}, nil
}

// newK8sExecutorInfo describes the Kubernetes executor: it has no concurrency
// limit of its own, and a resource with neither request nor limit set takes
// the executor's default as both.
func newK8sExecutorInfo(cfg *k8sExecutorConfig, runtime *executorRuntimeConfig, group pipelineexec.ExecutorGroup) (pipelineexec.ExecutorInfo, error) {
	ownerID, err := resolveK8sOwnerID(cfg)
	if err != nil {
		return pipelineexec.ExecutorInfo{}, err
	}

	defaults := pipelineservice.ExecutorResourceDefaults{
		CPURequest:    runtime.Runtime.CPURequest,
		CPULimit:      runtime.Runtime.CPULimit,
		MemoryRequest: runtime.Runtime.MemoryRequest,
		MemoryLimit:   runtime.Runtime.MemoryLimit,
	}

	if defaults.CPURequest == "" && defaults.CPULimit == "" && cfg.DefaultCPU != "" {
		defaults.CPURequest, defaults.CPULimit, defaults.CPUExecutorDefault = cfg.DefaultCPU, cfg.DefaultCPU, true
	}

	if defaults.MemoryRequest == "" && defaults.MemoryLimit == "" && cfg.DefaultMemory != "" {
		defaults.MemoryRequest, defaults.MemoryLimit, defaults.MemoryExecutorDefault = cfg.DefaultMemory, cfg.DefaultMemory, true
	}

	return pipelineexec.ExecutorInfo{
		ID:               ownerID,
		Type:             pipelineservice.ExecutorTypeK8s,
		Version:          buildinfo.Version(),
		ResourceDefaults: defaults,
		Group:            string(group),
	}, nil
}
