package app

import (
	"context"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"github.com/synclet-io/synclet/modules/auth/authservice"
	"go.uber.org/fx"
)

func authJobsModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		conditionalFxOptions(options.RunJobs, func() fx.Option {
			return pnpjobber.Module(newAuthTokenCleanupJob)
		}),
	)
}

func newAuthTokenCleanupJob(cfg *authConfig, cleanup *authservice.CleanupExpiredTokens, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "auth_token_cleanup",
		Interval:         cfg.TokenCleanupInterval,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := cleanup.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "auth token cleanup failed")
			}

			return nil
		},
	})
}
