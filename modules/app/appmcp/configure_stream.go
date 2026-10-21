package appmcp

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	pc "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// configureStreamTool sets one stream of a connection: it reads the configured
// catalog, replaces or inserts the stream matched by namespace and name (or
// drops it when enabled is false) and writes the whole list back.
func configureStreamTool(connections pc.ConnectionServiceHandler) toolDef {
	schema := messageSchema((&pipelinev1.ConfiguredStream{}).ProtoReflect().Descriptor())
	properties := schema["properties"].(map[string]any)
	properties["connection_id"] = map[string]any{"type": "string"}
	properties["enabled"] = map[string]any{
		"type":        "boolean",
		"default":     true,
		"description": "false removes the stream from the configuration.",
	}
	schema["required"] = []string{"connection_id", "stream_name"}

	return destructive(toolDef{
		name: "configure_stream",
		description: "Configure one stream of a connection, matched by namespace and stream_name, and keep every other stream as configured. " +
			"The stream's settings are replaced by the ones given and every field left out is cleared (sync modes reset to their defaults, cursor, primary key, selected fields and mappers emptied), " +
			"so send the full stream as get_configured_catalog returns it, with your change. " +
			"A stream that is not configured yet is added; enabled: false removes it.",
		kind:       kindWrite,
		procedures: []string{pc.ConnectionServiceGetConfiguredCatalogProcedure, pc.ConnectionServiceConfigureStreamsProcedure},
		schema:     schema,
		call: func(ctx context.Context, args json.RawMessage) ([]byte, error) {
			parsed, err := parseConfigureStreamArgs(args)
			if err != nil {
				return nil, err
			}

			streams, err := currentStreams(ctx, connections, parsed.connectionID)
			if err != nil {
				return nil, err
			}

			req := &pipelinev1.ConfigureStreamsRequest{
				ConnectionId: parsed.connectionID,
				Streams:      setStream(streams, parsed.stream, parsed.enabled),
			}

			resp, err := connections.ConfigureStreams(ctx, connect.NewRequest(req))
			if err != nil {
				return nil, err
			}

			return protojson.Marshal(resp.Msg)
		},
	})
}

// configureStreamArgs are the arguments of configure_stream.
type configureStreamArgs struct {
	connectionID string
	enabled      bool
	stream       *pipelinev1.ConfiguredStream
}

func parseConfigureStreamArgs(args json.RawMessage) (configureStreamArgs, error) {
	fields := map[string]json.RawMessage{}
	if len(args) > 0 && string(args) != "null" {
		if err := json.Unmarshal(args, &fields); err != nil {
			return configureStreamArgs{}, fmt.Errorf("%w: %w", errInvalidArguments, err)
		}
	}

	var connectionID string
	if raw, ok := fields["connection_id"]; ok {
		if err := json.Unmarshal(raw, &connectionID); err != nil {
			return configureStreamArgs{}, fmt.Errorf("%w: connection_id: %w", errInvalidArguments, err)
		}

		delete(fields, "connection_id")
	}

	enabled := true
	if raw, ok := fields["enabled"]; ok {
		if err := json.Unmarshal(raw, &enabled); err != nil {
			return configureStreamArgs{}, fmt.Errorf("%w: enabled: %w", errInvalidArguments, err)
		}

		delete(fields, "enabled")
	}

	rest, err := json.Marshal(fields)
	if err != nil {
		return configureStreamArgs{}, err
	}

	stream := &pipelinev1.ConfiguredStream{}
	if err := unmarshalArgs(rest, stream); err != nil {
		return configureStreamArgs{}, err
	}

	if stream.GetStreamName() == "" {
		return configureStreamArgs{}, fmt.Errorf("%w: stream_name is required", errInvalidArguments)
	}

	return configureStreamArgs{connectionID: connectionID, enabled: enabled, stream: stream}, nil
}

// setStream replaces the stream with stream's namespace and name, appends it
// when none matches, or drops it when enabled is false.
func setStream(streams []*pipelinev1.ConfiguredStream, stream *pipelinev1.ConfiguredStream, enabled bool) []*pipelinev1.ConfiguredStream {
	out := make([]*pipelinev1.ConfiguredStream, 0, len(streams)+1)
	found := false

	for _, s := range streams {
		if s.GetNamespace() != stream.GetNamespace() || s.GetStreamName() != stream.GetStreamName() {
			out = append(out, s)

			continue
		}

		found = true

		if enabled {
			out = append(out, stream)
		}
	}

	if !found && enabled {
		out = append(out, stream)
	}

	return out
}

