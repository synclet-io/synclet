package appmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
	"google.golang.org/protobuf/types/known/structpb"

	protocolv1 "github.com/synclet-io/synclet/gen/proto/synclet/protocol/v1"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	registryv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	statsv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
	"github.com/synclet-io/synclet/pkg/buildinfo"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

var (
	testWorkspaceID = uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	viewerUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	editorUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000002")
	outsiderUserID  = uuid.MustParse("00000000-0000-0000-0000-000000000003")
	otherWorkspace  = uuid.MustParse("00000000-0000-0000-0000-0000000000bb")
)

const (
	viewerKey   = "synclet_sk_viewer"
	editorKey   = "synclet_sk_editor"
	outsiderKey = "synclet_sk_outsider"
	expiredKey  = "synclet_sk_expired"
	unknownKey  = "synclet_sk_unknown"

	oauthReadKey    = "synclet_sk_oauth_read"
	oauthActionsKey = "synclet_sk_oauth_actions"
	oauthWriteKey   = "synclet_sk_oauth_write"

	testPublicURL = "https://mcp.example.com"
)

type fakeKeys struct{}

func (fakeKeys) ValidateMCPKey(_ context.Context, rawKey string) (MCPKey, error) {
	expiresAt := time.Now().Add(time.Hour)
	oauthKey := func(level string) MCPKey {
		return MCPKey{UserID: editorUserID.String(), WorkspaceID: testWorkspaceID.String(), Level: level, ExpiresAt: &expiresAt}
	}

	switch rawKey {
	case viewerKey:
		return MCPKey{UserID: viewerUserID.String(), WorkspaceID: testWorkspaceID.String()}, nil
	case editorKey:
		return MCPKey{UserID: editorUserID.String(), WorkspaceID: testWorkspaceID.String()}, nil
	case outsiderKey:
		return MCPKey{UserID: outsiderUserID.String(), WorkspaceID: otherWorkspace.String()}, nil
	case oauthReadKey:
		return oauthKey(LevelRead), nil
	case oauthActionsKey:
		return oauthKey(LevelActions), nil
	case oauthWriteKey:
		return oauthKey(LevelWrite), nil
	case expiredKey:
		return MCPKey{}, fmt.Errorf("API key expired: %w", connectutil.ErrInvalidCredential)
	default:
		return MCPKey{}, fmt.Errorf("unknown API key: %w", connectutil.ErrInvalidCredential)
	}
}

type fakeMembers struct{}

func (fakeMembers) IsMember(context.Context, uuid.UUID, uuid.UUID) (bool, error) { return true, nil }

func (fakeMembers) GetMemberRole(_ context.Context, workspaceID, userID uuid.UUID) (string, error) {
	if workspaceID != testWorkspaceID {
		return "", connectutil.ErrNotMember
	}

	if userID == editorUserID {
		return "Editor", nil
	}

	return "Viewer", nil
}

type recordedCall struct {
	method      string
	userID      uuid.UUID
	workspaceID uuid.UUID
	req         proto.Message
}

// fakeServices implements every handler the tools call and records each call.
type fakeServices struct {
	pipelinev1connect.UnimplementedSourceServiceHandler
	pipelinev1connect.UnimplementedDestinationServiceHandler
	pipelinev1connect.UnimplementedConnectionServiceHandler
	pipelinev1connect.UnimplementedJobServiceHandler
	pipelinev1connect.UnimplementedWorkflowServiceHandler
	statsv1connect.UnimplementedStatsServiceHandler
	registryv1connect.UnimplementedConnectorRegistryServiceHandler

	mu       sync.Mutex
	calls    []recordedCall
	logLines int

	configuredCatalog    *pipelinev1.GetConfiguredCatalogResponse
	configuredCatalogErr error
}

func (f *fakeServices) record(ctx context.Context, method string, req proto.Message) {
	f.mu.Lock()
	defer f.mu.Unlock()

	userID, _ := connectutil.UserIDFromContext(ctx)
	workspaceID, _ := connectutil.WorkspaceIDFromContext(ctx)
	f.calls = append(f.calls, recordedCall{method: method, userID: userID, workspaceID: workspaceID, req: req})
}

func (f *fakeServices) methods() []string {
	f.mu.Lock()
	defer f.mu.Unlock()

	out := make([]string, 0, len(f.calls))
	for _, c := range f.calls {
		out = append(out, c.method)
	}

	return out
}

func (f *fakeServices) last() recordedCall {
	f.mu.Lock()
	defer f.mu.Unlock()

	return f.calls[len(f.calls)-1]
}

func respond[Res any](ctx context.Context, f *fakeServices, method string, req proto.Message) (*connect.Response[Res], error) { //nolint:unparam // matches the handler signature
	f.record(ctx, method, req)

	return connect.NewResponse(new(Res)), nil
}

func (f *fakeServices) ListConnections(ctx context.Context, r *connect.Request[pipelinev1.ListConnectionsRequest]) (*connect.Response[pipelinev1.ListConnectionsResponse], error) {
	return respond[pipelinev1.ListConnectionsResponse](ctx, f, "ListConnections", r.Msg)
}

func (f *fakeServices) GetConnection(ctx context.Context, r *connect.Request[pipelinev1.GetConnectionRequest]) (*connect.Response[pipelinev1.GetConnectionResponse], error) {
	return respond[pipelinev1.GetConnectionResponse](ctx, f, "GetConnection", r.Msg)
}

func (f *fakeServices) ListSources(ctx context.Context, r *connect.Request[pipelinev1.ListSourcesRequest]) (*connect.Response[pipelinev1.ListSourcesResponse], error) {
	return respond[pipelinev1.ListSourcesResponse](ctx, f, "ListSources", r.Msg)
}

func (f *fakeServices) GetSource(ctx context.Context, r *connect.Request[pipelinev1.GetSourceRequest]) (*connect.Response[pipelinev1.GetSourceResponse], error) {
	return respond[pipelinev1.GetSourceResponse](ctx, f, "GetSource", r.Msg)
}

func (f *fakeServices) ListDestinations(ctx context.Context, r *connect.Request[pipelinev1.ListDestinationsRequest]) (*connect.Response[pipelinev1.ListDestinationsResponse], error) {
	return respond[pipelinev1.ListDestinationsResponse](ctx, f, "ListDestinations", r.Msg)
}

func (f *fakeServices) GetDestination(ctx context.Context, r *connect.Request[pipelinev1.GetDestinationRequest]) (*connect.Response[pipelinev1.GetDestinationResponse], error) {
	return respond[pipelinev1.GetDestinationResponse](ctx, f, "GetDestination", r.Msg)
}

