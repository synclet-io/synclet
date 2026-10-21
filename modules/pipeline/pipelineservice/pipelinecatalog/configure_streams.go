package pipelinecatalog

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"slices"
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
	audit   pipelineservice.AuditRecorder
}

// NewConfigureStreams creates a new ConfigureStreams use case.
func NewConfigureStreams(storage pipelineservice.Storage, audit pipelineservice.AuditRecorder) *ConfigureStreams {
	return &ConfigureStreams{storage: storage, audit: audit}
}

// Execute validates the configured streams against the latest discovered catalog
// and creates or updates the configured catalog for the connection.
func (uc *ConfigureStreams) Execute(ctx context.Context, params ConfigureStreamsParams) error {
	conn, err := uc.storage.Connections().First(ctx, &pipelineservice.ConnectionFilter{
		ID: filter.Equals(params.ConnectionID),
	})
	if err != nil {
		return fmt.Errorf("getting connection: %w", err)
	}

	if err := PrepareStreams(ctx, uc.storage, params.SourceID, params.Streams); err != nil {
		return err
	}

	previous, err := SaveStreams(ctx, uc.storage, params.ConnectionID, params.Streams)
	if err != nil {
		return err
	}

	uc.recordAudit(ctx, conn, previous, params.Streams)

	return nil
}

// PrepareStreams validates streams against the source's latest discovered
// catalog and fills in, in place, the schema, sync modes, cursor and primary
// key the catalog defines. An empty list is a *ValidationError, and a source
// with no discovered catalog yet is ErrCatalogNotReady.
func PrepareStreams(ctx context.Context, storage pipelineservice.Storage, sourceID uuid.UUID, streams []protocol.ConfiguredAirbyteStream) error {
	if len(streams) == 0 {
		return streamValidationError("at least one stream must be configured")
	}

	availableStreams, err := latestDiscoveredStreams(ctx, storage, sourceID)
	if errors.Is(err, pipelineservice.ErrCatalogDiscoveryNotFound) {
		return pipelineservice.ErrCatalogNotReady
	}

	if err != nil {
		return fmt.Errorf("getting latest catalog for validation: %w", err)
	}

	for i, configured := range streams {
		if err := prepareStream(availableStreams, &streams[i], configured); err != nil {
			return err
		}
	}

	return nil
}

func prepareStream(
	availableStreams map[string]protocol.AirbyteStream,
	stream *protocol.ConfiguredAirbyteStream,
	configured protocol.ConfiguredAirbyteStream,
) error {
	key := streamKey(configured.Stream.Namespace, configured.Stream.Name)

	available, ok := availableStreams[key]
	if !ok {
		return streamValidationError("stream %q not found in discovered catalog", key)
	}

	// Enrich configured stream with schema and metadata from the discovered catalog.
	// The Airbyte CDK requires json_schema.properties to be present during read.
	stream.Stream.JSONSchema = available.JSONSchema
	stream.Stream.SupportedSyncModes = available.SupportedSyncModes
	stream.Stream.SourceDefinedCursor = available.SourceDefinedCursor
	stream.Stream.DefaultCursorField = available.DefaultCursorField
	stream.Stream.SourceDefinedPrimaryKey = available.SourceDefinedPrimaryKey

	if !syncModeSupported(available.SupportedSyncModes, configured.SyncMode) {
		return streamValidationError("sync mode %q not supported for stream %q", configured.SyncMode, key)
	}

	// Reject incompatible sync mode combinations.
	if configured.SyncMode == protocol.SyncModeFullRefresh &&
		configured.DestinationSyncMode == protocol.DestinationSyncModeAppendDedup {
		return streamValidationError("sync mode full_refresh is incompatible with destination sync mode append_dedup for stream %q", key)
	}

	// Use source-defined primary key when available.
	if len(available.SourceDefinedPrimaryKey) > 0 {
		stream.PrimaryKey = available.SourceDefinedPrimaryKey
	}

	// Use source-defined cursor when available.
	if available.SourceDefinedCursor {
		stream.CursorField = available.DefaultCursorField
	}

	// Validate cursor field for incremental sync when source doesn't define one.
	if configured.SyncMode == protocol.SyncModeIncremental &&
		!available.SourceDefinedCursor &&
		len(configured.CursorField) == 0 {
		return streamValidationError("cursor field required for incremental sync on stream %q (source does not define cursor)", key)
	}

	// Validate primary key for append_dedup when source doesn't define one.
	if configured.DestinationSyncMode == protocol.DestinationSyncModeAppendDedup &&
		len(available.SourceDefinedPrimaryKey) == 0 &&
		len(configured.PrimaryKey) == 0 {
		return streamValidationError("primary key required for append_dedup on stream %q (source does not define primary key)", key)
	}

	// Validate and enhance selected fields.
	if len(configured.SelectedFields) > 0 {
		if err := pipelineservice.ValidateSelectedFields(configured.SelectedFields, available.JSONSchema); err != nil {
			return streamValidationError("invalid selected fields for stream %q: %v", key, err)
		}

		stream.SelectedFields = pipelineservice.ForceIncludeFields(
			configured.SelectedFields,
			stream.CursorField,
			stream.PrimaryKey,
		)
	}

	if pipelineservice.HasMappers(stream) {
		return validateStreamMappers(key, stream)
	}

	stream.Mappers = nil

	return nil
}

