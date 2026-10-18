package app

import (
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/connectrpc/pnprecoverconnectrpchandling"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/http/pnphttpserverrecovery"
	"go.uber.org/fx"
)

// internalHTTPServerModule creates a separate HTTP server for cluster-internal
// APIs. This server is not exposed via ingress. handlers mount what the
// process serves on it, with per-handler authentication; the probes of
// healthcheckModule are always among them, so a jobs-only process or an
// executor can be probed too. start is false when the process keeps the
// server unstarted.
func internalHTTPServerModule(start bool, handlers ...fx.Option) fx.Option {
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
			pnphttpserver.Start(start),
		),
		fx.Options(handlers...),
	)
}
