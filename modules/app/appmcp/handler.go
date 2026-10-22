package appmcp

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"connectrpc.com/connect"

	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/gorilla/mux"
	"github.com/modelcontextprotocol/go-sdk/mcp"

	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
	"github.com/synclet-io/synclet/pkg/buildinfo"
)

// Services are the public API handlers the tools call.
type Services struct {
	Sources      pipelinev1connect.SourceServiceHandler
	Destinations pipelinev1connect.DestinationServiceHandler
	Connections  pipelinev1connect.ConnectionServiceHandler
	Jobs         pipelinev1connect.JobServiceHandler
	Workflows    pipelinev1connect.WorkflowServiceHandler
	Stats        statsv1connect.StatsServiceHandler
	Registry     registryv1connect.ConnectorRegistryServiceHandler
}

// Authorizer refuses a caller whose role does not allow a procedure.
type Authorizer interface {
	Authorize(ctx context.Context, procedure string) error
}

// Handler serves the MCP endpoint at /mcp.
type Handler struct {
	endpoint http.Handler
}

// NewHandler serves the tools over Streamable HTTP without session state.
// Every request needs a bearer token one of authenticators accepts; one
// without answers 401 pointing at the protected resource metadata under
// cfg.PublicURL. Every tool call runs in that token's workspace, refused when
// the caller's role does not allow its procedures. A token's access level
// limits the tools it lists and calls: read lists the read tools, actions adds
// the action tools. Write tools are listed only when cfg.WriteTools is set.
// Every level serves the reference resources and sends serverInstructions on
// initialize.
func NewHandler(cfg Config, services Services, authorizer Authorizer, authenticators ...BearerAuthenticator) *Handler {
	defs := toolDefs(services)
	servers := map[string]*mcp.Server{}

	for _, level := range []string{LevelRead, LevelActions, LevelWrite} {
		server := mcp.NewServer(&mcp.Implementation{Name: "synclet", Version: buildinfo.Version()}, &mcp.ServerOptions{Instructions: serverInstructions})
		addReferenceResources(server)

		for _, def := range defs {
			if def.kind == kindWrite && !cfg.WriteTools || !allows(level, def.kind) {
				continue
			}

			server.AddTool(&mcp.Tool{
				Name:        def.name,
				Description: def.description,
				InputSchema: def.schema,
				Annotations: def.annotations(),
			}, toolHandler(def, authorizer))
		}

		servers[level] = server
	}

	endpoint := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server {
			if server, ok := servers[levelOf(r.Context())]; ok {
				return server
			}

			return servers[LevelWrite]
		},
		&mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true},
	)

	resourceMetadataURL := strings.TrimSuffix(cfg.PublicURL, "/") + "/.well-known/oauth-protected-resource"

	return &Handler{endpoint: requireBearer(resourceMetadataURL, authenticators, endpoint)}
}

func toolHandler(def toolDef, authorizer Authorizer) mcp.ToolHandler {
	return func(ctx context.Context, req *mcp.CallToolRequest) (*mcp.CallToolResult, error) {
		if !allows(levelOf(ctx), def.kind) {
			return errorResult(ctx, def.name, errAboveAccessLevel), nil
		}

		for _, procedure := range def.procedures {
			if err := authorizer.Authorize(ctx, procedure); err != nil {
				return errorResult(ctx, def.name, err), nil
			}
		}

		out, err := def.call(ctx, req.Params.Arguments)
		if err != nil {
			return errorResult(ctx, def.name, err), nil
		}

		return &mcp.CallToolResult{Content: []mcp.Content{&mcp.TextContent{Text: string(out)}}}, nil
	}
}

var errAboveAccessLevel = errors.New("this tool is above the access level granted to this client")

// errorResult reports a failed call to the client: invalid arguments and API
// errors with their code and message, anything else as an internal error.
func errorResult(ctx context.Context, tool string, err error) *mcp.CallToolResult {
	var (
		text       string
		connectErr *connect.Error
	)

	switch {
	case errors.Is(err, errInvalidArguments), errors.Is(err, errAboveAccessLevel):
		text = err.Error()
	case errors.As(err, &connectErr):
		text = fmt.Sprintf("%s: %s", connectErr.Code(), connectErr.Message())
	default:
		slog.Default().ErrorContext(ctx, "MCP tool call failed", "tool", tool, "error", err)

		text = "internal error"
	}

	return &mcp.CallToolResult{IsError: true, Content: []mcp.Content{&mcp.TextContent{Text: text}}}
}

func (h *Handler) Register(r *mux.Router) {
	r.Handle("/mcp", h.endpoint)
}

var _ pnphttpserver.MuxHandlerRegistrar = (*Handler)(nil)
