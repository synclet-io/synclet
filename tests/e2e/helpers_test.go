//go:build e2e

package e2e

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/exec"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"google.golang.org/protobuf/types/known/structpb"

	authv1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	workspacev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1/workspacev1connect"
)

const (
	testSourceImage = "synclet-test-source"
	testDestImage   = "synclet-test-dest"
)

// syncletServer manages a synclet server subprocess.
type syncletServer struct {
	cmd     *exec.Cmd
	baseURL string
	cancel  context.CancelFunc
}

func startSyncletServer(t *testing.T, env map[string]string) *syncletServer {
	t.Helper()

	// Find two free ports (public + internal HTTP servers).
	listener, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "finding free port for public server")
	port := listener.Addr().(*net.TCPAddr).Port
	listener.Close()

	listener2, err := net.Listen("tcp", ":0")
	require.NoError(t, err, "finding free port for internal server")
	internalPort := listener2.Addr().(*net.TCPAddr).Port
	listener2.Close()

	ctx, cancel := context.WithCancel(context.Background())
	cmd := exec.CommandContext(ctx, syncletBinary, "server", "--standalone")
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PUBLIC_HTTP_SERVER_ADDR=:%d", port),
		fmt.Sprintf("INTERNAL_HTTP_SERVER_ADDR=:%d", internalPort),
		fmt.Sprintf("DB_DSN=%s", testDSN),
		"SECRET_ENCRYPTION_KEY=PjsJSmk5MpLbSjsXYdIeflREhQ57B7+Sz0b0/VreDRk=",
		"JWT_SECRET=e2e-test-jwt-secret-key-that-is-long-enough",
		"INTERNAL_HTTP_SERVER_INTERNAL_API_SECRET=e2e-test-internal-secret",
	)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	require.NoError(t, cmd.Start(), "starting synclet server")

	baseURL := fmt.Sprintf("http://localhost:%d", port)
	waitForServer(t, baseURL, 30*time.Second)

	return &syncletServer{cmd: cmd, baseURL: baseURL, cancel: cancel}
}

func (s *syncletServer) Stop() {
	s.cancel()
	_ = s.cmd.Wait()
}

