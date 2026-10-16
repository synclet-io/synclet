package pipelinestate_test

import (
	"context"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/dbutil"
	"github.com/saturn4er/boilerplate-go/lib/filter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
)

type connStateStorage struct {
	*mockStorage
	connections *fakeConnections
}

func (s *connStateStorage) Connections() pipelineservice.ConnectionsStorage { return s.connections }

type fakeConnections struct {
	pipelineservice.ConnectionsStorage
	conn *pipelineservice.Connection
}

func (s *fakeConnections) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	if f.ID.(*filter.EqualsFilter[uuid.UUID]).Value != s.conn.ID ||
		f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]).Value != s.conn.WorkspaceID {
		return nil, pipelineservice.ErrConnectionNotFound
	}

	cp := *s.conn

	return &cp, nil
}

type recordingAudit struct {
	events []pipelineservice.AuditEvent
}

func (r *recordingAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	r.events = append(r.events, event)
}

func newUpdateStreamState(conn *pipelineservice.Connection, audit pipelineservice.AuditRecorder) (*pipelinestate.UpdateStreamState, *connStateStorage) {
	store := &connStateStorage{mockStorage: newMockStorage(), connections: &fakeConnections{conn: conn}}

	return pipelinestate.NewUpdateStreamState(
		pipelineconnections.NewGetConnection(store),
		pipelinestate.NewSaveSyncState(store),
		audit,
	), store
}

func TestUpdateStreamState_RecordsAudit(t *testing.T) {
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New(), Name: "pg -> bq"}

	for _, tc := range []struct {
		name      string
		namespace string
		want      string
	}{
		{name: "with namespace", namespace: "public", want: "public.users"},
		{name: "without namespace", want: "users"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			audit := &recordingAudit{}
			uc, store := newUpdateStreamState(conn, audit)

			err := uc.Execute(context.Background(), pipelinestate.UpdateStreamStateParams{
				ConnectionID:    conn.ID,
				WorkspaceID:     conn.WorkspaceID,
				StreamName:      "users",
				StreamNamespace: tc.namespace,
				StateData:       `{"cursor":"42"}`,
			})
			require.NoError(t, err)
			require.Contains(t, store.connStates.states, conn.ID)

			require.Len(t, audit.events, 1)
			event := audit.events[0]
			assert.Equal(t, conn.WorkspaceID, event.WorkspaceID)
			assert.Equal(t, "update", event.Action)
			assert.Equal(t, "connection", event.ResourceType)
			assert.Equal(t, conn.ID, event.ResourceID)
			assert.Equal(t, "pg -> bq", event.ResourceLabel)
			assert.Nil(t, event.Before)
			assert.Equal(t, map[string]any{"stream_state_set": tc.want}, event.After)
		})
	}
}

func TestUpdateStreamState_NoAuditOnFailure(t *testing.T) {
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New(), Name: "pg -> bq"}

	t.Run("invalid json", func(t *testing.T) {
		audit := &recordingAudit{}
		uc, _ := newUpdateStreamState(conn, audit)

		err := uc.Execute(context.Background(), pipelinestate.UpdateStreamStateParams{
			ConnectionID: conn.ID,
			WorkspaceID:  conn.WorkspaceID,
			StreamName:   "users",
			StateData:    `{not json`,
		})
		require.ErrorIs(t, err, pipelineservice.ErrStateDataInvalidJSON)
		assert.Empty(t, audit.events)
	})

	t.Run("foreign workspace", func(t *testing.T) {
		audit := &recordingAudit{}
		uc, _ := newUpdateStreamState(conn, audit)

		err := uc.Execute(context.Background(), pipelinestate.UpdateStreamStateParams{
			ConnectionID: conn.ID,
			WorkspaceID:  uuid.New(),
			StreamName:   "users",
			StateData:    `{}`,
		})
		require.Error(t, err)
		assert.Empty(t, audit.events)
	})
}
