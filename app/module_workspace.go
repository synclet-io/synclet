package app

import (
	"context"
	"fmt"

	"github.com/caarlos0/env/v10"
	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"go.uber.org/fx"
	"gorm.io/gorm"

	pipelinev1 "github.com/synclet-io/synclet/gen/proto/synclet/publicapi/pipeline/v1"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/workspace/v1/workspacev1connect"
	"github.com/synclet-io/synclet/modules/workspace/workspaceadapt"
	"github.com/synclet-io/synclet/modules/workspace/workspaceconnect"
	_ "github.com/synclet-io/synclet/modules/workspace/workspacedbstate"
	"github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	"github.com/synclet-io/synclet/modules/workspace/workspacestorage"
)

type workspaceConfig struct {
	WorkspacesMode string `env:"WORKSPACES_MODE" envDefault:"single"`
}

// workspacesMode is a named FX type for workspace mode injection.
type workspacesMode = pipelinev1.WorkspacesMode

func workspaceModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		fx.Provide(
			configutil.NewConfigProvider[workspaceConfig](env.Options{}),
			func(cfg *workspaceConfig) workspacesMode {
				switch cfg.WorkspacesMode {
				case "multi":
					return pipelinev1.WorkspacesMode_WORKSPACES_MODE_MULTI
				default:
					return pipelinev1.WorkspacesMode_WORKSPACES_MODE_SINGLE
				}
			},
			fx.Annotate(
				func(db *gorm.DB, logger *logging.Logger) *workspacestorage.Storages {
					return workspacestorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
				},
				fx.As(new(workspaceservice.Storage)),
			),
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
			fx.Annotate(
				func(
					storage workspaceservice.Storage,
					emailSender workspaceservice.EmailSender,
					userLookup workspaceservice.UserLookup,
					cfg *inviteConfig,
					logger *logging.Logger,
				) *workspaceservice.CreateInvite {
					return workspaceservice.NewCreateInvite(storage, emailSender, userLookup, cfg.InviteTTL, cfg.FrontendURL, logger)
				},
			),
			workspaceservice.NewAcceptInvite,
			workspaceservice.NewDeclineInvite,
			workspaceservice.NewRevokeInvite,
			fx.Annotate(
				func(
					storage workspaceservice.Storage,
					emailSender workspaceservice.EmailSender,
					userLookup workspaceservice.UserLookup,
					cfg *inviteConfig,
					logger *logging.Logger,
				) *workspaceservice.ResendInvite {
					return workspaceservice.NewResendInvite(storage, emailSender, userLookup, cfg.InviteTTL, cfg.FrontendURL, logger)
				},
			),
			workspaceservice.NewListInvites,
			workspaceservice.NewGetInviteByToken,

			// Cross-module adapters
			fx.Annotate(
				workspaceadapt.NewEmailSenderAdapter,
				fx.As(new(workspaceservice.EmailSender)),
			),
			fx.Annotate(
				workspaceadapt.NewUserLookupAdapter,
				fx.As(new(workspaceservice.UserLookup)),
			),
			workspaceadapt.NewMembershipChecker,
		),
		conditionalFxOption(options.RunPublicHTTPServer, func() fx.Option {
			return fx.Invoke(func(
				lc fx.Lifecycle,
				mode workspacesMode,
				bootstrap *workspaceservice.BootstrapDefaultWorkspace,
				logger *logging.Logger,
			) {
				lc.Append(fx.Hook{
					OnStart: func(ctx context.Context) error {
						if mode != pipelinev1.WorkspacesMode_WORKSPACES_MODE_SINGLE {
							return nil
						}

						ws, err := bootstrap.Execute(ctx)
						if err != nil {
							return fmt.Errorf("bootstrapping default workspace: %w", err)
						}

						if ws != nil {
							logger.WithField("id", ws.ID.String()).Info(ctx, "default workspace bootstrapped")
						}

						return nil
					},
				})
			})
		}),
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
