package appmcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"slices"
	"strconv"
	"strings"
	"sync"
	"testing"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/proto"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	registryv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	statsv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

var (
	testWorkspaceID = uuid.MustParse("00000000-0000-0000-0000-0000000000aa")
	viewerUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000001")
	editorUserID    = uuid.MustParse("00000000-0000-0000-0000-000000000002")
)

const (
	viewerKey  = "synclet_sk_viewer"
	editorKey  = "synclet_sk_editor"
	expiredKey = "synclet_sk_expired"
	unknownKey = "synclet_sk_unknown"
)

type fakeKeys struct{}

func (fakeKeys) ValidateAPIKey(_ context.Context, rawKey string) (userID, workspaceID string, err error) {
	switch rawKey {
	case viewerKey:
		return viewerUserID.String(), testWorkspaceID.String(), nil
	case editorKey:
		return editorUserID.String(), testWorkspaceID.String(), nil
	case expiredKey:
		return "", "", fmt.Errorf("API key expired: %w", connectutil.ErrInvalidCredential)
	default:
		return "", "", fmt.Errorf("unknown API key: %w", connectutil.ErrInvalidCredential)
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

func (f *fakeServices) ListManagedConnectors(ctx context.Context, r *connect.Request[registryv1.ListManagedConnectorsRequest]) (*connect.Response[registryv1.ListManagedConnectorsResponse], error) {
	return respond[registryv1.ListManagedConnectorsResponse](ctx, f, "ListManagedConnectors", r.Msg)
}

func (f *fakeServices) GetConnectorSpec(ctx context.Context, r *connect.Request[registryv1.GetConnectorSpecRequest]) (*connect.Response[registryv1.GetConnectorSpecResponse], error) {
	return respond[registryv1.GetConnectorSpecResponse](ctx, f, "GetConnectorSpec", r.Msg)
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
		Config{Enabled: true, WriteTools: writeTools},
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
		"get_workflow_run", "get_connection_stats", "get_schema_changes", "list_connectors", "get_connector_spec",
	}
	actionTools = []string{
		"trigger_sync", "cancel_job", "pause_connection", "resume_connection", "run_workflow", "rerun_workflow_run",
	}
	writeTools = []string{
		"create_source", "update_source", "create_destination", "update_destination", "create_connection",
		"update_connection", "configure_streams", "create_workflow", "update_workflow", "delete_source",
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

			_ = resp.Body.Close()

			assert.Equal(t, http.StatusUnauthorized, resp.StatusCode, name)
			assert.Equal(t, "Bearer", resp.Header.Get("WWW-Authenticate"), name)
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
	"list_connections":     "ListConnections",
	"get_connection":       "GetConnection",
	"list_sources":         "ListSources",
	"get_source":           "GetSource",
	"list_destinations":    "ListDestinations",
	"get_destination":      "GetDestination",
	"list_jobs":            "ListJobs",
	"get_job":              "GetJob",
	"list_workflows":       "ListWorkflows",
	"get_workflow":         "GetWorkflow",
	"list_workflow_runs":   "ListWorkflowRuns",
	"get_workflow_run":     "GetWorkflowRun",
	"get_connection_stats": "GetConnectionStats",
	"get_schema_changes":   "GetSchemaChanges",
	"list_connectors":      "ListManagedConnectors",
	"get_connector_spec":   "GetConnectorSpec",
	"trigger_sync":         "TriggerSync",
	"cancel_job":           "CancelJob",
	"pause_connection":     "DisableConnection",
	"resume_connection":    "EnableConnection",
	"run_workflow":         "RunWorkflow",
	"rerun_workflow_run":   "RerunWorkflowRun",
	"create_source":        "CreateSource",
	"update_source":        "UpdateSource",
	"create_destination":   "CreateDestination",
	"update_destination":   "UpdateDestination",
	"create_connection":    "CreateConnection",
	"update_connection":    "UpdateConnection",
	"configure_streams":    "ConfigureStreams",
	"create_workflow":      "CreateWorkflow",
	"update_workflow":      "UpdateWorkflow",
	"delete_source":        "DeleteSource",
	"delete_destination":   "DeleteDestination",
	"delete_connection":    "DeleteConnection",
	"delete_workflow":      "DeleteWorkflow",
}

func assertToolsCallHandlers(t *testing.T, names []string) {
	t.Helper()

	endpoint, fake := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, editorKey)

	for _, name := range names {
		res := callTool(t, session, name, nil)
		require.False(t, res.IsError, "%s: %s", name, resultText(t, res))

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

			if strings.HasPrefix(name, "delete_") {
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
	assert.Len(t, names, 35)

	for _, def := range toolDefs((&fakeServices{}).services()) {
		for _, procedure := range def.procedures {
			for _, forbidden := range []string{".auth.", ".workspace.", "APIKey", "Member"} {
				assert.NotContains(t, procedure, forbidden, def.name)
			}
		}
	}
}
