package auditstorage

import (
	time "time"

	uuid "github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	clause "gorm.io/gorm/clause"

	auditservice "github.com/synclet-io/synclet/modules/audit/auditservice"
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
func buildAuditEventFilterExpr(filter *auditservice.AuditEventFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
		dbutil.MappedColumnFilter[auditservice.ActorType, string]{
			Column: opts.columnPrefix + "actor_type",
			Filter: filter.ActorType,
			Mapper: convertActorTypeToDB,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "actor_id",
			Filter: filter.ActorID,
		},
		dbutil.MappedColumnFilter[auditservice.Action, string]{
			Column: opts.columnPrefix + "action",
			Filter: filter.Action,
			Mapper: convertActionToDB,
		},
		dbutil.MappedColumnFilter[auditservice.ResourceType, string]{
			Column: opts.columnPrefix + "resource_type",
			Filter: filter.ResourceType,
			Mapper: convertResourceTypeToDB,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "resource_id",
			Filter: filter.ResourceID,
		},
		dbutil.ColumnFilter[time.Time]{
			Column: opts.columnPrefix + "created_at",
			Filter: filter.CreatedAt,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildAuditEventFilterExpr(orFilter)
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
				expr, err := buildAuditEventFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