// SaveStreams stores streams as the connection's configured catalog and
// returns the streams it replaced, nil when the connection had none.
func SaveStreams(
	ctx context.Context,
	storage pipelineservice.Storage,
	connectionID uuid.UUID,
	streams []protocol.ConfiguredAirbyteStream,
) ([]protocol.ConfiguredAirbyteStream, error) {
	streamsJSON, err := json.Marshal(streams)
	if err != nil {
		return nil, fmt.Errorf("marshaling configured streams: %w", err)
	}

	now := time.Now()

	existing, err := storage.ConfiguredCatalogs().First(ctx, &pipelineservice.ConfiguredCatalogFilter{
		ConnectionID: filter.Equals(connectionID),
	})
	if err != nil {
		var nfe pipelineservice.NotFoundError
		if !errors.As(err, &nfe) {
			return nil, fmt.Errorf("getting configured catalog: %w", err)
		}

		record := &pipelineservice.ConfiguredCatalog{
			ID:           uuid.New(),
			ConnectionID: connectionID,
			StreamsJSON:  string(streamsJSON),
			CreatedAt:    now,
			UpdatedAt:    now,
		}

		if _, err := storage.ConfiguredCatalogs().Create(ctx, record); err != nil {
			return nil, fmt.Errorf("creating configured catalog: %w", err)
		}

		return nil, nil
	}

	var previousStreams []protocol.ConfiguredAirbyteStream
	if err := json.Unmarshal([]byte(existing.StreamsJSON), &previousStreams); err != nil {
		previousStreams = nil
	}

	existing.StreamsJSON = string(streamsJSON)
	existing.UpdatedAt = now

	if _, err := storage.ConfiguredCatalogs().Update(ctx, existing); err != nil {
		return nil, fmt.Errorf("updating configured catalog: %w", err)
	}

	return previousStreams, nil
}

func (uc *ConfigureStreams) recordAudit(
	ctx context.Context,
	conn *pipelineservice.Connection,
	before, after []protocol.ConfiguredAirbyteStream,
) {
	uc.audit.Record(ctx, pipelineservice.AuditEvent{
		WorkspaceID:   conn.WorkspaceID,
		Action:        "update",
		ResourceType:  "connection",
		ResourceID:    conn.ID,
		ResourceLabel: conn.Name,
		Before:        streamsAuditPayload(before),
		After:         streamsAuditPayload(after),
	})
}

// streamsAuditPayload lists configured streams as sorted
// "namespace.name:sync_mode/destination_sync_mode" entries.
func streamsAuditPayload(streams []protocol.ConfiguredAirbyteStream) map[string]any {
	entries := make([]string, len(streams))
	for i, stream := range streams {
		entries[i] = fmt.Sprintf("%s:%s/%s",
			streamKey(stream.Stream.Namespace, stream.Stream.Name),
			stream.SyncMode,
			stream.DestinationSyncMode,
		)
	}

	slices.Sort(entries)

	return map[string]any{"streams": entries}
}

// latestDiscoveredStreams returns the streams of the source's latest
// discovered catalog by stream key.
func latestDiscoveredStreams(ctx context.Context, storage pipelineservice.Storage, sourceID uuid.UUID) (map[string]protocol.AirbyteStream, error) {
	latestRecord, err := storage.CatalogDiscoveries().First(ctx, &pipelineservice.CatalogDiscoveryFilter{
		SourceID: filter.Equals(sourceID),
	}, dbutil.WithOrder(pipelineservice.CatalogDiscoveryFieldVersion, dbutil.OrderDirDesc))
	if err != nil {
		return nil, err
	}

	var latestCatalog protocol.AirbyteCatalog
	if err := json.Unmarshal([]byte(latestRecord.CatalogJSON), &latestCatalog); err != nil {
		return nil, fmt.Errorf("unmarshaling latest catalog: %w", err)
	}

	return streamsByKey(latestCatalog.Streams), nil
}

func streamsByKey(streams []protocol.AirbyteStream) map[string]protocol.AirbyteStream {
	out := make(map[string]protocol.AirbyteStream, len(streams))
	for _, stream := range streams {
		out[streamKey(stream.Namespace, stream.Name)] = stream
	}

	return out
}

// validateStreamMappers compiles the stream's mappers against its schema and
// returns a validation error naming each failing step.
func validateStreamMappers(key string, stream *protocol.ConfiguredAirbyteStream) error {
	_, result, err := pipelineservice.CompileStreamMappers(stream, stream.Stream.JSONSchema, nil)
	if err != nil {
		return streamValidationError("invalid mappers for stream %q: %v", key, err)
	}

	if len(result.Errors) > 0 {
		return streamValidationError("invalid mappers for stream %q: %s", key, pipelineservice.FormatMapperIssues(result.Errors))
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

func streamValidationError(format string, args ...any) error {
	return &pipelineservice.ValidationError{Field: "streams", Message: fmt.Sprintf(format, args...)}
}
