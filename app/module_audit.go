package app

import (
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

func auditModule() fx.Option {
	return fx.Module(
		"audit",
		logging.DecorateNamed("audit"),
		auditDependenciesModule(),
		auditUseCasesModule(),
	)
}

func auditDependenciesModule() fx.Option {
	return fx.Provide(
		fx.Annotate(newAuditStorage, fx.As(new(auditservice.Storage))),
	)
}

func auditUseCasesModule() fx.Option {
	return fx.Provide(
		auditservice.NewRecordAuditEvent,
		auditservice.NewListAuditEvents,
		auditservice.NewGetAuditEvent,
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

func newAuditStorage(db *gorm.DB, logger *logging.Logger) *auditstorage.Storages {
	return auditstorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
}
