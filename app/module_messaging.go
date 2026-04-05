package app

import (
	"github.com/ThreeDotsLabs/watermill/message"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"
	"github.com/go-pnp/jobber"
	"github.com/samber/lo"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox/txoutboxwatermill"
	"go.uber.org/fx"
)

func messagingModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		pnpwatermill.Module(pnpwatermill.WithStart(options.RunConsumers)),
		fx.Provide(
			newStorageMessageProcessors,
		),
		fx.Provide(
			fx.Annotate(newTxOutboxMessagesSender, fx.As(new(txoutbox.MessageSender))),
		),

		// Tx Outbox Publisher
		lo.Ternary(options.RunJobs, pnpjobber.Module(fx.Annotate(txoutbox.NewMessagesProcessor, fx.As(new(jobber.Job)))), fx.Options()),
	)
}
func newTxOutboxMessagesSender(publisher message.Publisher) (txoutbox.MessageSender, error) {
	return txoutboxwatermill.NewMessagesSender(nil, publisher), nil
}

func newStorageMessageProcessors() []txoutbox.MessageProcessor {
	return nil
}
