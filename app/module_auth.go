package app

import (
	"context"
	"time"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/connectrpc/pnpconnectrpchandling"
	"github.com/go-pnp/go-pnp/http/pnphttpserver"
	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/watermill/pnpwatermill"
	"github.com/synclet-io/synclet/gen/proto/synclet/publicapi/auth/v1/authv1connect"
	"github.com/synclet-io/synclet/modules/auth/authadapt"
	"github.com/synclet-io/synclet/modules/auth/authconnect"
	_ "github.com/synclet-io/synclet/modules/auth/authdbstate"
	"github.com/synclet-io/synclet/modules/auth/autheventhandle"
	"github.com/synclet-io/synclet/modules/auth/authhttp"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"github.com/synclet-io/synclet/modules/auth/authstorage"
	"github.com/synclet-io/synclet/pkg/connectutil"
	"go.uber.org/fx"
)

type JWTSecret string

func (s *JWTSecret) UnmarshalText(text []byte) error {
	if err := validateJWTSecret("AUTH_JWT_SECRET", text); err != nil {
		return err
	}

	*s = JWTSecret(text)

	return nil
}

type authConfig struct {
	JWTSecret            JWTSecret     `env:"JWT_SECRET,notEmpty"`
	AccessTokenTTL       time.Duration `env:"ACCESS_TOKEN_TTL" envDefault:"15m"`
	RefreshTokenTTL      time.Duration `env:"REFRESH_TOKEN_TTL" envDefault:"168h"`
	RegistrationEnabled  bool          `env:"REGISTRATION_ENABLED" envDefault:"true"`
	SecureCookies        bool          `env:"SECURE_COOKIES" envDefault:"false"`
	OIDCProviders        []string      `env:"OIDC_PROVIDERS" envSeparator:","`
	OIDCCallbackBaseURL  string        `env:"OIDC_CALLBACK_BASE_URL"`
	TokenCleanupInterval time.Duration `env:"TOKEN_CLEANUP_INTERVAL" envDefault:"1h"`
	OIDCStateTTL         time.Duration `env:"OIDC_STATE_TTL" envDefault:"10m"`
	MinPasswordLength    int           `env:"MIN_PASSWORD_LENGTH" envDefault:"8"`
	// PasswordResetCooldown is the minimum time between two password reset emails for one user.
	PasswordResetCooldown time.Duration `env:"PASSWORD_RESET_COOLDOWN" envDefault:"2m"`
}

func authModule(options *RunAppOptions) fx.Option {
	return fx.Module(
		"auth",
		logging.DecorateNamed("auth"),
		authConfigModule(),
		authDependenciesModule(),
		authUseCasesModule(),
		authEventHandlersModule(),
		authJobsModule(options),
	)
}

func authConfigModule() fx.Option {
	return fx.Provide(
		configutil.NewPrefixedConfigProvider[authConfig]("AUTH_"),
		configutil.NewPrefixedConfigInfoProvider[authConfig]("AUTH_"),
		newAuthServiceConfig,
		newCookieConfig,
		parseOIDCProviderConfigs,
	)
}

func authDependenciesModule() fx.Option {
	return fx.Provide(
		fx.Annotate(authstorage.NewStorages, fx.As(new(authservice.Storage))),
		fx.Annotate(authstorage.NewUserIDStorage, fx.As(new(authservice.UserIDStorage))),
		fx.Annotate(authadapt.NewWorkspaceAutoAssigner, fx.As(new(authservice.WorkspaceAutoAssigner))),
		fx.Annotate(authadapt.NewAuditRecorder, fx.As(new(authservice.AuditRecorder))),
		fx.Annotate(authadapt.NewWorkspaceMembershipLister, fx.As(new(authservice.WorkspaceMembershipLister))),
		fx.Annotate(authadapt.NewPasswordResetEmailSender, fx.As(new(authservice.PasswordResetEmailSender))),
		newOIDCProviders,
	)
}

