package pipelinecatalog

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// GetConfiguredCatalogParams holds parameters for getting a configured catalog.
type GetConfiguredCatalogParams struct {
	ConnectionID uuid.UUID
}

// GetConfiguredCatalog retrieves the configured catalog for a connection.
type GetConfiguredCatalog struct {
	storage pipelineservice.Storage
}

// NewGetConfiguredCatalog creates a new GetConfiguredCatalog use case.
func NewGetConfiguredCatalog(storage pipelineservice.Storage) *GetConfiguredCatalog {
	return &GetConfiguredCatalog{storage: storage}
}

// Execute returns the configured catalog for the given connection.
// It enriches each stream with json_schema and metadata from the latest
// discovered catalog so connectors receive full schema information.
func (uc *GetConfiguredCatalog) Execute(ctx context.Context, params GetConfiguredCatalogParams) (*protocol.ConfiguredAirbyteCatalog, error) {
	record, err := uc.storage.ConfiguredCatalogs().First(ctx, &pipelineservice.ConfiguredCatalogFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting configured catalog: %w", err)
	}

	var configuredStreams []protocol.ConfiguredAirbyteStream
	if err := json.Unmarshal([]byte(record.StreamsJSON), &configuredStreams); err != nil {
		return nil, fmt.Errorf("unmarshaling configured streams: %w", err)
	}

	// Look up the connection to find the source ID for discovered catalog enrichment.
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return nil, fmt.Errorf("loading connection: %w", err)
	}

	// Load the latest discovered catalog to enrich configured streams with
	// json_schema and stream metadata that connectors require at runtime.
	latestRecord, discoverErr := uc.storage.CatalogDiscoverys().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(conn.SourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if discoverErr == nil {
		var latestCatalog protocol.AirbyteCatalog
		if err := json.Unmarshal([]byte(latestRecord.CatalogJSON), &latestCatalog); err == nil {
			availableStreams := make(map[string]protocol.AirbyteStream, len(latestCatalog.Streams))
			for _, stream := range latestCatalog.Streams {
				availableStreams[streamKey(stream.Namespace, stream.Name)] = stream
			}

			for i, cs := range configuredStreams {
				key := streamKey(cs.Stream.Namespace, cs.Stream.Name)
				if available, ok := availableStreams[key]; ok {
					configuredStreams[i].Stream.JSONSchema = available.JSONSchema
					configuredStreams[i].Stream.SupportedSyncModes = available.SupportedSyncModes
					configuredStreams[i].Stream.SourceDefinedCursor = available.SourceDefinedCursor
					configuredStreams[i].Stream.DefaultCursorField = available.DefaultCursorField
					configuredStreams[i].Stream.SourceDefinedPrimaryKey = available.SourceDefinedPrimaryKey
				}
			}
		}
	}

	return &protocol.ConfiguredAirbyteCatalog{Streams: configuredStreams}, nil
}
