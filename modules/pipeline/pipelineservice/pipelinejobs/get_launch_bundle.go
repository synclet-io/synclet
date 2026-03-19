package pipelinejobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
)

// GetLaunchBundleParams holds parameters for loading the launch bundle.
type GetLaunchBundleParams struct {
	JobID uuid.UUID
}

// GetLaunchBundleResult contains all entities needed to launch a K8s sync pod.
type GetLaunchBundleResult struct {
	Job                    *pipelineservice.Job
	Connection             *pipelineservice.Connection
	Source                 *pipelineservice.Source
	Destination            *pipelineservice.Destination
	SourceManagedConnector *pipelineservice.ManagedConnector
	DestManagedConnector   *pipelineservice.ManagedConnector
}

// GetLaunchBundle loads the job and all related entities for K8s sync launch.
type GetLaunchBundle struct {
	storage pipelineservice.Storage
}

// NewGetLaunchBundle creates a new GetLaunchBundle use case.
func NewGetLaunchBundle(storage pipelineservice.Storage) *GetLaunchBundle {
	return &GetLaunchBundle{storage: storage}
}

// Execute loads the job, connection, source, destination, and both managed connectors.
func (uc *GetLaunchBundle) Execute(ctx context.Context, params GetLaunchBundleParams) (*GetLaunchBundleResult, error) {
	// Load the job.
	job, err := uc.storage.Jobs().First(ctx, &pipelineservice.JobFilter{
		ID: filter.Equals(params.JobID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading job: %w", err)
	}

	// Load connection.
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(job.ConnectionID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading connection: %w", err)
	}

	// Load source.
	src, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID: filter.Equals(conn.SourceID),
	})
	if err != nil {
		return nil, fmt.Errorf("resolving source: %w", err)
	}

	// Load source managed connector for image resolution.
	srcMC, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(src.ManagedConnectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading source managed connector: %w", err)
	}

	// Load destination.
	dest, err := uc.storage.Destinations().First(ctx, &pipelineservice.DestinationFilter{
		ID: filter.Equals(conn.DestinationID),
	})
	if err != nil {
		return nil, fmt.Errorf("resolving destination: %w", err)
	}

	// Load destination managed connector for image resolution.
	destMC, err := uc.storage.ManagedConnectors().First(ctx, &pipelineservice.ManagedConnectorFilter{
		ID: filter.Equals(dest.ManagedConnectorID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading destination managed connector: %w", err)
	}

	return &GetLaunchBundleResult{
		Job:                    job,
		Connection:             conn,
		Source:                 src,
		Destination:            dest,
		SourceManagedConnector: srcMC,
		DestManagedConnector:   destMC,
	}, nil
}
