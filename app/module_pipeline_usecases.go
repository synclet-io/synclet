package app

import (
	"github.com/go-pnp/go-pnp/logging"
	"go.uber.org/fx"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineexec/pipelineexecdocker"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinecatalog"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconfig"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnections"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelineconnectors"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinedestinations"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinejobs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinelogs"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinerepositories"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesettings"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestate"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinestats"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinetasks"
)

func pipelineUseCasesModule() fx.Option {
	return fx.Module(
		"pipeline.use_cases",
		logging.DecorateNamed("pipeline.use_cases"),

		// Connector use cases
		fx.Provide(
			pipelineconnectors.NewAddConnector,
			pipelineconnectors.NewGetConnector,
			pipelineconnectors.NewGetConnectorSpec,
			pipelineconnectors.NewListConnectors,
			pipelineconnectors.NewDeleteConnector,
			pipelineconnectors.NewUpdateManagedConnector,
			pipelineconnectors.NewBatchUpdateConnectors,
			pipelineconnectors.NewListConnectorsWithUpdateInfo,
			pipelineconnectors.NewGetConnectorWithUpdateInfo,
		),

		// Repository use cases
		fx.Provide(
			pipelinerepositories.NewValidateImage,
			pipelinerepositories.NewRegistryFetcher,
			pipelinerepositories.NewSyncRepository,
			pipelinerepositories.NewAddRepository,
			pipelinerepositories.NewListRepositories,
			pipelinerepositories.NewDeleteRepository,
			pipelinerepositories.NewListRepositoryConnectors,
			pipelinerepositories.NewGetConnectorVersions,
			pipelinerepositories.NewSyncAllRepositories,
			pipelinerepositories.NewCreateDefaultRepositories,
		),

		// Source use cases
		fx.Provide(
			pipelinesources.NewCreateSource,
			pipelinesources.NewUpdateSource,
			pipelinesources.NewDeleteSource,
			pipelinesources.NewGetSource,
			pipelinesources.NewListSources,
			pipelinesources.NewUpdateSourceInternal,
		),

		// Destination use cases
		fx.Provide(
			pipelinedestinations.NewCreateDestination,
			pipelinedestinations.NewUpdateDestination,
			pipelinedestinations.NewDeleteDestination,
			pipelinedestinations.NewGetDestination,
			pipelinedestinations.NewListDestinations,
			pipelinedestinations.NewUpdateDestinationInternal,
		),

		// Connection use cases
		fx.Provide(
			pipelineconnections.NewCreateConnection,
			pipelineconnections.NewUpdateConnection,
			pipelineconnections.NewDeleteConnection,
			pipelineconnections.NewGetConnection,
			pipelineconnections.NewListConnections,
			pipelineconnections.NewUpdateConnectionStatus,
			pipelineconnections.NewEnableConnection,
			pipelineconnections.NewDisableConnection,
		),

		// Config import/export use cases
		fx.Provide(
			pipelineconfig.NewExportConfig,
			pipelineconfig.NewImportConfig,
		),

		// Catalog use cases
		fx.Provide(
			pipelinecatalog.NewDiscoverCatalog,
			pipelinecatalog.NewGetConfiguredCatalog,
			pipelinecatalog.NewConfigureStreams,
			pipelinecatalog.NewDetectSchemaChanges,
			pipelinecatalog.NewGetDiscoveredCatalogForConnection,
			pipelinecatalog.NewGetSourceCatalog,
			pipelinecatalog.NewPopulateGenerationIDs,
		),

		// Job use cases
		fx.Provide(
			pipelinejobs.NewIsTaskActive,
			pipelinejobs.NewGetLaunchBundle,
			pipelinejobs.NewCountConnectionJobs,
			pipelinejobs.NewQueueJob,
			pipelinejobs.NewGetJob,
			pipelinejobs.NewListJobs,
			pipelinejobs.NewListJobAttempts,
			pipelinejobs.NewCancelJob,
			pipelinejobs.NewClaimJob,
			pipelinejobs.NewUpdateJobStatus,
			pipelinejobs.NewUpdateHeartbeat,
			pipelinejobs.NewFailStaleJobs,
			pipelinejobs.NewSetK8sJobName,
			pipelinejobs.NewTriggerSync,
			pipelinejobs.NewCancelJobForWorkspace,
			pipelinejobs.NewGetJobWithAttempts,
			pipelinejobs.NewListJobsWithAttempts,
			pipelinejobs.NewReportCompletion,
			pipelinejobs.NewHandleConfigUpdate,
			fx.Annotate(
				pipelinesettings.NewGetWorkspaceSettings,
				fx.As(new(pipelinejobs.SettingsProvider)),
				fx.As(fx.Self()),
			),
			pipelinesettings.NewUpdateWorkspaceSettings,
			pipelinejobs.NewCleanupOldJobs,
			pipelinejobs.NewClaimJobBundle,
			pipelinejobs.NewCheckJobCancelled,
		),

		// Connector task use cases
		fx.Provide(
			pipelinetasks.NewCreateCheckTask,
			pipelinetasks.NewCreateSpecTask,
			pipelinetasks.NewCreateDiscoverTask,
			pipelinetasks.NewGetTaskResult,
			pipelinetasks.NewWaitForTaskResult,
			pipelinetasks.NewClaimTask,
			pipelinetasks.NewReportTaskResult,
			pipelinetasks.NewRetainTasks,
			pipelinetasks.NewFailStaleTasks,
		),

		// State use cases
		fx.Provide(
			pipelinestate.NewGetSyncState,
			pipelinestate.NewSaveSyncState,
			pipelinestate.NewClearSyncState,
			pipelinestate.NewListSyncStates,
			pipelinestate.NewResetStreamState,
			pipelinestate.NewResetConnectionState,
			pipelinestate.NewUpdateStreamState,
			pipelinestate.NewListStreamStates,
		),

		// Log use cases
		fx.Provide(
			pipelinelogs.NewAppendJobLog,
			pipelinelogs.NewGetJobLog,
			pipelinelogs.NewBatchAppendJobLogs,
		),

		// Sync use cases
		fx.Provide(
			pipelineexecdocker.NewDockerSyncExecutor,
			pipelinejobs.NewSyncScheduler,
		),

		// Stats storage + use cases
		fx.Provide(
			pipelinestats.NewGetWorkspaceStats,
			pipelinestats.NewGetConnectionStats,
			pipelinestats.NewGetSyncTimeline,
			pipelinestats.NewComputeStatsRollup,
		),
	)
}
