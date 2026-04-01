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
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesecrets"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

// RegistrationEnabled is a named type for the registration toggle flag.
type RegistrationEnabled bool

// WorkspacesMode is a named type for the workspace mode (single/multi).
type WorkspacesMode = pipelinev1.WorkspacesMode

// SourceHandler implements the SourceService ConnectRPC handler.
type SourceHandler struct {
	pipelinev1connect.UnimplementedSourceServiceHandler

	createSource        *pipelinesources.CreateSource
	updateSource        *pipelinesources.UpdateSource
	deleteSource        *pipelinesources.DeleteSource
	getSource           *pipelinesources.GetSource
	listSources         *pipelinesources.ListSources
	createCheckTask     *pipelinetasks.CreateCheckTask
	waitForTaskResult   *pipelinetasks.WaitForTaskResult
	createDiscoverTask  *pipelinetasks.CreateDiscoverTask
	getSourceCatalog    *pipelinecatalog.GetSourceCatalog
	registrationEnabled RegistrationEnabled
	workspacesMode      WorkspacesMode
}

// NewSourceHandler creates a new source handler.
func NewSourceHandler(
	createSource *pipelinesources.CreateSource,
	updateSource *pipelinesources.UpdateSource,
	deleteSource *pipelinesources.DeleteSource,
	getSource *pipelinesources.GetSource,
	listSources *pipelinesources.ListSources,
	createCheckTask *pipelinetasks.CreateCheckTask,
	waitForTaskResult *pipelinetasks.WaitForTaskResult,
	createDiscoverTask *pipelinetasks.CreateDiscoverTask,
	getSourceCatalog *pipelinecatalog.GetSourceCatalog,
	registrationEnabled RegistrationEnabled,
	workspacesMode WorkspacesMode,
) *SourceHandler {
	return &SourceHandler{
		createSource:        createSource,
		updateSource:        updateSource,
		deleteSource:        deleteSource,
		getSource:           getSource,
		listSources:         listSources,
		createCheckTask:     createCheckTask,
		waitForTaskResult:   waitForTaskResult,
		createDiscoverTask:  createDiscoverTask,
		getSourceCatalog:    getSourceCatalog,
		registrationEnabled: registrationEnabled,
		workspacesMode:      workspacesMode,
	}
}

func (h *SourceHandler) CreateSource(ctx context.Context, req *connect.Request[pipelinev1.CreateSourceRequest]) (*connect.Response[pipelinev1.CreateSourceResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	managedConnectorID, err := uuid.Parse(req.Msg.GetManagedConnectorId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid managed_connector_id: %w", err))
	}

	config, err := json.Marshal(req.Msg.GetConfig().AsMap())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Blocking connection check before persist.
	if err := runConnectionCheck(ctx, h.createCheckTask, h.waitForTaskResult,
		workspaceID, managedConnectorID, config); err != nil {
		return nil, err
	}

	src, err := h.createSource.Execute(ctx, pipelinesources.CreateSourceParams{
		WorkspaceID:        workspaceID,
		Name:               req.Msg.GetName(),
		ManagedConnectorID: managedConnectorID,
		Config:             config,
	})
	if err != nil {
		return nil, mapError(err)
	}

	// Auto-trigger discover (fire-and-forget).
	var discoverTaskID *string

	taskResult, discoverErr := h.createDiscoverTask.Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{
		SourceID:    src.ID,
		WorkspaceID: workspaceID,
	})
	if discoverErr == nil && taskResult != nil {
		id := taskResult.TaskID.String()
		discoverTaskID = &id
	}

	return connect.NewResponse(&pipelinev1.CreateSourceResponse{
		Source:         sourceToProto(src),
		DiscoverTaskId: discoverTaskID,
	}), nil
}

func (h *SourceHandler) UpdateSource(ctx context.Context, req *connect.Request[pipelinev1.UpdateSourceRequest]) (*connect.Response[pipelinev1.UpdateSourceResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := pipelinesources.UpdateSourceParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}

	if req.Msg.Name != nil {
		if err := connectutil.ValidateStringLengths(
			connectutil.StringValidation{Field: "name", Value: req.Msg.GetName(), MaxLen: connectutil.MaxNameLength},
		); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		params.Name = req.Msg.Name
	}

	if req.Msg.GetConfig() != nil {
		config, err := json.Marshal(req.Msg.GetConfig().AsMap())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		raw := json.RawMessage(config)
		params.Config = &raw

		// Blocking connection check before persist.
		existingSource, err := h.getSource.Execute(ctx, pipelinesources.GetSourceParams{
			ID:          id,
			WorkspaceID: workspaceID,
		})
		if err != nil {
			return nil, mapError(err)
		}

		if err := runConnectionCheck(ctx, h.createCheckTask, h.waitForTaskResult,
			workspaceID, existingSource.ManagedConnectorID, raw); err != nil {
			return nil, err
		}
	}

	if req.Msg.RuntimeConfig != nil {
		params.RuntimeConfig = req.Msg.RuntimeConfig
	}

	src, err := h.updateSource.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	// Auto-trigger discover when config is updated.
	var discoverTaskID *string

	if req.Msg.GetConfig() != nil {
		taskResult, discoverErr := h.createDiscoverTask.Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{
			SourceID:    id,
			WorkspaceID: workspaceID,
		})
		if discoverErr == nil && taskResult != nil {
			tid := taskResult.TaskID.String()
			discoverTaskID = &tid
		}
	}

	return connect.NewResponse(&pipelinev1.UpdateSourceResponse{
		Source:         sourceToProto(src),
		DiscoverTaskId: discoverTaskID,
	}), nil
}

