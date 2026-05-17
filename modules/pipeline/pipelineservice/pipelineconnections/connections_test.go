package pipelineconnections_test

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
)

// Find filter helper.
func (s *createConnConnectionsStorage) First(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) (*pipelineservice.Connection, error) {
	for _, conn := range s.records {
		if f.ID != nil {
			if id, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.ID != id.Value {
				continue
			}
		}

		if f.WorkspaceID != nil {
			if wid, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.WorkspaceID != wid.Value {
				continue
			}
		}

		cp := *conn

		return &cp, nil
	}

	return nil, pipelineservice.ErrConnectionNotFound
}

func (s *createConnConnectionsStorage) Find(_ context.Context, f *pipelineservice.ConnectionFilter, _ ...optionutil.Option[dbutil.SelectOptions]) ([]*pipelineservice.Connection, error) {
	out := make([]*pipelineservice.Connection, 0, len(s.records))

	for _, conn := range s.records {
		if f.WorkspaceID != nil {
			if wid, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.WorkspaceID != wid.Value {
				continue
			}
		}

		cp := *conn
		out = append(out, &cp)
	}

	return out, nil
}

func (s *createConnConnectionsStorage) Update(_ context.Context, conn *pipelineservice.Connection) (*pipelineservice.Connection, error) {
	cp := *conn
	s.records[conn.ID] = &cp

	return &cp, nil
}

func (s *createConnConnectionsStorage) Delete(_ context.Context, f *pipelineservice.ConnectionFilter) error {
	for id, conn := range s.records {
		if f.ID != nil {
			if eq, ok := f.ID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.ID != eq.Value {
				continue
			}
		}

		if f.WorkspaceID != nil {
			if eq, ok := f.WorkspaceID.(*filter.EqualsFilter[uuid.UUID]); ok && conn.WorkspaceID != eq.Value {
				continue
			}
		}

		delete(s.records, id)
	}

	return nil
}

func seedConnection(t *testing.T, store *createConnStore, conn *pipelineservice.Connection) {
	t.Helper()

	store.connections.records[conn.ID] = conn
}

func TestGetConnection(t *testing.T) {
	t.Run("returns connection scoped to the workspace", func(t *testing.T) {
		workspaceID := uuid.New()
		sourceID := uuid.New()
		destID := uuid.New()
		store := newCreateConnStore(workspaceID, sourceID, destID)

		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: workspaceID, Name: "wanted"}
		seedConnection(t, store, conn)

		uc := pipelineconnections.NewGetConnection(store)
		got, err := uc.Execute(context.Background(), pipelineconnections.GetConnectionParams{
			ID:          conn.ID,
			WorkspaceID: workspaceID,
		})
		require.NoError(t, err)
		assert.Equal(t, "wanted", got.Name)
	})

	t.Run("returns not-found when the connection belongs to a different workspace", func(t *testing.T) {
		workspaceID := uuid.New()
		otherWorkspace := uuid.New()
		store := newCreateConnStore(workspaceID, uuid.New(), uuid.New())

		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: otherWorkspace, Name: "isolated"}
		seedConnection(t, store, conn)

		uc := pipelineconnections.NewGetConnection(store)
		_, err := uc.Execute(context.Background(), pipelineconnections.GetConnectionParams{
			ID:          conn.ID,
			WorkspaceID: workspaceID,
		})
		require.Error(t, err)
	})
}

func TestListConnections(t *testing.T) {
	t.Run("returns only connections within the workspace", func(t *testing.T) {
		ws1 := uuid.New()
		ws2 := uuid.New()
		store := newCreateConnStore(ws1, uuid.New(), uuid.New())
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws1, Name: "a"})
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws1, Name: "b"})
		seedConnection(t, store, &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws2, Name: "other"})

		uc := pipelineconnections.NewListConnections(store)
		got, err := uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws1})
		require.NoError(t, err)
		assert.Len(t, got, 2)

		for _, c := range got {
			assert.Equal(t, ws1, c.WorkspaceID)
		}
	})

	t.Run("returns empty slice when workspace has no connections", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())

		uc := pipelineconnections.NewListConnections(store)
		got, err := uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws})
		require.NoError(t, err)
		assert.Empty(t, got)
	})
}

