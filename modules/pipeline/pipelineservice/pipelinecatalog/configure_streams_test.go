package pipelinecatalog

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/protocol"
)

type configureFixture struct {
	store *fakeStore
	audit *recordingAudit
	uc    *ConfigureStreams
	conn  *pipelineservice.Connection
}

func newConfigureFixture(t *testing.T) *configureFixture {
	t.Helper()

	store := newFakeStore()
	conn := &pipelineservice.Connection{
		ID:          uuid.New(),
		WorkspaceID: uuid.New(),
		SourceID:    uuid.New(),
		Name:        "pg-to-bq",
	}
	store.connections.records[conn.ID] = conn

	seedDiscovery(t, store, conn.SourceID, 1, protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{
		{Namespace: "public", Name: "stale"},
	}})
	seedDiscovery(t, store, conn.SourceID, 2, protocol.AirbyteCatalog{Streams: []protocol.AirbyteStream{
		{
			Namespace:          "public",
			Name:               "users",
			JSONSchema:         json.RawMessage(nestedSchema),
			SupportedSyncModes: []protocol.SyncMode{protocol.SyncModeFullRefresh, protocol.SyncModeIncremental},
		},
		{
			Namespace:               "public",
			Name:                    "orders",
			JSONSchema:              json.RawMessage(`{"properties":{"id":{"type":"integer"},"ts":{"type":"string"}}}`),
			SupportedSyncModes:      []protocol.SyncMode{protocol.SyncModeFullRefresh, protocol.SyncModeIncremental},
			SourceDefinedCursor:     true,
			DefaultCursorField:      []string{"ts"},
			SourceDefinedPrimaryKey: [][]string{{"id"}},
		},
		{
			Name:               "events",
			SupportedSyncModes: []protocol.SyncMode{protocol.SyncModeFullRefresh},
		},
	}})

	audit := &recordingAudit{}

	return &configureFixture{
		store: store,
		audit: audit,
		uc:    NewConfigureStreams(store, audit),
		conn:  conn,
	}
}

func (f *configureFixture) execute(streams ...protocol.ConfiguredAirbyteStream) error {
	return f.uc.Execute(context.Background(), ConfigureStreamsParams{
		ConnectionID: f.conn.ID,
		SourceID:     f.conn.SourceID,
		Streams:      streams,
	})
}

func configured(namespace, name string, mode protocol.SyncMode, destMode protocol.DestinationSyncMode) protocol.ConfiguredAirbyteStream {
	return protocol.ConfiguredAirbyteStream{
		Stream:              protocol.AirbyteStream{Namespace: namespace, Name: name},
		SyncMode:            mode,
		DestinationSyncMode: destMode,
	}
}

func TestConfigureStreams_CreatesEnrichedCatalog(t *testing.T) {
	f := newConfigureFixture(t)

	users := configured("public", "users", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppendDedup)
	users.CursorField = []string{"updated_at"}
	users.PrimaryKey = [][]string{{"id"}}
	users.SelectedFields = fields([]string{"name"})

	orders := configured("public", "orders", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppendDedup)
	orders.CursorField = []string{"ignored"}

	require.NoError(t, f.execute(users, orders))

	assert.Equal(t, 1, f.store.configured.created)

	got := storedStreams(t, f.store, f.conn.ID)
	require.Len(t, got, 2)

	assert.JSONEq(t, nestedSchema, string(got[0].Stream.JSONSchema))
	assert.Equal(t, []protocol.SyncMode{protocol.SyncModeFullRefresh, protocol.SyncModeIncremental}, got[0].Stream.SupportedSyncModes)
	assert.Equal(t, fields([]string{"name"}, []string{"updated_at"}, []string{"id"}), got[0].SelectedFields)

	assert.Equal(t, []string{"ts"}, got[1].CursorField, "source-defined cursor overrides the requested one")
	assert.Equal(t, [][]string{{"id"}}, got[1].PrimaryKey, "source-defined primary key is applied")
	assert.True(t, got[1].Stream.SourceDefinedCursor)
}