// waitForServer polls the server until it responds or timeout expires.
func waitForServer(t *testing.T, baseURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	client := &http.Client{Timeout: 2 * time.Second}

	for time.Now().Before(deadline) {
		resp, err := client.Get(baseURL + "/health")
		if err == nil {
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("server at %s did not become ready within %s", baseURL, timeout)
}

// apiClient wraps ConnectRPC clients with auth token.
type apiClient struct {
	auth        authv1connect.AuthServiceClient
	workspace   workspacev1connect.WorkspaceServiceClient
	source      pipelinev1connect.SourceServiceClient
	destination pipelinev1connect.DestinationServiceClient
	connection  pipelinev1connect.ConnectionServiceClient
	job         pipelinev1connect.JobServiceClient
	token       string
	workspaceID string
}

func newAPIClient(baseURL string) *apiClient {
	httpClient := http.DefaultClient
	return &apiClient{
		auth:        authv1connect.NewAuthServiceClient(httpClient, baseURL),
		workspace:   workspacev1connect.NewWorkspaceServiceClient(httpClient, baseURL),
		source:      pipelinev1connect.NewSourceServiceClient(httpClient, baseURL),
		destination: pipelinev1connect.NewDestinationServiceClient(httpClient, baseURL),
		connection:  pipelinev1connect.NewConnectionServiceClient(httpClient, baseURL),
		job:         pipelinev1connect.NewJobServiceClient(httpClient, baseURL),
	}
}

// newAuthRequest creates a connect.Request with the auth and workspace headers set.
func newAuthRequest[T any](c *apiClient, msg *T) *connect.Request[T] {
	req := connect.NewRequest(msg)
	req.Header().Set("Authorization", "Bearer "+c.token)
	if c.workspaceID != "" {
		req.Header().Set("Workspace-Id", c.workspaceID)
	}
	return req
}

// RegisterAndLogin registers a test user, logs in, and stores the token and workspace ID.
func (c *apiClient) RegisterAndLogin(ctx context.Context, t *testing.T) {
	t.Helper()

	email := fmt.Sprintf("e2e-%d@test.local", time.Now().UnixNano())
	password := "TestPassword123!"

	// Register.
	regResp, err := c.auth.Register(ctx, connect.NewRequest(&authv1.RegisterRequest{
		Email:    email,
		Password: password,
		Name:     "E2E Test User",
	}))
	require.NoError(t, err, "registering test user")
	c.token = regResp.Msg.GetAccessToken()

	// Create a workspace for the test user.
	userID := regResp.Msg.GetUser().GetId()
	wsName := fmt.Sprintf("e2e-workspace-%d", time.Now().UnixNano())
	wsCreateResp, err := c.workspace.CreateWorkspace(ctx, newAuthRequest(c, &workspacev1.CreateWorkspaceRequest{
		Name:        wsName,
		OwnerUserId: userID,
	}))
	require.NoError(t, err, "creating workspace")
	c.workspaceID = wsCreateResp.Msg.GetWorkspace().GetId()
}

// CreateSource creates a source with the test source image and returns its ID.
func (c *apiClient) CreateSource(ctx context.Context, t *testing.T, name string, config map[string]any) string {
	t.Helper()
	configStruct := mapToStruct(t, config)

	resp, err := c.source.CreateSource(ctx, newAuthRequest(c, &pipelinev1.CreateSourceRequest{
		WorkspaceId:      c.workspaceID,
		Name:             name,
		ConnectorImage:   testSourceImage,
		ConnectorVersion: "latest",
		Config:           configStruct,
	}))
	require.NoError(t, err, "creating source")
	return resp.Msg.GetSource().GetId()
}

// CreateDestination creates a destination with the test dest image and returns its ID.
func (c *apiClient) CreateDestination(ctx context.Context, t *testing.T, name string, config map[string]any) string {
	t.Helper()
	configStruct := mapToStruct(t, config)

	resp, err := c.destination.CreateDestination(ctx, newAuthRequest(c, &pipelinev1.CreateDestinationRequest{
		WorkspaceId:      c.workspaceID,
		Name:             name,
		ConnectorImage:   testDestImage,
		ConnectorVersion: "latest",
		Config:           configStruct,
	}))
	require.NoError(t, err, "creating destination")
	return resp.Msg.GetDestination().GetId()
}

// CreateConnection creates a connection and returns its ID.
func (c *apiClient) CreateConnection(ctx context.Context, t *testing.T, name string, sourceID, destID string) string {
	t.Helper()
	resp, err := c.connection.CreateConnection(ctx, newAuthRequest(c, &pipelinev1.CreateConnectionRequest{
		WorkspaceId:   c.workspaceID,
		Name:          name,
		SourceId:      sourceID,
		DestinationId: destID,
	}))
	require.NoError(t, err, "creating connection")
	return resp.Msg.GetConnection().GetId()
}

// ConfigureStreams configures streams on a connection.
func (c *apiClient) ConfigureStreams(ctx context.Context, t *testing.T, connectionID string, streams []streamOpt) {
	t.Helper()

	var protoStreams []*pipelinev1.ConfiguredStream
	for _, s := range streams {
		cs := &pipelinev1.ConfiguredStream{
			StreamName:          s.Name,
			Namespace:           s.Namespace,
			SyncMode:            parseSyncMode(s.SyncMode),
			DestinationSyncMode: parseDestSyncMode(s.DestinationSyncMode),
		}
		if s.CursorField != "" {
			cs.CursorField = []string{s.CursorField}
		}
		protoStreams = append(protoStreams, cs)
	}

	_, err := c.connection.ConfigureStreams(ctx, newAuthRequest(c, &pipelinev1.ConfigureStreamsRequest{
		ConnectionId: connectionID,
		Streams:      protoStreams,
	}))
	require.NoError(t, err, "configuring streams")
}

// TriggerSync triggers a sync and returns the job ID.
func (c *apiClient) TriggerSync(ctx context.Context, t *testing.T, connectionID string) string {
	t.Helper()
	resp, err := c.job.TriggerSync(ctx, newAuthRequest(c, &pipelinev1.TriggerSyncRequest{
		ConnectionId: connectionID,
	}))
	require.NoError(t, err, "triggering sync")
	return resp.Msg.GetJob().GetId()
}

// WaitForJob polls GetJob until the job reaches a terminal state.
func (c *apiClient) WaitForJob(ctx context.Context, t *testing.T, jobID string, timeout time.Duration) *syncResult {
	t.Helper()
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			t.Fatalf("context cancelled while waiting for job %s", jobID)
		case <-ticker.C:
			if time.Now().After(deadline) {
				t.Fatalf("job %s did not complete within %s", jobID, timeout)
			}

			resp, err := c.job.GetJob(ctx, newAuthRequest(c, &pipelinev1.GetJobRequest{
				JobId: jobID,
			}))
			require.NoError(t, err, "getting job status")

			job := resp.Msg.GetJob()
			status := job.GetStatus()

			switch status {
			case pipelinev1.JobStatus_JOB_STATUS_COMPLETED:
				return &syncResult{
					JobStatus:   "completed",
					RecordsRead: extractRecordsRead(job),
				}
			case pipelinev1.JobStatus_JOB_STATUS_FAILED:
				return &syncResult{
					JobStatus: "failed",
					JobError:  job.GetError(),
					Err:       fmt.Errorf("job failed: %s", job.GetError()),
				}
			case pipelinev1.JobStatus_JOB_STATUS_CANCELLED:
				return &syncResult{
					JobStatus: "cancelled",
					Err:       fmt.Errorf("job cancelled"),
				}
			}
			// Still pending/running, keep polling.
		}
	}
}

