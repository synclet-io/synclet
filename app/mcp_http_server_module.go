package app

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/http/pnphttpserverrecovery"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1/pipelinev1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/registry/v1/registryv1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/stats/v1/statsv1connect"
	"github.com/synclet-io/synclet/modules/app/appmcp"
	"github.com/synclet-io/synclet/modules/auth/authadapt"
	"github.com/synclet-io/synclet/modules/auth/authhttp"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineconnect"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

const (
	mcpHTTPServerConfigPrefix = "MCP_HTTP_SERVER_"
	mcpHTTPServerModuleName   = "mcp-http-server"
)

// runsMCPServer reports whether a process with options serves MCP: only one
// that serves the public API, and only with MCP_ENABLED set.
func runsMCPServer(options *RunAppOptions) bool {
	return options.RunPublicHTTPServer && appmcp.EnabledFromEnv()
}

// mcpConfigModule describes the MCP configuration whether or not the server
// runs, so `generate dotenv` lists it.
func mcpConfigModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		fx.Provide(
			configutil.NewPrefixedConfigProvider[appmcp.Config](appmcp.ConfigPrefix),
			configutil.NewPrefixedConfigInfoProvider[appmcp.Config](appmcp.ConfigPrefix),
		),
		conditionalFxOptions(!options.RunMCPServer, func() fx.Option {
			return fx.Provide(configutil.NewPrefixedConfigInfoProvider[pnphttpserver.Config](mcpHTTPServerConfigPrefix))
		}),
	)
}

// mcpHTTPServerModule serves the MCP endpoint on its own HTTP server, so it
// can be exposed apart from the public API. Its tools call the public API
// handlers, provided here again for this server alone.
func mcpHTTPServerModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		mcpHTTPServerModuleName,
		pnphttpserverrecovery.Module(pnphttpserverrecovery.WithFxPrivate()),
		pnphttpserver.Module(
			pnphttpserver.WithFxPrivate(),
			pnphttpserver.WithConfigPrefix(mcpHTTPServerConfigPrefix),
			pnphttpserver.Start(true),
		),
		fx.Provide(
			pnphttpserver.HandlerMiddlewareProvider(newHTTPSecurityHeadersMiddleware),
			pnphttpserver.MuxHandlerRegistrarProvider(newMCPHandler),
			pnphttpserver.MuxHandlerRegistrarProvider(newOAuthHTTPHandler),
			newOAuthRegisterLimiter,

			fx.Annotate(pipelineconnect.NewSourceHandler, fx.As(new(pipelinev1connect.SourceServiceHandler))),
			fx.Annotate(pipelineconnect.NewDestinationHandler, fx.As(new(pipelinev1connect.DestinationServiceHandler))),
			fx.Annotate(pipelineconnect.NewConnectionHandler, fx.As(new(pipelinev1connect.ConnectionServiceHandler))),
			fx.Annotate(pipelineconnect.NewJobHandler, fx.As(new(pipelinev1connect.JobServiceHandler))),
			fx.Annotate(pipelineconnect.NewWorkflowHandler, fx.As(new(pipelinev1connect.WorkflowServiceHandler))),
			fx.Annotate(pipelineconnect.NewStatsHandler, fx.As(new(statsv1connect.StatsServiceHandler))),
			fx.Annotate(pipelineconnect.NewRegistryHandler, fx.As(new(registryv1connect.ConnectorRegistryServiceHandler))),
			connectutil.NewRoleInterceptor,
			authadapt.NewTokenValidator,
			fx.Private,
		),
		fx.Options(options.serverFxOptions[mcpHTTPServerModuleName]...),
	)
}

type mcpHandlerParams struct {
	fx.In

	Config          *appmcp.Config
	Sources         pipelinev1connect.SourceServiceHandler
	Destinations    pipelinev1connect.DestinationServiceHandler
	Connections     pipelinev1connect.ConnectionServiceHandler
	Jobs            pipelinev1connect.JobServiceHandler
	Workflows       pipelinev1connect.WorkflowServiceHandler
	Stats           statsv1connect.StatsServiceHandler
	Registry        registryv1connect.ConnectorRegistryServiceHandler
	RoleInterceptor *connectutil.RoleInterceptor
	TokenValidator  *authadapt.TokenValidator
}

func newMCPHandler(params mcpHandlerParams) *appmcp.Handler {
	return appmcp.NewHandler(
		*params.Config,
		appmcp.Services{
			Sources:      params.Sources,
			Destinations: params.Destinations,
			Connections:  params.Connections,
			Jobs:         params.Jobs,
			Workflows:    params.Workflows,
			Stats:        params.Stats,
			Registry:     params.Registry,
		},
		params.RoleInterceptor,
		appmcp.NewAPIKeyAuthenticator(mcpKeyValidator{params.TokenValidator}),
	)
}

// mcpKeyValidator adapts authadapt.TokenValidator to appmcp.APIKeyValidator.
type mcpKeyValidator struct {
	validator *authadapt.TokenValidator
}

func (v mcpKeyValidator) ValidateMCPKey(ctx context.Context, rawKey string) (appmcp.MCPKey, error) {
	key, err := v.validator.ValidateMCPKey(ctx, rawKey)
	if err != nil {
		return appmcp.MCPKey{}, err
	}

	return appmcp.MCPKey{UserID: key.UserID, WorkspaceID: key.WorkspaceID, Level: key.Level, ExpiresAt: key.ExpiresAt}, nil
}

// newOAuthRegisterLimiter limits dynamic client registration, which needs no
// credentials, to 10 a minute per client IP.
func newOAuthRegisterLimiter(store connectutil.RateLimitStore, cfg *rateLimitConfig) (*connectutil.RateLimitInterceptor, error) {
	trustedProxies, err := connectutil.ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return nil, fmt.Errorf("RATE_LIMIT_TRUSTED_PROXIES: %w", err)
	}

	limiter := connectutil.NewRateLimitInterceptor(
		store,
		cfg.limits(map[string]connectutil.RateLimitConfig{
			authhttp.RegisterPath: {Limit: 10, Window: time.Minute},
		}),
		connectutil.WithTrustedProxies(trustedProxies),
	)

	return limiter, nil
}

type oauthHTTPHandlerParams struct {
	fx.In

	Config     *appmcp.Config
	AuthConfig *authConfig
	Limiter    *connectutil.RateLimitInterceptor
	Register   *authservice.RegisterOAuthClient
	Exchange   *authservice.ExchangeOAuthCode
	Refresh    *authservice.RefreshOAuthToken
}

// newOAuthHTTPHandler serves the OAuth discovery, registration and token
// endpoints on the MCP server. The consent page is on the main UI, at
// MCP_UI_URL or else AUTH_OIDC_CALLBACK_BASE_URL.
func newOAuthHTTPHandler(params oauthHTTPHandlerParams) *authhttp.OAuthHTTPHandler {
	uiURL := params.Config.UIURL
	if uiURL == "" {
		uiURL = params.AuthConfig.OIDCCallbackBaseURL
	}

	return authhttp.NewOAuthHTTPHandler(
		authhttp.OAuthURLs{Public: strings.TrimSuffix(params.Config.PublicURL, "/"), UI: strings.TrimSuffix(uiURL, "/")},
		params.Limiter,
		params.Register,
		params.Exchange,
		params.Refresh,
	)
}
