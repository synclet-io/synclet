package pipelineconnect

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/pkg/connectutil"
	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func (h *ConnectionHandler) ValidateMappers(ctx context.Context, req *connect.Request[pipelinev1.ValidateMappersRequest]) (*connect.Response[pipelinev1.ValidateMappersResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	params, err := mapperSourceParams(req.Msg)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	mappers, err := mappersToJSON(req.Msg.GetMappers())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	var sample json.RawMessage
	if req.Msg.GetSampleRecord() != nil {
		sample, err = protojson.Marshal(req.Msg.GetSampleRecord())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid sample record: %w", err))
		}
	}

	selectedFields := make([]protocol.SelectedField, len(req.Msg.GetSelectedFields()))
	for i, f := range req.Msg.GetSelectedFields() {
		selectedFields[i] = protocol.SelectedField{FieldPath: f.GetFieldPath()}
	}

	params.WorkspaceID = workspaceID
	params.Stream = protocol.ConfiguredAirbyteStream{
		Stream: protocol.AirbyteStream{
			Name:      req.Msg.GetStreamName(),
			Namespace: req.Msg.GetNamespace(),
		},
		DestinationSyncMode: protoToDestinationSyncMode(req.Msg.GetDestinationSyncMode()),
		PrimaryKey:          compositeKeysToPaths(req.Msg.GetPrimaryKey()),
		SelectedFields:      selectedFields,
		Mappers:             mappers,
	}
	params.SampleRecord = sample

	result, err := h.validateMappers.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	resp := &pipelinev1.ValidateMappersResponse{
		Errors:           mapperIssuesToProto(result.Errors),
		Warnings:         mapperIssuesToProto(result.Warnings),
		OutputPrimaryKey: pathsToCompositeKeys(result.OutputPrimaryKey),
		SampleKept:       result.SampleKept,
		SampleError:      result.SampleError,
	}

	if result.OutputSchema != nil {
		schema := &structpb.Struct{}
		if err := protojson.Unmarshal(result.OutputSchema, schema); err != nil {
			return nil, mapError(fmt.Errorf("converting output schema: %w", err))
		}

		resp.OutputSchema = schema
	}

	if result.SampleOutput != nil {
		sampleOutput := &structpb.Struct{}
		if err := protojson.Unmarshal(result.SampleOutput, sampleOutput); err != nil {
			return nil, mapError(fmt.Errorf("converting sample output: %w", err))
		}

		resp.SampleOutput = sampleOutput
	}

	return connect.NewResponse(resp), nil
}

// mapperSourceParams reads which source the request validates against: a
// connection id, or a source id with the name of the connection being created.
func mapperSourceParams(msg *pipelinev1.ValidateMappersRequest) (pipelinecatalog.ValidateMappersParams, error) {
	switch {
	case msg.GetConnectionId() != "" && msg.GetSourceId() != "":
		return pipelinecatalog.ValidateMappersParams{}, errors.New("set either connection_id or source_id, not both")
	case msg.GetConnectionId() != "":
		connID, err := uuid.Parse(msg.GetConnectionId())
		if err != nil {
			return pipelinecatalog.ValidateMappersParams{}, fmt.Errorf("invalid connection_id: %w", err)
		}

		return pipelinecatalog.ValidateMappersParams{ConnectionID: connID}, nil
	case msg.GetSourceId() != "":
		sourceID, err := uuid.Parse(msg.GetSourceId())
		if err != nil {
			return pipelinecatalog.ValidateMappersParams{}, fmt.Errorf("invalid source_id: %w", err)
		}

		if msg.GetConnectionName() == "" {
			return pipelinecatalog.ValidateMappersParams{}, errors.New("connection_name is required with source_id")
		}

		return pipelinecatalog.ValidateMappersParams{SourceID: sourceID, ConnectionName: msg.GetConnectionName()}, nil
	default:
		return pipelinecatalog.ValidateMappersParams{}, errors.New("connection_id or source_id is required")
	}
}

// mappersToJSON encodes the mapper steps as a JSON array, or nil when there are none.
func mappersToJSON(steps []*structpb.Struct) (json.RawMessage, error) {
	if len(steps) == 0 {
		return nil, nil
	}

	raw := make([]json.RawMessage, len(steps))

	for i, step := range steps {
		encoded, err := protojson.Marshal(step)
		if err != nil {
			return nil, fmt.Errorf("invalid mapper step %d: %w", i, err)
		}

		raw[i] = encoded
	}

	out, err := json.Marshal(raw)
	if err != nil {
		return nil, fmt.Errorf("encoding mappers: %w", err)
	}

	return out, nil
}

func compositeKeysToPaths(keys []*pipelinev1.CompositeKey) [][]string {
	out := make([][]string, len(keys))
	for i, key := range keys {
		out[i] = key.GetFieldPath()
	}

	return out
}

func pathsToCompositeKeys(paths [][]string) []*pipelinev1.CompositeKey {
	out := make([]*pipelinev1.CompositeKey, len(paths))
	for i, path := range paths {
		out[i] = &pipelinev1.CompositeKey{FieldPath: path}
	}

	return out
}

func mapperIssuesToProto(issues []mapper.Issue) []*pipelinev1.MapperIssue {
	out := make([]*pipelinev1.MapperIssue, len(issues))
	for i, issue := range issues {
		out[i] = &pipelinev1.MapperIssue{
			Step:    int32(issue.Step), //nolint:gosec // step indexes are small
			Field:   issue.Field,
			Message: issue.Message,
		}
	}

	return out
}
