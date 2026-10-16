package app

import (
	"time"

	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pkg/ordering"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"
	"github.com/go-pnp/jobber"
	"github.com/samber/lo"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox/txoutboxwatermill"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/pkg/eventbus"
)

type eventsRetryConfig struct {
	MaxRetries      int           `env:"MAX_RETRIES" envDefault:"5"`
	InitialInterval time.Duration `env:"INITIAL_INTERVAL" envDefault:"1s"`
	MaxInterval     time.Duration `env:"MAX_INTERVAL" envDefault:"1m"`
}

type eventsConfig struct {
	Retry eventsRetryConfig `envPrefix:"RETRY_"`
}

func messagingModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		pnpwatermill.Module(pnpwatermill.WithStart(options.RunConsumers)),
		fx.Provide(
			configutil.NewPrefixedConfigProvider[eventsConfig]("EVENTS_"),
			configutil.NewPrefixedConfigInfoProvider[eventsConfig]("EVENTS_"),
			newStorageMessageProcessors,
			pnpwatermill.HandlerMiddlewareProvider(newEventBusMiddleware),
		),
		fx.Provide(
			fx.Annotate(newTxOutboxMessagesSender, fx.As(new(txoutbox.MessageSender))),
		),

		// Tx Outbox Publisher
		lo.Ternary(options.RunJobs, pnpjobber.Module(fx.Annotate(txoutbox.NewMessagesProcessor, fx.As(new(jobber.Job)))), fx.Options()),
	)
}

func newEventBusMiddleware(cfg *eventsConfig, publisher message.Publisher, logger *logging.Logger) (ordering.OrderedItem[message.HandlerMiddleware], error) {
	middleware, err := eventbus.Middleware(eventbus.RetryConfig{
		MaxRetries:      cfg.Retry.MaxRetries,
		InitialInterval: cfg.Retry.InitialInterval,
		MaxInterval:     cfg.Retry.MaxInterval,
	}, publisher, eventbus.NewLoggerAdapter(logger.Named("eventbus")))
	if err != nil {
		return ordering.OrderedItem[message.HandlerMiddleware]{}, err
	}

	return ordering.Ordered(0, middleware), nil
}

func newTxOutboxMessagesSender(publisher message.Publisher) (txoutbox.MessageSender, error) {
	return txoutboxwatermill.NewMessagesSender(nil, publisher), nil
}

func newStorageMessageProcessors() []txoutbox.MessageProcessor {
	return nil
}
