package appmcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	pc "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
)

type toolKind int

const (
	kindRead toolKind = iota
	kindAction
	kindWrite
)

// toolDef is one MCP tool over one or more public API procedures. Its input
// is the protojson form of the request message.
type toolDef struct {
	name        string
	description string
	kind        toolKind
	destructive bool
	procedures  []string
	schema      map[string]any
	call        func(ctx context.Context, args json.RawMessage) ([]byte, error)
}

// errInvalidArguments marks arguments that do not parse into the request.
var errInvalidArguments = errors.New("invalid arguments")

// rpcTool wraps one unary procedure: the arguments unmarshal strictly into
// Req, and the result is the protojson form of Res.
func rpcTool[Req, Res any](
	name, description string,
	kind toolKind,
	procedure string,
	handle func(context.Context, *connect.Request[Req]) (*connect.Response[Res], error),
) toolDef {
	return toolDef{
		name:        name,
		description: description,
		kind:        kind,
		procedures:  []string{procedure},
		schema:      messageSchema(protoMessage[Req]().ProtoReflect().Descriptor()),
		call: func(ctx context.Context, args json.RawMessage) ([]byte, error) {
			req := protoMessage[Req]()
			if err := unmarshalArgs(args, req); err != nil {
				return nil, err
			}

			resp, err := handle(ctx, connect.NewRequest(any(req).(*Req)))
			if err != nil {
				return nil, err
			}

			return protojson.Marshal(any(resp.Msg).(proto.Message))
		},
	}
}

func protoMessage[T any]() proto.Message {
	return any(new(T)).(proto.Message)
}

func unmarshalArgs(args json.RawMessage, msg proto.Message) error {
	if len(args) == 0 || string(args) == "null" {
		return nil
	}

	if err := protojson.Unmarshal(args, msg); err != nil {
		return fmt.Errorf("%w: %w", errInvalidArguments, err)
	}

	return nil
}

func (d toolDef) annotations() *mcp.ToolAnnotations {
	if d.kind == kindRead {
		return &mcp.ToolAnnotations{ReadOnlyHint: true}
	}

	destructive := d.destructive

	return &mcp.ToolAnnotations{ReadOnlyHint: false, DestructiveHint: &destructive}
}

