package pipelinecatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// GetSourceCatalogParams holds parameters for getting a cached source catalog.
type GetSourceCatalogParams struct {
	SourceID    uuid.UUID
	WorkspaceID uuid.UUID
}

// GetSourceCatalogResult holds the result of getting a cached source catalog.
type GetSourceCatalogResult struct {
	Catalog      *protocol.AirbyteCatalog
	Version      int
	DiscoveredAt time.Time
}

// GetSourceCatalog retrieves the latest cached catalog for a source.
type GetSourceCatalog struct {
	storage pipelineservice.Storage
}

// NewGetSourceCatalog creates a new GetSourceCatalog use case.
func NewGetSourceCatalog(storage pipelineservice.Storage) *GetSourceCatalog {
	return &GetSourceCatalog{storage: storage}
}

// Execute retrieves the latest cached catalog for the given source.
func (uc *GetSourceCatalog) Execute(ctx context.Context, params GetSourceCatalogParams) (*GetSourceCatalogResult, error) {
	// Verify source belongs to workspace.
	_, err := uc.storage.Sources().First(ctx, &pipelineservice.SourceFilter{
		ID:          filter.Equals(params.SourceID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("finding source: %w", err)
	}

	// Get the latest catalog discovery for this source.
	latest, err := uc.storage.CatalogDiscoverys().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(params.SourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err != nil {
		var nfe pipelineservice.NotFoundError
		if errors.As(err, &nfe) {
			return nil, pipelineservice.NotFoundError("no catalog discovered for this source")
		}

		return nil, fmt.Errorf("getting latest catalog: %w", err)
	}

	var catalog protocol.AirbyteCatalog
	if err := json.Unmarshal([]byte(latest.CatalogJSON), &catalog); err != nil {
		return nil, fmt.Errorf("parsing cached catalog: %w", err)
	}

	return &GetSourceCatalogResult{
		Catalog:      &catalog,
		Version:      latest.Version,
		DiscoveredAt: latest.DiscoveredAt,
	}, nil
}
