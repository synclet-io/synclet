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

// ConfigureStreamsParams holds parameters for configuring streams on a connection.
type ConfigureStreamsParams struct {
	ConnectionID uuid.UUID
	SourceID     uuid.UUID
	Streams      []protocol.ConfiguredAirbyteStream
}

// ConfigureStreams validates and stores the configured streams for a connection.
type ConfigureStreams struct {
	storage pipelineservice.Storage
}

// NewConfigureStreams creates a new ConfigureStreams use case.
func NewConfigureStreams(storage pipelineservice.Storage) *ConfigureStreams {
	return &ConfigureStreams{storage: storage}
}

// Execute validates the configured streams against the latest discovered catalog
// and creates or updates the configured catalog for the connection.
func (uc *ConfigureStreams) Execute(ctx context.Context, params ConfigureStreamsParams) error {
	// Get the latest discovered catalog to validate against.
	latestRecord, err := uc.storage.CatalogDiscoverys().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(params.SourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err != nil {
		return fmt.Errorf("getting latest catalog for validation: %w", err)
	}

	var latestCatalog protocol.AirbyteCatalog
	if err := json.Unmarshal([]byte(latestRecord.CatalogJSON), &latestCatalog); err != nil {
		return fmt.Errorf("unmarshaling latest catalog: %w", err)
	}

	// Build a lookup map of available streams.
	availableStreams := make(map[string]protocol.AirbyteStream, len(latestCatalog.Streams))
	for _, stream := range latestCatalog.Streams {
		key := streamKey(stream.Namespace, stream.Name)
		availableStreams[key] = stream
	}

	// Validate each configured stream.
	for i, configured := range params.Streams {
		key := streamKey(configured.Stream.Namespace, configured.Stream.Name)

		available, ok := availableStreams[key]
		if !ok {
			return fmt.Errorf("stream %q not found in discovered catalog", key)
		}

		// Enrich configured stream with schema and metadata from the discovered catalog.
		// The Airbyte CDK requires json_schema.properties to be present during read.
		params.Streams[i].Stream.JSONSchema = available.JSONSchema
		params.Streams[i].Stream.SupportedSyncModes = available.SupportedSyncModes
		params.Streams[i].Stream.SourceDefinedCursor = available.SourceDefinedCursor
		params.Streams[i].Stream.DefaultCursorField = available.DefaultCursorField
		params.Streams[i].Stream.SourceDefinedPrimaryKey = available.SourceDefinedPrimaryKey

		if !syncModeSupported(available.SupportedSyncModes, configured.SyncMode) {
			return fmt.Errorf("sync mode %q not supported for stream %q", configured.SyncMode, key)
		}

		// Reject incompatible sync mode combinations.
		if configured.SyncMode == protocol.SyncModeFullRefresh &&
			configured.DestinationSyncMode == protocol.DestinationSyncModeAppendDedup {
			return fmt.Errorf("sync mode full_refresh is incompatible with destination sync mode append_dedup for stream %q", key)
		}

		// Use source-defined primary key when available.
		if len(available.SourceDefinedPrimaryKey) > 0 {
			params.Streams[i].PrimaryKey = available.SourceDefinedPrimaryKey
		}

		// Use source-defined cursor when available.
		if available.SourceDefinedCursor {
			params.Streams[i].CursorField = available.DefaultCursorField
		}

		// Validate cursor field for incremental sync when source doesn't define one.
		if configured.SyncMode == protocol.SyncModeIncremental &&
			!available.SourceDefinedCursor &&
			len(configured.CursorField) == 0 {
			return fmt.Errorf("cursor field required for incremental sync on stream %q (source does not define cursor)", key)
		}

		// Validate primary key for append_dedup when source doesn't define one.
		if configured.DestinationSyncMode == protocol.DestinationSyncModeAppendDedup &&
			len(available.SourceDefinedPrimaryKey) == 0 &&
			len(configured.PrimaryKey) == 0 {
			return fmt.Errorf("primary key required for append_dedup on stream %q (source does not define primary key)", key)
		}

		// Validate and enhance selected fields.
		if len(configured.SelectedFields) > 0 {
			if err := ValidateSelectedFields(configured.SelectedFields, available.JSONSchema); err != nil {
				return fmt.Errorf("invalid selected fields for stream %q: %w", key, err)
			}

			params.Streams[i].SelectedFields = ForceIncludeFields(
				configured.SelectedFields,
				configured.CursorField,
				configured.PrimaryKey,
			)
		}
	}

	streamsJSON, err := json.Marshal(params.Streams)
	if err != nil {
		return fmt.Errorf("marshaling configured streams: %w", err)
	}

	now := time.Now()

	// Check if configured catalog already exists for this connection.
	existing, err := uc.storage.ConfiguredCatalogs().First(ctx, &pipelineservice.ConfiguredCatalogFilter{
		ConnectionID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		var nfe pipelineservice.NotFoundError
		if !errors.As(err, &nfe) {
			return fmt.Errorf("getting configured catalog: %w", err)
		}

		// Not found -- create new.
		record := &pipelineservice.ConfiguredCatalog{
			ID:           uuid.New(),
			ConnectionID: params.ConnectionID,
			StreamsJSON:  string(streamsJSON),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if _, err := uc.storage.ConfiguredCatalogs().Create(ctx, record); err != nil {
			return fmt.Errorf("creating configured catalog: %w", err)
		}

		return nil
	}

	// Update existing.
	existing.StreamsJSON = string(streamsJSON)
	existing.UpdatedAt = now

	if _, err := uc.storage.ConfiguredCatalogs().Update(ctx, existing); err != nil {
		return fmt.Errorf("updating configured catalog: %w", err)
	}

	return nil
}

// streamKey returns a unique key for a stream based on namespace and name.
func streamKey(namespace, name string) string {
	if namespace == "" {
		return name
	}

	return namespace + "." + name
}

// syncModeSupported checks if a sync mode is in the list of supported modes.
func syncModeSupported(supported []protocol.SyncMode, mode protocol.SyncMode) bool {
	for _, s := range supported {
		if s == mode {
			return true
		}
	}

	return false
}