func TestDeleteConnection(t *testing.T) {
	t.Run("deletes a connection within the workspace", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws}
		seedConnection(t, store, conn)

		uc := pipelineconnections.NewDeleteConnection(store)
		err := uc.Execute(context.Background(), pipelineconnections.DeleteConnectionParams{
			ID:          conn.ID,
			WorkspaceID: ws,
		})
		require.NoError(t, err)

		_, exists := store.connections.records[conn.ID]
		assert.False(t, exists)
	})

	t.Run("does not delete a connection that belongs to another workspace", func(t *testing.T) {
		ws := uuid.New()
		otherWs := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: otherWs}
		seedConnection(t, store, conn)

		uc := pipelineconnections.NewDeleteConnection(store)
		err := uc.Execute(context.Background(), pipelineconnections.DeleteConnectionParams{
			ID:          conn.ID,
			WorkspaceID: ws,
		})
		require.NoError(t, err)

		_, exists := store.connections.records[conn.ID]
		assert.True(t, exists, "delete must be workspace-scoped")
	})
}

func TestUpdateConnection(t *testing.T) {
	t.Run("updates only the provided fields", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		original := &pipelineservice.Connection{
			ID:                 uuid.New(),
			WorkspaceID:        ws,
			Name:               "old",
			SchemaChangePolicy: pipelineservice.SchemaChangePolicyPause,
			MaxAttempts:        3,
		}
		seedConnection(t, store, original)

		newName := "new"
		newAttempts := 7
		uc := pipelineconnections.NewUpdateConnection(store)
		updated, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID:          original.ID,
			WorkspaceID: ws,
			Name:        &newName,
			MaxAttempts: &newAttempts,
		})
		require.NoError(t, err)
		assert.Equal(t, "new", updated.Name)
		assert.Equal(t, 7, updated.MaxAttempts)
		// Untouched field stays the same.
		assert.Equal(t, pipelineservice.SchemaChangePolicyPause, updated.SchemaChangePolicy)
	})

	t.Run("rejects an invalid cron schedule", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws}
		seedConnection(t, store, conn)

		invalid := "every blue moon"
		invalidPtr := &invalid
		uc := pipelineconnections.NewUpdateConnection(store)
		_, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID:          conn.ID,
			WorkspaceID: ws,
			Schedule:    &invalidPtr,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("returns error when connection is in a different workspace", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New()}
		seedConnection(t, store, conn)

		newName := "x"
		uc := pipelineconnections.NewUpdateConnection(store)
		_, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID:          conn.ID,
			WorkspaceID: ws,
			Name:        &newName,
		})
		require.Error(t, err)
	})
}

func TestEnableDisableConnection(t *testing.T) {
	ws := uuid.New()

	t.Run("enable transitions inactive to active", func(t *testing.T) {
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Status: pipelineservice.ConnectionStatusInactive}
		seedConnection(t, store, conn)

		getter := pipelineconnections.NewGetConnection(store)
		statusUC := pipelineconnections.NewUpdateConnectionStatus(store)
		uc := pipelineconnections.NewEnableConnection(getter, statusUC)
		out, err := uc.Execute(context.Background(), pipelineconnections.EnableConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  ws,
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectionStatusActive, out.Status)
	})

	t.Run("disable transitions active to inactive", func(t *testing.T) {
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Status: pipelineservice.ConnectionStatusActive}
		seedConnection(t, store, conn)

		getter := pipelineconnections.NewGetConnection(store)
		statusUC := pipelineconnections.NewUpdateConnectionStatus(store)
		uc := pipelineconnections.NewDisableConnection(getter, statusUC)
		out, err := uc.Execute(context.Background(), pipelineconnections.DisableConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  ws,
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectionStatusInactive, out.Status)
	})

	t.Run("enable rejects connection from a different workspace", func(t *testing.T) {
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New(), Status: pipelineservice.ConnectionStatusInactive}
		seedConnection(t, store, conn)

		getter := pipelineconnections.NewGetConnection(store)
		statusUC := pipelineconnections.NewUpdateConnectionStatus(store)
		uc := pipelineconnections.NewEnableConnection(getter, statusUC)
		_, err := uc.Execute(context.Background(), pipelineconnections.EnableConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  ws,
		})
		require.Error(t, err)
	})
}
