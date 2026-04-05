package app

import (
	"context"

	"github.com/go-pnp/go-pnp/logging"
	"github.com/go-pnp/go-pnp/pnpjobber"
	"github.com/go-pnp/jobber"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinerepositories"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

func pipelineJobsModule(options *RunAppOptions) fx.Option {
	return fx.Options(
		// Background jobs (only when RunJobs is enabled)
		conditionalFxOptions(options.RunJobs, func() fx.Option {
			return fx.Options(
				// Worker modules registered AFTER recovery hook so they start polling
				// only after stale jobs and orphan containers have been cleaned up.
				pnpjobber.Module(newPipelineSyncSchedulerJob),
				pnpjobber.Module(newPipelineStaleJobWatchdogJob),
				pnpjobber.Module(newPipelineStatsRollupJob),
				pnpjobber.Module(newPipelineJobRetentionJob),
				pnpjobber.Module(newConnectorTaskRetentionJob),
				pnpjobber.Module(newStaleConnectorTaskWatchdogJob),
				pnpjobber.Module(newPipelineRepositorySyncJob),
			)
		}),
	)
}

func newPipelineStaleJobWatchdogJob(cfg *pipelineConfig, failStale *pipelinejobs.FailStaleJobs, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "stale_job_watchdog",
		Interval:         cfg.Jobs.WatchdogInterval,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			failed, err := failStale.Execute(ctx)
			if err != nil {
				return err
			}

			if failed > 0 {
				logger.WithField("count", failed).Info(ctx, "watchdog: failed stale jobs")
			}

			return nil
		},
	})
}

func newPipelineSyncSchedulerJob(cfg *pipelineConfig, syncScheduler *pipelinejobs.SyncScheduler) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "sync_scheduler",
		Interval:         cfg.Jobs.SchedulerInterval,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			return syncScheduler.Execute(ctx)
		},
	})
}

func newPipelineJobRetentionJob(cfg *pipelineConfig, cleanup *pipelinejobs.CleanupOldJobs, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "job_retention",
		Interval:         cfg.Jobs.JobRetention,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := cleanup.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "job retention cleanup failed")
			}

			return nil
		},
	})
}

func newPipelineStatsRollupJob(cfg *pipelineConfig, computeRollup *pipelinestats.ComputeStatsRollup, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "stats_rollup",
		Interval:         cfg.Jobs.StatsRollup,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			if err := computeRollup.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "stats rollup failed")
			}

			return nil
		},
	})
}

func newPipelineRepositorySyncJob(cfg *pipelineConfig, syncAll *pipelinerepositories.SyncAllRepositories, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "repository_sync",
		Interval:         cfg.Jobs.RepositorySync,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := syncAll.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "repository sync failed")
			}

			return nil
		},
	})
}

func newConnectorTaskRetentionJob(cfg *pipelineConfig, retain *pipelinetasks.RetainTasks, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "connector_task_retention",
		Interval:         cfg.Jobs.ConnectorTaskRetention,
		StartImmediately: false,
		Job: func(ctx context.Context) error {
			if err := retain.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "connector task retention failed")
			}

			return nil
		},
	})
}

func newStaleConnectorTaskWatchdogJob(cfg *pipelineConfig, failStale *pipelinetasks.FailStaleTasks, logger *logging.Logger) jobber.Job {
	return jobber.NewIntervalJob(jobber.IntervalJobParams{
		Name:             "stale_connector_task_watchdog",
		Interval:         cfg.Jobs.StaleConnectorTaskWatchdog,
		StartImmediately: true,
		Job: func(ctx context.Context) error {
			if err := failStale.Execute(ctx); err != nil {
				logger.WithError(err).Error(ctx, "stale connector task watchdog failed")
			}

			return nil
		},
	})
}
