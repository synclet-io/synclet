package pipelinesync

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"time"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineroute"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// SyncBundle contains all pre-loaded data needed to execute a sync.
// Populated by ClaimJobBundleResult from ExecutorBackend.
type SyncBundle struct {
	Job                   *pipelineservice.Job
	ConnectionID          uuid.UUID
	WorkspaceID           uuid.UUID
	SourceID              uuid.UUID
	DestinationID         uuid.UUID
	SourceImage           string
	SourceConfig          json.RawMessage // Already decrypted
	DestImage             string
	DestConfig            json.RawMessage                    // Already decrypted
	ConfiguredCatalog     *protocol.ConfiguredAirbyteCatalog // Already unmarshaled
	StateBlob             json.RawMessage                    // May be nil
	SourceRuntimeConfig   string                             // JSON
	DestRuntimeConfig     string                             // JSON
	NamespaceDefinition   pipelineservice.NamespaceDefinition
	CustomNamespaceFormat *string
	StreamPrefix          *string
}

// SyncExecutor orchestrates a sync job using pre-loaded data from SyncBundle.
// It no longer accesses storage for data loading — all data arrives via the bundle.
type SyncExecutor struct {
	sourceReader       pipelineservice.SourceReader
	destWriter         pipelineservice.DestinationWriter
	saveSyncState      *pipelinestate.SaveSyncState
	appendJobLog       *pipelinelogs.AppendJobLog
	handleConfigUpdate *pipelinejobs.HandleConfigUpdate
	runtimeDefaults    pipelineservice.RuntimeDefaults
	idleTimeout        time.Duration
	logger             *logging.Logger
}

// SyncExecutorParams holds all dependencies for SyncExecutor.
type SyncExecutorParams struct {
	SourceReader       pipelineservice.SourceReader
	DestWriter         pipelineservice.DestinationWriter
	SaveSyncState      *pipelinestate.SaveSyncState
	AppendJobLog       *pipelinelogs.AppendJobLog
	HandleConfigUpdate *pipelinejobs.HandleConfigUpdate
	RuntimeDefaults    pipelineservice.RuntimeDefaults
	IdleTimeout        time.Duration
	Logger             *logging.Logger
}

// NewSyncExecutor creates a new sync executor.
func NewSyncExecutor(params SyncExecutorParams) *SyncExecutor {
	return &SyncExecutor{
		sourceReader:       params.SourceReader,
		destWriter:         params.DestWriter,
		saveSyncState:      params.SaveSyncState,
		appendJobLog:       params.AppendJobLog,
		handleConfigUpdate: params.HandleConfigUpdate,
		runtimeDefaults:    params.RuntimeDefaults,
		idleTimeout:        params.IdleTimeout,
		logger:             params.Logger.Named("sync-executor"),
	}
}

// Execute runs a sync for the given bundle of pre-loaded data.
func (e *SyncExecutor) Execute(ctx context.Context, bundle *SyncBundle) (*pipelineservice.SyncStats, error) {
	catalog := bundle.ConfiguredCatalog

	// Labels for container tracking and orphan cleanup.
	labels := map[string]string{
		"synclet.io/managed":       "true",
		"synclet.io/job-id":        bundle.Job.ID.String(),
		"synclet.io/connection-id": bundle.ConnectionID.String(),
	}

	// Resolve runtime config for source and set resource limits on the reader.
	srcRuntimeCfg := pipelineservice.ResolveRuntimeConfig(e.runtimeDefaults, pipelineservice.ParseRuntimeConfig(&bundle.SourceRuntimeConfig))
	srcMemLimit, srcCPULimit, _, _ := pipelineservice.ToContainerResources(srcRuntimeCfg)
	if rc, ok := e.sourceReader.(pipelineservice.ResourceConfigurable); ok {
		rc.SetResourceLimits(srcMemLimit, srcCPULimit)
	}

	// Start source read.
	sourceStdout, sourceCleanup, err := e.sourceReader.Read(ctx, bundle.SourceImage, bundle.SourceConfig, catalog, bundle.StateBlob, labels)
	if err != nil {
		return nil, fmt.Errorf("starting source read: %w", err)
	}
	defer sourceCleanup()

	// Build destination catalog with filtered schemas for selected fields.
	destCatalog, err := pipelinecatalog.BuildDestinationCatalog(catalog)
	if err != nil {
		sourceCleanup()
		return nil, fmt.Errorf("building destination catalog: %w", err)
	}

	// Apply namespace rewriting and stream prefix to dest catalog.
	// Source catalog retains original namespaces for filtering and rewriter mapping.
	pipelinecatalog.ApplyNamespaceAndPrefix(destCatalog, bundle.NamespaceDefinition, bundle.CustomNamespaceFormat, bundle.StreamPrefix)

	// Create pipe for source-to-dest message routing.
	srcPipeReader, srcPipeWriter := io.Pipe()

	// Resolve runtime config for destination and set resource limits on the writer.
	destRuntimeCfg := pipelineservice.ResolveRuntimeConfig(e.runtimeDefaults, pipelineservice.ParseRuntimeConfig(&bundle.DestRuntimeConfig))
	destMemLimit, destCPULimit, _, _ := pipelineservice.ToContainerResources(destRuntimeCfg)
	if rc, ok := e.destWriter.(pipelineservice.ResourceConfigurable); ok {
		rc.SetResourceLimits(destMemLimit, destCPULimit)
	}

	// Start destination write with pipe reader as stdin.
	destStdout, destCleanup, err := e.destWriter.Write(ctx, bundle.DestImage, bundle.DestConfig, destCatalog, srcPipeReader, labels)
	if err != nil {
		_ = srcPipeWriter.Close()
		_ = srcPipeReader.Close()
		return nil, fmt.Errorf("starting destination write: %w", err)
	}
	defer destCleanup()

	// Create handler for Docker mode side effects.
	handler := pipelineroute.NewDockerHandler(pipelineroute.DockerHandlerParams{
		ConnectionID:       bundle.ConnectionID,
		SourceID:           bundle.SourceID,
		DestinationID:      bundle.DestinationID,
		JobID:              bundle.Job.ID,
		HandleConfigUpdate: e.handleConfigUpdate,
		SaveSyncState:      e.saveSyncState,
		AppendJobLog:       e.appendJobLog,
		Logger:             e.logger,
	})

	// Wrap source output to filter record data by selected fields.
	filteredSource := pipelineroute.NewFilteringReader(sourceStdout, catalog)

	// Create namespace rewriter from the ORIGINAL source catalog (before namespace rewriting).
	// This maps source namespace/stream names to destination namespace/stream names in RECORD
	// and STATE messages before they are forwarded to the destination connector.
	rewriter := pipelineroute.NewNamespaceRewriter(catalog, bundle.NamespaceDefinition, bundle.CustomNamespaceFormat, bundle.StreamPrefix)

	// Route messages using the shared router.
	routeStats, routeErr := pipelineroute.Run(ctx, filteredSource, srcPipeWriter, destStdout, handler, pipelineroute.RunConfig{
		IdleTimeout: e.idleTimeout,
		Rewriter:    rewriter,
	}, e.logger.Named("message-router"))

	// Map router stats to SyncStats.
	stats := &pipelineservice.SyncStats{}
	if routeStats != nil {
		stats.RecordsRead = routeStats.RecordsRead
		stats.BytesSynced = routeStats.BytesSynced
		stats.Duration = routeStats.Duration
	}

	return stats, routeErr
}