func (f *fakeServices) ListJobs(ctx context.Context, r *connect.Request[pipelinev1.ListJobsRequest]) (*connect.Response[pipelinev1.ListJobsResponse], error) {
	return respond[pipelinev1.ListJobsResponse](ctx, f, "ListJobs", r.Msg)
}

func (f *fakeServices) GetJob(ctx context.Context, r *connect.Request[pipelinev1.GetJobRequest]) (*connect.Response[pipelinev1.GetJobResponse], error) {
	f.record(ctx, "GetJob", r.Msg)

	return connect.NewResponse(&pipelinev1.GetJobResponse{Job: &pipelinev1.Job{Id: r.Msg.GetJobId()}}), nil
}

// GetJobLogs serves logLines lines numbered from 1, at most 5000 per page.
func (f *fakeServices) GetJobLogs(ctx context.Context, r *connect.Request[pipelinev1.GetJobLogsRequest]) (*connect.Response[pipelinev1.GetJobLogsResponse], error) {
	f.record(ctx, "GetJobLogs", r.Msg)

	limit := int(r.Msg.GetLimit())
	if limit <= 0 || limit > 5000 {
		limit = 5000
	}

	start := int(r.Msg.GetAfterId())
	end := min(start+limit, f.logLines)

	lines := make([]string, 0, end-start)
	for i := start + 1; i <= end; i++ {
		lines = append(lines, "line "+strconv.Itoa(i))
	}

	return connect.NewResponse(&pipelinev1.GetJobLogsResponse{
		Lines:   lines,
		LastId:  int64(end),
		HasMore: end < f.logLines,
	}), nil
}

func (f *fakeServices) ListWorkflows(ctx context.Context, r *connect.Request[pipelinev1.ListWorkflowsRequest]) (*connect.Response[pipelinev1.ListWorkflowsResponse], error) {
	return respond[pipelinev1.ListWorkflowsResponse](ctx, f, "ListWorkflows", r.Msg)
}

func (f *fakeServices) GetWorkflow(ctx context.Context, r *connect.Request[pipelinev1.GetWorkflowRequest]) (*connect.Response[pipelinev1.GetWorkflowResponse], error) {
	return respond[pipelinev1.GetWorkflowResponse](ctx, f, "GetWorkflow", r.Msg)
}

func (f *fakeServices) ListWorkflowRuns(ctx context.Context, r *connect.Request[pipelinev1.ListWorkflowRunsRequest]) (*connect.Response[pipelinev1.ListWorkflowRunsResponse], error) {
	return respond[pipelinev1.ListWorkflowRunsResponse](ctx, f, "ListWorkflowRuns", r.Msg)
}

func (f *fakeServices) GetWorkflowRun(ctx context.Context, r *connect.Request[pipelinev1.GetWorkflowRunRequest]) (*connect.Response[pipelinev1.GetWorkflowRunResponse], error) {
	return respond[pipelinev1.GetWorkflowRunResponse](ctx, f, "GetWorkflowRun", r.Msg)
}

func (f *fakeServices) GetConnectionStats(ctx context.Context, r *connect.Request[statsv1.GetConnectionStatsRequest]) (*connect.Response[statsv1.GetConnectionStatsResponse], error) {
	return respond[statsv1.GetConnectionStatsResponse](ctx, f, "GetConnectionStats", r.Msg)
}

func (f *fakeServices) GetSchemaChanges(ctx context.Context, r *connect.Request[pipelinev1.GetSchemaChangesRequest]) (*connect.Response[pipelinev1.GetSchemaChangesResponse], error) {
	return respond[pipelinev1.GetSchemaChangesResponse](ctx, f, "GetSchemaChanges", r.Msg)
}

var (
	configuredCatalog = &pipelinev1.GetConfiguredCatalogResponse{Catalog: mustStruct(map[string]any{
		"streams": []any{
			map[string]any{
				"stream":                map[string]any{"name": "users", "namespace": "public", "json_schema": map[string]any{}},
				"sync_mode":             "incremental",
				"destination_sync_mode": "append_dedup",
				"cursor_field":          []any{"updated_at"},
				"primary_key":           []any{[]any{"id"}},
				"selected_fields":       []any{map[string]any{"field_path": []any{"id"}}, map[string]any{"field_path": []any{"updated_at"}}},
				"mappers":               []any{map[string]any{"type": "hash", "field": "email"}},
				"generation_id":         0,
				"minimum_generation_id": 0,
				"sync_id":               0,
			},
			map[string]any{
				"stream":                map[string]any{"name": "orders", "namespace": "public", "json_schema": map[string]any{}},
				"sync_mode":             "full_refresh",
				"destination_sync_mode": "overwrite",
				"generation_id":         0,
				"minimum_generation_id": 0,
				"sync_id":               0,
			},
		},
	})}
	configuredUsers = &pipelinev1.ConfiguredStream{
		StreamName:          "users",
		Namespace:           "public",
		SyncMode:            pipelinev1.SyncMode_SYNC_MODE_INCREMENTAL,
		DestinationSyncMode: pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND_DEDUP,
		CursorField:         []string{"updated_at"},
		PrimaryKey:          []*pipelinev1.CompositeKey{{FieldPath: []string{"id"}}},
		SelectedFields:      []*pipelinev1.SelectedField{{FieldPath: []string{"id"}}, {FieldPath: []string{"updated_at"}}},
		Mappers:             []*structpb.Struct{mustStruct(map[string]any{"type": "hash", "field": "email"})},
	}
	configuredOrders = &pipelinev1.ConfiguredStream{
		StreamName:          "orders",
		Namespace:           "public",
		SyncMode:            pipelinev1.SyncMode_SYNC_MODE_FULL_REFRESH,
		DestinationSyncMode: pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_OVERWRITE,
	}
	discoveredCatalog = &pipelinev1.GetDiscoveredCatalogResponse{Catalog: &protocolv1.AirbyteCatalog{
		Streams: []*protocolv1.AirbyteStream{{
			Name:               "users",
			Namespace:          "public",
			SupportedSyncModes: []string{"full_refresh", "incremental"},
			DefaultCursorField: []string{"updated_at"},
		}},
	}}
)

var sourceCatalog = &pipelinev1.GetSourceCatalogResponse{Catalog: discoveredCatalog.GetCatalog(), Version: 3}

func mustStruct(m map[string]any) *structpb.Struct {
	s, err := structpb.NewStruct(m)
	if err != nil {
		panic(err)
	}

	return s
}

