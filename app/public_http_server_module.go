package app

import (
	"context"
	"log/slog"
	"time"

	"connectrpc.com/connect"
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

func publicHTTPServerModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		"http-server",
		pnphttpserverrecovery.Module(pnphttpserverrecovery.WithFxPrivate()),
		pnphttpservercors.Module(pnphttpservercors.WithFxPrivate()),
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

			// Security headers middleware (order -2, before other middleware).
			pnphttpserver.HandlerMiddlewareProvider(func() ordering.OrderedItem[pnphttpserver.HandlerMiddleware] {
				return ordering.OrderedItem[pnphttpserver.HandlerMiddleware]{
					Value: connectutil.SecurityHeadersMiddleware,
					Order: -2,
				}
			}),

			// Rate limiting interceptor (before auth, ordering -1).
			pnpconnectrpchandling.InterceptorProvider(func(lc fx.Lifecycle) ordering.OrderedItem[connect.Interceptor] {
				rl := connectutil.NewRateLimitInterceptor(
					map[string]connectutil.RateLimitConfig{
						"/synclet.publicapi.auth.v1.AuthService/Login":        {Rate: rate.Every(6 * time.Second), Burst: 10}, // ~10/min
						"/synclet.publicapi.auth.v1.AuthService/Register":     {Rate: rate.Every(20 * time.Second), Burst: 3}, // ~3/min
						"/synclet.publicapi.auth.v1.AuthService/RefreshToken": {Rate: rate.Every(3 * time.Second), Burst: 20}, // ~20/min
					},
				)
				lc.Append(fx.Hook{
					OnStop: func(ctx context.Context) error {
						rl.Stop()
						return nil
					},
				})
				return ordering.Ordered[connect.Interceptor](-1, rl)
			}),

			// Auth interceptor (validates tokens, populates user/workspace context).
			pnpconnectrpchandling.InterceptorProvider(func(
				validateAccessToken *authservice.ValidateAccessToken,
				validateAPIKey *authservice.ValidateAPIKey,
			) ordering.OrderedItem[connect.Interceptor] {
				return ordering.Ordered[connect.Interceptor](0, connectutil.NewAuthInterceptor(
					authadapt.NewTokenValidator(validateAccessToken, validateAPIKey),
				))
			}),

			// Role interceptor (enforces proto-annotated role requirements, ordering 1).
			pnpconnectrpchandling.InterceptorProvider(func(
				checker connectutil.MembershipChecker,
			) ordering.OrderedItem[connect.Interceptor] {
				return ordering.Ordered[connect.Interceptor](1, connectutil.NewRoleInterceptor(checker))
			}),

			// Error classification interceptor (after role check, ordering 2).
			// Logs unhandled errors and converts domain errors to connect codes.
			pnpconnectrpchandling.InterceptorProvider(func() ordering.OrderedItem[connect.Interceptor] {
				return ordering.Ordered[connect.Interceptor](2, connectutil.NewErrorInterceptor(slog.Default()))
			}),

			fx.Private,
		),

		// MembershipChecker adapter as standalone FX type for downstream handler injection.
		fx.Provide(
			fx.Annotate(
				func(mc *workspaceadapt.MembershipChecker) connectutil.MembershipChecker {
					return mc
				},
				fx.As(new(connectutil.MembershipChecker)),
			),
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
	)
}
