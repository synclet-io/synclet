package pipelineconnect

import (
	"context"
	"encoding/json"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/internal/pipelinefake"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/pkg/connectutil"
	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestValidateMappersSourceResolution(t *testing.T) {
	store := pipelinefake.New()
	workspaceID := uuid.New()
	ctx := connectutil.ContextWithWorkspaceID(context.Background(), workspaceID)

	sourceID := uuid.New()
	store.SourcesTable.Insert(&pipelineservice.Source{ID: sourceID, WorkspaceID: workspaceID, Name: "pg"})

	foreignSourceID := uuid.New()
	store.SourcesTable.Insert(&pipelineservice.Source{ID: foreignSourceID, WorkspaceID: uuid.New(), Name: "other"})

	catalog, err := json.Marshal(protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{{
		Namespace:  "public",
		Name:       "users",
		JSONSchema: json.RawMessage(`{"type":"object","properties":{"id":{"type":"integer"},"name":{"type":"string"}}}`),
	}}})
	require.NoError(t, err)

	for _, id := range []uuid.UUID{sourceID, foreignSourceID} {
		store.CatalogDiscoveriesTable.Insert(&pipelineservice.CatalogDiscovery{ID: uuid.New(), SourceID: id, Version: 1, CatalogJSON: string(catalog)})
	}

	connID := uuid.New()
	store.ConnectionsTable.Insert(&pipelineservice.Connection{ID: connID, WorkspaceID: workspaceID, SourceID: sourceID, Name: "pg-to-bq"})

	handler := &ConnectionHandler{
		validateMappers: pipelinecatalog.NewValidateMappers(store),
	}

	step, err := structpb.NewStruct(map[string]any{"set": map[string]any{"conn": "sync.connection"}})
	require.NoError(t, err)

	sample, err := structpb.NewStruct(map[string]any{"id": 1, "name": "Ann"})
	require.NoError(t, err)

	request := func(mutate func(*pipelinev1.ValidateMappersRequest)) *connect.Request[pipelinev1.ValidateMappersRequest] {
		msg := &pipelinev1.ValidateMappersRequest{
			Namespace:    "public",
			StreamName:   "users",
			Mappers:      []*structpb.Struct{step},
			SampleRecord: sample,
		}
		mutate(msg)

		return connect.NewRequest(msg)
	}

	byConnection, err := handler.ValidateMappers(ctx, request(func(m *pipelinev1.ValidateMappersRequest) {
		m.ConnectionId = connID.String()
	}))
	require.NoError(t, err)

	bySource, err := handler.ValidateMappers(ctx, request(func(m *pipelinev1.ValidateMappersRequest) {
		m.SourceId = sourceID.String()
		m.ConnectionName = "pg-to-bq"
	}))
	require.NoError(t, err, "a source of the caller's workspace with a connection name validates")

	assert.Empty(t, bySource.Msg.GetErrors())
	assert.Equal(t, "pg-to-bq", bySource.Msg.GetSampleOutput().GetFields()["conn"].GetStringValue(), "the connection name reaches the run")
	assert.Equal(t, byConnection.Msg.GetSampleOutput().AsMap(), bySource.Msg.GetSampleOutput().AsMap(), "both ways validate the same")
	assert.Equal(t, byConnection.Msg.GetOutputSchema().AsMap(), bySource.Msg.GetOutputSchema().AsMap())

	_, err = handler.ValidateMappers(ctx, request(func(m *pipelinev1.ValidateMappersRequest) {
		m.SourceId = foreignSourceID.String()
		m.ConnectionName = "pg-to-bq"
	}))
	assert.Equal(t, connect.CodeNotFound, connect.CodeOf(err), "a source of another workspace is not found")

	_, err = handler.ValidateMappers(ctx, request(func(*pipelinev1.ValidateMappersRequest) {}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "neither a connection nor a source is rejected")

	_, err = handler.ValidateMappers(ctx, request(func(m *pipelinev1.ValidateMappersRequest) {
		m.ConnectionId = connID.String()
		m.SourceId = sourceID.String()
		m.ConnectionName = "pg-to-bq"
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "both a connection and a source are rejected")

	_, err = handler.ValidateMappers(ctx, request(func(m *pipelinev1.ValidateMappersRequest) {
		m.SourceId = sourceID.String()
	}))
	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err), "a source without a connection name is rejected")
}