func (f *fakeServices) GetConfiguredCatalog(ctx context.Context, r *connect.Request[pipelinev1.GetConfiguredCatalogRequest]) (*connect.Response[pipelinev1.GetConfiguredCatalogResponse], error) {
	f.record(ctx, "GetConfiguredCatalog", r.Msg)

	if f.configuredCatalogErr != nil {
		return nil, f.configuredCatalogErr
	}

	if f.configuredCatalog != nil {
		return connect.NewResponse(f.configuredCatalog), nil
	}

	return connect.NewResponse(configuredCatalog), nil
}

func (f *fakeServices) GetDiscoveredCatalog(ctx context.Context, r *connect.Request[pipelinev1.GetDiscoveredCatalogRequest]) (*connect.Response[pipelinev1.GetDiscoveredCatalogResponse], error) {
	f.record(ctx, "GetDiscoveredCatalog", r.Msg)

	return connect.NewResponse(discoveredCatalog), nil
}

func (f *fakeServices) GetSourceCatalog(ctx context.Context, r *connect.Request[pipelinev1.GetSourceCatalogRequest]) (*connect.Response[pipelinev1.GetSourceCatalogResponse], error) {
	f.record(ctx, "GetSourceCatalog", r.Msg)

	return connect.NewResponse(sourceCatalog), nil
}

func (f *fakeServices) ListManagedConnectors(ctx context.Context, r *connect.Request[registryv1.ListManagedConnectorsRequest]) (*connect.Response[registryv1.ListManagedConnectorsResponse], error) {
	return respond[registryv1.ListManagedConnectorsResponse](ctx, f, "ListManagedConnectors", r.Msg)
}

func (f *fakeServices) GetConnectorSpec(ctx context.Context, r *connect.Request[registryv1.GetConnectorSpecRequest]) (*connect.Response[registryv1.GetConnectorSpecResponse], error) {
	return respond[registryv1.GetConnectorSpecResponse](ctx, f, "GetConnectorSpec", r.Msg)
}

func (f *fakeServices) GetSystemInfo(ctx context.Context, r *connect.Request[pipelinev1.GetSystemInfoRequest]) (*connect.Response[pipelinev1.GetSystemInfoResponse], error) {
	f.record(ctx, "GetSystemInfo", r.Msg)

	return connect.NewResponse(&pipelinev1.GetSystemInfoResponse{
		RegistrationEnabled: true,
		WorkspacesMode:      pipelinev1.WorkspacesMode_WORKSPACES_MODE_SINGLE,
		EmailEnabled:        true,
	}), nil
}

func (f *fakeServices) GetRuntimeDefaults(ctx context.Context, r *connect.Request[pipelinev1.GetRuntimeDefaultsRequest]) (*connect.Response[pipelinev1.GetRuntimeDefaultsResponse], error) {
	f.record(ctx, "GetRuntimeDefaults", r.Msg)

	return connect.NewResponse(&pipelinev1.GetRuntimeDefaultsResponse{
		RuntimeDefaults: &pipelinev1.RuntimeDefaults{MaxConcurrentJobs: 4},
	}), nil
}

func (f *fakeServices) ListExecutors(ctx context.Context, r *connect.Request[pipelinev1.ListExecutorsRequest]) (*connect.Response[pipelinev1.ListExecutorsResponse], error) {
	f.record(ctx, "ListExecutors", r.Msg)

	return connect.NewResponse(&pipelinev1.ListExecutorsResponse{
		Executors: []*pipelinev1.ExecutorInstance{
			{Id: "exec-1", Type: pipelinev1.ExecutorType_EXECUTOR_TYPE_DOCKER, Version: "v0.15.0"},
		},
	}), nil
}

func (f *fakeServices) GetPipelineSettings(ctx context.Context, r *connect.Request[pipelinev1.GetPipelineSettingsRequest]) (*connect.Response[pipelinev1.GetPipelineSettingsResponse], error) {
	f.record(ctx, "GetPipelineSettings", r.Msg)

	return connect.NewResponse(&pipelinev1.GetPipelineSettingsResponse{
		Settings: &pipelinev1.PipelineSettings{MaxJobsPerWorkspace: 7},
	}), nil
}

func (f *fakeServices) TriggerSync(ctx context.Context, r *connect.Request[pipelinev1.TriggerSyncRequest]) (*connect.Response[pipelinev1.TriggerSyncResponse], error) {
	return respond[pipelinev1.TriggerSyncResponse](ctx, f, "TriggerSync", r.Msg)
}

func (f *fakeServices) CancelJob(ctx context.Context, r *connect.Request[pipelinev1.CancelJobRequest]) (*connect.Response[pipelinev1.CancelJobResponse], error) {
	return respond[pipelinev1.CancelJobResponse](ctx, f, "CancelJob", r.Msg)
}

func (f *fakeServices) DisableConnection(ctx context.Context, r *connect.Request[pipelinev1.DisableConnectionRequest]) (*connect.Response[pipelinev1.DisableConnectionResponse], error) {
	return respond[pipelinev1.DisableConnectionResponse](ctx, f, "DisableConnection", r.Msg)
}

func (f *fakeServices) EnableConnection(ctx context.Context, r *connect.Request[pipelinev1.EnableConnectionRequest]) (*connect.Response[pipelinev1.EnableConnectionResponse], error) {
	return respond[pipelinev1.EnableConnectionResponse](ctx, f, "EnableConnection", r.Msg)
}

func (f *fakeServices) RunWorkflow(ctx context.Context, r *connect.Request[pipelinev1.RunWorkflowRequest]) (*connect.Response[pipelinev1.RunWorkflowResponse], error) {
	return respond[pipelinev1.RunWorkflowResponse](ctx, f, "RunWorkflow", r.Msg)
}

func (f *fakeServices) RerunWorkflowRun(ctx context.Context, r *connect.Request[pipelinev1.RerunWorkflowRunRequest]) (*connect.Response[pipelinev1.RerunWorkflowRunResponse], error) {
	return respond[pipelinev1.RerunWorkflowRunResponse](ctx, f, "RerunWorkflowRun", r.Msg)
}

func (f *fakeServices) CreateSource(ctx context.Context, r *connect.Request[pipelinev1.CreateSourceRequest]) (*connect.Response[pipelinev1.CreateSourceResponse], error) {
	return respond[pipelinev1.CreateSourceResponse](ctx, f, "CreateSource", r.Msg)
}

func (f *fakeServices) UpdateSource(ctx context.Context, r *connect.Request[pipelinev1.UpdateSourceRequest]) (*connect.Response[pipelinev1.UpdateSourceResponse], error) {
	return respond[pipelinev1.UpdateSourceResponse](ctx, f, "UpdateSource", r.Msg)
}