func (h *SourceHandler) DeleteSource(ctx context.Context, req *connect.Request[pipelinev1.DeleteSourceRequest]) (*connect.Response[pipelinev1.DeleteSourceResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.deleteSource.Execute(ctx, pipelinesources.DeleteSourceParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.DeleteSourceResponse{}), nil
}

func (h *SourceHandler) GetSource(ctx context.Context, req *connect.Request[pipelinev1.GetSourceRequest]) (*connect.Response[pipelinev1.GetSourceResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.GetId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	src, err := h.getSource.Execute(ctx, pipelinesources.GetSourceParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetSourceResponse{
		Source: sourceToProto(src),
	}), nil
}

func (h *SourceHandler) ListSources(ctx context.Context, req *connect.Request[pipelinev1.ListSourcesRequest]) (*connect.Response[pipelinev1.ListSourcesResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	sources, err := h.listSources.Execute(ctx, pipelinesources.ListSourcesParams{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoSources := make([]*pipelinev1.Source, len(sources))
	for i, s := range sources {
		protoSources[i] = sourceToProto(s)
	}

	paginated, total := paginateSlice(protoSources, req.Msg.GetPageSize(), req.Msg.GetOffset())

	return connect.NewResponse(&pipelinev1.ListSourcesResponse{
		Sources: paginated,
		Total:   total,
	}), nil
}

func (h *SourceHandler) TestSourceConnection(ctx context.Context, req *connect.Request[pipelinev1.TestSourceConnectionRequest]) (*connect.Response[pipelinev1.TestSourceConnectionResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := pipelinetasks.CreateCheckTaskParams{
		WorkspaceID: workspaceID,
	}

	if req.Msg.GetManagedConnectorId() != "" && req.Msg.GetConfig() != nil {
		// Direct config path: test without creating a source.
		mcID, err := uuid.Parse(req.Msg.GetManagedConnectorId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid managed_connector_id: %w", err))
		}

		configJSON, err := json.Marshal(req.Msg.GetConfig().AsMap())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid config: %w", err))
		}

		params.ManagedConnectorID = &mcID
		params.Config = configJSON
	} else if req.Msg.GetId() != "" {
		// Existing source path.
		id, err := uuid.Parse(req.Msg.GetId())
		if err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		params.SourceID = &id
	} else {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("either id or managed_connector_id+config must be provided"))
	}

	result, err := h.createCheckTask.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.TestSourceConnectionResponse{
		TaskId: result.TaskID.String(),
	}), nil
}

func (h *SourceHandler) DiscoverSourceSchema(ctx context.Context, req *connect.Request[pipelinev1.DiscoverSourceSchemaRequest]) (*connect.Response[pipelinev1.DiscoverSourceSchemaResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	sourceID, err := uuid.Parse(req.Msg.GetSourceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid source_id: %w", err))
	}

	result, err := h.createDiscoverTask.Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{
		SourceID:    sourceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.DiscoverSourceSchemaResponse{
		TaskId: result.TaskID.String(),
	}), nil
}

func (h *SourceHandler) GetSourceCatalog(ctx context.Context, req *connect.Request[pipelinev1.GetSourceCatalogRequest]) (*connect.Response[pipelinev1.GetSourceCatalogResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	sourceID, err := uuid.Parse(req.Msg.GetSourceId())
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	result, err := h.getSourceCatalog.Execute(ctx, pipelinecatalog.GetSourceCatalogParams{
		SourceID:    sourceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	pbCatalog := catalogToProto(result.Catalog)

	return connect.NewResponse(&pipelinev1.GetSourceCatalogResponse{
		Catalog:      pbCatalog,
		Version:      int32(result.Version),
		DiscoveredAt: timestamppb.New(result.DiscoveredAt),
	}), nil
}

func (h *SourceHandler) GetSystemInfo(ctx context.Context, req *connect.Request[pipelinev1.GetSystemInfoRequest]) (*connect.Response[pipelinev1.GetSystemInfoResponse], error) {
	return connect.NewResponse(&pipelinev1.GetSystemInfoResponse{
		RegistrationEnabled: bool(h.registrationEnabled),
		WorkspacesMode:      h.workspacesMode,
	}), nil
}

func sourceToProto(source *pipelineservice.Source) *pipelinev1.Source {
	var config *structpb.Struct

	if source.Config != "" {
		// Mask secret references before returning to API
		maskedConfig, err := pipelinesecrets.MaskConfigSecrets(source.Config)
		if err != nil {
			maskedConfig = source.Config
		}

		var m map[string]any
		if json.Unmarshal([]byte(maskedConfig), &m) == nil {
			config, _ = structpb.NewStruct(m)
		}
	}

	proto := &pipelinev1.Source{
		Id:                 source.ID.String(),
		WorkspaceId:        source.WorkspaceID.String(),
		Name:               source.Name,
		ManagedConnectorId: source.ManagedConnectorID.String(),
		Config:             config,
		CreatedAt:          timestamppb.New(source.CreatedAt),
		UpdatedAt:          timestamppb.New(source.UpdatedAt),
	}
	if source.RuntimeConfig != nil {
		proto.RuntimeConfig = source.RuntimeConfig
	}

	return proto
}
