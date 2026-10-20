package pipelineconnections_test

import (
	"context"
	"fmt"
	"slices"
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
		if !matchesUUID(f.WorkspaceID, conn.WorkspaceID) ||
			!matchesUUID(f.SourceID, conn.SourceID) ||
			!matchesUUID(f.DestinationID, conn.DestinationID) ||
			!matchesTags(f.Tags, conn.Tags) {
			continue
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

func matchesUUID(f filter.Filter[uuid.UUID], value uuid.UUID) bool {
	switch typed := f.(type) {
	case nil:
		return true
	case *filter.EqualsFilter[uuid.UUID]:
		return typed.Value == value
	case *filter.InFilter[uuid.UUID]:
		return slices.Contains(typed.Values, value)
	default:
		panic(fmt.Sprintf("unsupported filter %T", f))
	}
}

func matchesTags(f filter.ArrayFilter[string], tags []string) bool {
	switch typed := f.(type) {
	case nil:
		return true
	case *filter.ArrayContainsFilter[string]:
		for _, want := range typed.Values {
			if !slices.Contains(tags, want) {
				return false
			}
		}

		return true
	default:
		panic(fmt.Sprintf("unsupported filter %T", f))
	}
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

	t.Run("filters by source and destination within the workspace", func(t *testing.T) {
		ws := uuid.New()
		otherWs := uuid.New()
		srcA, srcB := uuid.New(), uuid.New()
		dstA, dstB := uuid.New(), uuid.New()
		store := newCreateConnStore(ws, srcA, dstA)

		aa := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcA, DestinationID: dstA}
		ab := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcA, DestinationID: dstB}
		ba := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcB, DestinationID: dstA}
		foreign := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: otherWs, SourceID: srcA, DestinationID: dstA}

		for _, c := range []*pipelineservice.Connection{aa, ab, ba, foreign} {
			seedConnection(t, store, c)
		}

		ids := func(conns []*pipelineservice.Connection) []uuid.UUID {
			out := make([]uuid.UUID, 0, len(conns))
			for _, c := range conns {
				out = append(out, c.ID)
			}

			return out
		}

		uc := pipelineconnections.NewListConnections(store)

		got, err := uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{aa.ID, ab.ID, ba.ID}, ids(got))

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, SourceID: &srcA})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{aa.ID, ab.ID}, ids(got))

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, DestinationID: &dstA})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{aa.ID, ba.ID}, ids(got))

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, SourceID: &srcB, DestinationID: &dstA})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{ba.ID}, ids(got))

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, SourceID: &srcB, DestinationID: &dstB})
		require.NoError(t, err)
		assert.Empty(t, got)
	})

	t.Run("filters by tags, matching all of them, within the workspace", func(t *testing.T) {
		ws := uuid.New()
		otherWs := uuid.New()
		srcA, srcB := uuid.New(), uuid.New()
		store := newCreateConnStore(ws, srcA, uuid.New())

		finance := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcA, Tags: []string{"finance"}}
		both := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcA, Tags: []string{"finance", "prod"}}
		bothOtherSource := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcB, Tags: []string{"finance", "prod", "x"}}
		prod := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcA, Tags: []string{"prod"}}
		untagged := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, SourceID: srcA}
		foreign := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: otherWs, SourceID: srcA, Tags: []string{"finance", "prod", "elsewhere"}}

		for _, c := range []*pipelineservice.Connection{finance, both, bothOtherSource, prod, untagged, foreign} {
			seedConnection(t, store, c)
		}

		ids := func(conns []*pipelineservice.Connection) []uuid.UUID {
			out := make([]uuid.UUID, 0, len(conns))
			for _, c := range conns {
				out = append(out, c.ID)
			}

			return out
		}

		uc := pipelineconnections.NewListConnections(store)

		got, err := uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, Tags: []string{"finance"}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{finance.ID, both.ID, bothOtherSource.ID}, ids(got))

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, Tags: []string{"finance", "prod"}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{both.ID, bothOtherSource.ID}, ids(got))

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, Tags: []string{" PROD ", "Finance"}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{both.ID, bothOtherSource.ID}, ids(got), "filter tags are normalised")

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, SourceID: &srcA, Tags: []string{"finance", "prod"}})
		require.NoError(t, err)
		assert.ElementsMatch(t, []uuid.UUID{both.ID}, ids(got))

		got, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, Tags: []string{"elsewhere"}})
		require.NoError(t, err)
		assert.Empty(t, got)

		_, err = uc.Execute(context.Background(), pipelineconnections.ListConnectionsParams{WorkspaceID: ws, Tags: []string{"no spaces"}})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "tags", validationErr.Field)
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

		uc := pipelineconnections.NewDeleteConnection(store, pipelineservice.NoopAuditRecorder{})
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

		audit := &recordingAudit{}
		uc := pipelineconnections.NewDeleteConnection(store, audit)
		err := uc.Execute(context.Background(), pipelineconnections.DeleteConnectionParams{
			ID:          conn.ID,
			WorkspaceID: ws,
		})
		require.ErrorIs(t, err, pipelineservice.ErrConnectionNotFound)
		assert.Empty(t, audit.events)

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
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
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
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
		_, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID:          conn.ID,
			WorkspaceID: ws,
			Schedule:    &invalidPtr,
		})
		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "schedule", validationErr.Field)
		assert.Contains(t, err.Error(), "invalid cron expression")
	})

	t.Run("empty optional strings clear the stored value", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		schedule := "0 * * * *"
		format := "${SOURCE_NAMESPACE}_raw"
		prefix := "raw_"
		conn := &pipelineservice.Connection{
			ID:                    uuid.New(),
			WorkspaceID:           ws,
			Status:                pipelineservice.ConnectionStatusActive,
			Schedule:              &schedule,
			CustomNamespaceFormat: &format,
			StreamPrefix:          &prefix,
		}
		seedConnection(t, store, conn)

		empty := ""
		emptyPtr := &empty
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
		updated, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID:                    conn.ID,
			WorkspaceID:           ws,
			Schedule:              &emptyPtr,
			CustomNamespaceFormat: &emptyPtr,
			StreamPrefix:          &emptyPtr,
		})
		require.NoError(t, err)
		assert.Nil(t, updated.Schedule)
		assert.Nil(t, updated.NextScheduledAt)
		assert.Nil(t, updated.CustomNamespaceFormat)
		assert.Nil(t, updated.StreamPrefix)
	})

	t.Run("returns error when connection is in a different workspace", func(t *testing.T) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: uuid.New()}
		seedConnection(t, store, conn)

		newName := "x"
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
		_, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID:          conn.ID,
			WorkspaceID: ws,
			Name:        &newName,
		})
		require.Error(t, err)
	})
}