func (f *fakeServices) DeleteSource(ctx context.Context, r *connect.Request[pipelinev1.DeleteSourceRequest]) (*connect.Response[pipelinev1.DeleteSourceResponse], error) {
	return respond[pipelinev1.DeleteSourceResponse](ctx, f, "DeleteSource", r.Msg)
}

func (f *fakeServices) CreateDestination(ctx context.Context, r *connect.Request[pipelinev1.CreateDestinationRequest]) (*connect.Response[pipelinev1.CreateDestinationResponse], error) {
	return respond[pipelinev1.CreateDestinationResponse](ctx, f, "CreateDestination", r.Msg)
}

func (f *fakeServices) UpdateDestination(ctx context.Context, r *connect.Request[pipelinev1.UpdateDestinationRequest]) (*connect.Response[pipelinev1.UpdateDestinationResponse], error) {
	return respond[pipelinev1.UpdateDestinationResponse](ctx, f, "UpdateDestination", r.Msg)
}

func (f *fakeServices) DeleteDestination(ctx context.Context, r *connect.Request[pipelinev1.DeleteDestinationRequest]) (*connect.Response[pipelinev1.DeleteDestinationResponse], error) {
	return respond[pipelinev1.DeleteDestinationResponse](ctx, f, "DeleteDestination", r.Msg)
}

func (f *fakeServices) CreateConnection(ctx context.Context, r *connect.Request[pipelinev1.CreateConnectionRequest]) (*connect.Response[pipelinev1.CreateConnectionResponse], error) {
	return respond[pipelinev1.CreateConnectionResponse](ctx, f, "CreateConnection", r.Msg)
}

func (f *fakeServices) UpdateConnection(ctx context.Context, r *connect.Request[pipelinev1.UpdateConnectionRequest]) (*connect.Response[pipelinev1.UpdateConnectionResponse], error) {
	return respond[pipelinev1.UpdateConnectionResponse](ctx, f, "UpdateConnection", r.Msg)
}

func (f *fakeServices) DeleteConnection(ctx context.Context, r *connect.Request[pipelinev1.DeleteConnectionRequest]) (*connect.Response[pipelinev1.DeleteConnectionResponse], error) {
	return respond[pipelinev1.DeleteConnectionResponse](ctx, f, "DeleteConnection", r.Msg)
}

func (f *fakeServices) ConfigureStreams(ctx context.Context, r *connect.Request[pipelinev1.ConfigureStreamsRequest]) (*connect.Response[pipelinev1.ConfigureStreamsResponse], error) {
	return respond[pipelinev1.ConfigureStreamsResponse](ctx, f, "ConfigureStreams", r.Msg)
}

func (f *fakeServices) CreateWorkflow(ctx context.Context, r *connect.Request[pipelinev1.CreateWorkflowRequest]) (*connect.Response[pipelinev1.CreateWorkflowResponse], error) {
	return respond[pipelinev1.CreateWorkflowResponse](ctx, f, "CreateWorkflow", r.Msg)
}

func (f *fakeServices) UpdateWorkflow(ctx context.Context, r *connect.Request[pipelinev1.UpdateWorkflowRequest]) (*connect.Response[pipelinev1.UpdateWorkflowResponse], error) {
	return respond[pipelinev1.UpdateWorkflowResponse](ctx, f, "UpdateWorkflow", r.Msg)
}

func (f *fakeServices) DeleteWorkflow(ctx context.Context, r *connect.Request[pipelinev1.DeleteWorkflowRequest]) (*connect.Response[pipelinev1.DeleteWorkflowResponse], error) {
	return respond[pipelinev1.DeleteWorkflowResponse](ctx, f, "DeleteWorkflow", r.Msg)
}

func (f *fakeServices) services() Services {
	return Services{
		Sources:      f,
		Destinations: f,
		Connections:  f,
		Jobs:         f,
		Workflows:    f,
		Stats:        f,
		Registry:     f,
	}
}

func newTestServer(t *testing.T, writeTools bool) (string, *fakeServices) {
	t.Helper()

	fake := &fakeServices{}
	handler := NewHandler(
		Config{Enabled: true, WriteTools: writeTools, PublicURL: testPublicURL},
		fake.services(),
		connectutil.NewRoleInterceptor(fakeMembers{}),
		NewAPIKeyAuthenticator(fakeKeys{}),
	)

	router := mux.NewRouter()
	handler.Register(router)

	srv := httptest.NewServer(router)
	t.Cleanup(srv.Close)

	return srv.URL + "/mcp", fake
}

type headerTransport struct {
	key  string
	mu   sync.Mutex
	seen []http.Header
}

func (h *headerTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req = req.Clone(req.Context())
	req.Header.Set("Authorization", "Bearer "+h.key)

	resp, err := http.DefaultTransport.RoundTrip(req)
	if err == nil {
		h.mu.Lock()
		h.seen = append(h.seen, resp.Header.Clone())
		h.mu.Unlock()
	}

	return resp, err
}

func connectClient(t *testing.T, endpoint, key string) (*mcp.ClientSession, *headerTransport) {
	t.Helper()

	transport := &headerTransport{key: key}
	client := mcp.NewClient(&mcp.Implementation{Name: "test", Version: "v0"}, nil)

	session, err := client.Connect(t.Context(), &mcp.StreamableClientTransport{
		Endpoint:             endpoint,
		HTTPClient:           &http.Client{Transport: transport},
		DisableStandaloneSSE: true,
		MaxRetries:           -1,
	}, nil)
	require.NoError(t, err)
	t.Cleanup(func() { _ = session.Close() })

	return session, transport
}

func callTool(t *testing.T, session *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()

	if args == nil {
		args = map[string]any{}
	}

	res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: name, Arguments: args})
	require.NoError(t, err)

	return res
}

func resultText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	require.NotEmpty(t, res.Content)

	text, ok := res.Content[0].(*mcp.TextContent)
	require.True(t, ok)

	return text.Text
}

func listTools(t *testing.T, session *mcp.ClientSession) map[string]*mcp.Tool {
	t.Helper()

	res, err := session.ListTools(t.Context(), nil)
	require.NoError(t, err)

	tools := map[string]*mcp.Tool{}
	for _, tool := range res.Tools {
		tools[tool.Name] = tool
	}

	return tools
}

