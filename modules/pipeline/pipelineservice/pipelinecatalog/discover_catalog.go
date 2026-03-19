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

// DiscoverCatalogParams holds parameters for discovering a catalog.
type DiscoverCatalogParams struct {
	SourceID uuid.UUID
	Image    string
	Config   json.RawMessage
}

// DiscoverCatalog runs discovery on a source connector, stores the result, and returns the catalog.
type DiscoverCatalog struct {
	storage   pipelineservice.Storage
	connector pipelineservice.ConnectorDiscoverer
}

// NewDiscoverCatalog creates a new DiscoverCatalog use case.
func NewDiscoverCatalog(storage pipelineservice.Storage, connector pipelineservice.ConnectorDiscoverer) *DiscoverCatalog {
	return &DiscoverCatalog{storage: storage, connector: connector}
}

// Execute discovers the catalog from a source connector and persists it.
func (uc *DiscoverCatalog) Execute(ctx context.Context, params DiscoverCatalogParams) (*protocol.AirbyteCatalog, error) {
	catalog, err := uc.connector.Discover(ctx, params.Image, params.Config)
	if err != nil {
		return nil, fmt.Errorf("discovering catalog: %w", err)
	}

	catalogJSON, err := json.Marshal(catalog)
	if err != nil {
		return nil, fmt.Errorf("marshaling catalog: %w", err)
	}

	// Compute next version number.
	nextVersion := 1

	latest, err := uc.storage.CatalogDiscoverys().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(params.SourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err != nil {
		var nfe pipelineservice.NotFoundError
		if !errors.As(err, &nfe) {
			return nil, fmt.Errorf("getting latest catalog: %w", err)
		}
	} else {
		nextVersion = latest.Version + 1
	}

	record := &pipelineservice.CatalogDiscovery{
		ID:           uuid.New(),
		SourceID:     params.SourceID,
		Version:      nextVersion,
		CatalogJSON:  string(catalogJSON),
		DiscoveredAt: time.Now(),
	}

	if _, err := uc.storage.CatalogDiscoverys().Create(ctx, record); err != nil {
		return nil, fmt.Errorf("storing catalog: %w", err)
	}

	return catalog, nil
}
