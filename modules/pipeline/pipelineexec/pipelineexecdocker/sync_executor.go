package pipelineexecdocker

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/go-pnp/go-pnp/logging"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	pkgcontainer "github.com/synclet-io/synclet/pkg/container"
	"github.com/synclet-io/synclet/pkg/mapper"
)

// dockerVolumesToMounts converts user-defined docker volumes into bind mounts.
func dockerVolumesToMounts(volumes []pipelineservice.DockerVolume) []pkgcontainer.BindMount {
	if len(volumes) == 0 {
		return nil
	}

	mounts := make([]pkgcontainer.BindMount, 0, len(volumes))
	for _, v := range volumes {
		mounts = append(mounts, pkgcontainer.BindMount{
			HostPath:      v.HostPath,
			ContainerPath: v.ContainerPath,
			ReadOnly:      v.ReadOnly,
		})
	}

	return mounts
}

// DockerSyncExecutor runs a sync job from the pre-loaded data of a SyncBundle
// and reports state, config updates and logs through the SyncReporter.
type DockerSyncExecutor struct {
	client          *ConnectorClient
	reporter        pipelineexec.SyncReporter
	runtimeDefaults pipelineservice.RuntimeDefaults
	idleTimeout     time.Duration
	tempDirRoot     string
	localRoot       string
	logger          *logging.Logger
}

// DockerSyncExecutorParams holds all dependencies for DockerSyncExecutor.
type DockerSyncExecutorParams struct {
	Client          *ConnectorClient
	Reporter        pipelineexec.SyncReporter
	RuntimeDefaults pipelineservice.RuntimeDefaults
	IdleTimeout     time.Duration
	// TempDirRoot is where the job's /local directory is created; see
	// NewContainerRunner.
	TempDirRoot string
	// LocalRoot, when set, is the host directory mounted at /local in every
	// job instead of a per-job directory; it is never removed.
	LocalRoot string
	Logger    *logging.Logger
}

// NewDockerSyncExecutor creates a new sync executor.
func NewDockerSyncExecutor(params DockerSyncExecutorParams) *DockerSyncExecutor {
	return &DockerSyncExecutor{
		client:          params.Client,
		reporter:        params.Reporter,
		runtimeDefaults: params.RuntimeDefaults,
		idleTimeout:     params.IdleTimeout,
		tempDirRoot:     params.TempDirRoot,
		localRoot:       params.LocalRoot,
		logger:          params.Logger.Named("sync-executor"),
	}
}

