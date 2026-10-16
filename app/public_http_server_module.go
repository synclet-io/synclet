package app

import (
	"context"
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
	"golang.org/x/time/rate"

	"github.com/synclet-io/synclet/modules/app/apphttp"
	"github.com/synclet-io/synclet/modules/auth/authadapt"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/workspace/workspaceadapt"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

const publicCORSConfigPrefix = "PUBLIC_HTTP_SERVER_CORS_"

func publicHTTPServerModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		"http-server",
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
}

func newConnectRPCRateLimitInterceptor(lc fx.Lifecycle, cfg *rateLimitConfig) (ordering.OrderedItem[connect.Interceptor], error) {
	trustedProxies, err := connectutil.ParseTrustedProxies(cfg.TrustedProxies)
	if err != nil {
		return ordering.OrderedItem[connect.Interceptor]{}, fmt.Errorf("RATE_LIMIT_TRUSTED_PROXIES: %w", err)
	}

	rateLimiter := connectutil.NewRateLimitInterceptor(
		map[string]connectutil.RateLimitConfig{
			"/synclet.publicapi.auth.v1.AuthService/Login":        {Rate: rate.Every(6 * time.Second), Burst: 10}, // ~10/min
			"/synclet.publicapi.auth.v1.AuthService/Register":     {Rate: rate.Every(20 * time.Second), Burst: 3}, // ~3/min
			"/synclet.publicapi.auth.v1.AuthService/RefreshToken": {Rate: rate.Every(3 * time.Second), Burst: 20}, // ~20/min
		},
		connectutil.WithTrustedProxies(trustedProxies),
	)

	lc.Append(fx.Hook{
		OnStop: func(ctx context.Context) error {
			rateLimiter.Stop()

			return nil
		},
	})

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