// currentStreams returns the configured streams of a connection; a connection
// without a configured catalog has none.
func currentStreams(ctx context.Context, connections pc.ConnectionServiceHandler, connectionID string) ([]*pipelinev1.ConfiguredStream, error) {
	current, err := connections.GetConfiguredCatalog(ctx, connect.NewRequest(&pipelinev1.GetConfiguredCatalogRequest{
		ConnectionId: connectionID,
	}))
	if connect.CodeOf(err) == connect.CodeNotFound {
		return nil, nil
	}

	if err != nil {
		return nil, err
	}

	return configuredStreams(current.Msg.GetCatalog())
}

// configuredStreams converts the configured catalog, as GetConfiguredCatalog
// returns it, into the streams of a ConfigureStreams request.
func configuredStreams(catalog *structpb.Struct) ([]*pipelinev1.ConfiguredStream, error) {
	raw, err := protojson.Marshal(catalog)
	if err != nil {
		return nil, err
	}

	var parsed struct {
		Streams []storedStream `json:"streams"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, fmt.Errorf("decoding configured catalog: %w", err)
	}

	streams := make([]*pipelinev1.ConfiguredStream, 0, len(parsed.Streams))

	for _, s := range parsed.Streams {
		stream, err := configuredStream(s)
		if err != nil {
			return nil, err
		}

		streams = append(streams, stream)
	}

	return streams, nil
}

// storedStream is the part of a configured catalog stream that a
// ConfigureStreams request carries.
type storedStream struct {
	Stream struct {
		Name      string `json:"name"`
		Namespace string `json:"namespace"`
	} `json:"stream"`
	SyncMode            protocol.SyncMode            `json:"sync_mode"`
	DestinationSyncMode protocol.DestinationSyncMode `json:"destination_sync_mode"`
	CursorField         []string                     `json:"cursor_field"`
	PrimaryKey          [][]string                   `json:"primary_key"`
	SelectedFields      []protocol.SelectedField     `json:"selected_fields"`
	Mappers             json.RawMessage              `json:"mappers"`
}

func configuredStream(stored storedStream) (*pipelinev1.ConfiguredStream, error) {
	syncMode, ok := syncModes[stored.SyncMode]
	if !ok {
		return nil, unsupportedMode(stored, "sync mode", string(stored.SyncMode))
	}

	destinationSyncMode, ok := destinationSyncModes[stored.DestinationSyncMode]
	if !ok {
		return nil, unsupportedMode(stored, "destination sync mode", string(stored.DestinationSyncMode))
	}

	out := &pipelinev1.ConfiguredStream{
		StreamName:          stored.Stream.Name,
		Namespace:           stored.Stream.Namespace,
		SyncMode:            syncMode,
		DestinationSyncMode: destinationSyncMode,
		CursorField:         stored.CursorField,
	}

	for _, key := range stored.PrimaryKey {
		out.PrimaryKey = append(out.PrimaryKey, &pipelinev1.CompositeKey{FieldPath: key})
	}

	for _, field := range stored.SelectedFields {
		out.SelectedFields = append(out.SelectedFields, &pipelinev1.SelectedField{FieldPath: field.FieldPath})
	}

	if len(stored.Mappers) > 0 {
		var steps []json.RawMessage
		if err := json.Unmarshal(stored.Mappers, &steps); err != nil {
			return nil, fmt.Errorf("decoding mappers of stream %s: %w", stored.Stream.Name, err)
		}

		for _, step := range steps {
			mapper := &structpb.Struct{}
			if err := protojson.Unmarshal(step, mapper); err != nil {
				return nil, fmt.Errorf("decoding mappers of stream %s: %w", stored.Stream.Name, err)
			}

			out.Mappers = append(out.Mappers, mapper)
		}
	}

	return out, nil
}

func unsupportedMode(stored storedStream, what, mode string) error {
	return connect.NewError(connect.CodeFailedPrecondition,
		fmt.Errorf("stream %s.%s has %s %q, which the API cannot set; change the streams in the UI", stored.Stream.Namespace, stored.Stream.Name, what, mode))
}

var (
	syncModes = map[protocol.SyncMode]pipelinev1.SyncMode{
		protocol.SyncModeFullRefresh: pipelinev1.SyncMode_SYNC_MODE_FULL_REFRESH,
		protocol.SyncModeIncremental: pipelinev1.SyncMode_SYNC_MODE_INCREMENTAL,
	}
	destinationSyncModes = map[protocol.DestinationSyncMode]pipelinev1.DestinationSyncMode{
		protocol.DestinationSyncModeOverwrite:   pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_OVERWRITE,
		protocol.DestinationSyncModeAppend:      pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND,
		protocol.DestinationSyncModeAppendDedup: pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND_DEDUP,
	}
)
