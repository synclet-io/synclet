package appmcp

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

//go:embed reference/*.md
var referenceFiles embed.FS

// referenceTopic is one page of the reference for the JSON fields a client
// cannot infer from the tool schemas.
type referenceTopic struct {
	name        string
	title       string
	description string
	text        string
}

func referenceTopics() []referenceTopic {
	return []referenceTopic{
		{
			name:        "mappers",
			title:       "Stream mappers",
			description: "The mappers of a configured stream: every step kind with its options and examples, and the CEL expressions they use.",
			text:        referenceText("mappers"),
		},
		{
			name:        "runtime_config",
			title:       "Runtime config",
			description: "The runtime_config of a source or destination: the Docker and Kubernetes keys, their units and examples.",
			text:        referenceText("runtime_config"),
		},
	}
}

func referenceText(name string) string {
	data, err := referenceFiles.ReadFile("reference/" + name + ".md")
	if err != nil {
		panic(fmt.Sprintf("reference %s is not embedded: %v", name, err))
	}

	return string(data)
}

func referenceURI(name string) string {
	return "synclet://reference/" + name
}

func referenceTopicNames() []string {
	topics := referenceTopics()
	names := make([]string, len(topics))

	for i, topic := range topics {
		names[i] = topic.name
	}

	return names
}

var serverInstructions = "Synclet syncs data from sources to destinations through connections. " +
	"Some fields hold JSON whose format the tool schemas do not describe: a stream's mappers and a source's or destination's runtime_config. " +
	`Before writing one, read its reference with get_reference(topic: "mappers") or get_reference(topic: "runtime_config"), ` +
	"also served as the resources " + referenceURI("mappers") + " and " + referenceURI("runtime_config") + ". " +
	"Before any write, read the current state: get_source or get_destination before an update, get_configured_catalog before configure_stream or configure_streams, " +
	"and send back what you keep. Check mappers with validate_mappers before saving them."

func referenceTool() toolDef {
	names := referenceTopicNames()

	return toolDef{
		name: "get_reference",
		description: "Get the reference for a JSON field the tool schemas do not describe, as Markdown: " +
			`"mappers" for the mappers of configure_stream, configure_streams and create_connection, ` +
			`"runtime_config" for the runtime_config of update_source and update_destination. Read it before writing that field.`,
		kind: kindRead,
		schema: map[string]any{
			"type": "object",
			"properties": map[string]any{
				"topic": map[string]any{"type": "string", "enum": names},
			},
			"required": []string{"topic"},
		},
		call: func(_ context.Context, args json.RawMessage) ([]byte, error) {
			var parsed struct {
				Topic string `json:"topic"`
			}

			if len(args) > 0 {
				if err := json.Unmarshal(args, &parsed); err != nil {
					return nil, fmt.Errorf("%w: %w", errInvalidArguments, err)
				}
			}

			for _, topic := range referenceTopics() {
				if topic.name == parsed.Topic {
					return []byte(topic.text), nil
				}
			}

			return nil, fmt.Errorf("%w: unknown topic %q, use one of: %s", errInvalidArguments, parsed.Topic, strings.Join(names, ", "))
		},
	}
}

func addReferenceResources(server *mcp.Server) {
	for _, topic := range referenceTopics() {
		uri := referenceURI(topic.name)

		server.AddResource(&mcp.Resource{
			URI:         uri,
			Name:        topic.name,
			Title:       topic.title,
			Description: topic.description,
			MIMEType:    "text/markdown",
		}, func(context.Context, *mcp.ReadResourceRequest) (*mcp.ReadResourceResult, error) {
			return &mcp.ReadResourceResult{Contents: []*mcp.ResourceContents{{URI: uri, MIMEType: "text/markdown", Text: topic.text}}}, nil
		})
	}
}