var (
	readTools = []string{
		"list_connections", "get_connection", "list_sources", "get_source", "list_destinations",
		"get_destination", "list_jobs", "get_job", "list_workflows", "get_workflow", "list_workflow_runs",
		"get_workflow_run", "get_connection_stats", "get_schema_changes", "get_configured_catalog", "get_discovered_catalog", "get_source_catalog", "list_connectors",
		"get_connector_spec", "get_system_info", "get_reference", "validate_mappers",
	}
	actionTools = []string{
		"trigger_sync", "cancel_job", "pause_connection", "resume_connection", "run_workflow", "rerun_workflow_run",
	}
	writeTools = []string{
		"create_source", "update_source", "create_destination", "update_destination", "create_connection",
		"update_connection", "configure_streams", "configure_stream", "create_workflow", "update_workflow", "delete_source",
		"delete_destination", "delete_connection", "delete_workflow",
	}
)

func TestMCP_Stateless(t *testing.T) {
	endpoint, _ := newTestServer(t, true)

	for range 2 {
		session, transport := connectClient(t, endpoint, viewerKey)
		assert.NotEmpty(t, listTools(t, session))

		transport.mu.Lock()
		for _, header := range transport.seen {
			assert.Empty(t, header.Get("Mcp-Session-Id"))
		}
		transport.mu.Unlock()
	}
}

func TestMCP_Auth(t *testing.T) {
	endpoint, fake := newTestServer(t, true)

	t.Run("rejected credentials", func(t *testing.T) {
		for name, header := range map[string]string{
			"no header":   "",
			"not bearer":  "Basic abc",
			"garbage":     "Bearer garbage",
			"unknown key": "Bearer " + unknownKey,
			"expired key": "Bearer " + expiredKey,
		} {
			req, err := http.NewRequestWithContext(t.Context(), http.MethodPost, endpoint,
				strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list"}`))
			require.NoError(t, err)
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")

			if header != "" {
				req.Header.Set("Authorization", header)
			}

			resp, err := http.DefaultClient.Do(req)
			require.NoError(t, err)

			body, err := io.ReadAll(resp.Body)
			require.NoError(t, err)

			_ = resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, name)
			assert.NotContains(t, string(body), "expired", name)
			assert.NotContains(t, string(body), "unknown", name)
			assert.Equal(t, `Bearer resource_metadata="https://mcp.example.com/.well-known/oauth-protected-resource"`,
				resp.Header.Get("WWW-Authenticate"), name)
		}
	})

	t.Run("viewer reads in the key's workspace", func(t *testing.T) {
		session, _ := connectClient(t, endpoint, viewerKey)

		res := callTool(t, session, "list_connections", nil)
		require.False(t, res.IsError, resultText(t, res))

		call := fake.last()
		assert.Equal(t, "ListConnections", call.method)
		assert.Equal(t, viewerUserID, call.userID)
		assert.Equal(t, testWorkspaceID, call.workspaceID)
	})

	t.Run("viewer is refused editor actions", func(t *testing.T) {
		session, _ := connectClient(t, endpoint, viewerKey)
		before := len(fake.methods())

		for _, name := range []string{"trigger_sync", "create_source"} {
			res := callTool(t, session, name, nil)
			assert.True(t, res.IsError, name)
			assert.Contains(t, resultText(t, res), "permission_denied", name)
		}

		assert.Len(t, fake.methods(), before)
	})
}

var toolMethods = map[string]string{
	"list_connections":       "ListConnections",
	"get_connection":         "GetConnection",
	"list_sources":           "ListSources",
	"get_source":             "GetSource",
	"list_destinations":      "ListDestinations",
	"get_destination":        "GetDestination",
	"list_jobs":              "ListJobs",
	"get_job":                "GetJob",
	"list_workflows":         "ListWorkflows",
	"get_workflow":           "GetWorkflow",
	"list_workflow_runs":     "ListWorkflowRuns",
	"get_workflow_run":       "GetWorkflowRun",
	"get_connection_stats":   "GetConnectionStats",
	"get_schema_changes":     "GetSchemaChanges",
	"get_configured_catalog": "GetConfiguredCatalog",
	"get_discovered_catalog": "GetDiscoveredCatalog",
	"get_source_catalog":     "GetSourceCatalog",
	"list_connectors":        "ListManagedConnectors",
	"get_connector_spec":     "GetConnectorSpec",
	"get_system_info":        "GetSystemInfo",
	"validate_mappers":       "ValidateMappers",
	"trigger_sync":           "TriggerSync",
	"cancel_job":             "CancelJob",
	"pause_connection":       "DisableConnection",
	"resume_connection":      "EnableConnection",
	"run_workflow":           "RunWorkflow",
	"rerun_workflow_run":     "RerunWorkflowRun",
	"create_source":          "CreateSource",
	"update_source":          "UpdateSource",
	"create_destination":     "CreateDestination",
	"update_destination":     "UpdateDestination",
	"create_connection":      "CreateConnection",
	"update_connection":      "UpdateConnection",
	"configure_streams":      "ConfigureStreams",
	"configure_stream":       "ConfigureStreams",
	"create_workflow":        "CreateWorkflow",
	"update_workflow":        "UpdateWorkflow",
	"delete_source":          "DeleteSource",
	"delete_destination":     "DeleteDestination",
	"delete_connection":      "DeleteConnection",
	"delete_workflow":        "DeleteWorkflow",
}

var toolArgs = map[string]map[string]any{
	"configure_stream": {"stream_name": "events"},
	"get_reference":    {"topic": "mappers"},
}

func assertToolsCallHandlers(t *testing.T, names []string) {
	t.Helper()

	endpoint, fake := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, editorKey)

	for _, name := range names {
		res := callTool(t, session, name, toolArgs[name])
		require.False(t, res.IsError, "%s: %s", name, resultText(t, res))

		if name == "get_reference" {
			continue
		}

		assert.Contains(t, fake.methods(), toolMethods[name], name)
		assert.Equal(t, editorUserID, fake.last().userID, name)
	}
}