// toolDefs lists every tool. Write tools come last.
func toolDefs(services Services) []toolDef {
	return []toolDef{
		rpcTool("list_connections", "List the workspace's connections with their status, schedule, source and destination.", kindRead,
			pc.ConnectionServiceListConnectionsProcedure, services.Connections.ListConnections),
		rpcTool("get_connection", "Get one connection by id, with its status and schedule.", kindRead,
			pc.ConnectionServiceGetConnectionProcedure, services.Connections.GetConnection),
		rpcTool("list_sources", "List the workspace's sources. Secret config values are masked.", kindRead,
			pc.SourceServiceListSourcesProcedure, services.Sources.ListSources),
		rpcTool("get_source", "Get one source by id. Secret config values are masked.", kindRead,
			pc.SourceServiceGetSourceProcedure, services.Sources.GetSource),
		rpcTool("list_destinations", "List the workspace's destinations. Secret config values are masked.", kindRead,
			pc.DestinationServiceListDestinationsProcedure, services.Destinations.ListDestinations),
		rpcTool("get_destination", "Get one destination by id. Secret config values are masked.", kindRead,
			pc.DestinationServiceGetDestinationProcedure, services.Destinations.GetDestination),
		rpcTool("list_jobs", "List the sync jobs of a connection, newest first, with their attempts.", kindRead,
			pc.JobServiceListJobsProcedure, services.Jobs.ListJobs),
		jobTool(services.Jobs),
		rpcTool("list_workflows", "List the workspace's workflows.", kindRead,
			pc.WorkflowServiceListWorkflowsProcedure, services.Workflows.ListWorkflows),
		rpcTool("get_workflow", "Get one workflow by id, with its steps.", kindRead,
			pc.WorkflowServiceGetWorkflowProcedure, services.Workflows.GetWorkflow),
		rpcTool("list_workflow_runs", "List the runs of a workflow.", kindRead,
			pc.WorkflowServiceListWorkflowRunsProcedure, services.Workflows.ListWorkflowRuns),
		rpcTool("get_workflow_run", "Get one workflow run with its step runs.", kindRead,
			pc.WorkflowServiceGetWorkflowRunProcedure, services.Workflows.GetWorkflowRun),
		rpcTool("get_connection_stats", "Get the sync statistics of a connection.", kindRead,
			statsv1connect.StatsServiceGetConnectionStatsProcedure, services.Stats.GetConnectionStats),
		rpcTool("get_schema_changes", "Get the pending source schema changes of a connection.", kindRead,
			pc.ConnectionServiceGetSchemaChangesProcedure, services.Connections.GetSchemaChanges),
		rpcTool("list_connectors", "List the connectors installed in the workspace; their ids are the managed_connector_id of sources and destinations.", kindRead,
			registryv1connect.ConnectorRegistryServiceListManagedConnectorsProcedure, services.Registry.ListManagedConnectors),
		rpcTool("get_connector_spec", "Get the configuration spec (JSON Schema) of a connector.", kindRead,
			registryv1connect.ConnectorRegistryServiceGetConnectorSpecProcedure, services.Registry.GetConnectorSpec),

		rpcTool("trigger_sync", "Start a sync of a connection now.", kindAction,
			pc.JobServiceTriggerSyncProcedure, services.Jobs.TriggerSync),
		destructive(rpcTool("cancel_job", "Cancel a running or queued job.", kindAction,
			pc.JobServiceCancelJobProcedure, services.Jobs.CancelJob)),
		rpcTool("pause_connection", "Pause a connection: it stops syncing on its schedule.", kindAction,
			pc.ConnectionServiceDisableConnectionProcedure, services.Connections.DisableConnection),
		rpcTool("resume_connection", "Resume a paused connection.", kindAction,
			pc.ConnectionServiceEnableConnectionProcedure, services.Connections.EnableConnection),
		rpcTool("run_workflow", "Start a run of a workflow.", kindAction,
			pc.WorkflowServiceRunWorkflowProcedure, services.Workflows.RunWorkflow),
		rpcTool("rerun_workflow_run", "Rerun a failed workflow run from its failed steps.", kindAction,
			pc.WorkflowServiceRerunWorkflowRunProcedure, services.Workflows.RerunWorkflowRun),

		rpcTool("create_source", "Create a source from a connector and its config.", kindWrite,
			pc.SourceServiceCreateSourceProcedure, services.Sources.CreateSource),
		destructive(rpcTool("update_source", "Update a source. Masked secret values passed back unchanged keep the stored secrets.", kindWrite,
			pc.SourceServiceUpdateSourceProcedure, services.Sources.UpdateSource)),
		rpcTool("create_destination", "Create a destination from a connector and its config.", kindWrite,
			pc.DestinationServiceCreateDestinationProcedure, services.Destinations.CreateDestination),
		destructive(rpcTool("update_destination", "Update a destination. Masked secret values passed back unchanged keep the stored secrets.", kindWrite,
			pc.DestinationServiceUpdateDestinationProcedure, services.Destinations.UpdateDestination)),
		rpcTool("create_connection", "Create a connection between a source and a destination.", kindWrite,
			pc.ConnectionServiceCreateConnectionProcedure, services.Connections.CreateConnection),
		destructive(rpcTool("update_connection", "Update a connection's settings and schedule.", kindWrite,
			pc.ConnectionServiceUpdateConnectionProcedure, services.Connections.UpdateConnection)),
		destructive(rpcTool("configure_streams", "Set which streams a connection syncs and how.", kindWrite,
			pc.ConnectionServiceConfigureStreamsProcedure, services.Connections.ConfigureStreams)),
		rpcTool("create_workflow", "Create a workflow.", kindWrite,
			pc.WorkflowServiceCreateWorkflowProcedure, services.Workflows.CreateWorkflow),
		destructive(rpcTool("update_workflow", "Update a workflow.", kindWrite,
			pc.WorkflowServiceUpdateWorkflowProcedure, services.Workflows.UpdateWorkflow)),
		destructive(rpcTool("delete_source", "Delete a source.", kindWrite,
			pc.SourceServiceDeleteSourceProcedure, services.Sources.DeleteSource)),
		destructive(rpcTool("delete_destination", "Delete a destination.", kindWrite,
			pc.DestinationServiceDeleteDestinationProcedure, services.Destinations.DeleteDestination)),
		destructive(rpcTool("delete_connection", "Delete a connection.", kindWrite,
			pc.ConnectionServiceDeleteConnectionProcedure, services.Connections.DeleteConnection)),
		destructive(rpcTool("delete_workflow", "Delete a workflow.", kindWrite,
			pc.WorkflowServiceDeleteWorkflowProcedure, services.Workflows.DeleteWorkflow)),
	}
}

