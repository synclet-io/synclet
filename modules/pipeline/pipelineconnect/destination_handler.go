package pipelineconnect

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// DestinationHandler implements the DestinationService ConnectRPC handler.
type DestinationHandler struct {
	pipelinev1connect.UnimplementedDestinationServiceHandler

	createDestination *pipelinedestinations.CreateDestination
	updateDestination *pipelinedestinations.UpdateDestination
	deleteDestination *pipelinedestinations.DeleteDestination
	getDestination    *pipelinedestinations.GetDestination
	listDestinations  *pipelinedestinations.ListDestinations
	createCheckTask   *pipelinetasks.CreateCheckTask
	waitForTaskResult *pipelinetasks.WaitForTaskResult
}

// NewDestinationHandler creates a new destination handler.
func NewDestinationHandler(
	createDestination *pipelinedestinations.CreateDestination,
	updateDestination *pipelinedestinations.UpdateDestination,
	deleteDestination *pipelinedestinations.DeleteDestination,
	getDestination *pipelinedestinations.GetDestination,
	listDestinations *pipelinedestinations.ListDestinations,
	createCheckTask *pipelinetasks.CreateCheckTask,
	waitForTaskResult *pipelinetasks.WaitForTaskResult,
) *DestinationHandler {
	return &DestinationHandler{
		createDestination: createDestination,
		updateDestination: updateDestination,
		deleteDestination: deleteDestination,
		getDestination:    getDestination,
		listDestinations:  listDestinations,
		createCheckTask:   createCheckTask,
		waitForTaskResult: waitForTaskResult,
	}
}

func (h *DestinationHandler) CreateDestination(ctx context.Context, req *connect.Request[pipelinev1.CreateDestinationRequest]) (*connect.Response[pipelinev1.CreateDestinationResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.Name, MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	managedConnectorID, err := uuid.Parse(req.Msg.ManagedConnectorId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid managed_connector_id: %w", err))
	}

	config, err := json.Marshal(req.Msg.Config.AsMap())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Blocking connection check before persist.
	if err := runConnectionCheck(ctx, h.createCheckTask, h.waitForTaskResult,
		workspaceID, managedConnectorID, config); err != nil {
		return nil, err
	}

	dest, err := h.createDestination.Execute(ctx, pipelinedestinations.CreateDestinationParams{
		WorkspaceID:        workspaceID,
		Name:               req.Msg.Name,
		ManagedConnectorID: managedConnectorID,
		Config:             config,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.CreateDestinationResponse{
		Destination: destinationToProto(dest),
	}), nil
}

func (h *DestinationHandler) UpdateDestination(ctx context.Context, req *connect.Request[pipelinev1.UpdateDestinationRequest]) (*connect.Response[pipelinev1.UpdateDestinationResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := pipelinedestinations.UpdateDestinationParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}
	if req.Msg.Name != nil {
		if err := connectutil.ValidateStringLengths(
			connectutil.StringValidation{Field: "name", Value: *req.Msg.Name, MaxLen: connectutil.MaxNameLength},
		); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		params.Name = req.Msg.Name
	}
	if req.Msg.Config != nil {
		config, err := json.Marshal(req.Msg.Config.AsMap())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		raw := json.RawMessage(config)
		params.Config = &raw

		// Blocking connection check before persist.
		existingDest, err := h.getDestination.Execute(ctx, pipelinedestinations.GetDestinationParams{
			ID:          id,
			WorkspaceID: workspaceID,
		})
		if err != nil {
			return nil, mapError(err)
		}
		if err := runConnectionCheck(ctx, h.createCheckTask, h.waitForTaskResult,
			workspaceID, existingDest.ManagedConnectorID, raw); err != nil {
			return nil, err
		}
	}
	if req.Msg.RuntimeConfig != nil {
		params.RuntimeConfig = req.Msg.RuntimeConfig
	}

	dest, err := h.updateDestination.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.UpdateDestinationResponse{
		Destination: destinationToProto(dest),
	}), nil
}

func (h *DestinationHandler) DeleteDestination(ctx context.Context, req *connect.Request[pipelinev1.DeleteDestinationRequest]) (*connect.Response[pipelinev1.DeleteDestinationResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.deleteDestination.Execute(ctx, pipelinedestinations.DeleteDestinationParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.DeleteDestinationResponse{}), nil
}

func (h *DestinationHandler) GetDestination(ctx context.Context, req *connect.Request[pipelinev1.GetDestinationRequest]) (*connect.Response[pipelinev1.GetDestinationResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	dest, err := h.getDestination.Execute(ctx, pipelinedestinations.GetDestinationParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetDestinationResponse{
		Destination: destinationToProto(dest),
	}), nil
}

func (h *DestinationHandler) ListDestinations(ctx context.Context, req *connect.Request[pipelinev1.ListDestinationsRequest]) (*connect.Response[pipelinev1.ListDestinationsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	dests, err := h.listDestinations.Execute(ctx, pipelinedestinations.ListDestinationsParams{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoDests := make([]*pipelinev1.Destination, len(dests))
	for i, d := range dests {
		protoDests[i] = destinationToProto(d)
	}

	paginated, total := paginateSlice(protoDests, req.Msg.PageSize, req.Msg.Offset)

	return connect.NewResponse(&pipelinev1.ListDestinationsResponse{
		Destinations: paginated,
		Total:        total,
	}), nil
}

func (h *DestinationHandler) TestDestinationConnection(ctx context.Context, req *connect.Request[pipelinev1.TestDestinationConnectionRequest]) (*connect.Response[pipelinev1.TestDestinationConnectionResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := pipelinetasks.CreateCheckTaskParams{
		WorkspaceID: workspaceID,
	}

	if req.Msg.ManagedConnectorId != "" && req.Msg.Config != nil {
		// Direct config path: test without creating a destination.
		mcID, err := uuid.Parse(req.Msg.ManagedConnectorId)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid managed_connector_id: %w", err))
		}
		configJSON, err := json.Marshal(req.Msg.Config.AsMap())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid config: %w", err))
		}
		params.ManagedConnectorID = &mcID
		params.Config = configJSON
	} else if req.Msg.Id != "" {
		// Existing destination path.
		id, err := uuid.Parse(req.Msg.Id)
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		params.DestinationID = &id
	} else {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("either id or managed_connector_id+config must be provided"))
	}

	result, err := h.createCheckTask.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.TestDestinationConnectionResponse{
		TaskId: result.TaskID.String(),
	}), nil
}

func destinationToProto(d *pipelineservice.Destination) *pipelinev1.Destination {
	var config *structpb.Struct
	if d.Config != "" {
		// Mask secret references before returning to API
		maskedConfig, err := pipelinesecrets.MaskConfigSecrets(d.Config)
		if err != nil {
			maskedConfig = d.Config
		}
		var m map[string]any
		if json.Unmarshal([]byte(maskedConfig), &m) == nil {
			config, _ = structpb.NewStruct(m)
		}
	}

	proto := &pipelinev1.Destination{
		Id:                 d.ID.String(),
		WorkspaceId:        d.WorkspaceID.String(),
		Name:               d.Name,
		ManagedConnectorId: d.ManagedConnectorID.String(),
		Config:             config,
		CreatedAt:          timestamppb.New(d.CreatedAt),
		UpdatedAt:          timestamppb.New(d.UpdatedAt),
	}
	if d.RuntimeConfig != nil {
		proto.RuntimeConfig = d.RuntimeConfig
	}
	return proto
}