func TestMCP_ReadTools(t *testing.T) {
	assertToolsCallHandlers(t, readTools)

	t.Run("request is parsed from the arguments", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)

		res := callTool(t, session, "get_connection", map[string]any{"id": "c-1"})
		require.False(t, res.IsError, resultText(t, res))
		assert.Equal(t, "c-1", fake.last().req.(*pipelinev1.GetConnectionRequest).GetId())

		res = callTool(t, session, "get_connection", map[string]any{"bogus": 1})
		assert.True(t, res.IsError)
	})

	t.Run("get_source_catalog returns a source's streams to a viewer", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)

		res := callTool(t, session, "get_source_catalog", map[string]any{"source_id": "s-1"})
		require.False(t, res.IsError, resultText(t, res))

		call := fake.last()
		assert.Equal(t, "GetSourceCatalog", call.method)
		assert.Equal(t, viewerUserID, call.userID)
		assert.Equal(t, "s-1", call.req.(*pipelinev1.GetSourceCatalogRequest).GetSourceId())

		wantJSON, err := protojson.Marshal(sourceCatalog)
		require.NoError(t, err)
		assert.JSONEq(t, string(wantJSON), resultText(t, res))
	})

	t.Run("catalog tools return the public API's data to a viewer", func(t *testing.T) {
		for name, want := range map[string]proto.Message{
			"get_configured_catalog": configuredCatalog,
			"get_discovered_catalog": discoveredCatalog,
		} {
			endpoint, fake := newTestServer(t, true)
			session, _ := connectClient(t, endpoint, viewerKey)

			res := callTool(t, session, name, map[string]any{"connection_id": "c-1"})
			require.False(t, res.IsError, "%s: %s", name, resultText(t, res))

			call := fake.last()
			assert.Equal(t, toolMethods[name], call.method, name)
			assert.Equal(t, viewerUserID, call.userID, name)
			assert.Equal(t, "c-1", call.req.(interface{ GetConnectionId() string }).GetConnectionId(), name)

			wantJSON, err := protojson.Marshal(want)
			require.NoError(t, err)
			assert.JSONEq(t, string(wantJSON), resultText(t, res), name)
		}
	})

	t.Run("get_job returns the tail of the logs", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		fake.logLines = 7000
		session, _ := connectClient(t, endpoint, viewerKey)

		res := callTool(t, session, "get_job", map[string]any{"jobId": "j-1", "log_lines": 50})
		require.False(t, res.IsError, resultText(t, res))

		var out struct {
			Job struct {
				Job struct {
					ID string `json:"id"`
				} `json:"job"`
			} `json:"job"`
			LogLines []string `json:"log_lines"`
		}
		require.NoError(t, json.Unmarshal([]byte(resultText(t, res)), &out))

		assert.Equal(t, "j-1", out.Job.Job.ID)
		require.Len(t, out.LogLines, 50)
		assert.Equal(t, "line 6951", out.LogLines[0])
		assert.Equal(t, "line 7000", out.LogLines[49])

		res = callTool(t, session, "get_job", map[string]any{"jobId": "j-1", "log_lines": 1001})
		assert.True(t, res.IsError)
	})
}

func TestMCP_ActionTools(t *testing.T) {
	assertToolsCallHandlers(t, actionTools)

	endpoint, _ := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, viewerKey)
	tools := listTools(t, session)

	for _, name := range actionTools {
		require.NotNil(t, tools[name].Annotations, name)
		assert.False(t, tools[name].Annotations.ReadOnlyHint, name)
	}
}

func TestMCP_WriteTools(t *testing.T) {
	assertToolsCallHandlers(t, writeTools)

	t.Run("annotations", func(t *testing.T) {
		endpoint, _ := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)
		tools := listTools(t, session)

		for _, name := range writeTools {
			annotations := tools[name].Annotations
			require.NotNil(t, annotations, name)
			assert.False(t, annotations.ReadOnlyHint, name)

			if strings.HasPrefix(name, "delete_") || strings.HasPrefix(name, "configure_") {
				require.NotNil(t, annotations.DestructiveHint, name)
				assert.True(t, *annotations.DestructiveHint, name)
			}
		}

		for _, name := range readTools {
			require.NotNil(t, tools[name].Annotations, name)
			assert.True(t, tools[name].Annotations.ReadOnlyHint, name)
		}
	})

	t.Run("switched off", func(t *testing.T) {
		endpoint, _ := newTestServer(t, false)
		session, _ := connectClient(t, endpoint, editorKey)
		tools := listTools(t, session)

		for _, name := range writeTools {
			assert.NotContains(t, tools, name)
		}

		assert.Len(t, tools, len(readTools)+len(actionTools))
	})

	t.Run("configure_streams says it replaces the whole configuration", func(t *testing.T) {
		endpoint, _ := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)
		description := listTools(t, session)["configure_streams"].Description

		assert.Contains(t, description, "replaces the whole stream configuration")
		assert.Contains(t, description, "get_configured_catalog")
		assert.Contains(t, description, "every stream")
	})

	t.Run("configure_stream says omitted fields are cleared", func(t *testing.T) {
		endpoint, _ := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, editorKey)

		description := listTools(t, session)["configure_stream"].Description
		assert.Contains(t, description, "every field left out is cleared")
		assert.Contains(t, description, "send the full stream as get_configured_catalog returns it")
	})

	t.Run("configure_streams points to configure_stream", func(t *testing.T) {
		endpoint, _ := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)

		assert.Contains(t, listTools(t, session)["configure_streams"].Description, "configure_stream ")
	})

	t.Run("create_connection takes streams", func(t *testing.T) {
		endpoint, _ := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)
		tool := listTools(t, session)["create_connection"]

		raw, err := json.Marshal(tool.InputSchema)
		require.NoError(t, err)

		var schema struct {
			Properties map[string]struct {
				Type string `json:"type"`
			} `json:"properties"`
		}
		require.NoError(t, json.Unmarshal(raw, &schema))

		assert.Equal(t, "array", schema.Properties["streams"].Type)
		assert.Contains(t, tool.Description, "Pass streams")
		assert.Contains(t, tool.Description, "cannot sync")
	})

	t.Run("create_source schema takes config as an object", func(t *testing.T) {
		endpoint, _ := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)
		tools := listTools(t, session)

		raw, err := json.Marshal(tools["create_source"].InputSchema)
		require.NoError(t, err)

		var schema struct {
			Type       string `json:"type"`
			Properties map[string]struct {
				Type string `json:"type"`
			} `json:"properties"`
		}
		require.NoError(t, json.Unmarshal(raw, &schema))

		assert.Equal(t, "object", schema.Type)
		assert.Equal(t, "object", schema.Properties["config"].Type)
	})
}

func configureStream(t *testing.T, args map[string]any) []*pipelinev1.ConfiguredStream {
	t.Helper()

	return configureStreamOn(t, func(*fakeServices) {}, args)
}

