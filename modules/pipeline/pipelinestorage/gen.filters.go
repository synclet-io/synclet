package pipelinestorage

import (
	time "time"

	uuid "github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	clause "gorm.io/gorm/clause"

	pipelineservice "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	// user code 'imports'
	// end user code 'imports'
)

type filterOptions struct {
	columnPrefix string
}

func withFilterColumnPrefix(prefix string) func(*filterOptions) {
	return func(f *filterOptions) {
		f.columnPrefix = prefix
	}
}
func buildManagedConnectorFilterExpr(filter *pipelineservice.ManagedConnectorFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "workspace_id",
			Filter: filter.WorkspaceID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "docker_image",
			Filter: filter.DockerImage,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "name",
			Filter: filter.Name,
		},
		dbutil.MappedColumnFilter[pipelineservice.ConnectorType, string]{
			Column: opts.columnPrefix + "connector_type",
			Filter: filter.ConnectorType,
			Mapper: convertConnectorTypeToDB,
		},
		dbutil.ColumnFilter[*uuid.UUID]{
			Column: opts.columnPrefix + "repository_id",
			Filter: filter.RepositoryID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildManagedConnectorFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildManagedConnectorFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildRepositoryFilterExpr(filter *pipelineservice.RepositoryFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "workspace_id",
			Filter: filter.WorkspaceID,
		},
		dbutil.MappedColumnFilter[pipelineservice.RepositoryStatus, string]{
			Column: opts.columnPrefix + "status",
			Filter: filter.Status,
			Mapper: convertRepositoryStatusToDB,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildRepositoryFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildRepositoryFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildRepositoryConnectorFilterExpr(filter *pipelineservice.RepositoryConnectorFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "repository_id",
			Filter: filter.RepositoryID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "docker_repository",
			Filter: filter.DockerRepository,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "name",
			Filter: filter.Name,
		},
		dbutil.MappedColumnFilter[pipelineservice.ConnectorType, string]{
			Column: opts.columnPrefix + "connector_type",
			Filter: filter.ConnectorType,
			Mapper: convertConnectorTypeToDB,
		},
		dbutil.MappedColumnFilter[pipelineservice.SupportLevel, string]{
			Column: opts.columnPrefix + "support_level",
			Filter: filter.SupportLevel,
			Mapper: convertSupportLevelToDB,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "license",
			Filter: filter.License,
		},
		dbutil.MappedColumnFilter[pipelineservice.SourceType, string]{
			Column: opts.columnPrefix + "source_type",
			Filter: filter.SourceType,
			Mapper: convertSourceTypeToDB,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildRepositoryConnectorFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildRepositoryConnectorFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildSourceFilterExpr(filter *pipelineservice.SourceFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "workspace_id",
			Filter: filter.WorkspaceID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "name",
			Filter: filter.Name,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "managed_connector_id",
			Filter: filter.ManagedConnectorID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildSourceFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildSourceFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildDestinationFilterExpr(filter *pipelineservice.DestinationFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "workspace_id",
			Filter: filter.WorkspaceID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "name",
			Filter: filter.Name,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "managed_connector_id",
			Filter: filter.ManagedConnectorID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildDestinationFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildDestinationFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildConnectionFilterExpr(filter *pipelineservice.ConnectionFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "workspace_id",
			Filter: filter.WorkspaceID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "name",
			Filter: filter.Name,
		},
		dbutil.MappedColumnFilter[pipelineservice.ConnectionStatus, string]{
			Column: opts.columnPrefix + "status",
			Filter: filter.Status,
			Mapper: convertConnectionStatusToDB,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "source_id",
			Filter: filter.SourceID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "destination_id",
			Filter: filter.DestinationID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildConnectionFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildConnectionFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildJobFilterExpr(filter *pipelineservice.JobFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "connection_id",
			Filter: filter.ConnectionID,
		},
		dbutil.MappedColumnFilter[pipelineservice.JobStatus, string]{
			Column: opts.columnPrefix + "status",
			Filter: filter.Status,
			Mapper: convertJobStatusToDB,
		},
		dbutil.MappedColumnFilter[pipelineservice.JobType, string]{
			Column: opts.columnPrefix + "job_type",
			Filter: filter.JobType,
			Mapper: convertJobTypeToDB,
		},
		dbutil.ColumnFilter[*time.Time]{
			Column: opts.columnPrefix + "started_at",
			Filter: filter.StartedAt,
		},
		dbutil.ColumnFilter[*time.Time]{
			Column: opts.columnPrefix + "heartbeat_at",
			Filter: filter.HeartbeatAt,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildJobFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildJobFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildJobAttemptFilterExpr(filter *pipelineservice.JobAttemptFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "job_id",
			Filter: filter.JobID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildJobAttemptFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildJobAttemptFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildCatalogDiscoveryFilterExpr(filter *pipelineservice.CatalogDiscoveryFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "source_id",
			Filter: filter.SourceID,
		},
		dbutil.ColumnFilter[int]{
			Column: opts.columnPrefix + "version",
			Filter: filter.Version,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildCatalogDiscoveryFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildCatalogDiscoveryFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildConfiguredCatalogFilterExpr(filter *pipelineservice.ConfiguredCatalogFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "connection_id",
			Filter: filter.ConnectionID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildConfiguredCatalogFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildConfiguredCatalogFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildJobLogFilterExpr(filter *pipelineservice.JobLogFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "job_id",
			Filter: filter.JobID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildJobLogFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildJobLogFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildConnectionStateFilterExpr(filter *pipelineservice.ConnectionStateFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "connection_id",
			Filter: filter.ConnectionID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildConnectionStateFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildConnectionStateFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}

func buildWorkspaceSettingsFilterExpr(filter *pipelineservice.WorkspaceSettingsFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "workspace_id",
			Filter: filter.WorkspaceID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildWorkspaceSettingsFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildWorkspaceSettingsFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildConnectorTaskFilterExpr(filter *pipelineservice.ConnectorTaskFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "id",
			Filter: filter.ID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "workspace_id",
			Filter: filter.WorkspaceID,
		},
		dbutil.MappedColumnFilter[pipelineservice.ConnectorTaskType, string]{
			Column: opts.columnPrefix + "task_type",
			Filter: filter.TaskType,
			Mapper: convertConnectorTaskTypeToDB,
		},
		dbutil.MappedColumnFilter[pipelineservice.ConnectorTaskStatus, string]{
			Column: opts.columnPrefix + "status",
			Filter: filter.Status,
			Mapper: convertConnectorTaskStatusToDB,
		},
		dbutil.ColumnFilter[time.Time]{
			Column: opts.columnPrefix + "created_at",
			Filter: filter.CreatedAt,
		},
		dbutil.ColumnFilter[time.Time]{
			Column: opts.columnPrefix + "updated_at",
			Filter: filter.UpdatedAt,
		},
		dbutil.ColumnFilter[*time.Time]{
			Column: opts.columnPrefix + "completed_at",
			Filter: filter.CompletedAt,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildConnectorTaskFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildConnectorTaskFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildStreamGenerationFilterExpr(filter *pipelineservice.StreamGenerationFilter, options ...func(*filterOptions)) (clause.Expression, error) {
	if filter == nil {
		return nil, nil
	}

	opts := &filterOptions{}
	for _, opt := range options {
		opt(opts)
	}

	return dbutil.BuildFilterExpression(
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "connection_id",
			Filter: filter.ConnectionID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "stream_namespace",
			Filter: filter.StreamNamespace,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "stream_name",
			Filter: filter.StreamName,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildStreamGenerationFilterExpr(orFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.Or(exprs...), nil
		}),
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.And == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.And))
			for _, andFilter := range filter.And {
				expr, err := buildStreamGenerationFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
