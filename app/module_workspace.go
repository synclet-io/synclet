package app

import (
	"context"
	"fmt"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1/workspacev1connect"
	"github.com/synclet-io/synclet/modules/workspace/workspaceadapt"
	"github.com/synclet-io/synclet/modules/workspace/workspaceconnect"
	_ "github.com/synclet-io/synclet/modules/workspace/workspacedbstate"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/modules/workspace/workspacestorage"
)

const (
	WorkspaceModeSingle WorkspaceMode = iota
	WorkspaceModeMulti
)

type WorkspaceMode byte

func (w *WorkspaceMode) UnmarshalText(text []byte) error {
	switch string(text) {
	case "single":
		*w = WorkspaceModeSingle
	case "multi":
		*w = WorkspaceModeMulti
	default:
		return fmt.Errorf("invalid workspace mode: %s", text)
	}

	return nil
}

type workspaceConfig struct {
	Mode        WorkspaceMode `env:"MODE" envDefault:"single"`
	InviteTTL   time.Duration `env:"INVITE_TTL" envDefault:"168h"`
	FrontendURL string        `env:"FRONTEND_URL" envDefault:"http://localhost:5173"`
}

func workspaceModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		"workspace",
		logging.DecorateNamed("workspace"),
		workspaceConfigModule(),
		workspaceDependenciesModule(),
		workspaceUseCasesModule(),
		conditionalFxOptions(options.RunJobs, func() fx.Option {
			return fx.Invoke(invokeWorkspaceBootstrap)
		}),
	)
}

func workspaceConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[workspaceConfig]("WORKSPACE_"),
		configutil.NewPrefixedConfigInfoProvider[workspaceConfig]("WORKSPACE_"),
		newWorkspaceServiceConfig,
	)
}

func workspaceDependenciesModule() fx.Option {
	return fx.Provide(
		fx.Annotate(newWorkspaceStorage, fx.As(new(workspaceservice.Storage))),
		fx.Annotate(workspaceadapt.NewEmailSenderAdapter, fx.As(new(workspaceservice.EmailSender))),
		fx.Annotate(workspaceadapt.NewUserLookupAdapter, fx.As(new(workspaceservice.UserLookup))),
		workspaceadapt.NewMembershipChecker,
	)
}

func workspaceUseCasesModule() fx.Option {
	return fx.Provide(
		workspaceservice.NewAutoAssignMember,
		workspaceservice.NewCreateWorkspace,
		workspaceservice.NewBootstrapDefaultWorkspace,
		workspaceservice.NewUpdateWorkspace,
		workspaceservice.NewDeleteWorkspace,
		workspaceservice.NewGetWorkspace,
		workspaceservice.NewListWorkspacesForUser,
		workspaceservice.NewRemoveMember,
		workspaceservice.NewGetMembership,
		workspaceservice.NewListMembers,

		// Invite use cases
		workspaceservice.NewCreateInvite,
		workspaceservice.NewAcceptInvite,
		workspaceservice.NewDeclineInvite,
		workspaceservice.NewRevokeInvite,
		workspaceservice.NewResendInvite,
		workspaceservice.NewListInvites,
		workspaceservice.NewGetInviteByToken,
	)
}

func workspaceHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(workspaceconnect.NewHandler, fx.As(new(workspacev1connect.WorkspaceServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(workspacev1connect.NewWorkspaceServiceHandler),
			fx.Private,
		),
	)
}

func newWorkspaceStorage(db *gorm.DB, logger *logging.Logger) *workspacestorage.Storages {
	return workspacestorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
}

func newWorkspaceServiceConfig(cfg *workspaceConfig) workspaceservice.Config {
	return workspaceservice.Config{
		InviteTTL:   cfg.InviteTTL,
		FrontendURL: cfg.FrontendURL,
	}
}

func invokeWorkspaceBootstrap(
	lc fx.Lifecycle,
	wsCfg *workspaceConfig,
	bootstrap *workspaceservice.BootstrapDefaultWorkspace,
	logger *logging.Logger,
) {
	lc.Append(fx.Hook{
		OnStart: func(ctx context.Context) error {
			if WorkspaceModeMulti == wsCfg.Mode {
				return nil
			}

			workspace, err := bootstrap.Execute(ctx)
			if err != nil {
				return fmt.Errorf("bootstrapping default workspace: %w", err)
			}

			if workspace != nil {
				logger.WithField("id", workspace.ID.String()).Info(ctx, "default workspace bootstrapped")
			}

			return nil
		},
	})
}