func TestUpdateConnection_Tags(t *testing.T) {
	newStore := func() (*createConnStore, *pipelineservice.Connection) {
		ws := uuid.New()
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Name: "c", Tags: []string{"finance", "prod"}}
		seedConnection(t, store, conn)

		return store, conn
	}

	t.Run("unset tags keep the stored tags", func(t *testing.T) {
		store, conn := newStore()
		name := "renamed"
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
		updated, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID: conn.ID, WorkspaceID: conn.WorkspaceID, Name: &name,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"finance", "prod"}, updated.Tags)
	})

	t.Run("set tags replace the stored tags, normalised", func(t *testing.T) {
		store, conn := newStore()
		tags := []string{" Team:Data", "prod", "PROD"}
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
		updated, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID: conn.ID, WorkspaceID: conn.WorkspaceID, Tags: &tags,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"prod", "team:data"}, updated.Tags)
		assert.Equal(t, []string{"prod", "team:data"}, store.connections.records[conn.ID].Tags)
	})

	t.Run("an empty list clears the tags", func(t *testing.T) {
		store, conn := newStore()
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
		updated, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID: conn.ID, WorkspaceID: conn.WorkspaceID, Tags: &[]string{},
		})
		require.NoError(t, err)
		assert.Empty(t, updated.Tags)
	})

	t.Run("an invalid tag is a validation error and changes nothing", func(t *testing.T) {
		store, conn := newStore()
		tags := []string{"ok", "-bad"}
		uc := pipelineconnections.NewUpdateConnection(store, pipelineservice.NoopAuditRecorder{})
		_, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
			ID: conn.ID, WorkspaceID: conn.WorkspaceID, Tags: &tags,
		})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "tags", validationErr.Field)
		assert.Contains(t, err.Error(), `"-bad"`)
		assert.Equal(t, []string{"finance", "prod"}, store.connections.records[conn.ID].Tags)
	})
}

