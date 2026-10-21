package pipelineconnect

import (
	"context"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/internal/pipelinefake"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

func TestConfigureStreamsRejectsEmptyList(t *testing.T) {
	store := pipelinefake.New()
	workspaceID := uuid.New()
	ctx := connectutil.ContextWithWorkspaceID(context.Background(), workspaceID)

	connID := uuid.New()
	store.ConnectionsTable.Insert(&pipelineservice.Connection{ID: connID, WorkspaceID: workspaceID, SourceID: uuid.New(), Name: "pg-to-bq"})

	handler := &ConnectionHandler{
		getConnection:    pipelineconnections.NewGetConnection(store),
		configureStreams: pipelinecatalog.NewConfigureStreams(store, pipelineservice.NoopAuditRecorder{}),
	}

	_, err := handler.ConfigureStreams(ctx, connect.NewRequest(&pipelinev1.ConfigureStreamsRequest{ConnectionId: connID.String()}))

	assert.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
	assert.Contains(t, err.Error(), "at least one stream")
	assert.Empty(t, store.ConfiguredCatalogsTable.All())
}
