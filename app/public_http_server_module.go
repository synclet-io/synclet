package app

import (
	"fmt"
	"log/slog"
	"time"

	"connectrpc.com/connect"
	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/connectrpc/pnprecoverconnectrpchandling"
	"github.com/go-pnp/go-pnp/http/pnphttphealthcheck"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/http/pnphttpservercors"
	"github.com/go-pnp/go-pnp/http/pnphttpserverrecovery"
	"github.com/go-pnp/go-pnp/pkg/ordering"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/app/apphttp"
	"github.com/synclet-io/synclet/modules/auth/authadapt"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceadapt"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

const (
	publicCORSConfigPrefix     = "PUBLIC_HTTP_SERVER_CORS_"
	publicHTTPServerModuleName = "http-server"
)

func publicHTTPServerModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		publicHTTPServerModuleName,
		pnphttpserverrecovery.Module(pnphttpserverrecovery.WithFxPrivate()),
		pnphttpservercors.Module(
			pnphttpservercors.WithFxPrivate(),
			pnphttpservercors.WithConfigPrefix(publicCORSConfigPrefix),
		),
		pnphttphealthcheck.Module(pnphttphealthcheck.WithFxPrivate()),
		pnpconnectrpchandling.Module(pnpconnectrpchandling.WithFxPrivate()),
		pnprecoverconnectrpchandling.Module(pnprecoverconnectrpchandling.WithFxPrivate()),
		pnphttpserver.Module(
			pnphttpserver.WithFxPrivate(),
			pnphttpserver.WithConfigPrefix("PUBLIC_HTTP_SERVER_"),
			pnphttpserver.Start(options.RunPublicHTTPServer),
		),

		// Shared infrastructure for all handlers.
		fx.Provide(
			pnphttpserver.MuxHandlerRegistrarProvider(pnpconnectrpchandling.NewMuxHandlersRegistrar),
			pnphttpserver.HandlerMiddlewareProvider(newHTTPSecurityHeadersMiddleware),
			pnpconnectrpchandling.InterceptorProvider(newConnectRPCRateLimitInterceptor),
			pnpconnectrpchandling.InterceptorProvider(newConnectRPCAuthInterceptor),
			pnpconnectrpchandling.InterceptorProvider(newConnectRPCRoleInterceptor),
			pnpconnectrpchandling.InterceptorProvider(newConnectRPCErrorInterceptor),
			fx.Private,
		),

		fx.Provide(
			configutil.NewPrefixedConfigProvider[rateLimitConfig]("RATE_LIMIT_"),
			configutil.NewPrefixedConfigInfoProvider[rateLimitConfig]("RATE_LIMIT_"),
		),

		// MembershipChecker adapter as standalone FX type for downstream handler injection.
		fx.Provide(
			fx.Annotate(membershipCheckerAdapter, fx.As(new(connectutil.MembershipChecker))),
		),

		// SPA frontend handler.
		fx.Provide(
			pnphttpserver.MuxHandlerRegistrarProvider(apphttp.NewHandler),
			fx.Private,
		),

		// Per-module handler registration.
		authHTTPServerModule(),
		workspaceHTTPServerModule(),
		pipelineHTTPServerModule(),
		notifyHTTPServerModule(),
		auditHTTPServerModule(),
		fx.Options(options.serverFxOptions[publicHTTPServerModuleName]...),
	)
}

func newHTTPSecurityHeadersMiddleware() ordering.OrderedItem[pnphttpserver.HandlerMiddleware] {
	return ordering.OrderedItem[pnphttpserver.HandlerMiddleware]{
		Value: connectutil.SecurityHeadersMiddleware,
		Order: -2,
	}
}

type rateLimitConfig struct {
	// TrustedProxies lists the IPs or CIDR ranges of reverse proxies whose
	// X-Forwarded-For and X-Real-IP headers identify the client.
	TrustedProxies []string `env:"TRUSTED_PROXIES" envSeparator:","`
	// Disabled turns off the per-IP limits on the auth endpoints and the MCP
	// OAuth client registration. For test environments only.
	Disabled bool `env:"DISABLED" envDefault:"false"`
}

// limits returns the given per-procedure limits, or none when rate limiting
// is disabled.
func (c *rateLimitConfig) limits(limits map[string]connectutil.RateLimitConfig) map[string]connectutil.RateLimitConfig {
	if c.Disabled {
		return nil
	}

	return limits
}

func newConnectRPCRateLimitInterceptor(store connectutil.RateLimitStore, cfg *rateLimitConfig) (ordering.OrderedItem[connect.Interceptor], error) {
	trustedProxies, err := connectutil.ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return ordering.OrderedItem[connect.Interceptor]{}, fmt.Errorf("RATE_LIMIT_TRUSTED_PROXIES: %w", err)
	}

	rateLimiter := connectutil.NewRateLimitInterceptor(
		store,
		cfg.limits(map[string]connectutil.RateLimitConfig{
			"/synclet.publicapi.auth.v1.AuthService/Login":                {Limit: 10, Window: time.Minute},
			"/synclet.publicapi.auth.v1.AuthService/Register":             {Limit: 3, Window: time.Minute},
			"/synclet.publicapi.auth.v1.AuthService/RefreshToken":         {Limit: 20, Window: time.Minute},
			"/synclet.publicapi.auth.v1.AuthService/RequestPasswordReset": {Limit: 10, Window: time.Minute},
			"/synclet.publicapi.auth.v1.AuthService/ResetPassword":        {Limit: 10, Window: time.Minute},
		}),
		connectutil.WithTrustedProxies(trustedProxies),
	)

	return ordering.Ordered[connect.Interceptor](-1, rateLimiter), nil
}

func newConnectRPCAuthInterceptor(
	validateAccessToken *authservice.ValidateAccessToken,
	validateAPIKey *authservice.ValidateAPIKey,
) ordering.OrderedItem[connect.Interceptor] {
	return ordering.Ordered[connect.Interceptor](0, connectutil.NewAuthInterceptor(
		authadapt.NewTokenValidator(validateAccessToken, validateAPIKey),
	))
}

func newConnectRPCRoleInterceptor(checker connectutil.MembershipChecker) ordering.OrderedItem[connect.Interceptor] {
	return ordering.Ordered[connect.Interceptor](1, connectutil.NewRoleInterceptor(checker))
}

func newConnectRPCErrorInterceptor() ordering.OrderedItem[connect.Interceptor] {
	return ordering.Ordered[connect.Interceptor](2, connectutil.NewErrorInterceptor(slog.Default()))
}

func membershipCheckerAdapter(mc *workspaceadapt.MembershipChecker) connectutil.MembershipChecker {
	return mc
}
