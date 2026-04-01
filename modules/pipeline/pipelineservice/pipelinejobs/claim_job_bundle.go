package pipelinejobs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
)

// ClaimJobBundleResult bundles everything an executor needs to run a sync.
type ClaimJobBundleResult struct {
	Job                   *pipelineservice.Job
	ConnectionID          uuid.UUID
	WorkspaceID           uuid.UUID
	SourceID              uuid.UUID
	DestinationID         uuid.UUID
	SourceImage           string
	SourceConfig          []byte // Decrypted JSON
	DestImage             string
	DestConfig            []byte // Decrypted JSON
	ConfiguredCatalog     []byte // JSON-marshaled
	StateBlob             []byte // JSON, may be nil
	SourceRuntimeConfig   string
	DestRuntimeConfig     string
	NamespaceDefinition   pipelineservice.NamespaceDefinition
	CustomNamespaceFormat string
	StreamPrefix          string
	MaxAttempts           int
}

// ClaimJobBundle claims a job and assembles the full executor bundle in one call.
// This keeps all storage access in the service layer so the handler never touches storage.
type ClaimJobBundle struct {
	claimJob              *ClaimJob
	getConfiguredCatalog  *pipelinecatalog.GetConfiguredCatalog
	populateGenerationIDs *pipelinecatalog.PopulateGenerationIDs
	getSyncState          *pipelinestate.GetSyncState
	storage               pipelineservice.Storage
	secrets               pipelineservice.SecretsProvider
	logger                *logging.Logger
}

// NewClaimJobBundle creates a new ClaimJobBundle use case.
func NewClaimJobBundle(
	claimJob *ClaimJob,
	getConfiguredCatalog *pipelinecatalog.GetConfiguredCatalog,
	populateGenerationIDs *pipelinecatalog.PopulateGenerationIDs,
	getSyncState *pipelinestate.GetSyncState,
	storage pipelineservice.Storage,
	secrets pipelineservice.SecretsProvider,
	logger *logging.Logger,
) *ClaimJobBundle {
	return &ClaimJobBundle{
		claimJob:              claimJob,
		getConfiguredCatalog:  getConfiguredCatalog,
		populateGenerationIDs: populateGenerationIDs,
		getSyncState:          getSyncState,
		storage:               storage,
		secrets:               secrets,
		logger:                logger.Named("claim-job-bundle"),
	}
}

// Execute claims a job and loads the full bundle. Returns nil, nil when no jobs available.
func (uc *ClaimJobBundle) Execute(ctx context.Context, params ClaimJobParams) (*ClaimJobBundleResult, error) {
	// 1. Claim job via existing ClaimJob use case.
	job, err := uc.claimJob.Execute(ctx, params)
	if err != nil {
		return nil, fmt.Errorf("claiming job: %w", err)
	}

	if job == nil {
		return nil, nil
	}

	// 2. Load connection.
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(job.ConnectionID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading connection: %w", err)
	}

	// 3. Load source (WorkspaceID filter for defense-in-depth).
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(conn.SourceID),
		WorkspaceID: filter.Equals(conn.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading source: %w", err)
	}

	// 3b. Load source managed connector for image resolution.
	srcMC, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(src.ManagedConnectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading source managed connector: %w", err)
	}

	// 4. Load destination (WorkspaceID filter for defense-in-depth).
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID:          filter.Equals(conn.DestinationID),
		WorkspaceID: filter.Equals(conn.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading destination: %w", err)
	}

	// 4b. Load destination managed connector for image resolution.
	destMC, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(dest.ManagedConnectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading destination managed connector: %w", err)
	}

	// 5. Decrypt source config.
	decryptedSourceConfig, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, src.Config)
	if err != nil {
		return nil, fmt.Errorf("decrypting source config: %w", err)
	}

	// 6. Decrypt dest config.
	decryptedDestConfig, err := pipelinesecrets.DecryptConfigSecrets(ctx, uc.secrets, dest.Config)
	if err != nil {
		return nil, fmt.Errorf("decrypting destination config: %w", err)
	}

	// 7. Get configured catalog and marshal to JSON.
	catalog, err := uc.getConfiguredCatalog.Execute(ctx, pipelinecatalog.GetConfiguredCatalogParams{
		ConnectionID: job.ConnectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("loading configured catalog: %w", err)
	}

	// Populate sync metadata (sync_id, generation_id, minimum_generation_id) so
	// Airbyte CDK connectors receive valid non-zero integers instead of defaults.
	jobCount, err := uc.storage.Jobs().Count(ctx, &pipelineservice.JobFilter{
		ConnectionID: filter.Equals(job.ConnectionID),
	})
	if err != nil {
		return nil, fmt.Errorf("counting jobs for sync_id: %w", err)
	}

	if err := uc.populateGenerationIDs.Execute(ctx, pipelinecatalog.PopulateGenerationIDsParams{
		ConnectionID: job.ConnectionID,
		Catalog:      catalog,
		SyncID:       int64(jobCount),
	}); err != nil {
		return nil, fmt.Errorf("populating generation IDs: %w", err)
	}

	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return nil, fmt.Errorf("marshaling catalog: %w", err)
	}

	// 8. Get sync state.
	stateBlob, err := uc.getSyncState.Execute(ctx, pipelinestate.GetSyncStateParams{
		ConnectionID: job.ConnectionID,
	})
	if err != nil {
		return nil, fmt.Errorf("loading state: %w", err)
	}

	// 9. Build result with all fields.
	result := &ClaimJobBundleResult{
		Job:                   job,
		ConnectionID:          conn.ID,
		WorkspaceID:           conn.WorkspaceID,
		SourceID:              conn.SourceID,
		DestinationID:         conn.DestinationID,
		SourceImage:           srcMC.DockerImage + ":" + srcMC.DockerTag,
		SourceConfig:          []byte(decryptedSourceConfig),
		DestImage:             destMC.DockerImage + ":" + destMC.DockerTag,
		DestConfig:            []byte(decryptedDestConfig),
		ConfiguredCatalog:     catalogJSON,
		StateBlob:             stateBlob,
		SourceRuntimeConfig:   derefStringPtr(src.RuntimeConfig),
		DestRuntimeConfig:     derefStringPtr(dest.RuntimeConfig),
		NamespaceDefinition:   conn.NamespaceDefinition,
		CustomNamespaceFormat: derefStringPtr(conn.CustomNamespaceFormat),
		StreamPrefix:          derefStringPtr(conn.StreamPrefix),
		MaxAttempts:           conn.MaxAttempts,
	}

	return result, nil
}

func derefStringPtr(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}
