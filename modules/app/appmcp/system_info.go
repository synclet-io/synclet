package appmcp

import (
	"context"
	"encoding/json"

	"connectrpc.com/connect"
	"google.golang.org/protobuf/encoding/protojson"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	pc "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/pkg/buildinfo"
)

func systemInfoTool(sources pc.SourceServiceHandler, connections pc.ConnectionServiceHandler) toolDef {
	return toolDef{
		name: "get_system_info",
		description: "Get what this Synclet instance is: its version, whether registration and email are enabled, " +
			"single or multi workspace mode, the runtime defaults and pipeline settings that shape every sync, " +
			"and the connected executors with their versions.",
		kind: kindRead,
		procedures: []string{
			pc.SourceServiceGetSystemInfoProcedure,
			pc.SourceServiceGetRuntimeDefaultsProcedure,
			pc.SourceServiceListExecutorsProcedure,
			pc.ConnectionServiceGetPipelineSettingsProcedure,
		},
		schema: messageSchema((&pipelinev1.GetSystemInfoRequest{}).ProtoReflect().Descriptor()),
		call: func(ctx context.Context, args json.RawMessage) ([]byte, error) {
			req := &pipelinev1.GetSystemInfoRequest{}
			if err := unmarshalArgs(args, req); err != nil {
				return nil, err
			}

			info, err := sources.GetSystemInfo(ctx, connect.NewRequest(req))
			if err != nil {
				return nil, err
			}

			defaults, err := sources.GetRuntimeDefaults(ctx, connect.NewRequest(&pipelinev1.GetRuntimeDefaultsRequest{}))
			if err != nil {
				return nil, err
			}

			settings, err := connections.GetPipelineSettings(ctx, connect.NewRequest(&pipelinev1.GetPipelineSettingsRequest{}))
			if err != nil {
				return nil, err
			}

			executors, err := sources.ListExecutors(ctx, connect.NewRequest(&pipelinev1.ListExecutorsRequest{}))
			if err != nil {
				return nil, err
			}

			infoJSON, err := protojson.Marshal(info.Msg)
			if err != nil {
				return nil, err
			}

			defaultsJSON, err := protojson.Marshal(defaults.Msg.GetRuntimeDefaults())
			if err != nil {
				return nil, err
			}

			settingsJSON, err := protojson.Marshal(settings.Msg.GetSettings())
			if err != nil {
				return nil, err
			}

			executorsJSON := make([]json.RawMessage, 0, len(executors.Msg.GetExecutors()))

			for _, executor := range executors.Msg.GetExecutors() {
				executorJSON, err := protojson.Marshal(executor)
				if err != nil {
					return nil, err
				}

				executorsJSON = append(executorsJSON, executorJSON)
			}

			return json.Marshal(struct {
				Version          string            `json:"version"`
				SystemInfo       json.RawMessage   `json:"system_info"`
				RuntimeDefaults  json.RawMessage   `json:"runtime_defaults"`
				PipelineSettings json.RawMessage   `json:"pipeline_settings"`
				Executors        []json.RawMessage `json:"executors"`
			}{
				Version:          buildinfo.Version(),
				SystemInfo:       infoJSON,
				RuntimeDefaults:  defaultsJSON,
				PipelineSettings: settingsJSON,
				Executors:        executorsJSON,
			})
		},
	}
}
