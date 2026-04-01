package app

import (
	"context"
	"time"

	"github.com/caarlos0/env/v10"
	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/notify/v1/notifyv1connect"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/webhook/v1/webhookv1connect"
	"github.com/synclet-io/synclet/modules/notify/notifyadapt"
	"github.com/synclet-io/synclet/modules/notify/notifyconnect"
	_ "github.com/synclet-io/synclet/modules/notify/notifydbstate"
	"github.com/synclet-io/synclet/modules/notify/notifyservice"
	"github.com/synclet-io/synclet/modules/notify/notifystorage"
)

type smtpConfig struct {
	SMTPHost     string `env:"SMTP_HOST"`
	SMTPPort     int    `env:"SMTP_PORT" envDefault:"587"`
	SMTPUser     string `env:"SMTP_USER"`
	SMTPPassword string `env:"SMTP_PASSWORD"`
	SMTPFrom     string `env:"SMTP_FROM" envDefault:"noreply@synclet.io"`
}

type inviteConfig struct {
	InviteTTL   time.Duration `env:"INVITE_TTL" envDefault:"168h"`
	FrontendURL string        `env:"FRONTEND_URL" envDefault:"http://localhost:5173"`
}

func notifyModule() fx.Option {
	return fx.Options(
		fx.Provide(
			configutil.NewConfigProvider[smtpConfig](env.Options{}),
			configutil.NewConfigProvider[inviteConfig](env.Options{}),
			fx.Annotate(
				func(db *gorm.DB, logger *logging.Logger) *notifystorage.Storages {
					return notifystorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
				},
				fx.As(new(notifyservice.Storage)),
			),
			// Secrets adapter.
			fx.Annotate(notifyadapt.NewSecretsAdapter, fx.As(new(notifyservice.SecretsProvider))),
			// Webhook use cases.
			notifyservice.NewCreateWebhook,
			notifyservice.NewUpdateWebhook,
			notifyservice.NewDeleteWebhook,
			notifyservice.NewListWebhooks,
			notifyservice.NewDeliverWebhook,
			// Notification channel use cases.
			notifyservice.NewCreateChannel,
			notifyservice.NewUpdateChannel,
			notifyservice.NewDeleteChannel,
			notifyservice.NewListChannels,
			notifyservice.NewCreateNotificationRule,
			notifyservice.NewUpdateNotificationRule,
			notifyservice.NewDeleteNotificationRule,
			notifyservice.NewListNotificationRules,
			// Channel deliverers.
			notifyservice.NewSlackChannel,
			notifyservice.NewTelegramChannel,
			notifyservice.NewEmailChannel,
			// Deliverer map for dispatch.
			func(slack *notifyservice.SlackChannel, email *notifyservice.EmailChannel, telegram *notifyservice.TelegramChannel) map[notifyservice.ChannelType]notifyservice.ChannelDeliverer {
				return map[notifyservice.ChannelType]notifyservice.ChannelDeliverer{
					notifyservice.ChannelTypeSlack:    slack,
					notifyservice.ChannelTypeEmail:    email,
					notifyservice.ChannelTypeTelegram: telegram,
				}
			},
			// DeliverNotification and TestChannel use cases.
			func(storage notifyservice.Storage, deliverers map[notifyservice.ChannelType]notifyservice.ChannelDeliverer, logger *logging.Logger) *notifyservice.DeliverNotification {
				return notifyservice.NewDeliverNotification(storage, deliverers, logger.Named("notify"))
			},
			notifyservice.NewTestChannel,
			// EmailSender: SMTP when configured, NoOp otherwise.
			fx.Annotate(
				func(cfg *smtpConfig, logger *logging.Logger) notifyservice.EmailSender {
					if cfg.SMTPHost == "" {
						logger.Named("notify").Warn(context.Background(), "SMTP not configured: email delivery disabled. Set SMTP_HOST to enable.")

						return notifyservice.NewNoOpEmailSender()
					}

					return notifyservice.NewSMTPEmailSender(notifyservice.SMTPConfig{
						Host:     cfg.SMTPHost,
						Port:     cfg.SMTPPort,
						Username: cfg.SMTPUser,
						Password: cfg.SMTPPassword,
						From:     cfg.SMTPFrom,
					})
				},
			),
		),
	)
}

func notifyHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(notifyconnect.NewHandler, fx.As(new(webhookv1connect.WebhookServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(webhookv1connect.NewWebhookServiceHandler),
			fx.Annotate(notifyconnect.NewNotificationHandler, fx.As(new(notifyv1connect.NotificationServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(notifyv1connect.NewNotificationServiceHandler),
			fx.Private,
		),
	)
}
