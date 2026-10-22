package appmcp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"
	"testing"

	"connectrpc.com/connect"
	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/mapper"
	"github.com/synclet-io/synclet/pkg/protocol"
)

var referenceURIs = map[string]string{
	"mappers":        "synclet://reference/mappers",
	"runtime_config": "synclet://reference/runtime_config",
}

func TestReferenceTool(t *testing.T) {
	endpoint, _ := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, oauthReadKey)

	require.Contains(t, listTools(t, session), "get_reference")

	for _, topic := range referenceTopics() {
		res := callTool(t, session, "get_reference", map[string]any{"topic": topic.name})
		require.False(t, res.IsError, "%s: %s", topic.name, resultText(t, res))
		assert.Equal(t, topic.text, resultText(t, res), topic.name)
		assert.NotEmpty(t, topic.text, topic.name)
	}

	res := callTool(t, session, "get_reference", map[string]any{"topic": "nope"})
	assert.True(t, res.IsError)
	assert.Contains(t, resultText(t, res), "mappers")
	assert.Contains(t, resultText(t, res), "runtime_config")
}

func TestReferenceResources(t *testing.T) {
	endpoint, _ := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, oauthReadKey)

	listed, err := session.ListResources(t.Context(), nil)
	require.NoError(t, err)

	uris := make([]string, 0, len(listed.Resources))
	for _, resource := range listed.Resources {
		uris = append(uris, resource.URI)
	}

	assert.ElementsMatch(t, []string{referenceURIs["mappers"], referenceURIs["runtime_config"]}, uris)

	for name, uri := range referenceURIs {
		read, err := session.ReadResource(t.Context(), &mcp.ReadResourceParams{URI: uri})
		require.NoError(t, err, name)
		require.Len(t, read.Contents, 1, name)

		tool := callTool(t, session, "get_reference", map[string]any{"topic": name})
		assert.Equal(t, resultText(t, tool), read.Contents[0].Text, name)
		assert.Equal(t, "text/markdown", read.Contents[0].MIMEType, name)
	}
}

func TestToolDescriptionsNameReference(t *testing.T) {
	endpoint, _ := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, oauthWriteKey)
	tools := listTools(t, session)

	for name, topic := range map[string]string{
		"configure_stream":   "mappers",
		"configure_streams":  "mappers",
		"create_source":      "runtime_config",
		"update_source":      "runtime_config",
		"create_destination": "runtime_config",
		"update_destination": "runtime_config",
	} {
		require.Contains(t, tools, name)
		description := tools[name].Description
		assert.Contains(t, description, "get_reference", name)
		assert.Contains(t, description, `"`+topic+`"`, name)
	}
}

func TestInitializeInstructions(t *testing.T) {
	endpoint, _ := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, oauthReadKey)

	instructions := session.InitializeResult().Instructions
	assert.Contains(t, instructions, "get_reference")
	assert.Contains(t, instructions, referenceURIs["mappers"])
	assert.Contains(t, instructions, referenceURIs["runtime_config"])
	assert.Contains(t, instructions, "get_configured_catalog")
	assert.Contains(t, instructions, "validate_mappers")
}

// referenceSampleSchema is the sample stream the mapper reference describes.
const referenceSampleSchema = `{
	"type": "object",
	"properties": {
		"id": {"type": "integer"},
		"email": {"type": ["string", "null"]},
		"amount": {"type": "number"},
		"status": {"type": "string"},
		"created_at": {"type": "string", "format": "date-time"}
	}
}`

var exampleBlock = regexp.MustCompile("(?s)```([^\\n]*)\\n(.*?)```")

// referenceExamples returns the bodies of the fenced blocks whose info string is info.
func referenceExamples(t *testing.T, topic, info string) []string {
	t.Helper()

	var text string

	for _, candidate := range referenceTopics() {
		if candidate.name == topic {
			text = candidate.text
		}
	}

	require.NotEmpty(t, text, topic)

	var out []string

	for _, match := range exampleBlock.FindAllStringSubmatch(text, -1) {
		if strings.TrimSpace(match[1]) == info {
			out = append(out, match[2])
		}
	}

	require.NotEmpty(t, out, "no %q blocks in %s", info, topic)

	return out
}

