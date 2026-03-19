package pipelineconnect

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"
	"google.golang.org/protobuf/types/known/timestamppb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconfig"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesettings"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/connectutil"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ConnectionHandler implements the ConnectionService ConnectRPC handler.
type ConnectionHandler struct {
	pipelinev1connect.UnimplementedConnectionServiceHandler

	// Connection use cases
	createConnection *pipelineconnections.CreateConnection
	updateConnection *pipelineconnections.UpdateConnection
	deleteConnection *pipelineconnections.DeleteConnection
	getConnection    *pipelineconnections.GetConnection
	listConnections  *pipelineconnections.ListConnections

	// Composite connection use cases (with workspace verification)
	enableConnection  *pipelineconnections.EnableConnection
	disableConnection *pipelineconnections.DisableConnection

	// Async task use cases
	createDiscoverTask *pipelinetasks.CreateDiscoverTask

	// Composite catalog use cases
	getDiscoveredCatalogForConnection *pipelinecatalog.GetDiscoveredCatalogForConnection
	configureStreams                  *pipelinecatalog.ConfigureStreams
	getConfiguredCatalog              *pipelinecatalog.GetConfiguredCatalog
	detectSchemaChanges               *pipelinecatalog.DetectSchemaChanges

	// Composite state use cases (with workspace verification)
	resetStreamState     *pipelinestate.ResetStreamState
	resetConnectionState *pipelinestate.ResetConnectionState
	updateStreamState    *pipelinestate.UpdateStreamState
	listStreamStates     *pipelinestate.ListStreamStates

	// Config import/export use cases
	exportConfig *pipelineconfig.ExportConfig
	importConfig *pipelineconfig.ImportConfig

	// Settings use cases
	getWorkspaceSettings    *pipelinesettings.GetWorkspaceSettings
	updateWorkspaceSettings *pipelinesettings.UpdateWorkspaceSettings
}

// NewConnectionHandler creates a new connection handler.
func NewConnectionHandler(
	createConnection *pipelineconnections.CreateConnection,
	updateConnection *pipelineconnections.UpdateConnection,
	deleteConnection *pipelineconnections.DeleteConnection,
	getConnection *pipelineconnections.GetConnection,
	listConnections *pipelineconnections.ListConnections,
	enableConnection *pipelineconnections.EnableConnection,
	disableConnection *pipelineconnections.DisableConnection,
	createDiscoverTask *pipelinetasks.CreateDiscoverTask,
	getDiscoveredCatalogForConnection *pipelinecatalog.GetDiscoveredCatalogForConnection,
	configureStreams *pipelinecatalog.ConfigureStreams,
	getConfiguredCatalog *pipelinecatalog.GetConfiguredCatalog,
	detectSchemaChanges *pipelinecatalog.DetectSchemaChanges,
	resetStreamState *pipelinestate.ResetStreamState,
	resetConnectionState *pipelinestate.ResetConnectionState,
	updateStreamState *pipelinestate.UpdateStreamState,
	listStreamStates *pipelinestate.ListStreamStates,
	exportConfig *pipelineconfig.ExportConfig,
	importConfig *pipelineconfig.ImportConfig,
	getWorkspaceSettings *pipelinesettings.GetWorkspaceSettings,
	updateWorkspaceSettings *pipelinesettings.UpdateWorkspaceSettings,
) *ConnectionHandler {
	return &ConnectionHandler{
		createConnection:                  createConnection,
		updateConnection:                  updateConnection,
		deleteConnection:                  deleteConnection,
		getConnection:                     getConnection,
		listConnections:                   listConnections,
		enableConnection:                  enableConnection,
		disableConnection:                 disableConnection,
		createDiscoverTask:                createDiscoverTask,
		getDiscoveredCatalogForConnection: getDiscoveredCatalogForConnection,
		configureStreams:                  configureStreams,
		getConfiguredCatalog:              getConfiguredCatalog,
		detectSchemaChanges:               detectSchemaChanges,
		resetStreamState:                  resetStreamState,
		resetConnectionState:              resetConnectionState,
		updateStreamState:                 updateStreamState,
		listStreamStates:                  listStreamStates,
		exportConfig:                      exportConfig,
		importConfig:                      importConfig,
		getWorkspaceSettings:              getWorkspaceSettings,
		updateWorkspaceSettings:           updateWorkspaceSettings,
	}
}

