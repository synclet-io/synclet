package app

import (
	"context"

	"github.com/caarlos0/env/v10"
	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/gorilla/mux"
	"github.com/saturn4er/boilerplate-go/lib/txoutbox"
	"go.uber.org/fx"
	"gorm.io/gorm"

	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
	"github.com/synclet-io/synclet/modules/auth/authadapt"
	"github.com/synclet-io/synclet/modules/auth/authconnect"
	_ "github.com/synclet-io/synclet/modules/auth/authdbstate"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/auth/authstorage"
	"github.com/synclet-io/synclet/pkg/connectutil"
)

type authConfig struct {
	JWTSecret           string `env:"JWT_SECRET,notEmpty"`
	RegistrationEnabled bool   `env:"REGISTRATION_ENABLED" envDefault:"true"`
	SecureCookies       bool   `env:"SECURE_COOKIES" envDefault:"false"`
}

// oidcCallbackBaseURL is a named type for FX injection disambiguation.
type oidcCallbackBaseURL string

// registrationEnabled is a named type for FX injection of the registration toggle.
type registrationEnabled bool

func authModule() fx.Option {
	return fx.Options(
		fx.Provide(
			configutil.NewConfigProvider[authConfig](env.Options{}),
			fx.Annotate(
				func(db *gorm.DB, logger *logging.Logger) *authstorage.Storages {
					return authstorage.NewStorages(db, logger, []txoutbox.MessageProcessor{})
				},
				fx.As(new(authservice.Storage)),
			),
			// Auth config (shared by use cases that need it).
			func(cfg *authConfig) authservice.Config {
				config := authservice.DefaultConfig()
				config.JWTSecret = cfg.JWTSecret

				return config
			},
			func(cfg *authConfig) registrationEnabled {
				return registrationEnabled(cfg.RegistrationEnabled)
			},
			func(cfg *authConfig) connectutil.CookieConfig {
				return connectutil.CookieConfig{Secure: cfg.SecureCookies}
			},
			// Auth use cases.
			authservice.NewRegister,
			authservice.NewLogin,
			authservice.NewRefreshTokenUC,
			authservice.NewLogout,
			authservice.NewGetUserByID,
			authservice.NewGetUserByEmail,
			authservice.NewUpdateProfile,
			authservice.NewChangePassword,
			authservice.NewCreateAPIKey,
			authservice.NewRevokeAPIKey,
			authservice.NewListAPIKeys,
			authservice.NewValidateAPIKey,
			authservice.NewValidateAccessToken,
			authservice.NewCleanupExpiredTokens,
			fx.Annotate(authadapt.NewWorkspaceAutoAssigner, fx.As(new(authservice.WorkspaceAutoAssigner))),
			func(register *authservice.Register, login *authservice.Login, autoAssigner authservice.WorkspaceAutoAssigner, cfg *workspaceConfig) *authservice.RegisterAndLogin {
				return authservice.NewRegisterAndLogin(register, login, autoAssigner, cfg.WorkspacesMode != "multi")
			},
			authservice.NewLoginWithUserInfo,
			// OIDC: parse config, create providers + state store, provide use cases.
			authservice.NewStateStore,
			func(logger *logging.Logger) (map[string]*authservice.OIDCProvider, oidcCallbackBaseURL) {
				oidcLogger := logger.Named("oidc")

				oidcCfg, err := parseOIDCConfig()
				if err != nil {
					oidcLogger.WithError(err).Error(context.Background(), "failed to parse OIDC config")

					return nil, ""
				}

				if oidcCfg == nil {
					oidcLogger.Info(context.Background(), "OIDC disabled (OIDC_PROVIDERS not set)")

					return nil, ""
				}

				providers := make(map[string]*authservice.OIDCProvider)

				for _, cfg := range oidcCfg.Providers {
					provider, err := authservice.NewOIDCProvider(context.Background(), cfg, oidcCfg.CallbackBaseURL)
					if err != nil {
						oidcLogger.WithError(err).Error(context.Background(), "failed to initialize OIDC provider", "provider", cfg.Slug)

						continue
					}

					providers[cfg.Slug] = provider
					oidcLogger.Info(context.Background(), "OIDC provider initialized", "provider", cfg.Slug)
				}

				return providers, oidcCallbackBaseURL(oidcCfg.CallbackBaseURL)
			},
			authservice.NewGetOIDCProviders,
			authservice.NewStartOIDCLogin,
			func(providers map[string]*authservice.OIDCProvider, stateStore *authservice.StateStore, storage authservice.Storage, config authservice.Config, autoAssigner authservice.WorkspaceAutoAssigner, cfg *workspaceConfig, logger *logging.Logger) *authservice.HandleOIDCCallback {
				return authservice.NewHandleOIDCCallback(providers, stateStore, storage, config, autoAssigner, cfg.WorkspacesMode != "multi", logger)
			},
		),
	)
}

func authHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			func(re registrationEnabled) authconnect.RegistrationEnabled {
				return authconnect.RegistrationEnabled(re)
			},
			fx.Annotate(authconnect.NewHandler, fx.As(new(authv1connect.AuthServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(authv1connect.NewAuthServiceHandler),
			// OIDC HTTP routes (login/callback redirects).
			pnphttpserver.MuxHandlerRegistrarProvider(func(
				startOIDCLogin *authservice.StartOIDCLogin,
				handleOIDCCallback *authservice.HandleOIDCCallback,
				getOIDCProviders *authservice.GetOIDCProviders,
				callbackBaseURL oidcCallbackBaseURL,
				cookieCfg connectutil.CookieConfig,
				logger *logging.Logger,
			) pnphttpserver.MuxHandlerRegistrar {
				if len(getOIDCProviders.Execute()) == 0 {
					// No-op registrar when OIDC is disabled.
					return pnphttpserver.MuxHandlerRegistrarFunc(func(router *mux.Router) {})
				}

				handler := authconnect.NewOIDCHTTPHandler(startOIDCLogin, handleOIDCCallback, string(callbackBaseURL), logger, cookieCfg)

				return handler.RegisterRoutes()
			}),
			fx.Private,
		),
	)
}
