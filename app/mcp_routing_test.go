package app

import (
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/go-pnp/go-pnp/pkg/optionutil"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

const mcpInitializeRequest = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18","capabilities":{},"clientInfo":{"name":"test","version":"0"}}}`

// serverHandlers builds the server command's graph with MCP enabled, against
// a database it never connects to, and returns the handler of the public and
// the MCP HTTP server, and the router of the public server.
func serverHandlers(t *testing.T) (public, mcp http.Handler, publicRouter *mux.Router) {
	t.Helper()

	t.Setenv("MCP_ENABLED", "true")
	t.Setenv("ENVIRONMENT", "development")
	t.Setenv("AUTH_JWT_SECRET", "0123456789abcdef0123456789abcdef")
	t.Setenv("SECRET_ENCRYPTION_KEY", "MDEyMzQ1Njc4OWFiY2RlZjAxMjM0NTY3ODlhYmNkZWY=")

	options := optionutil.ApplyOptions(serverDefaultOptions(), ServerCommandOptions(false)...)
	options.RunMCPServer = runsMCPServer(options)
	require.True(t, options.RunMCPServer)

	capture := func(target *http.Handler) []fx.Option {
		return []fx.Option{fx.Invoke(func(server *http.Server) { *target = server.Handler })}
	}
	options.serverFxOptions = map[string][]fx.Option{
		publicHTTPServerModuleName: append(capture(&public), fx.Invoke(func(handler http.Handler) {
			publicRouter, _ = handler.(*mux.Router)
		})),
		mcpHTTPServerModuleName: capture(&mcp),
	}

	app := fx.New(
		NewFxAppOptions(options),
		fx.Decorate(func() (*gorm.DB, error) {
			return gorm.Open(postgres.Open("postgres://synclet@127.0.0.1:1/synclet"), &gorm.Config{DisableAutomaticPing: true})
		}),
	)
	require.NoError(t, app.Err())
	require.NotNil(t, public)
	require.NotNil(t, mcp)
	require.NotNil(t, publicRouter)

	return public, mcp, publicRouter
}

// routesMatching returns every top-level route of router that matches the
// request, whatever the method, so a route registered after another one for
// the same path is still found.
func routesMatching(t *testing.T, router *mux.Router, req *http.Request) []*mux.Route {
	t.Helper()

	var matched []*mux.Route

	err := router.Walk(func(route *mux.Route, _ *mux.Router, ancestors []*mux.Route) error {
		if len(ancestors) > 0 {
			return nil
		}

		var match mux.RouteMatch
		if route.Match(req, &match) || errors.Is(match.MatchErr, mux.ErrMethodMismatch) {
			matched = append(matched, route)
		}

		return nil
	})
	require.NoError(t, err)

	return matched
}

func serve(t *testing.T, handler http.Handler, method, path, body string) (code int, respBody string) {
	t.Helper()

	req := httptest.NewRequestWithContext(t.Context(), method, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")

	rec := httptest.NewRecorder()
	handler.ServeHTTP(rec, req)

	out, err := io.ReadAll(rec.Result().Body)
	require.NoError(t, err)

	return rec.Code, string(out)
}

// TestMCPRouting asserts that only the MCP server answers MCP, and that it
// answers nothing else.
func TestMCPRouting(t *testing.T) {
	public, mcp, publicRouter := serverHandlers(t)

	t.Run("public server never serves MCP", func(t *testing.T) {
		for _, path := range []string{"/mcp", "/mcp/", "/mcp/tools"} {
			req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, path, nil)
			routes := routesMatching(t, publicRouter, req)
			require.Len(t, routes, 1, "%s: only the not-found route may match", path)

			rec := httptest.NewRecorder()
			routes[0].GetHandler().ServeHTTP(rec, req)
			assert.Equal(t, http.StatusNotFound, rec.Code, path)

			code, body := serve(t, public, http.MethodPost, path, mcpInitializeRequest)
			assert.Equal(t, http.StatusNotFound, code, path)
			assert.NotContains(t, body, `"jsonrpc"`, path)
			assert.NotContains(t, body, `"tools"`, path)
		}
	})

	t.Run("MCP server serves only /mcp", func(t *testing.T) {
		code, _ := serve(t, mcp, http.MethodPost, "/mcp", mcpInitializeRequest)
		assert.Equal(t, http.StatusUnauthorized, code, "the MCP endpoint asks for a key")

		for _, path := range []string{
			"/",
			"/index.html",
			"/live",
			"/synclet.publicapi.pipeline.v1.ConnectionService/ListConnections",
			"/synclet.publicapi.auth.v1.AuthService/Login",
		} {
			code, body := serve(t, mcp, http.MethodPost, path, "{}")
			assert.Equal(t, http.StatusNotFound, code, path)
			assert.NotContains(t, body, "<html", path)
		}
	})
}
