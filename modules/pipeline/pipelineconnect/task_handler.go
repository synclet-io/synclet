package pipelineconnect

import (
	"context"
	"encoding/json"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"google.golang.org/protobuf/types/known/structpb"

	protocolv1 "github.com/synclet-io/synclet/gen/proto/synclet/protocol/v1"
	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
	"github.com/synclet-io/synclet/pkg/connectutil"
	"github.com/synclet-io/synclet/pkg/protocol"
)

// ConnectorTaskHandler implements the ConnectorTaskService ConnectRPC handler.
type ConnectorTaskHandler struct {
	pipelinev1connect.UnimplementedConnectorTaskServiceHandler
	getTaskResult *pipelinetasks.GetTaskResult
}

// NewConnectorTaskHandler creates a new ConnectorTaskHandler.
func NewConnectorTaskHandler(getTaskResult *pipelinetasks.GetTaskResult) *ConnectorTaskHandler {
	return &ConnectorTaskHandler{getTaskResult: getTaskResult}
}

// GetConnectorTaskResult returns the status and typed result of a connector task.
func (h *ConnectorTaskHandler) GetConnectorTaskResult(ctx context.Context, req *connect.Request[pipelinev1.GetConnectorTaskResultRequest]) (*connect.Response[pipelinev1.GetConnectorTaskResultResponse], error) {
	workspaceID, err := connectutil.WorkspaceIDFromContext(ctx)
	if err != nil {
		return nil, connect.NewError(connect.CodeUnauthenticated, err)
	}

	taskID, err := uuid.Parse(req.Msg.TaskId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, fmt.Errorf("invalid task_id: %w", err))
	}

	result, err := h.getTaskResult.Execute(ctx, pipelinetasks.GetTaskResultParams{
		TaskID:      taskID,
		WorkspaceID: workspaceID,
	})
	if err != nil {
		return nil, mapError(err)
	}

	resp := &pipelinev1.GetConnectorTaskResultResponse{
		Status:       mapTaskStatus(result.Status),
		TaskType:     mapTaskType(result.TaskType),
		ErrorMessage: result.ErrorMessage,
	}

	// Map typed result via one_of interface type switch.
	if result.Result != nil {
		switch r := result.Result.(type) {
		case *pipelineservice.CheckResult:
			resp.Result = &pipelinev1.GetConnectorTaskResultResponse_CheckResult{
				CheckResult: &pipelinev1.CheckTaskResult{
					Success: r.Success,
					Message: r.Message,
				},
			}
		case *pipelineservice.SpecResult:
			resp.Result = &pipelinev1.GetConnectorTaskResultResponse_SpecResult{
				SpecResult: &pipelinev1.SpecTaskResult{Spec: specResultToProto(r)},
			}
		case *pipelineservice.DiscoverResult:
			resp.Result = &pipelinev1.GetConnectorTaskResultResponse_DiscoverResult{
				DiscoverResult: &pipelinev1.DiscoverTaskResult{Catalog: catalogStringToProto(r.Catalog)},
			}
		}
	}

	return connect.NewResponse(resp), nil
}

// mapTaskStatus converts a domain ConnectorTaskStatus to the proto enum.
func mapTaskStatus(s pipelineservice.ConnectorTaskStatus) pipelinev1.ConnectorTaskStatus {
	switch s {
	case pipelineservice.ConnectorTaskStatusPending:
		return pipelinev1.ConnectorTaskStatus_CONNECTOR_TASK_STATUS_PENDING
	case pipelineservice.ConnectorTaskStatusRunning:
		return pipelinev1.ConnectorTaskStatus_CONNECTOR_TASK_STATUS_RUNNING
	case pipelineservice.ConnectorTaskStatusCompleted:
		return pipelinev1.ConnectorTaskStatus_CONNECTOR_TASK_STATUS_COMPLETED
	case pipelineservice.ConnectorTaskStatusFailed:
		return pipelinev1.ConnectorTaskStatus_CONNECTOR_TASK_STATUS_FAILED
	default:
		return pipelinev1.ConnectorTaskStatus_CONNECTOR_TASK_STATUS_UNSPECIFIED
	}
}

// mapTaskType converts a domain ConnectorTaskType to the proto enum.
func mapTaskType(t pipelineservice.ConnectorTaskType) pipelinev1.ConnectorTaskType {
	switch t {
	case pipelineservice.ConnectorTaskTypeCheck:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_CHECK
	case pipelineservice.ConnectorTaskTypeSpec:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_SPEC
	case pipelineservice.ConnectorTaskTypeDiscover:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_DISCOVER
	default:
		return pipelinev1.ConnectorTaskType_CONNECTOR_TASK_TYPE_UNSPECIFIED
	}
}

// specResultToProto converts a domain SpecResult to a typed proto ConnectorSpecification.
func specResultToProto(r *pipelineservice.SpecResult) *protocolv1.ConnectorSpecification {
	spec := &protocolv1.ConnectorSpecification{
		DocumentationUrl:      r.DocumentationURL,
		ChangelogUrl:          r.ChangelogURL,
		SupportsIncremental:   r.SupportsIncremental,
		SupportsNormalization: r.SupportsNormalization,
		SupportsDbt:           r.SupportsDBT,
		ProtocolVersion:       r.ProtocolVersion,
	}

	// ConnectionSpecification: jsonb string -> structpb.Struct
	if r.ConnectionSpecification != "" {
		spec.ConnectionSpecification = rawJSONToStruct(json.RawMessage(r.ConnectionSpecification))
	}

	// SupportedDestinationSyncModes: jsonb string -> []string
	if r.SupportedDestinationSyncModes != "" {
		var modes []string
		if json.Unmarshal([]byte(r.SupportedDestinationSyncModes), &modes) == nil {
			spec.SupportedDestinationSyncModes = modes
		}
	}

	// AdvancedAuth: jsonb string -> protocolv1.AdvancedAuth
	if r.AdvancedAuth != "" {
		spec.AdvancedAuth = parseAdvancedAuth(r.AdvancedAuth)
	}

	return spec
}