func TestConfigureStreams_UpdatesExistingCatalog(t *testing.T) {
	f := newConfigureFixture(t)
	seedConfigured(t, f.store, f.conn.ID, []protocol.ConfiguredAirbyteStream{
		configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
	})

	require.NoError(t, f.execute(configured("", "events", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeAppend)))

	assert.Equal(t, 0, f.store.configured.created)
	assert.Equal(t, 1, f.store.configured.updated)

	got := storedStreams(t, f.store, f.conn.ID)
	require.Len(t, got, 1)
	assert.Equal(t, "events", got[0].Stream.Name)
}

func TestConfigureStreams_RecordsAuditEvent(t *testing.T) {
	t.Run("update lists previous and new streams", func(t *testing.T) {
		f := newConfigureFixture(t)
		seedConfigured(t, f.store, f.conn.ID, []protocol.ConfiguredAirbyteStream{
			configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
			configured("", "events", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeAppend),
		})

		require.NoError(t, f.execute(
			configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeAppend),
			configured("public", "orders", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppendDedup),
		))

		require.Len(t, f.audit.events, 1)
		assert.Equal(t, pipelineservice.AuditEvent{
			WorkspaceID:   f.conn.WorkspaceID,
			Action:        "update",
			ResourceType:  "connection",
			ResourceID:    f.conn.ID,
			ResourceLabel: "pg-to-bq",
			Before: map[string]any{"streams": []string{
				"events:full_refresh/append",
				"public.users:full_refresh/overwrite",
			}},
			After: map[string]any{"streams": []string{
				"public.orders:incremental/append_dedup",
				"public.users:full_refresh/append",
			}},
		}, f.audit.events[0])
	})

	t.Run("first configuration has empty before list", func(t *testing.T) {
		f := newConfigureFixture(t)

		require.NoError(t, f.execute(configured("", "events", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeAppend)))

		require.Len(t, f.audit.events, 1)
		assert.Equal(t, map[string]any{"streams": []string{}}, f.audit.events[0].Before)
		assert.Equal(t, map[string]any{"streams": []string{"events:full_refresh/append"}}, f.audit.events[0].After)
	})

	t.Run("failed write records nothing", func(t *testing.T) {
		f := newConfigureFixture(t)
		f.store.configured.createErr = errStorage

		require.ErrorIs(t, f.execute(configured("", "events", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeAppend)), errStorage)
		assert.Empty(t, f.audit.events)
	})
}

func TestConfigureStreams_ValidationErrors(t *testing.T) {
	withCursor := func(s protocol.ConfiguredAirbyteStream, cursor ...string) protocol.ConfiguredAirbyteStream {
		s.CursorField = cursor

		return s
	}

	withSelected := func(s protocol.ConfiguredAirbyteStream, paths ...[]string) protocol.ConfiguredAirbyteStream {
		s.SelectedFields = fields(paths...)

		return s
	}

	tests := []struct {
		name    string
		stream  protocol.ConfiguredAirbyteStream
		wantErr string
	}{
		{
			name:    "stream missing from latest discovery",
			stream:  configured("public", "stale", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
			wantErr: `stream "public.stale" not found`,
		},
		{
			name:    "unsupported sync mode",
			stream:  configured("", "events", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppend),
			wantErr: `sync mode "incremental" not supported`,
		},
		{
			name:    "full refresh with append dedup",
			stream:  configured("public", "orders", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeAppendDedup),
			wantErr: "incompatible with destination sync mode append_dedup",
		},
		{
			name:    "incremental without cursor",
			stream:  configured("public", "users", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppend),
			wantErr: "cursor field required",
		},
		{
			name:    "append dedup without primary key",
			stream:  withCursor(configured("public", "users", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppendDedup), "updated_at"),
			wantErr: "primary key required",
		},
		{
			name:    "unknown selected field",
			stream:  withSelected(configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite), []string{"nope"}),
			wantErr: "invalid selected fields",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfigureFixture(t)

			err := f.execute(tt.stream)
			require.ErrorContains(t, err, tt.wantErr)

			var validationErr *pipelineservice.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Empty(t, f.store.configured.records)
			assert.Empty(t, f.audit.events)
		})
	}
}