// Execute runs a sync for the given bundle of pre-loaded data. onStarted is
// called once both the source and destination containers have started.
func (e *DockerSyncExecutor) Execute(ctx context.Context, bundle *pipelineexec.SyncBundle, onStarted func()) (*pipelineservice.SyncStats, error) {
	catalog := bundle.ConfiguredCatalog

	transformer, err := pipelineexec.NewRecordTransformer(catalog, bundle.MapperSalt, mapper.SyncInfo{
		StartedAt:  time.Now().UTC(),
		JobID:      bundle.Job.ID.String(),
		Connection: bundle.ConnectionID.String(),
	})
	if err != nil {
		return nil, fmt.Errorf("compiling stream mappers: %w", err)
	}

	// Labels for container tracking and orphan cleanup.
	labels := map[string]string{
		"synclet.io/managed":       "true",
		"synclet.io/job-id":        bundle.Job.ID.String(),
		"synclet.io/connection-id": bundle.ConnectionID.String(),
	}

	// The source and destination of the job share one /local directory: the
	// local root, or a per-job directory removed after both containers are.
	localDir, releaseLocalDir, err := acquireLocalDir(e.tempDirRoot, e.localRoot)
	if err != nil {
		return nil, fmt.Errorf("creating the job's local dir: %w", err)
	}
	defer releaseLocalDir()

	// Resolve runtime config for source and set resource limits.
	srcRuntimeCfg := pipelineservice.ResolveRuntimeConfig(e.runtimeDefaults, pipelineservice.ParseRuntimeConfig(&bundle.SourceRuntimeConfig))

	srcMemLimit, srcCPULimit, _, _ := pipelineservice.ToContainerResources(srcRuntimeCfg)

	// One redactor serves the whole run, so secrets of a config replaced
	// through CONTROL are masked in connector stderr as well.
	redactor := pipelineexec.NewRedactor(bundle.SourceConfig, bundle.DestConfig)

	// Start source read.
	sourceStdout, sourceCleanup, err := e.client.Read(ctx, bundle.SourceImage, bundle.SourceConfig, redactor, pipelineservice.BuildSourceCatalog(catalog), bundle.StateBlob, labels, ResourceLimits{
		MemoryLimit: srcMemLimit,
		CPULimit:    srcCPULimit,
		NetworkMode: srcRuntimeCfg.DockerNetwork,
		ExtraMounts: dockerVolumesToMounts(srcRuntimeCfg.DockerVolumes),
		LocalDir:    localDir,
	})
	if err != nil {
		return nil, fmt.Errorf("starting source read: %w", err)
	}
	defer sourceCleanup()

	// Build destination catalog with filtered schemas for selected fields.
	destCatalog, err := pipelineservice.BuildDestinationCatalog(catalog)
	if err != nil {
		return nil, fmt.Errorf("building destination catalog: %w", err)
	}

	// Apply namespace rewriting and stream prefix to dest catalog.
	// Source catalog retains original namespaces for filtering and rewriter mapping.
	pipelineservice.ApplyNamespaceAndPrefix(destCatalog, bundle.NamespaceDefinition, bundle.CustomNamespaceFormat, bundle.StreamPrefix)

	// Create pipe for source-to-dest message routing.
	srcPipeReader, srcPipeWriter := io.Pipe()

	// Resolve runtime config for destination and set resource limits.
	destRuntimeCfg := pipelineservice.ResolveRuntimeConfig(e.runtimeDefaults, pipelineservice.ParseRuntimeConfig(&bundle.DestRuntimeConfig))

	destMemLimit, destCPULimit, _, _ := pipelineservice.ToContainerResources(destRuntimeCfg)

	// Start destination write with pipe reader as stdin.
	destStdout, destCleanup, err := e.client.Write(ctx, bundle.DestImage, bundle.DestConfig, redactor, destCatalog, srcPipeReader, labels, ResourceLimits{
		MemoryLimit: destMemLimit,
		CPULimit:    destCPULimit,
		NetworkMode: destRuntimeCfg.DockerNetwork,
		ExtraMounts: dockerVolumesToMounts(destRuntimeCfg.DockerVolumes),
		LocalDir:    localDir,
	})
	if err != nil {
		_ = srcPipeWriter.Close()
		_ = srcPipeReader.Close()

		return nil, fmt.Errorf("starting destination write: %w", err)
	}
	defer destCleanup()

	onStarted()

	// Create handler for Docker mode side effects.
	handler := NewDockerHandler(DockerHandlerParams{
		ConnectionID:  bundle.ConnectionID,
		SourceID:      bundle.SourceID,
		DestinationID: bundle.DestinationID,
		JobID:         bundle.Job.ID,
		Reporter:      e.reporter,
		Logger:        e.logger,
	})

	stopLogFlush := handler.StartLogFlush(ctx)
	defer stopLogFlush()

	// Create namespace rewriter from the ORIGINAL source catalog (before namespace rewriting).
	// This maps source namespace/stream names to destination namespace/stream names in RECORD
	// and STATE messages before they are forwarded to the destination connector.
	rewriter := pipelineexec.NewNamespaceRewriter(catalog, bundle.NamespaceDefinition, bundle.CustomNamespaceFormat, bundle.StreamPrefix)

	// Route messages using the shared router.
	routeStats, routeErr := pipelineexec.Run(ctx, sourceStdout, srcPipeWriter, destStdout, handler, pipelineexec.RunConfig{
		IdleTimeout: e.idleTimeout,
		Redactor:    redactor,
		Rewriter:    rewriter,
		Catalog:     catalog,
		Transformer: transformer,
		// SourceSucceeded stays unset: the source reader ends with EOF only after the container exited 0.
	}, e.logger.Named("message-router"))

	// Map router stats to SyncStats.
	stats := &pipelineservice.SyncStats{}
	if routeStats != nil {
		stats.RecordsRead = routeStats.RecordsRead
		stats.RecordsFiltered = routeStats.RecordsFiltered
		stats.BytesSynced = routeStats.BytesSynced
		stats.Duration = routeStats.Duration
	}

	return stats, routeErr
}