func configureStreamOn(t *testing.T, setup func(*fakeServices), args map[string]any) []*pipelinev1.ConfiguredStream {
	t.Helper()

	endpoint, fake := newTestServer(t, true)
	setup(fake)

	session, _ := connectClient(t, endpoint, editorKey)

	res := callTool(t, session, "configure_stream", args)
	require.False(t, res.IsError, resultText(t, res))

	assert.Equal(t, []string{"GetConfiguredCatalog", "ConfigureStreams"}, fake.methods())

	call := fake.last()
	assert.Equal(t, editorUserID, call.userID)

	req, ok := call.req.(*pipelinev1.ConfigureStreamsRequest)
	require.True(t, ok)
	assert.Equal(t, "c-1", req.GetConnectionId())

	return req.GetStreams()
}

func assertStreams(t *testing.T, want, got []*pipelinev1.ConfiguredStream) {
	t.Helper()

	require.Len(t, got, len(want))

	for i := range want {
		assert.True(t, proto.Equal(want[i], got[i]), "stream %d: want %v, got %v", i, want[i], got[i])
	}
}

func TestMCP_ConfigureStream(t *testing.T) {
	t.Run("changes one stream and keeps the other", func(t *testing.T) {
		streams := configureStream(t, map[string]any{
			"connection_id":         "c-1",
			"namespace":             "public",
			"stream_name":           "orders",
			"sync_mode":             "SYNC_MODE_INCREMENTAL",
			"destination_sync_mode": "DESTINATION_SYNC_MODE_APPEND",
			"cursor_field":          []any{"placed_at"},
		})

		assertStreams(t, []*pipelinev1.ConfiguredStream{
			configuredUsers,
			{
				StreamName:          "orders",
				Namespace:           "public",
				SyncMode:            pipelinev1.SyncMode_SYNC_MODE_INCREMENTAL,
				DestinationSyncMode: pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND,
				CursorField:         []string{"placed_at"},
			},
		}, streams)
	})

	t.Run("adds a stream that is not configured", func(t *testing.T) {
		streams := configureStream(t, map[string]any{
			"connection_id":         "c-1",
			"namespace":             "public",
			"stream_name":           "events",
			"sync_mode":             "SYNC_MODE_INCREMENTAL",
			"destination_sync_mode": "DESTINATION_SYNC_MODE_APPEND",
			"cursor_field":          []any{"created_at"},
		})

		assertStreams(t, []*pipelinev1.ConfiguredStream{
			configuredUsers,
			configuredOrders,
			{
				StreamName:          "events",
				Namespace:           "public",
				SyncMode:            pipelinev1.SyncMode_SYNC_MODE_INCREMENTAL,
				DestinationSyncMode: pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND,
				CursorField:         []string{"created_at"},
			},
		}, streams)
	})

	t.Run("enabled false removes the stream", func(t *testing.T) {
		streams := configureStream(t, map[string]any{
			"connection_id": "c-1",
			"namespace":     "public",
			"stream_name":   "users",
			"enabled":       false,
		})

		assertStreams(t, []*pipelinev1.ConfiguredStream{configuredOrders}, streams)
	})

	t.Run("a stream is matched by namespace and name", func(t *testing.T) {
		streams := configureStream(t, map[string]any{
			"connection_id": "c-1",
			"namespace":     "other",
			"stream_name":   "users",
			"enabled":       false,
		})

		assertStreams(t, []*pipelinev1.ConfiguredStream{configuredUsers, configuredOrders}, streams)
	})

	t.Run("adds the first stream to a connection without a configuration", func(t *testing.T) {
		streams := configureStreamOn(t, func(f *fakeServices) {
			f.configuredCatalogErr = connect.NewError(connect.CodeNotFound, errors.New("configured catalog not found"))
		}, map[string]any{
			"connection_id":         "c-1",
			"namespace":             "public",
			"stream_name":           "events",
			"sync_mode":             "SYNC_MODE_FULL_REFRESH",
			"destination_sync_mode": "DESTINATION_SYNC_MODE_OVERWRITE",
		})

		assertStreams(t, []*pipelinev1.ConfiguredStream{{
			StreamName:          "events",
			Namespace:           "public",
			SyncMode:            pipelinev1.SyncMode_SYNC_MODE_FULL_REFRESH,
			DestinationSyncMode: pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_OVERWRITE,
		}}, streams)
	})

	t.Run("numbers in an untouched stream's mapper config survive", func(t *testing.T) {
		streams := configureStreamOn(t, func(f *fakeServices) {
			f.configuredCatalog = &pipelinev1.GetConfiguredCatalogResponse{Catalog: mustStruct(map[string]any{
				"streams": []any{
					map[string]any{
						"stream":                map[string]any{"name": "users", "namespace": "public"},
						"sync_mode":             "full_refresh",
						"destination_sync_mode": "overwrite",
						"mappers":               []any{map[string]any{"type": "truncate", "field": "bio", "length": 120, "ratio": 0.25}},
					},
				},
			})}
		}, map[string]any{
			"connection_id":         "c-1",
			"namespace":             "public",
			"stream_name":           "orders",
			"sync_mode":             "SYNC_MODE_FULL_REFRESH",
			"destination_sync_mode": "DESTINATION_SYNC_MODE_OVERWRITE",
		})

		require.Len(t, streams, 2)
		require.Len(t, streams[0].GetMappers(), 1)
		assert.True(t, proto.Equal(
			mustStruct(map[string]any{"type": "truncate", "field": "bio", "length": 120, "ratio": 0.25}),
			streams[0].GetMappers()[0],
		), "mapper: %v", streams[0].GetMappers()[0])
	})

	for _, mode := range []string{"update", "soft_delete"} {
		t.Run("stored destination sync mode "+mode+" fails without writing", func(t *testing.T) {
			endpoint, fake := newTestServer(t, true)
			fake.configuredCatalog = &pipelinev1.GetConfiguredCatalogResponse{Catalog: mustStruct(map[string]any{
				"streams": []any{
					map[string]any{
						"stream":                map[string]any{"name": "users", "namespace": "public"},
						"sync_mode":             "incremental",
						"destination_sync_mode": mode,
					},
				},
			})}
			session, _ := connectClient(t, endpoint, editorKey)

			res := callTool(t, session, "configure_stream", map[string]any{"connection_id": "c-1", "namespace": "public", "stream_name": "orders"})
			assert.True(t, res.IsError)
			assert.Contains(t, resultText(t, res), "failed_precondition")
			assert.Equal(t, []string{"GetConfiguredCatalog"}, fake.methods())
		})
	}

	t.Run("stream_name is required", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, editorKey)

		res := callTool(t, session, "configure_stream", map[string]any{"connection_id": "c-1"})
		assert.True(t, res.IsError)
		assert.Empty(t, fake.methods())
	})

	t.Run("viewer is refused", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)

		res := callTool(t, session, "configure_stream", map[string]any{"connection_id": "c-1", "stream_name": "users"})
		assert.True(t, res.IsError)
		assert.Contains(t, resultText(t, res), "permission_denied")
		assert.Empty(t, fake.methods())
	})

	t.Run("hidden with write tools off", func(t *testing.T) {
		endpoint, _ := newTestServer(t, false)
		session, _ := connectClient(t, endpoint, editorKey)

		assert.NotContains(t, listTools(t, session), "configure_stream")
	})
}