// parseAdvancedAuth converts an AdvancedAuth JSON string to a typed proto AdvancedAuth.
func parseAdvancedAuth(jsonStr string) *protocolv1.AdvancedAuth {
	var raw protocol.AdvancedAuth
	if err := json.Unmarshal([]byte(jsonStr), &raw); err != nil {
		return nil
	}

	auth := &protocolv1.AdvancedAuth{
		AuthFlowType:   raw.AuthFlowType,
		PredicateKey:   raw.PredicateKey,
		PredicateValue: raw.PredicateValue,
	}

	if raw.OAuthConfigSpecification != nil {
		auth.OauthConfigSpecification = &protocolv1.OAuthConfigSpecification{
			OauthUserInputFromConnectorConfigSpecification: rawJSONToStruct(raw.OAuthConfigSpecification.OAuthUserInputFromConnectorConfigSpecification),
			CompleteOauthOutputSpecification:               rawJSONToStruct(raw.OAuthConfigSpecification.CompleteOAuthOutputSpecification),
			CompleteOauthServerInputSpecification:          rawJSONToStruct(raw.OAuthConfigSpecification.CompleteOAuthServerInputSpecification),
			CompleteOauthServerOutputSpecification:         rawJSONToStruct(raw.OAuthConfigSpecification.CompleteOAuthServerOutputSpecification),
		}
	}

	return auth
}

// rawJSONToStruct converts raw JSON bytes to a structpb.Struct. Returns nil on failure.
func rawJSONToStruct(data json.RawMessage) *structpb.Struct {
	if len(data) == 0 {
		return nil
	}
	var m map[string]any
	if json.Unmarshal(data, &m) != nil {
		return nil
	}
	s, err := structpb.NewStruct(m)
	if err != nil {
		return nil
	}
	return s
}

// catalogStringToProto converts a catalog JSON string to a typed proto AirbyteCatalog.
func catalogStringToProto(catalogJSON string) *protocolv1.AirbyteCatalog {
	if catalogJSON == "" {
		return nil
	}

	var catalog protocol.AirbyteCatalog
	if err := json.Unmarshal([]byte(catalogJSON), &catalog); err != nil {
		return nil
	}

	return catalogToProto(&catalog)
}

// catalogToProto converts a domain AirbyteCatalog to a typed proto AirbyteCatalog.
func catalogToProto(catalog *protocol.AirbyteCatalog) *protocolv1.AirbyteCatalog {
	if catalog == nil {
		return nil
	}

	pbCatalog := &protocolv1.AirbyteCatalog{}
	for _, s := range catalog.Streams {
		pbStream := &protocolv1.AirbyteStream{
			Name:                s.Name,
			SourceDefinedCursor: s.SourceDefinedCursor,
			DefaultCursorField:  s.DefaultCursorField,
			Namespace:           s.Namespace,
			IsResumable:         s.IsResumable,
			IsFileBased:         s.IsFileBased,
		}

		// JSONSchema: json.RawMessage -> structpb.Struct
		if len(s.JSONSchema) > 0 {
			pbStream.JsonSchema = rawJSONToStruct(s.JSONSchema)
		}

		// SupportedSyncModes: []SyncMode -> []string
		for _, mode := range s.SupportedSyncModes {
			pbStream.SupportedSyncModes = append(pbStream.SupportedSyncModes, string(mode))
		}

		// SourceDefinedPrimaryKey: [][]string -> []*StringList
		for _, pk := range s.SourceDefinedPrimaryKey {
			pbStream.SourceDefinedPrimaryKey = append(pbStream.SourceDefinedPrimaryKey, &protocolv1.StringList{Values: pk})
		}

		pbCatalog.Streams = append(pbCatalog.Streams, pbStream)
	}

	return pbCatalog
}

// specJSONToProto converts a JSON spec string (from ManagedConnector.Spec) to a typed proto ConnectorSpecification.
func specJSONToProto(specJSON string) (*protocolv1.ConnectorSpecification, error) {
	var spec protocol.ConnectorSpecification
	if err := json.Unmarshal([]byte(specJSON), &spec); err != nil {
		return nil, fmt.Errorf("parsing spec JSON: %w", err)
	}

	pbSpec := &protocolv1.ConnectorSpecification{
		DocumentationUrl:      spec.DocumentationURL,
		ChangelogUrl:          spec.ChangelogURL,
		SupportsIncremental:   spec.SupportsIncremental,
		SupportsNormalization: spec.SupportsNormalization,
		SupportsDbt:           spec.SupportsDBT,
		ProtocolVersion:       spec.ProtocolVersion,
	}

	if len(spec.ConnectionSpecification) > 0 {
		pbSpec.ConnectionSpecification = rawJSONToStruct(spec.ConnectionSpecification)
	}

	for _, mode := range spec.SupportedDestinationSyncModes {
		pbSpec.SupportedDestinationSyncModes = append(pbSpec.SupportedDestinationSyncModes, string(mode))
	}

	if spec.AdvancedAuth != nil {
		authJSON, _ := json.Marshal(spec.AdvancedAuth)
		pbSpec.AdvancedAuth = parseAdvancedAuth(string(authJSON))
	}

	return pbSpec, nil
}