func (h *ConnectionHandler) CreateConnection(ctx context.Context, req *connect.Request[pipelinev1.CreateConnectionRequest]) (*connect.Response[pipelinev1.CreateConnectionResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if err := connectutil.ValidateStringLengths(
		connectutil.StringValidation{Field: "name", Value: req.Msg.Name, MaxLen: connectutil.MaxNameLength},
	); err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}
	if req.Msg.Schedule != nil {
		if err := connectutil.ValidateStringLengths(
			connectutil.StringValidation{Field: "schedule", Value: *req.Msg.Schedule, MaxLen: 128},
		); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
	}

	var schedule *string
	if req.Msg.Schedule != nil {
		s := *req.Msg.Schedule
		schedule = &s
	}

	sourceID, err := uuid.Parse(req.Msg.SourceId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid source_id: %w", err))
	}

	destinationID, err := uuid.Parse(req.Msg.DestinationId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid destination_id: %w", err))
	}

	conn, err := h.createConnection.Execute(ctx, pipelineconnections.CreateConnectionParams{
		WorkspaceID:           workspaceID,
		Name:                  req.Msg.Name,
		SourceID:              sourceID,
		DestinationID:         destinationID,
		Schedule:              schedule,
		SchemaChangePolicy:    protoToSchemaChangePolicy(req.Msg.SchemaChangePolicy),
		MaxAttempts:           int(req.Msg.MaxAttempts),
		NamespaceDefinition:   protoToNamespaceDefinition(req.Msg.NamespaceDefinition),
		CustomNamespaceFormat: stringPtrFromProto(req.Msg.CustomNamespaceFormat),
		StreamPrefix:          stringPtrFromProto(req.Msg.StreamPrefix),
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.CreateConnectionResponse{
		Connection: connectionToProto(conn),
	}), nil
}

func (h *ConnectionHandler) UpdateConnection(ctx context.Context, req *connect.Request[pipelinev1.UpdateConnectionRequest]) (*connect.Response[pipelinev1.UpdateConnectionResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	params := pipelineconnections.UpdateConnectionParams{
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
	if req.Msg.Schedule != nil {
		if err := connectutil.ValidateStringLengths(
			connectutil.StringValidation{Field: "schedule", Value: *req.Msg.Schedule, MaxLen: 128},
		); err != nil {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}
		params.Schedule = &req.Msg.Schedule
	}
	if req.Msg.SchemaChangePolicy != nil {
		v := protoToSchemaChangePolicy(*req.Msg.SchemaChangePolicy)
		params.SchemaChangePolicy = &v
	}
	if req.Msg.MaxAttempts != nil {
		v := int(*req.Msg.MaxAttempts)
		params.MaxAttempts = &v
	}
	if req.Msg.NamespaceDefinition != nil {
		v := protoToNamespaceDefinition(*req.Msg.NamespaceDefinition)
		params.NamespaceDefinition = &v
	}
	if req.Msg.CustomNamespaceFormat != nil {
		v := stringPtrFromProto(*req.Msg.CustomNamespaceFormat)
		params.CustomNamespaceFormat = &v
	}
	if req.Msg.StreamPrefix != nil {
		v := stringPtrFromProto(*req.Msg.StreamPrefix)
		params.StreamPrefix = &v
	}

	conn, err := h.updateConnection.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.UpdateConnectionResponse{
		Connection: connectionToProto(conn),
	}), nil
}

func (h *ConnectionHandler) DeleteConnection(ctx context.Context, req *connect.Request[pipelinev1.DeleteConnectionRequest]) (*connect.Response[pipelinev1.DeleteConnectionResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if err := h.deleteConnection.Execute(ctx, pipelineconnections.DeleteConnectionParams{
		ID:          id,
		WorkspaceID: workspaceID,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.DeleteConnectionResponse{}), nil
}

func (h *ConnectionHandler) GetConnection(ctx context.Context, req *connect.Request[pipelinev1.GetConnectionRequest]) (*connect.Response[pipelinev1.GetConnectionResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	conn, err := h.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          id,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetConnectionResponse{
		Connection: connectionToProto(conn),
	}), nil
}

func (h *ConnectionHandler) ListConnections(ctx context.Context, req *connect.Request[pipelinev1.ListConnectionsRequest]) (*connect.Response[pipelinev1.ListConnectionsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	conns, err := h.listConnections.Execute(ctx, pipelineconnections.ListConnectionsParams{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoConns := make([]*pipelinev1.Connection, len(conns))
	for i, c := range conns {
		protoConns[i] = connectionToProto(c)
	}

	paginated, total := paginateSlice(protoConns, req.Msg.PageSize, req.Msg.Offset)

	return connect.NewResponse(&pipelinev1.ListConnectionsResponse{
		Connections: paginated,
		Total:       total,
	}), nil
}

func (h *ConnectionHandler) EnableConnection(ctx context.Context, req *connect.Request[pipelinev1.EnableConnectionRequest]) (*connect.Response[pipelinev1.EnableConnectionResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("workspace context required"))
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	conn, err := h.enableConnection.Execute(ctx, pipelineconnections.EnableConnectionParams{
		ConnectionID: id,
		WorkspaceID:  wsID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.EnableConnectionResponse{
		Connection: connectionToProto(conn),
	}), nil
}

func (h *ConnectionHandler) DisableConnection(ctx context.Context, req *connect.Request[pipelinev1.DisableConnectionRequest]) (*connect.Response[pipelinev1.DisableConnectionResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("workspace context required"))
	}

	id, err := uuid.Parse(req.Msg.Id)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	conn, err := h.disableConnection.Execute(ctx, pipelineconnections.DisableConnectionParams{
		ConnectionID: id,
		WorkspaceID:  wsID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.DisableConnectionResponse{
		Connection: connectionToProto(conn),
	}), nil
}

func (h *ConnectionHandler) DiscoverSchema(ctx context.Context, req *connect.Request[pipelinev1.DiscoverSchemaRequest]) (*connect.Response[pipelinev1.DiscoverSchemaResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	// Resolve connection to get source ID
	conn, err := h.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{
		ID:          connID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	// Create async discover task for the connection's source
	result, err := h.createDiscoverTask.Execute(ctx, pipelinetasks.CreateDiscoverTaskParams{
		SourceID:    conn.SourceID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.DiscoverSchemaResponse{
		TaskId: result.TaskID.String(),
	}), nil
}

func (h *ConnectionHandler) ConfigureStreams(ctx context.Context, req *connect.Request[pipelinev1.ConfigureStreamsRequest]) (*connect.Response[pipelinev1.ConfigureStreamsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
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

	streams := make([]protocol.ConfiguredAirbyteStream, len(req.Msg.Streams))
	for i, s := range req.Msg.Streams {
		pk := make([][]string, len(s.PrimaryKey))
		for j, ck := range s.PrimaryKey {
			pk[j] = ck.FieldPath
		}

		sf := make([]protocol.SelectedField, len(s.SelectedFields))
		for j, f := range s.SelectedFields {
			sf[j] = protocol.SelectedField{FieldPath: f.FieldPath}
		}

		streams[i] = protocol.ConfiguredAirbyteStream{
			Stream: protocol.AirbyteStream{
				Name:      s.StreamName,
				Namespace: s.Namespace,
			},
			SyncMode:            protoToSyncMode(s.SyncMode),
			DestinationSyncMode: protoToDestinationSyncMode(s.DestinationSyncMode),
			CursorField:         s.CursorField,
			PrimaryKey:          pk,
			SelectedFields:      sf,
		}
	}

	if err := h.configureStreams.Execute(ctx, pipelinecatalog.ConfigureStreamsParams{
		ConnectionID: connID,
		SourceID:     conn.SourceID,
		Streams:      streams,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.ConfigureStreamsResponse{}), nil
}

func (h *ConnectionHandler) GetDiscoveredCatalog(ctx context.Context, req *connect.Request[pipelinev1.GetDiscoveredCatalogRequest]) (*connect.Response[pipelinev1.GetDiscoveredCatalogResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	catalog, err := h.getDiscoveredCatalogForConnection.Execute(ctx, pipelinecatalog.GetDiscoveredCatalogForConnectionParams{
		ConnectionID: connID,
		WorkspaceID:  workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	pbCatalog := catalogToProto(catalog)

	return connect.NewResponse(&pipelinev1.GetDiscoveredCatalogResponse{
		Catalog: pbCatalog,
	}), nil
}

func (h *ConnectionHandler) GetConfiguredCatalog(ctx context.Context, req *connect.Request[pipelinev1.GetConfiguredCatalogRequest]) (*connect.Response[pipelinev1.GetConfiguredCatalogResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("workspace context required"))
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	if _, err := h.getConnection.Execute(ctx, pipelineconnections.GetConnectionParams{ID: connID, WorkspaceID: wsID}); err != nil {
		return nil, mapError(err)
	}

	configuredCatalog, err := h.getConfiguredCatalog.Execute(ctx, pipelinecatalog.GetConfiguredCatalogParams{
		ConnectionID: connID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	catalogJSON, err := json.Marshal(configuredCatalog)
	if err != nil {
		return nil, mapError(err)
	}

	var catalogMap map[string]any
	if err := json.Unmarshal(catalogJSON, &catalogMap); err != nil {
		return nil, mapError(err)
	}

	pbStruct, err := structpb.NewStruct(catalogMap)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetConfiguredCatalogResponse{
		Catalog: pbStruct,
	}), nil
}

func (h *ConnectionHandler) GetSchemaChanges(ctx context.Context, req *connect.Request[pipelinev1.GetSchemaChangesRequest]) (*connect.Response[pipelinev1.GetSchemaChangesResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
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

	changes, err := h.detectSchemaChanges.Execute(ctx, pipelinecatalog.DetectSchemaChangesParams{
		ConnectionID: connID,
		SourceID:     conn.SourceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoChanges := make([]*pipelinev1.SchemaChange, len(changes))
	for i, c := range changes {
		protoChanges[i] = &pipelinev1.SchemaChange{
			Type:       string(c.Type),
			StreamName: c.StreamName,
			Namespace:  c.Namespace,
			ColumnName: c.ColumnName,
			OldType:    c.OldType,
			NewType:    c.NewType,
		}
	}

	return connect.NewResponse(&pipelinev1.GetSchemaChangesResponse{
		Changes: protoChanges,
	}), nil
}

func (h *ConnectionHandler) ResetStreamState(ctx context.Context, req *connect.Request[pipelinev1.ResetStreamStateRequest]) (*connect.Response[pipelinev1.ResetStreamStateResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("workspace context required"))
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
	}

	if req.Msg.StreamName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("stream_name is required"))
	}

	if err := h.resetStreamState.Execute(ctx, pipelinestate.ResetStreamStateParams{
		ConnectionID:    connID,
		WorkspaceID:     wsID,
		StreamName:      req.Msg.StreamName,
		StreamNamespace: req.Msg.StreamNamespace,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.ResetStreamStateResponse{}), nil
}

func (h *ConnectionHandler) ResetConnectionState(ctx context.Context, req *connect.Request[pipelinev1.ResetConnectionStateRequest]) (*connect.Response[pipelinev1.ResetConnectionStateResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("workspace context required"))
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
	}

	if err := h.resetConnectionState.Execute(ctx, pipelinestate.ResetConnectionStateParams{
		ConnectionID: connID,
		WorkspaceID:  wsID,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.ResetConnectionStateResponse{}), nil
}

func (h *ConnectionHandler) ListStreamStates(ctx context.Context, req *connect.Request[pipelinev1.ListStreamStatesRequest]) (*connect.Response[pipelinev1.ListStreamStatesResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("workspace context required"))
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
	}

	result, err := h.listStreamStates.Execute(ctx, pipelinestate.ListStreamStatesParams{
		ConnectionID: connID,
		WorkspaceID:  wsID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	protoStates := make([]*pipelinev1.StreamState, len(result.StreamStates))
	for i, s := range result.StreamStates {
		protoStates[i] = &pipelinev1.StreamState{
			StreamName:      s.StreamName,
			StreamNamespace: s.StreamNamespace,
			StateData:       string(s.StateData),
		}
	}

	return connect.NewResponse(&pipelinev1.ListStreamStatesResponse{
		States:    protoStates,
		StateType: result.StateType,
	}), nil
}

func (h *ConnectionHandler) UpdateStreamState(ctx context.Context, req *connect.Request[pipelinev1.UpdateStreamStateRequest]) (*connect.Response[pipelinev1.UpdateStreamStateResponse], error) {
	wsID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, fmt.Errorf("workspace context required"))
	}

	connID, err := uuid.Parse(req.Msg.ConnectionId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid connection_id: %w", err))
	}

	if req.Msg.StreamName == "" {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("stream_name is required"))
	}

	if err := h.updateStreamState.Execute(ctx, pipelinestate.UpdateStreamStateParams{
		ConnectionID:    connID,
		WorkspaceID:     wsID,
		StreamName:      req.Msg.StreamName,
		StreamNamespace: req.Msg.StreamNamespace,
		StateData:       req.Msg.StateData,
	}); err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.UpdateStreamStateResponse{}), nil
}

func (h *ConnectionHandler) ExportWorkspaceConfig(ctx context.Context, req *connect.Request[pipelinev1.ExportWorkspaceConfigRequest]) (*connect.Response[pipelinev1.ExportWorkspaceConfigResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	yamlData, err := h.exportConfig.Execute(ctx, pipelineconfig.ExportConfigParams{
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.ExportWorkspaceConfigResponse{
		ConfigYaml: yamlData,
	}), nil
}

func (h *ConnectionHandler) ImportWorkspaceConfig(ctx context.Context, req *connect.Request[pipelinev1.ImportWorkspaceConfigRequest]) (*connect.Response[pipelinev1.ImportWorkspaceConfigResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	if len(req.Msg.ConfigYaml) == 0 {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("config_yaml is required"))
	}

	result, err := h.importConfig.Execute(ctx, pipelineconfig.ImportConfigParams{
		WorkspaceID: workspaceID,
		ConfigYAML:  req.Msg.ConfigYaml,
	})
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&pipelinev1.ImportWorkspaceConfigResponse{
		Created: int32(result.Created),
		Updated: int32(result.Updated),
		Errors:  result.Errors,
	}), nil
}

func (h *ConnectionHandler) GetPipelineSettings(
	ctx context.Context,
	req *connect.Request[pipelinev1.GetPipelineSettingsRequest],
) (*connect.Response[pipelinev1.GetPipelineSettingsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	settings, err := h.getWorkspaceSettings.Execute(ctx, workspaceID)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.GetPipelineSettingsResponse{
		Settings: &pipelinev1.PipelineSettings{
			MaxJobsPerWorkspace: int32(settings.MaxJobsPerWorkspace),
		},
	}), nil
}

func (h *ConnectionHandler) UpdatePipelineSettings(
	ctx context.Context,
	req *connect.Request[pipelinev1.UpdatePipelineSettingsRequest],
) (*connect.Response[pipelinev1.UpdatePipelineSettingsResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	params := pipelinesettings.UpdateWorkspaceSettingsParams{
		WorkspaceID: workspaceID,
	}
	if req.Msg.MaxJobsPerWorkspace != nil {
		v := int(*req.Msg.MaxJobsPerWorkspace)
		params.MaxJobsPerWorkspace = &v
	}

	settings, err := h.updateWorkspaceSettings.Execute(ctx, params)
	if err != nil {
		return nil, mapError(err)
	}

	return connect.NewResponse(&pipelinev1.UpdatePipelineSettingsResponse{
		Settings: &pipelinev1.PipelineSettings{
			MaxJobsPerWorkspace: int32(settings.MaxJobsPerWorkspace),
		},
	}), nil
}

func connectionToProto(c *pipelineservice.Connection) *pipelinev1.Connection {
	schedule := ""
	if c.Schedule != nil {
		schedule = *c.Schedule
	}

	return &pipelinev1.Connection{
		Id:                    c.ID.String(),
		WorkspaceId:           c.WorkspaceID.String(),
		Name:                  c.Name,
		Status:                connectionStatusToProto(c.Status),
		SourceId:              c.SourceID.String(),
		DestinationId:         c.DestinationID.String(),
		Schedule:              schedule,
		SchemaChangePolicy:    schemaChangePolicyToProto(c.SchemaChangePolicy),
		CreatedAt:             timestamppb.New(c.CreatedAt),
		UpdatedAt:             timestamppb.New(c.UpdatedAt),
		MaxAttempts:           int32(c.MaxAttempts),
		NamespaceDefinition:   namespaceDefinitionToProto(c.NamespaceDefinition),
		CustomNamespaceFormat: stringPtrOrEmpty(c.CustomNamespaceFormat),
		StreamPrefix:          stringPtrOrEmpty(c.StreamPrefix),
		NextScheduledAt:       timestampPtrOrNil(c.NextScheduledAt),
	}
}

func stringPtrOrEmpty(s *string) string {
	if s == nil {
		return ""
	}

	return *s
}

func timestampPtrOrNil(t *time.Time) *timestamppb.Timestamp {
	if t == nil {
		return nil
	}
	return timestamppb.New(*t)
}

func stringPtrFromProto(s string) *string {
	if s == "" {
		return nil
	}

	return &s
}
