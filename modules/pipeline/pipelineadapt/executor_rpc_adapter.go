package pipelineadapt

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"connectrpc.com/connect"
	"github.com/google/uuid"

	executorv1 "github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	protocolv1 "github.com/synclet-io/synclet/gen/proto/synclet/protocol/v1"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesync"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// tokenInterceptor attaches the shared secret to outgoing requests per D-06.
type tokenInterceptor struct {
	token string
}

func (i *tokenInterceptor) WrapUnary(next connect.UnaryFunc) connect.UnaryFunc {
	return func(ctx context.Context, req connect.AnyRequest) (connect.AnyResponse, error) {
		req.Header().Set("X-Internal-Secret", i.token)
		return next(ctx, req)
	}
}

func (i *tokenInterceptor) WrapStreamingClient(next connect.StreamingClientFunc) connect.StreamingClientFunc {
	return next
}

func (i *tokenInterceptor) WrapStreamingHandler(next connect.StreamingHandlerFunc) connect.StreamingHandlerFunc {
	return next
}

// RPCExecutorBackend implements pipelinesync.ExecutorBackend by calling
// ExecutorService RPCs over HTTP. Used in distributed mode (per D-15).
type RPCExecutorBackend struct {
	client executorv1connect.ExecutorServiceClient
}

// NewRPCExecutorBackend creates a new RPCExecutorBackend with the given API URL and token.
func NewRPCExecutorBackend(apiURL, token string) *RPCExecutorBackend {
	httpClient := &http.Client{Timeout: 30 * time.Second}
	var opts []connect.ClientOption
	if token != "" {
		opts = append(opts, connect.WithInterceptors(&tokenInterceptor{token: token}))
	}
	client := executorv1connect.NewExecutorServiceClient(httpClient, apiURL, opts...)
	return &RPCExecutorBackend{client: client}
}