func TestReferenceMapperExamples(t *testing.T) {
	kinds := map[string]bool{}

	for i, example := range referenceExamples(t, "mappers", "json mappers") {
		var steps []mapper.Step
		require.NoError(t, json.Unmarshal([]byte(example), &steps), "example %d", i)

		var raw []map[string]json.RawMessage
		require.NoError(t, json.Unmarshal([]byte(example), &raw), "example %d", i)

		for n, step := range raw {
			where := fmt.Sprintf("example %d step %d", i, n)
			assertKnownKeys(t, step, reflect.TypeFor[mapper.Step](), where)

			for key := range step {
				if key != "on_error" {
					kinds[key] = true
				}
			}

			assertSpecKeys(t, step["mask"], reflect.TypeFor[mapper.MaskSpec](), where+" mask")
			assertSpecKeys(t, step["cast"], reflect.TypeFor[mapper.CastSpec](), where+" cast")
		}

		stream := &protocol.ConfiguredAirbyteStream{
			Mappers:             json.RawMessage(example),
			SyncMode:            protocol.SyncModeFullRefresh,
			DestinationSyncMode: protocol.DestinationSyncModeAppend,
		}

		_, result, err := pipelineservice.CompileStreamMappers(stream, json.RawMessage(referenceSampleSchema), []byte("salt"))
		require.NoError(t, err, "example %d", i)
		assert.Empty(t, result.Errors, "example %d: %s", i, pipelineservice.FormatMapperIssues(result.Errors))
	}

	for _, kind := range []string{"rename", "cast", "set", "hash", "mask", "filter", "drop"} {
		assert.True(t, kinds[kind], "no example of %s", kind)
	}
}

// assertKnownKeys fails for each key of object that is not a JSON field of typ.
func assertKnownKeys(t *testing.T, object map[string]json.RawMessage, typ reflect.Type, where string) {
	t.Helper()

	known := map[string]bool{}

	for field := range typ.Fields() {
		name, _, _ := strings.Cut(field.Tag.Get("json"), ",")
		known[name] = true
	}

	for key := range object {
		assert.True(t, known[key], "%s: unknown key %q", where, key)
	}
}

// assertSpecKeys checks the keys of every spec object in a field → spec map;
// a spec given as a bare string has none.
func assertSpecKeys(t *testing.T, specs json.RawMessage, typ reflect.Type, where string) {
	t.Helper()

	if specs == nil {
		return
	}

	var byField map[string]json.RawMessage
	require.NoError(t, json.Unmarshal(specs, &byField), where)

	for _, spec := range byField {
		if bytes.HasPrefix(bytes.TrimSpace(spec), []byte("{")) {
			var object map[string]json.RawMessage
			require.NoError(t, json.Unmarshal(spec, &object), where)
			assertKnownKeys(t, object, typ, where)
		}
	}
}

func TestReferenceRuntimeConfigExamples(t *testing.T) {
	var docker, kubernetes bool

	for i, example := range referenceExamples(t, "runtime_config", "json runtime_config") {
		decoder := json.NewDecoder(bytes.NewReader([]byte(example)))
		decoder.DisallowUnknownFields()

		var cfg pipelineservice.RuntimeConfig
		require.NoError(t, decoder.Decode(&cfg), "example %d", i)
		require.NoError(t, pipelineservice.ValidateRuntimeConfig(&cfg), "example %d", i)

		docker = docker || cfg.DockerNetwork != "" || len(cfg.DockerVolumes) > 0
		kubernetes = kubernetes || len(cfg.Tolerations) > 0 || len(cfg.NodeSelector) > 0 || len(cfg.Affinity) > 0
	}

	assert.True(t, docker, "no example sets a Docker key")
	assert.True(t, kubernetes, "no example sets a Kubernetes key")
}

func TestValidateMappersTool(t *testing.T) {
	endpoint, fake := newTestServer(t, true)
	session, _ := connectClient(t, endpoint, oauthReadKey)

	tool, ok := listTools(t, session)["validate_mappers"]
	require.True(t, ok)
	require.NotNil(t, tool.Annotations)
	assert.True(t, tool.Annotations.ReadOnlyHint)

	res := callTool(t, session, "validate_mappers", map[string]any{
		"connection_id": "c-1",
		"stream_name":   "users",
		"mappers":       []any{map[string]any{"drop": []any{"nosuch"}}},
	})
	require.False(t, res.IsError, resultText(t, res))
	assert.Contains(t, resultText(t, res), `field \"nosuch\" does not exist`)

	call := fake.last()
	assert.Equal(t, "ValidateMappers", call.method)

	req := call.req.(*pipelinev1.ValidateMappersRequest)
	assert.Equal(t, "c-1", req.GetConnectionId())
	require.Len(t, req.GetMappers(), 1)
	assert.Contains(t, req.GetMappers()[0].AsMap(), "drop")
	assert.NotContains(t, fake.methods(), "ConfigureStreams")
}

func (f *fakeServices) ValidateMappers(ctx context.Context, r *connect.Request[pipelinev1.ValidateMappersRequest]) (*connect.Response[pipelinev1.ValidateMappersResponse], error) {
	f.record(ctx, "ValidateMappers", r.Msg)

	return connect.NewResponse(&pipelinev1.ValidateMappersResponse{
		Errors: []*pipelinev1.MapperIssue{{Step: 0, Field: "nosuch", Message: `field "nosuch" does not exist`}},
	}), nil
}