func TestMCP_ToolSet(t *testing.T) {
	endpoint, _ := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, viewerKey)

	tools := listTools(t, session)

	names := make([]string, 0, len(tools))
	for name := range tools {
		names = append(names, name)
	}

	want := slices.Concat(readTools, actionTools, writeTools)
	assert.ElementsMatch(t, want, names)
	assert.Len(t, names, 42)

	for _, def := range toolDefs((&fakeServices{}).services()) {
		for _, procedure := range def.procedures {
			for _, forbidden := range []string{".auth.", ".workspace.", "APIKey", "Member"} {
				assert.NotContains(t, procedure, forbidden, def.name)
			}
		}
	}
}

func TestMCP_OAuthLevels(t *testing.T) {
	endpoint, fake := newTestServer(t, true)

	toolNames := func(t *testing.T, key string) []string {
		t.Helper()

		session, _ := connectClient(t, endpoint, key)

		return slices.Sorted(maps.Keys(listTools(t, session)))
	}
	sorted := func(groups ...[]string) []string { return slices.Sorted(slices.Values(slices.Concat(groups...))) }

	assert.Equal(t, sorted(readTools), toolNames(t, oauthReadKey))
	assert.Equal(t, sorted(readTools, actionTools), toolNames(t, oauthActionsKey))
	assert.Equal(t, sorted(readTools, actionTools, writeTools), toolNames(t, oauthWriteKey))
	assert.Equal(t, sorted(readTools, actionTools, writeTools), toolNames(t, editorKey))

	refused := func(t *testing.T, key, tool string, args map[string]any) {
		t.Helper()

		session, _ := connectClient(t, endpoint, key)
		before := len(fake.methods())

		res, err := session.CallTool(t.Context(), &mcp.CallToolParams{Name: tool, Arguments: args})
		if err == nil {
			assert.True(t, res.IsError, tool)
		}

		assert.Len(t, fake.methods(), before, "%s reached the API", tool)
	}

	refused(t, oauthReadKey, "trigger_sync", map[string]any{"connection_id": uuid.NewString()})
	refused(t, oauthReadKey, "delete_source", map[string]any{"id": uuid.NewString()})
	refused(t, oauthActionsKey, "delete_source", map[string]any{"id": uuid.NewString()})

	session, _ := connectClient(t, endpoint, oauthActionsKey)
	res := callTool(t, session, "trigger_sync", map[string]any{"connection_id": uuid.NewString()})
	require.False(t, res.IsError, resultText(t, res))
	assert.Equal(t, "TriggerSync", fake.last().method)
	assert.Equal(t, editorUserID, fake.last().userID)
}

func TestMCP_GetSystemInfo(t *testing.T) {
	t.Run("viewer gets every part", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)

		res := callTool(t, session, "get_system_info", nil)
		require.False(t, res.IsError, resultText(t, res))

		var out struct {
			Version    string `json:"version"`
			SystemInfo struct {
				RegistrationEnabled bool   `json:"registrationEnabled"`
				WorkspacesMode      string `json:"workspacesMode"`
				EmailEnabled        bool   `json:"emailEnabled"`
			} `json:"system_info"`
			RuntimeDefaults struct {
				MaxConcurrentJobs int `json:"maxConcurrentJobs"`
			} `json:"runtime_defaults"`
			PipelineSettings struct {
				MaxJobsPerWorkspace int `json:"maxJobsPerWorkspace"`
			} `json:"pipeline_settings"`
			Executors []struct {
				ID      string `json:"id"`
				Version string `json:"version"`
			} `json:"executors"`
		}
		require.NoError(t, json.Unmarshal([]byte(resultText(t, res)), &out))

		assert.Equal(t, buildinfo.Version(), out.Version)
		assert.True(t, out.SystemInfo.RegistrationEnabled)
		assert.Equal(t, "WORKSPACES_MODE_SINGLE", out.SystemInfo.WorkspacesMode)
		assert.True(t, out.SystemInfo.EmailEnabled)
		assert.Equal(t, 4, out.RuntimeDefaults.MaxConcurrentJobs)
		assert.Equal(t, 7, out.PipelineSettings.MaxJobsPerWorkspace)
		require.Len(t, out.Executors, 1)
		assert.Equal(t, "exec-1", out.Executors[0].ID)
		assert.Equal(t, "v0.15.0", out.Executors[0].Version)

		assert.ElementsMatch(t,
			[]string{"GetSystemInfo", "GetRuntimeDefaults", "GetPipelineSettings", "ListExecutors"},
			fake.methods())
		assert.Equal(t, viewerUserID, fake.last().userID)
	})

	t.Run("a non-member is refused", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, outsiderKey)

		res := callTool(t, session, "get_system_info", nil)
		assert.True(t, res.IsError)
		assert.Contains(t, resultText(t, res), "permission_denied")
		assert.Empty(t, fake.methods())
	})

	t.Run("each part is authorized by its procedure", func(t *testing.T) {
		var procedures []string

		for _, def := range toolDefs((&fakeServices{}).services()) {
			if def.name == "get_system_info" {
				procedures = def.procedures
			}
		}

		assert.ElementsMatch(t, []string{
			pipelinev1connect.SourceServiceGetSystemInfoProcedure,
			pipelinev1connect.SourceServiceGetRuntimeDefaultsProcedure,
			pipelinev1connect.SourceServiceListExecutorsProcedure,
			pipelinev1connect.ConnectionServiceGetPipelineSettingsProcedure,
		}, procedures)
	})

	t.Run("unknown arguments are refused", func(t *testing.T) {
		endpoint, fake := newTestServer(t, true)
		session, _ := connectClient(t, endpoint, viewerKey)

		res := callTool(t, session, "get_system_info", map[string]any{"x": 1})
		assert.True(t, res.IsError)
		assert.Contains(t, resultText(t, res), "invalid arguments")
		assert.Empty(t, fake.methods())
	})
}

func TestMCP_ServerVersion(t *testing.T) {
	endpoint, _ := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, viewerKey)

	info := session.InitializeResult().ServerInfo
	assert.Equal(t, "synclet", info.Name)
	assert.Equal(t, buildinfo.Version(), info.Version)
}
