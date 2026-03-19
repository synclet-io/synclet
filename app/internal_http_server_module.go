package app

import (
	"connectrpc.com/connect"
	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/http/pnphttpserverrecovery"
	"github.com/go-pnp/go-pnp/pkg/ordering"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/gen/proto/synclet/internalapi/executor/v1/executorv1connect"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineconnect"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

type internalHTTPServerConfig struct {
	ExecutorAPIToken string `env:"EXECUTOR_API_TOKEN"`
}

// internalHTTPServerModule creates a separate HTTP server for cluster-internal APIs.
// This server is not exposed via ingress. A shared token (EXECUTOR_API_TOKEN) is required
// when configured; executors must send the token in the X-Internal-Secret header.
// Empty token = skip validation (dev mode).
func internalHTTPServerModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		"internal-http-server",
		pnphttpserverrecovery.Module(
			pnphttpserverrecovery.WithFxPrivate(),
		),
		pnpconnectrpchandling.Module(
			pnpconnectrpchandling.WithFxPrivate(),
		),
		pnphttpserver.Module(
			pnphttpserver.WithFxPrivate(),
			pnphttpserver.WithConfigPrefix("INTERNAL_HTTP_SERVER_"),
			pnphttpserver.Start(options.RunInternalHTTPServer),
		),

		// Executor handlers with token auth.
		fx.Provide(
			configutil.NewPrefixedConfigProvider[internalHTTPServerConfig]("INTERNAL_HTTP_SERVER_"),
			configutil.NewPrefixedConfigInfoProvider[internalHTTPServerConfig]("INTERNAL_HTTP_SERVER_"),
			pnphttpserver.MuxHandlerRegistrarProvider(pnpconnectrpchandling.NewMuxHandlersRegistrar),

			// Token interceptor for internal API per D-06.
			pnpconnectrpchandling.InterceptorProvider(func(cfg *internalHTTPServerConfig) ordering.OrderedItem[connect.Interceptor] {
				return ordering.Ordered[connect.Interceptor](0, connectutil.NewInternalSecretInterceptor(cfg.ExecutorAPIToken))
			}),

			fx.Annotate(pipelineconnect.NewExecutorHandler, fx.As(new(executorv1connect.ExecutorServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(executorv1connect.NewExecutorServiceHandler),

			fx.Private,
		),
	)
}
