package notifystorage

import (
	uuid "github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	clause "gorm.io/gorm/clause"

	notifyservice "github.com/synclet-io/synclet/modules/notify/notifyservice"
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
func buildWebhookFilterExpr(filter *notifyservice.WebhookFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
		dbutil.ColumnFilter[bool]{
			Column: opts.columnPrefix + "enabled",
			Filter: filter.Enabled,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildWebhookFilterExpr(orFilter)
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
				expr, err := buildWebhookFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}

			return clause.And(exprs...), nil
		}),
	)
}
func buildNotificationChannelFilterExpr(filter *notifyservice.NotificationChannelFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
		dbutil.MappedColumnFilter[notifyservice.ChannelType, string]{
			Column: opts.columnPrefix + "channel_type",
			Filter: filter.ChannelType,
			Mapper: convertChannelTypeToDB,
		},
		dbutil.ColumnFilter[bool]{
			Column: opts.columnPrefix + "enabled",
			Filter: filter.Enabled,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildNotificationChannelFilterExpr(orFilter)
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
				expr, err := buildNotificationChannelFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}

			return clause.And(exprs...), nil
		}),
	)
}
func buildNotificationRuleFilterExpr(filter *notifyservice.NotificationRuleFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "channel_id",
			Filter: filter.ChannelID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "connection_id",
			Filter: filter.ConnectionID,
		},
		dbutil.ColumnFilter[bool]{
			Column: opts.columnPrefix + "enabled",
			Filter: filter.Enabled,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildNotificationRuleFilterExpr(orFilter)
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
				expr, err := buildNotificationRuleFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}

			return clause.And(exprs...), nil
		}),
	)
}