func destructive(d toolDef) toolDef {
	d.destructive = true

	return d
}

const (
	defaultLogLines = 100
	maxLogLines     = 1000
	logPageSize     = 5000
)

// jobTool gets a job with its attempts and the last log_lines lines of its log.
func jobTool(jobs pc.JobServiceHandler) toolDef {
	schema := messageSchema((&pipelinev1.GetJobRequest{}).ProtoReflect().Descriptor())
	schema["properties"].(map[string]any)["log_lines"] = map[string]any{
		"type":        "integer",
		"minimum":     1,
		"maximum":     maxLogLines,
		"default":     defaultLogLines,
		"description": "How many of the job's last log lines to return.",
	}

	return toolDef{
		name:        "get_job",
		description: "Get a job with its attempts and the tail of its log.",
		kind:        kindRead,
		procedures:  []string{pc.JobServiceGetJobProcedure, pc.JobServiceGetJobLogsProcedure},
		schema:      schema,
		call: func(ctx context.Context, args json.RawMessage) ([]byte, error) {
			logLines, rest, err := splitLogLines(args)
			if err != nil {
				return nil, err
			}

			req := &pipelinev1.GetJobRequest{}
			if err := unmarshalArgs(rest, req); err != nil {
				return nil, err
			}

			job, err := jobs.GetJob(ctx, connect.NewRequest(req))
			if err != nil {
				return nil, err
			}

			tail, err := logTail(ctx, jobs, req.GetJobId(), logLines)
			if err != nil {
				return nil, err
			}

			jobJSON, err := protojson.Marshal(job.Msg)
			if err != nil {
				return nil, err
			}

			return json.Marshal(struct {
				Job      json.RawMessage `json:"job"`
				LogLines []string        `json:"log_lines"`
			}{Job: jobJSON, LogLines: tail})
		},
	}
}

// splitLogLines takes log_lines out of the arguments and returns the rest.
func splitLogLines(args json.RawMessage) (int, json.RawMessage, error) {
	if len(args) == 0 || string(args) == "null" {
		return defaultLogLines, nil, nil
	}

	var fields map[string]json.RawMessage
	if err := json.Unmarshal(args, &fields); err != nil {
		return 0, nil, fmt.Errorf("%w: %w", errInvalidArguments, err)
	}

	logLines := defaultLogLines

	if raw, ok := fields["log_lines"]; ok {
		if err := json.Unmarshal(raw, &logLines); err != nil {
			return 0, nil, fmt.Errorf("%w: log_lines: %w", errInvalidArguments, err)
		}

		delete(fields, "log_lines")
	}

	if logLines < 1 || logLines > maxLogLines {
		return 0, nil, fmt.Errorf("%w: log_lines must be between 1 and %d", errInvalidArguments, maxLogLines)
	}

	rest, err := json.Marshal(fields)
	if err != nil {
		return 0, nil, err
	}

	return logLines, rest, nil
}

// logTail pages through a job's log and keeps its last n lines.
func logTail(ctx context.Context, jobs pc.JobServiceHandler, jobID string, n int) ([]string, error) {
	var (
		tail    []string
		afterID int64
	)

	for {
		page, err := jobs.GetJobLogs(ctx, connect.NewRequest(&pipelinev1.GetJobLogsRequest{
			JobId:   jobID,
			AfterId: afterID,
			Limit:   logPageSize,
		}))
		if err != nil {
			return nil, err
		}

		tail = append(tail, page.Msg.GetLines()...)
		if len(tail) > n {
			tail = tail[len(tail)-n:]
		}

		if !page.Msg.GetHasMore() || page.Msg.GetLastId() <= afterID {
			return tail, nil
		}

		afterID = page.Msg.GetLastId()
	}
}
