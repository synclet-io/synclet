package pipelineconnect

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/pkg/connectutil"
	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func (h *ConnectionHandler) ValidateMappers(ctx context.Context, req *connect.Request[pipelinev1.ValidateMappersRequest]) (*connect.Response[pipelinev1.ValidateMappersResponse], error) {
	workspaceID, err := connectutil.RequireWorkspaceID(ctx)
	if err != nil {
		return nil, err
	}

	connID, err := uuid.Parse(req.Msg.GetConnectionId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	conn, err := h.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          connID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
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

	result, err := h.validateMappers.Execute(ctx, pipelinecatalog.ValidateMappersParams{
		SourceID:       conn.SourceID,
		ConnectionName: conn.Name,
		Stream: protocol.ConfiguredAirbyteStream{
			Stream: protocol.AirbyteStream{
				Name:      req.Msg.GetStreamName(),
				Namespace: req.Msg.GetNamespace(),
			},
			DestinationSyncMode: protoToDestinationSyncMode(req.Msg.GetDestinationSyncMode()),
			PrimaryKey:          compositeKeysToPaths(req.Msg.GetPrimaryKey()),
			SelectedFields:      selectedFields,
			Mappers:             mappers,
		},
		SampleRecord: sample,
	})
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