func TestUpdateConnection_TagChangeIsAudited(t *testing.T) {
	ws := uuid.New()
	store := newCreateConnStore(ws, uuid.New(), uuid.New())
	conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Name: "c", Tags: []string{"finance"}}
	seedConnection(t, store, conn)

	audit := &recordingAudit{}
	tags := []string{"finance", "prod"}
	uc := pipelineconnections.NewUpdateConnection(store, audit)
	_, err := uc.Execute(context.Background(), pipelineconnections.UpdateConnectionParams{
		ID: conn.ID, WorkspaceID: ws, Tags: &tags,
	})
	require.NoError(t, err)

	require.Len(t, audit.events, 1)
	event := audit.events[0]
	assert.Equal(t, "update", event.Action)
	assert.Equal(t, "connection", event.ResourceType)
	assert.Equal(t, conn.ID, event.ResourceID)

	before, ok := event.Before.(map[string]any)
	require.True(t, ok)
	after, ok := event.After.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, []string{"finance"}, before["tags"])
	assert.Equal(t, []string{"finance", "prod"}, after["tags"])
}

func TestEnableDisableConnection(t *testing.T) {
	ws := uuid.New()

	t.Run("enable transitions inactive to active", func(t *testing.T) {
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Status: pipelineservice.ConnectionStatusInactive}
		seedConnection(t, store, conn)

		getter := pipelineconnections.NewGetConnection(store)
		statusUC := pipelineconnections.NewUpdateConnectionStatus(store)
		uc := pipelineconnections.NewEnableConnection(getter, statusUC, pipelineservice.NoopAuditRecorder{})
		out, err := uc.Execute(context.Background(), pipelineconnections.EnableConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  ws,
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectionStatusActive, out.Status)
	})

	t.Run("enable clears the pause reason", func(t *testing.T) {
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		reason := `stream "public.users": mapper step 1: unknown field "name"`
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Status: pipelineservice.ConnectionStatusPaused, StatusReason: &reason}
		seedConnection(t, store, conn)

		getter := pipelineconnections.NewGetConnection(store)
		statusUC := pipelineconnections.NewUpdateConnectionStatus(store)
		uc := pipelineconnections.NewEnableConnection(getter, statusUC, pipelineservice.NoopAuditRecorder{})
		out, err := uc.Execute(context.Background(), pipelineconnections.EnableConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  ws,
		})
		require.NoError(t, err)
		assert.Equal(t, pipelineservice.ConnectionStatusActive, out.Status)
		assert.Nil(t, out.StatusReason)
	})

	t.Run("disable transitions active to inactive", func(t *testing.T) {
		store := newCreateConnStore(ws, uuid.New(), uuid.New())
		conn := &pipelineservice.Connection{ID: uuid.New(), WorkspaceID: ws, Status: pipelineservice.ConnectionStatusActive}
		seedConnection(t, store, conn)

		getter := pipelineconnections.NewGetConnection(store)
		statusUC := pipelineconnections.NewUpdateConnectionStatus(store)
		uc := pipelineconnections.NewDisableConnection(getter, statusUC, pipelineservice.NoopAuditRecorder{})
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
		uc := pipelineconnections.NewEnableConnection(getter, statusUC, pipelineservice.NoopAuditRecorder{})
		_, err := uc.Execute(context.Background(), pipelineconnections.EnableConnectionParams{
			ConnectionID: conn.ID,
			WorkspaceID:  ws,
		})
		require.Error(t, err)
	})
}

type recordingAudit struct {
	events []pipelineservice.AuditEvent
}

func (r *recordingAudit) Record(_ context.Context, event pipelineservice.AuditEvent) {
	r.events = append(r.events, event)
}
