package app

import (
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/connectrpc/pnprecoverconnectrpchandling"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/http/pnphttpserverrecovery"
	"go.uber.org/fx"
)

// internalHTTPServerModule creates a separate HTTP server for cluster-internal APIs.
// This server is not exposed via ingress. Individual modules register their handlers
// via MuxHandlerRegistrarProvider with per-handler authentication.
func internalHTTPServerModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		"internal-http-server",
		pnphttpserverrecovery.Module(
			pnphttpserverrecovery.WithFxPrivate(),
		),
		pnpconnectrpchandling.Module(pnpconnectrpchandling.WithFxPrivate()),
		pnprecoverconnectrpchandling.Module(pnprecoverconnectrpchandling.WithFxPrivate()),
		pnphttpserver.Module(
			pnphttpserver.WithFxPrivate(),
			pnphttpserver.WithConfigPrefix("INTERNAL_HTTP_SERVER_"),
			pnphttpserver.Start(options.RunInternalHTTPServer),
		),

		// Per-module handler registration.
		pipelineInternalHTTPServerModule(),
	)
}