// ClaimJob calls the ClaimJob RPC and maps the proto response to the domain type.
func (a *RPCExecutorBackend) ClaimJob(ctx context.Context, workerID string) (*pipelinejobs.ClaimJobBundleResult, error) {
	var result *pipelinejobs.ClaimJobBundleResult
	err := a.sendWithRetry(ctx, "ClaimJob", func(ctx context.Context) error {
		resp, callErr := a.client.ClaimJob(ctx, connect.NewRequest(&executorv1.ClaimJobRequest{
			WorkerId: workerID,
		}))
		if callErr != nil {
			return callErr
		}

		if !resp.Msg.HasJob {
			result = nil
			return nil
		}

		// Parse all UUID fields explicitly.
		parsedJobID, parseErr := uuid.Parse(resp.Msg.JobId)
		if parseErr != nil {
			return fmt.Errorf("parsing job_id: %w", parseErr)
		}
		parsedConnID, parseErr := uuid.Parse(resp.Msg.ConnectionId)
		if parseErr != nil {
			return fmt.Errorf("parsing connection_id: %w", parseErr)
		}
		parsedWorkspaceID, parseErr := uuid.Parse(resp.Msg.WorkspaceId)
		if parseErr != nil {
			return fmt.Errorf("parsing workspace_id: %w", parseErr)
		}
		parsedSourceID, parseErr := uuid.Parse(resp.Msg.SourceId)
		if parseErr != nil {
			return fmt.Errorf("parsing source_id: %w", parseErr)
		}
		parsedDestID, parseErr := uuid.Parse(resp.Msg.DestinationId)
		if parseErr != nil {
			return fmt.Errorf("parsing destination_id: %w", parseErr)
		}

		// Map all proto response fields explicitly to ClaimJobBundleResult.
		result = &pipelinejobs.ClaimJobBundleResult{
			Job: &pipelineservice.Job{
				ID:           parsedJobID,
				ConnectionID: parsedConnID,
				JobType:      protoJobTypeToDomain(resp.Msg.JobType),
				MaxAttempts:  int(resp.Msg.MaxAttempts),
				// Other Job fields (Status, StartedAt, AttemptCount, etc.) are zero-valued.
				// This is intentional: executor only needs ID, ConnectionID, MaxAttempts.
				// Status tracking happens server-side via UpdateJobStatus RPC.
			},
			ConnectionID:          parsedConnID,
			WorkspaceID:           parsedWorkspaceID,
			SourceID:              parsedSourceID,
			DestinationID:         parsedDestID,
			SourceImage:           resp.Msg.SourceImage,
			SourceConfig:          resp.Msg.SourceConfig,
			DestImage:             resp.Msg.DestImage,
			DestConfig:            resp.Msg.DestConfig,
			ConfiguredCatalog:     resp.Msg.ConfiguredCatalog,
			StateBlob:             resp.Msg.StateBlob,
			SourceRuntimeConfig:   resp.Msg.SourceRuntimeConfig,
			DestRuntimeConfig:     resp.Msg.DestRuntimeConfig,
			NamespaceDefinition:   protoNamespaceDefToDomain(resp.Msg.NamespaceDefinition),
			CustomNamespaceFormat: resp.Msg.CustomNamespaceFormat,
			StreamPrefix:          resp.Msg.StreamPrefix,
			MaxAttempts:           int(resp.Msg.MaxAttempts),
		}

		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// UpdateJobStatus calls the UpdateJobStatus RPC with retry.
func (a *RPCExecutorBackend) UpdateJobStatus(ctx context.Context, params pipelinesync.UpdateJobStatusParams) error {
	return a.sendWithRetry(ctx, "UpdateJobStatus", func(ctx context.Context) error {
		_, err := a.client.UpdateJobStatus(ctx, connect.NewRequest(&executorv1.UpdateJobStatusRequest{
			JobId:        params.JobID.String(),
			Success:      params.Success,
			ErrorMessage: params.ErrorMessage,
			RecordsRead:  params.RecordsRead,
			BytesSynced:  params.BytesSynced,
			DurationMs:   params.DurationMs,
		}))
		return err
	})
}

// Heartbeat calls the Heartbeat RPC with retry and returns cancellation status.
func (a *RPCExecutorBackend) Heartbeat(ctx context.Context, jobID uuid.UUID, recordsRead, bytesSynced int64) (*pipelinesync.HeartbeatResult, error) {
	var result *pipelinesync.HeartbeatResult
	err := a.sendWithRetry(ctx, "Heartbeat", func(ctx context.Context) error {
		resp, callErr := a.client.Heartbeat(ctx, connect.NewRequest(&executorv1.HeartbeatRequest{
			JobId:       jobID.String(),
			RecordsRead: recordsRead,
			BytesSynced: bytesSynced,
		}))
		if callErr != nil {
			return callErr
		}
		result = &pipelinesync.HeartbeatResult{Cancelled: resp.Msg.Cancelled}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ReportState serializes the state message and sends it via RPC.
// Best-effort: no retry (state will be re-sent on next checkpoint).
func (a *RPCExecutorBackend) ReportState(ctx context.Context, connectionID, jobID uuid.UUID, stateMsg *protocol.AirbyteStateMessage) error {
	stateData, err := json.Marshal(stateMsg)
	if err != nil {
		return fmt.Errorf("marshaling state message: %w", err)
	}

	_, callErr := a.client.ReportState(ctx, connect.NewRequest(&executorv1.ReportStateRequest{
		JobId:        jobID.String(),
		ConnectionId: connectionID.String(),
		StateData:    stateData,
		StateType:    airbyteStateTypeToProto(stateMsg.Type),
	}))
	return callErr
}

// ReportCompletion calls the ReportCompletion RPC with retry.
func (a *RPCExecutorBackend) ReportCompletion(ctx context.Context, params pipelinesync.ReportCompletionParams) error {
	return a.sendWithRetry(ctx, "ReportCompletion", func(ctx context.Context) error {
		_, err := a.client.ReportCompletion(ctx, connect.NewRequest(&executorv1.ReportCompletionRequest{
			JobId:        params.JobID.String(),
			ConnectionId: params.ConnectionID.String(),
			Success:      params.Success,
			ErrorMessage: params.ErrorMessage,
			RecordsRead:  params.RecordsRead,
			BytesSynced:  params.BytesSynced,
			DurationMs:   params.DurationMs,
		}))
		return err
	})
}

// ReportConfigUpdate calls the ReportConfigUpdate RPC.
// Best-effort: no retry (config updates are rare and can be re-triggered).
func (a *RPCExecutorBackend) ReportConfigUpdate(ctx context.Context, connectorType pipelineservice.ConnectorType, connectorID uuid.UUID, config []byte) error {
	_, err := a.client.ReportConfigUpdate(ctx, connect.NewRequest(&executorv1.ReportConfigUpdateRequest{
		ConnectorType: domainConnectorTypeToExecutorProto(connectorType),
		ConnectorId:   connectorID.String(),
		Config:        config,
	}))
	return err
}

// ReportLog calls the ReportLog RPC.
// Best-effort: no retry (log lines are non-critical).
func (a *RPCExecutorBackend) ReportLog(ctx context.Context, jobID uuid.UUID, lines []string) error {
	_, err := a.client.ReportLog(ctx, connect.NewRequest(&executorv1.ReportLogRequest{
		JobId:    jobID.String(),
		LogLines: lines,
	}))
	return err
}

// ClaimConnectorTask calls the ClaimConnectorTask RPC with retry (D-15).
func (a *RPCExecutorBackend) ClaimConnectorTask(ctx context.Context, workerID string) (*pipelinesync.ClaimConnectorTaskResult, error) {
	var result *pipelinesync.ClaimConnectorTaskResult
	err := a.sendWithRetry(ctx, "ClaimConnectorTask", func(ctx context.Context) error {
		resp, callErr := a.client.ClaimConnectorTask(ctx, connect.NewRequest(&executorv1.ClaimConnectorTaskRequest{
			WorkerId: workerID,
		}))
		if callErr != nil {
			return callErr
		}

		if !resp.Msg.HasTask {
			result = nil
			return nil
		}

		parsedTaskID, parseErr := uuid.Parse(resp.Msg.TaskId)
		if parseErr != nil {
			return fmt.Errorf("parsing task_id: %w", parseErr)
		}
		parsedWorkspaceID, parseErr := uuid.Parse(resp.Msg.WorkspaceId)
		if parseErr != nil {
			return fmt.Errorf("parsing workspace_id: %w", parseErr)
		}

		result = &pipelinesync.ClaimConnectorTaskResult{
			TaskID:      parsedTaskID,
			TaskType:    protoTaskTypeToDomain(resp.Msg.TaskType),
			Image:       resp.Msg.Image,
			Config:      resp.Msg.Config,
			WorkspaceID: parsedWorkspaceID,
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return result, nil
}

// ReportConnectorTaskResult calls the ReportConnectorTaskResult RPC with retry (D-18/D-19).
func (a *RPCExecutorBackend) ReportConnectorTaskResult(ctx context.Context, params pipelinesync.ReportConnectorTaskResultParams) error {
	return a.sendWithRetry(ctx, "ReportConnectorTaskResult", func(ctx context.Context) error {
		_, err := a.client.ReportConnectorTaskResult(ctx, connect.NewRequest(&executorv1.ReportConnectorTaskResultRequest{
			TaskId:       params.TaskID.String(),
			Success:      params.Success,
			ErrorMessage: params.ErrorMessage,
			Result:       params.Result,
		}))
		return err
	})
}

// IsJobActive calls the IsJobActive RPC.
// Best-effort: no retry (orphan cleanup can re-check later).
func (a *RPCExecutorBackend) IsJobActive(ctx context.Context, jobID string) (bool, error) {
	resp, err := a.client.IsJobActive(ctx, connect.NewRequest(&executorv1.IsJobActiveRequest{
		JobId: jobID,
	}))
	if err != nil {
		return false, err
	}
	return resp.Msg.Active, nil
}

// sendWithRetry retries a function with exponential backoff per D-12.
// Used for critical operations: ClaimJob, UpdateJobStatus, Heartbeat, ReportCompletion.
func (a *RPCExecutorBackend) sendWithRetry(ctx context.Context, name string, fn func(context.Context) error) error {
	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		err := fn(ctx)
		if err == nil {
			return nil
		}

		slog.Error("rpc backend: call failed, retrying", "rpc", name, "error", err, "backoff", backoff)

		select {
		case <-ctx.Done():
			return fmt.Errorf("%s: %w (last error: %w)", name, ctx.Err(), err)
		case <-time.After(backoff):
		}

		backoff *= 2
		if backoff > maxBackoff {
			backoff = maxBackoff
		}
	}
}

// protoJobTypeToDomain converts an executor proto JobType to the domain enum.
//
//nolint:unparam // maps proto values, will have more cases
func protoJobTypeToDomain(t executorv1.JobType) pipelineservice.JobType {
	switch t {
	case executorv1.JobType_JOB_TYPE_SYNC:
		return pipelineservice.JobTypeSync
	default:
		return pipelineservice.JobTypeSync
	}
}

// protoNamespaceDefToDomain converts an executor proto NamespaceDefinition to the domain enum.
func protoNamespaceDefToDomain(n executorv1.NamespaceDefinition) pipelineservice.NamespaceDefinition {
	switch n {
	case executorv1.NamespaceDefinition_NAMESPACE_DEFINITION_SOURCE:
		return pipelineservice.NamespaceDefinitionSource
	case executorv1.NamespaceDefinition_NAMESPACE_DEFINITION_DESTINATION:
		return pipelineservice.NamespaceDefinitionDestination
	case executorv1.NamespaceDefinition_NAMESPACE_DEFINITION_CUSTOM:
		return pipelineservice.NamespaceDefinitionCustom
	default:
		return pipelineservice.NamespaceDefinitionSource
	}
}

// domainConnectorTypeToExecutorProto converts a domain ConnectorType to the executor proto enum.
func domainConnectorTypeToExecutorProto(t pipelineservice.ConnectorType) executorv1.ConnectorType {
	switch t {
	case pipelineservice.ConnectorTypeSource:
		return executorv1.ConnectorType_CONNECTOR_TYPE_SOURCE
	case pipelineservice.ConnectorTypeDestination:
		return executorv1.ConnectorType_CONNECTOR_TYPE_DESTINATION
	default:
		return executorv1.ConnectorType_CONNECTOR_TYPE_UNSPECIFIED
	}
}

// protoTaskTypeToDomain converts a pipeline proto ConnectorTaskType to the domain enum.
func protoTaskTypeToDomain(t pipelinev1.ConnectorTaskType) pipelineservice.ConnectorTaskType {
	switch t {
	case pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_CHECK:
		return pipelineservice.ConnectorTaskTypeCheck
	case pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_SPEC:
		return pipelineservice.ConnectorTaskTypeSpec
	case pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_DISCOVER:
		return pipelineservice.ConnectorTaskTypeDiscover
	default:
		return pipelineservice.ConnectorTaskTypeCheck
	}
}

// airbyteStateTypeToProto converts an Airbyte state type to the proto StateType enum.
func airbyteStateTypeToProto(t protocol.AirbyteStateType) protocolv1.StateType {
	switch t {
	case protocol.StateTypeStream:
		return protocolv1.StateType_STATE_TYPE_STREAM
	case protocol.StateTypeGlobal:
		return protocolv1.StateType_STATE_TYPE_GLOBAL
	case protocol.StateTypeLegacy:
		return protocolv1.StateType_STATE_TYPE_LEGACY
	default:
		return protocolv1.StateType_STATE_TYPE_STREAM
	}
}
