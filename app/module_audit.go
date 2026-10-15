package app

import (
	"errors"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/audit/v1/auditv1connect"
	"github.com/synclet-io/synclet/modules/audit/auditconnect"
	_ "github.com/synclet-io/synclet/modules/audit/auditdbstate"
	"github.com/synclet-io/synclet/modules/audit/auditservice"
	"github.com/synclet-io/synclet/modules/audit/auditstorage"
)

type auditConfig struct {
	RetentionDays            int           `env:"RETENTION_DAYS" envDefault:"90"`
	RetentionMaxDays         int           `env:"RETENTION_MAX_DAYS" envDefault:"365"`
	RetentionCleanupInterval time.Duration `env:"RETENTION_CLEANUP_INTERVAL" envDefault:"1h"`
}

func auditModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		"audit",
		logging.DecorateNamed("audit"),
		auditConfigModule(),
		auditDependenciesModule(),
		auditUseCasesModule(),
		auditJobsModule(options),
	)
}

func auditConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[auditConfig]("AUDIT_"),
		configutil.NewPrefixedConfigInfoProvider[auditConfig]("AUDIT_"),
		newAuditServiceConfig,
	)
}

func auditDependenciesModule() fx.Option {
	return fx.Provide(
		fx.Annotate(newAuditStorage, fx.As(new(auditservice.Storage))),
		fx.Annotate(auditstorage.NewRetentionStorage, fx.As(new(auditservice.RetentionStorage))),
	)
}

func auditUseCasesModule() fx.Option {
	return fx.Provide(
		auditservice.NewRecordAuditEvent,
		auditservice.NewListAuditEvents,
		auditservice.NewGetAuditEvent,
		auditservice.NewGetAuditRetention,
		auditservice.NewUpdateAuditRetention,
		auditservice.NewCleanupOldAuditEvents,
	)
}

func auditHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(auditconnect.NewHandler, fx.As(new(auditv1connect.AuditServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(auditv1connect.NewAuditServiceHandler),
			fx.Private,
		),
	)
}

func newAuditServiceConfig(cfg *auditConfig) (auditservice.Config, error) {
	if cfg.RetentionDays < 1 {
		return auditservice.Config{}, errors.New("AUDIT_RETENTION_DAYS must be at least 1")
	}

	if cfg.RetentionMaxDays < cfg.RetentionDays {
		return auditservice.Config{}, fmt.Errorf(
			"AUDIT_RETENTION_MAX_DAYS (%d) must not be lower than AUDIT_RETENTION_DAYS (%d)",
			cfg.RetentionMaxDays, cfg.RetentionDays,
		)
	}

	return auditservice.Config{
		DefaultRetentionDays: cfg.RetentionDays,
		MaxRetentionDays:     cfg.RetentionMaxDays,
	}, nil
}

func newAuditStorage(db *gorm.DB, logger *logging.Logger) *auditstorage.Storages {
	return auditstorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
}
