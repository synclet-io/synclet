package pipelinesync

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/google/uuid"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// K8sJobCreator abstracts the K8s runner for creating sync jobs.
type K8sJobCreator interface {
	CreateSyncJob(ctx context.Context, opts K8sSyncJobOptions) (string, error)
}

// K8sSyncJobOptions contains all fields needed to construct the K8s pod spec.
type K8sSyncJobOptions struct {
	JobID          uuid.UUID
	ConnectionID   uuid.UUID
	SourceID       uuid.UUID
	DestinationID  uuid.UUID
	SourceImage    string
	DestImage      string
	SourceConfig   []byte // JSON config
	DestConfig     []byte // JSON config
	SourceCatalog  []byte // JSON configured catalog (original namespaces)
	DestCatalog    []byte // JSON configured catalog (namespace-rewritten)
	State          []byte // JSON state blob (may be nil)
	RuntimeConfig  string // JSON runtime config for resource limits
	InternalAPIURL string // URL for gRPC calls back to main server

	// Namespace/prefix rewriting for coordinator's message router.
	NamespaceDefinition   string
	CustomNamespaceFormat string
	StreamPrefix          string
}

// K8sSyncWorker is a fire-and-forget jobber that claims scheduled jobs
// and creates K8s Jobs with 3-container pods (coordinator + source + dest).
// Per D-09, it does NOT wait for pod completion -- it returns immediately
// after submitting the K8s Job. Uses ExecutorBackend for all server
// communication per D-14.
type K8sSyncWorker struct {
	backend        ExecutorBackend
	setK8sJobName  *pipelinejobs.SetK8sJobName // K8s-specific, keep
	k8sRunner      K8sJobCreator
	internalAPIURL string
	workerID       string
	logger         *logging.Logger
}

// NewK8sSyncWorker creates a new K8sSyncWorker.
func NewK8sSyncWorker(
	backend ExecutorBackend,
	setK8sJobName *pipelinejobs.SetK8sJobName,
	k8sRunner K8sJobCreator,
	internalAPIURL string,
	workerID string,
	logger *logging.Logger,
) *K8sSyncWorker {
	return &K8sSyncWorker{
		backend:        backend,
		setK8sJobName:  setK8sJobName,
		k8sRunner:      k8sRunner,
		internalAPIURL: internalAPIURL,
		workerID:       workerID,
		logger:         logger,
	}
}

// Execute claims the next scheduled job and creates a K8s Job with a 3-container pod.
// Fire-and-forget per D-09: returns immediately after K8s Job creation.
func (w *K8sSyncWorker) Execute(ctx context.Context) error {
	// Claim next scheduled job via ExecutorBackend (returns full bundle).
	result, err := w.backend.ClaimJob(ctx, w.workerID)
	if err != nil {
		return fmt.Errorf("claiming job: %w", err)
	}
	if result == nil {
		return nil
	}

	// Build destination catalog with namespace/prefix rewriting.
	// Source catalog keeps original namespaces; destination catalog gets rewritten.
	var catalog protocol.ConfiguredAirbyteCatalog
	if err := json.Unmarshal(result.ConfiguredCatalog, &catalog); err != nil {
		w.failJob(ctx, result.Job.ID, fmt.Errorf("unmarshaling catalog: %w", err))
		return nil
	}
	destCatalog, err := pipelinecatalog.BuildDestinationCatalog(&catalog)
	if err != nil {
		w.failJob(ctx, result.Job.ID, fmt.Errorf("building destination catalog: %w", err))
		return nil
	}
	pipelinecatalog.ApplyNamespaceAndPrefix(destCatalog, result.NamespaceDefinition, nilIfEmpty(result.CustomNamespaceFormat), nilIfEmpty(result.StreamPrefix))
	destCatalogJSON, err := json.Marshal(destCatalog)
	if err != nil {
		w.failJob(ctx, result.Job.ID, fmt.Errorf("marshaling dest catalog: %w", err))
		return nil
	}

	// Build K8s job options directly from ClaimJobBundleResult bundle.
	opts := K8sSyncJobOptions{
		JobID:                 result.Job.ID,
		ConnectionID:          result.ConnectionID,
		SourceID:              result.SourceID,
		DestinationID:         result.DestinationID,
		SourceImage:           result.SourceImage,
		DestImage:             result.DestImage,
		SourceConfig:          result.SourceConfig,
		DestConfig:            result.DestConfig,
		SourceCatalog:         result.ConfiguredCatalog,
		DestCatalog:           destCatalogJSON,
		State:                 result.StateBlob,
		RuntimeConfig:         result.SourceRuntimeConfig,
		InternalAPIURL:        w.internalAPIURL,
		NamespaceDefinition:   result.NamespaceDefinition.String(),
		CustomNamespaceFormat: result.CustomNamespaceFormat,
		StreamPrefix:          result.StreamPrefix,
	}

	// Create K8s Job -- fire-and-forget per D-09, do NOT wait for completion.
	k8sJobName, err := w.k8sRunner.CreateSyncJob(ctx, opts)
	if err != nil {
		w.failJob(ctx, result.Job.ID, fmt.Errorf("creating k8s job: %w", err))
		return nil
	}

	// Store the K8s job name on the job record.
	if err := w.setK8sJobName.Execute(ctx, pipelinejobs.SetK8sJobNameParams{
		ID:         result.Job.ID,
		K8sJobName: k8sJobName,
	}); err != nil {
		if w.logger != nil {
			w.logger.WithError(err).WithFields(map[string]interface{}{"job_id": result.Job.ID.String(), "k8s_job_name": k8sJobName}).Error(ctx, "failed to set k8s job name")
		}
	}

	if w.logger != nil {
		w.logger.WithFields(map[string]interface{}{"job_id": result.Job.ID.String(), "connection_id": result.ConnectionID.String(), "k8s_job_name": k8sJobName}).Info(ctx, "launched k8s sync job")
	}

	return nil
}

// failJob updates the job status to failed with the given error via ExecutorBackend.
func (w *K8sSyncWorker) failJob(ctx context.Context, jobID uuid.UUID, reason error) {
	if w.logger != nil {
		w.logger.WithError(reason).WithField("job_id", jobID.String()).Error(ctx, "k8s sync worker: job failed")
	}

	if err := w.backend.UpdateJobStatus(ctx, UpdateJobStatusParams{
		JobID:        jobID,
		Success:      false,
		ErrorMessage: reason.Error(),
	}); err != nil {
		if w.logger != nil {
			w.logger.WithError(err).WithField("job_id", jobID.String()).Error(ctx, "failed to update job status")
		}
	}
}
