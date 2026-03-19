package pipelinecatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// DetectSchemaChangesParams holds parameters for detecting schema changes.
type DetectSchemaChangesParams struct {
	ConnectionID uuid.UUID
	SourceID     uuid.UUID
}

// DetectSchemaChanges compares the latest discovered catalog for a source
// with the currently configured catalog for a connection and returns any changes.
type DetectSchemaChanges struct {
	storage   pipelineservice.Storage
	connector pipelineservice.ConnectorDiscoverer
}

// NewDetectSchemaChanges creates a new DetectSchemaChanges use case.
func NewDetectSchemaChanges(storage pipelineservice.Storage, connector pipelineservice.ConnectorDiscoverer) *DetectSchemaChanges {
	return &DetectSchemaChanges{storage: storage, connector: connector}
}

// Execute detects schema changes between the latest catalog and the configured catalog.
func (uc *DetectSchemaChanges) Execute(ctx context.Context, params DetectSchemaChangesParams) ([]SchemaChange, error) {
	// Get the latest discovered catalog.
	latestRecord, err := uc.storage.CatalogDiscoverys().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(params.SourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err != nil {
		return nil, fmt.Errorf("getting latest catalog: %w", err)
	}

	var latestCatalog protocol.AirbyteCatalog
	if err := json.Unmarshal([]byte(latestRecord.CatalogJSON), &latestCatalog); err != nil {
		return nil, fmt.Errorf("unmarshaling latest catalog: %w", err)
	}

	// Get the configured catalog.
	configuredRecord, err := uc.storage.ConfiguredCatalogs().First(ctx, &pipelineservice.ConfiguredCatalogFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		// If no configured catalog exists yet, there's nothing to compare — no changes.
		var nfe pipelineservice.NotFoundError
		if errors.As(err, &nfe) {
			return nil, nil
		}

		return nil, fmt.Errorf("getting configured catalog: %w", err)
	}

	var configuredStreams []protocol.ConfiguredAirbyteStream
	if err := json.Unmarshal([]byte(configuredRecord.StreamsJSON), &configuredStreams); err != nil {
		return nil, fmt.Errorf("unmarshaling configured streams: %w", err)
	}

	// Build a catalog from configured streams for comparison.
	oldCatalog := &protocol.AirbyteCatalog{
		Streams: make([]protocol.AirbyteStream, len(configuredStreams)),
	}
	for i, cs := range configuredStreams {
		oldCatalog.Streams[i] = cs.Stream
	}

	changes := ComputeSchemaDiff(oldCatalog, &latestCatalog)
	if len(changes) == 0 {
		return nil, nil
	}

	return changes, nil
}