// ListStreamStates returns stream states for a connection keyed by "namespace.stream_name",
// along with the overall state type (STREAM, GLOBAL, LEGACY).
func (c *apiClient) ListStreamStates(ctx context.Context, t *testing.T, connectionID string) (map[string]map[string]any, string) {
	t.Helper()
	resp, err := c.connection.ListStreamStates(ctx, newAuthRequest(c, &pipelinev1.ListStreamStatesRequest{
		ConnectionId: connectionID,
	}))
	require.NoError(t, err, "listing stream states")

	states := make(map[string]map[string]any)
	for _, s := range resp.Msg.GetStates() {
		key := s.GetStreamName()
		if ns := s.GetStreamNamespace(); ns != "" {
			key = ns + "." + key
		}
		var stateData map[string]any
		if err := json.Unmarshal([]byte(s.GetStateData()), &stateData); err == nil {
			states[key] = stateData
		}
	}
	return states, resp.Msg.GetStateType()
}

// syncResult holds the outcome of a sync run.
type syncResult struct {
	// API-based modes (Docker, K8s).
	JobStatus    string
	JobError     string
	RecordsRead  int64
	StreamStates map[string]map[string]any
	StateType    string // "STREAM", "GLOBAL", or "LEGACY"

	// CLI mode.
	ExitCode  int
	StateFile string

	// Common.
	Err error
}

// syncOpts configures a sync run across all modes.
type syncOpts struct {
	SourceConfig map[string]any
	DestConfig   map[string]any
	Streams      []streamOpt
	FullRefresh  bool
}

// streamOpt configures a single stream.
type streamOpt struct {
	Name                string
	Namespace           string
	SyncMode            string // "full_refresh" or "incremental"
	DestinationSyncMode string // "overwrite", "append", "append_dedup"
	CursorField         string
}

// modeRunner abstracts sync execution across Docker, CLI, and K8s modes.
type modeRunner interface {
	RunSync(t *testing.T, ctx context.Context, opts syncOpts) *syncResult
	Cleanup(t *testing.T)
}

// parseSyncMode converts a string sync mode to proto enum.
func parseSyncMode(mode string) pipelinev1.SyncMode {
	switch mode {
	case "incremental":
		return pipelinev1.SyncMode_SYNC_MODE_INCREMENTAL
	default:
		return pipelinev1.SyncMode_SYNC_MODE_FULL_REFRESH
	}
}

// parseDestSyncMode converts a string destination sync mode to proto enum.
func parseDestSyncMode(mode string) pipelinev1.DestinationSyncMode {
	switch mode {
	case "overwrite":
		return pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_OVERWRITE
	case "append_dedup":
		return pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND_DEDUP
	default:
		return pipelinev1.DestinationSyncMode_DESTINATION_SYNC_MODE_APPEND
	}
}

// DiscoverSourceSchema triggers schema discovery for a source.
func (c *apiClient) DiscoverSourceSchema(ctx context.Context, t *testing.T, sourceID string) {
	t.Helper()
	_, err := c.source.DiscoverSourceSchema(ctx, newAuthRequest(c, &pipelinev1.DiscoverSourceSchemaRequest{
		SourceId: sourceID,
	}))
	require.NoError(t, err, "discovering source schema")
}

// extractRecordsRead sums RecordsRead from all job attempts.
func extractRecordsRead(job *pipelinev1.Job) int64 {
	var total int64
	for _, attempt := range job.GetAttempts() {
		if stats := attempt.GetSyncStats(); stats != nil {
			total += stats.GetRecordsRead()
		}
	}
	return total
}

// mapToStruct converts a Go map to a protobuf Struct.
func mapToStruct(t *testing.T, m map[string]any) *structpb.Struct {
	t.Helper()
	data, err := json.Marshal(m)
	require.NoError(t, err, "marshaling config to JSON")

	s := &structpb.Struct{}
	require.NoError(t, s.UnmarshalJSON(data), "unmarshaling JSON to protobuf Struct")
	return s
}