func authUseCasesModule() fx.Option {
	return fx.Provide(
		authservice.NewRegister,
		authservice.NewLogin,
		authservice.NewRefreshTokenUC,
		authservice.NewLogout,
		authservice.NewGetUserByID,
		authservice.NewFindExistingUserIDs,
		authservice.NewListUserWorkspaces,
		authservice.NewGetUserByEmail,
		authservice.NewUpdateProfile,
		authservice.NewChangePassword,
		authservice.NewIssuePasswordResetLink,
		authservice.NewRequestPasswordReset,
		authservice.NewResetPassword,
		authservice.NewCreateAPIKey,
		authservice.NewRevokeAPIKey,
		authservice.NewListAPIKeys,
		authservice.NewValidateAPIKey,
		authservice.NewValidateAccessToken,
		authservice.NewCleanupExpiredTokens,
		authservice.NewRegisterAndLogin,
		authservice.NewLoginWithUserInfo,
		authservice.NewStateStore,
		authservice.NewGetOIDCProviders,
		authservice.NewStartOIDCLogin,
		authservice.NewHandleOIDCCallback,
		authservice.NewPurgeWorkspaceAPIKeys,
	)
}

func authEventHandlersModule() fx.Option {
	return fx.Provide(
		pnpwatermill.HandlerProvider(autheventhandle.NewWorkspaceDeletedHandler),
	)
}

func authHTTPServerModule() fx.Option {
	return fx.Options(
		fx.Provide(
			fx.Annotate(authconnect.NewHandler, fx.As(new(authv1connect.AuthServiceHandler))),
			pnpconnectrpchandling.ConnectHandlerConstructorProvider(authv1connect.NewAuthServiceHandler),
			pnphttpserver.MuxHandlerRegistrarProvider(newOIDCHTTPHandler),
			fx.Private,
		),
	)
}

func newAuthServiceConfig(cfg *authConfig, wsCfg *workspaceConfig, notifyCfg *notifyConfig) authservice.Config {
	return authservice.Config{
		JWTSecret:             string(cfg.JWTSecret),
		AccessTokenTTL:        cfg.AccessTokenTTL,
		RefreshTokenTTL:       cfg.RefreshTokenTTL,
		SingleWorkspaceMode:   WorkspaceModeSingle == wsCfg.Mode,
		RegistrationEnabled:   cfg.RegistrationEnabled,
		OIDCCallbackBaseURL:   cfg.OIDCCallbackBaseURL,
		OIDCStateTTL:          cfg.OIDCStateTTL,
		MinPasswordLength:     cfg.MinPasswordLength,
		EmailEnabled:          notifyCfg.SMTP.Host != "",
		FrontendURL:           wsCfg.FrontendURL,
		PasswordResetCooldown: cfg.PasswordResetCooldown,
	}
}

func newCookieConfig(cfg *authConfig) connectutil.CookieConfig {
	return connectutil.CookieConfig{
		Secure: cfg.SecureCookies,
	}
}

type newOIDCProvidersParams struct {
	fx.In

	ProviderConfigs []authservice.OIDCProviderConfig
	Config          authservice.Config
	Logger          *logging.Logger
}

func newOIDCProviders(params newOIDCProvidersParams) map[string]*authservice.OIDCProvider {
	oidcLogger := params.Logger.Named("oidc")

	if len(params.ProviderConfigs) == 0 {
		return nil
	}

	providers := make(map[string]*authservice.OIDCProvider)

	for _, cfg := range params.ProviderConfigs {
		provider, err := authservice.NewOIDCProvider(context.Background(), cfg, params.Config.OIDCCallbackBaseURL)
		if err != nil {
			oidcLogger.WithError(err).Error(context.Background(), "failed to initialize OIDC provider", "provider", cfg.Slug)

			continue
		}

		providers[cfg.Slug] = provider
		oidcLogger.Info(context.Background(), "OIDC provider initialized", "provider", cfg.Slug)
	}

	return providers
}

type newOIDCHTPHandlerParams struct {
	fx.In

	StartOIDCLogin     *authservice.StartOIDCLogin
	HandleOIDCCallback *authservice.HandleOIDCCallback
	Config             authservice.Config
	CookieCfg          connectutil.CookieConfig
	Logger             *logging.Logger
}

func newOIDCHTTPHandler(params newOIDCHTPHandlerParams) *authhttp.OIDCHTTPHandler {
	return authhttp.NewOIDCHTTPHandler(params.StartOIDCLogin, params.HandleOIDCCallback, params.Config.OIDCCallbackBaseURL, params.Logger, params.CookieCfg)
}
