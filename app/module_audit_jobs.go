package app

import (
	"context"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/audit/auditservice"
)

func auditJobsModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		conditionalFxOptions(options.RunJobs, func() fx.Option {
			return pnpjobber.Module(newAuditRetentionCleanupJob)
		}),
	)
}

func newAuditRetentionCleanupJob(cfg *auditConfig, cleanup *auditservice.CleanupOldAuditEvents, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "audit_retention_cleanup",
		Interval:         cfg.RetentionCleanupInterval,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := cleanup.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "audit retention cleanup failed")
			}

			return nil
		},
	})
}
