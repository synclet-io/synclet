package app

import (
	"context"
	"time"

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

type notifySMTPConfig struct {
	Host     string `env:"HOST"`
	Port     int    `env:"PORT" envDefault:"587"`
	User     string `env:"USER"`
	Password string `env:"PASSWORD"`
	From     string `env:"FROM" envDefault:"noreply@synclet.io"`
}
type notifyWebhookConfig struct {
	HTTPTimeout time.Duration `env:"HTTP_TIMEOUT" envDefault:"10s"`
	MaxRetries  int           `env:"MAX_RETRIES" envDefault:"3"`
}
type notifyConfig struct {
	SMTP    notifySMTPConfig    `envPrefix:"SMTP_"`
	Webhook notifyWebhookConfig `envPrefix:"WEBHOOK_"`
}

func notifyModule() fx.Option {
	return fx.Module(
		"notify",
		logging.DecorateNamed("notify"),
		notifyConfigModule(),
		notifyDependenciesModule(),
		notifyUseCasesModule(),
	)
}

func notifyConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[notifyConfig]("NOTIFY_"),
		configutil.NewPrefixedConfigInfoProvider[notifyConfig]("NOTIFY_"),
		newNotifyServiceConfig,
	)
}

func newNotifyServiceConfig(cfg *notifyConfig) notifyservice.Config {
	return notifyservice.Config{
		WebhookHTTPTimeout: cfg.Webhook.HTTPTimeout,
		WebhookMaxRetries:  cfg.Webhook.MaxRetries,
	}
}

func notifyDependenciesModule() fx.Option {
	return fx.Provide(
		fx.Annotate(newNotifyStorage, fx.As(new(notifyservice.Storage))),
		fx.Annotate(notifyadapt.NewSecretsAdapter, fx.As(new(notifyservice.SecretsProvider))),

		newSMTPEmailSender,
	)
}

func notifyUseCasesModule() fx.Option {
	return fx.Provide(
		// Webhook use cases
		notifyservice.NewCreateWebhook,
		notifyservice.NewUpdateWebhook,
		notifyservice.NewDeleteWebhook,
		notifyservice.NewListWebhooks,
		notifyservice.NewDeliverWebhook,

		// Notification channel use cases
		notifyservice.NewCreateChannel,
		notifyservice.NewUpdateChannel,
		notifyservice.NewDeleteChannel,
		notifyservice.NewListChannels,
		notifyservice.NewCreateNotificationRule,
		notifyservice.NewUpdateNotificationRule,
		notifyservice.NewDeleteNotificationRule,
		notifyservice.NewListNotificationRules,

		// Channel deliverers
		notifyservice.NewSlackChannel,
		notifyservice.NewTelegramChannel,
		notifyservice.NewEmailChannel,

		// Deliverer map for dispatch
		newNotifyChannelDeliverers,

		// DeliverNotification and TestChannel
		newNotifyDeliverNotification,
		notifyservice.NewTestChannel,
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

func newNotifyStorage(db *gorm.DB, logger *logging.Logger) *notifystorage.Storages {
	return notifystorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
}

func newNotifyChannelDeliverers(
	slack *notifyservice.SlackChannel,
	email *notifyservice.EmailChannel,
	telegram *notifyservice.TelegramChannel,
) map[notifyservice.ChannelType]notifyservice.ChannelDeliverer {
	return map[notifyservice.ChannelType]notifyservice.ChannelDeliverer{
		notifyservice.ChannelTypeSlack:    slack,
		notifyservice.ChannelTypeEmail:    email,
		notifyservice.ChannelTypeTelegram: telegram,
	}
}

func newNotifyDeliverNotification(
	storage notifyservice.Storage,
	deliverers map[notifyservice.ChannelType]notifyservice.ChannelDeliverer,
	logger *logging.Logger,
) *notifyservice.DeliverNotification {
	return notifyservice.NewDeliverNotification(storage, deliverers, logger.Named("notify"))
}

func newSMTPEmailSender(cfg *notifyConfig, logger *logging.Logger) notifyservice.EmailSender {
	if cfg.SMTP.Host == "" {
		logger.Named("notify").Info(context.Background(), "SMTP not configured: email delivery disabled. Set NOTIFY_SMTP_HOST to enable.")

		return notifyservice.NewNoOpEmailSender()
	}

	return notifyservice.NewSMTPEmailSender(notifyservice.SMTPConfig{
		Host:     cfg.SMTP.Host,
		Port:     cfg.SMTP.Port,
		Username: cfg.SMTP.User,
		Password: cfg.SMTP.Password,
		From:     cfg.SMTP.From,
	})
}
