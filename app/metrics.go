package app

import (
	"github.com/go-pnp/go-pnp/http/pnppromhttp"
	"github.com/go-pnp/go-pnp/prometheus/pnpprometheus"
	"go.uber.org/fx"
)

func metricsModule() fx.Option {
	return fx.Options(
		pnpprometheus.Module(),
		pnppromhttp.Module(pnppromhttp.WithFxPrivate()),
	)
}