func TestConfigureStreams_StorageErrors(t *testing.T) {
	t.Run("connection not found", func(t *testing.T) {
		f := newConfigureFixture(t)
		delete(f.store.connections.records, f.conn.ID)

		require.ErrorIs(t, f.execute(), pipelineservice.ErrConnectionNotFound)
	})

	t.Run("no discovered catalog", func(t *testing.T) {
		f := newConfigureFixture(t)
		f.store.discoveries.records = nil

		require.ErrorIs(t, f.execute(), pipelineservice.ErrCatalogDiscoveryNotFound)
	})

	t.Run("corrupt discovered catalog", func(t *testing.T) {
		f := newConfigureFixture(t)
		f.store.discoveries.records[1].CatalogJSON = "{"

		require.ErrorContains(t, f.execute(), "unmarshaling latest catalog")
	})

	t.Run("configured catalog lookup fails", func(t *testing.T) {
		f := newConfigureFixture(t)
		f.store.configured.firstErr = errStorage

		require.ErrorIs(t, f.execute(), errStorage)
	})

	t.Run("update fails", func(t *testing.T) {
		f := newConfigureFixture(t)
		seedConfigured(t, f.store, f.conn.ID, nil)
		f.store.configured.updateErr = errStorage

		require.ErrorIs(t, f.execute(), errStorage)
		assert.Empty(t, f.audit.events)
	})
}

func TestConfigureStreams_Mappers(t *testing.T) {
	withMappers := func(s protocol.ConfiguredAirbyteStream, raw string) protocol.ConfiguredAirbyteStream {
		s.Mappers = json.RawMessage(raw)

		return s
	}

	withSelectedFields := func(s protocol.ConfiguredAirbyteStream, paths ...[]string) protocol.ConfiguredAirbyteStream {
		s.SelectedFields = fields(paths...)

		return s
	}

	t.Run("valid mappers are stored and read back", func(t *testing.T) {
		f := newConfigureFixture(t)

		const mappers = `[{"rename":{"name":"full_name"}},{"set":{"label":"r.full_name + '!'"}}]`

		users := withMappers(
			configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
			mappers,
		)
		require.NoError(t, f.execute(users))

		got := storedStreams(t, f.store, f.conn.ID)
		require.Len(t, got, 1)
		assert.JSONEq(t, mappers, string(got[0].Mappers))

		catalog, err := NewGetConfiguredCatalog(f.store).Execute(context.Background(), GetConfiguredCatalogParams{ConnectionID: f.conn.ID})
		require.NoError(t, err)
		require.Len(t, catalog.Streams, 1)
		assert.JSONEq(t, mappers, string(catalog.Streams[0].Mappers))
	})

	tests := []struct {
		name    string
		stream  protocol.ConfiguredAirbyteStream
		wantErr string
	}{
		{
			name: "set reads an unknown field",
			stream: withMappers(
				configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
				`[{"rename":{"name":"full_name"}},{"set":{"label":"r.nope + 'x'"}}]`,
			),
			wantErr: `invalid mappers for stream "public.users": step 1`,
		},
		{
			name: "rename onto an existing field",
			stream: withMappers(
				configured("public", "orders", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppendDedup),
				`[{"rename":{"id":"ts"}}]`,
			),
			wantErr: `invalid mappers for stream "public.orders": step 0: field "ts" already exists`,
		},
		{
			name: "source-defined primary key outside the selection is force-included",
			stream: withMappers(
				withSelectedFields(configured("public", "orders", protocol.SyncModeIncremental, protocol.DestinationSyncModeAppendDedup), []string{"ts"}),
				`[{"rename":{"id":"order_id"}}]`,
			),
		},
		{
			name: "mappers are not a list of steps",
			stream: withMappers(
				configured("public", "users", protocol.SyncModeFullRefresh, protocol.DestinationSyncModeOverwrite),
				`{"rename":{"name":"full_name"}}`,
			),
			wantErr: `invalid mappers for stream "public.users"`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			f := newConfigureFixture(t)

			err := f.execute(tt.stream)
			if tt.wantErr == "" {
				require.NoError(t, err)
				assert.Equal(t, fields([]string{"ts"}, []string{"id"}), storedStreams(t, f.store, f.conn.ID)[0].SelectedFields)

				return
			}

			require.ErrorContains(t, err, tt.wantErr)

			var validationErr *pipelineservice.ValidationError
			require.ErrorAs(t, err, &validationErr)
			assert.Empty(t, f.store.configured.records)
		})
	}
}
