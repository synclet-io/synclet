package workspacestorage

import (
	uuid "github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	clause "gorm.io/gorm/clause"

	workspaceservice "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
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
func buildWorkspaceFilterExpr(filter *workspaceservice.WorkspaceFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "name",
			Filter: filter.Name,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "slug",
			Filter: filter.Slug,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildWorkspaceFilterExpr(orFilter)
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
				expr, err := buildWorkspaceFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}

			return clause.And(exprs...), nil
		}),
	)
}
func buildWorkspaceMemberFilterExpr(filter *workspaceservice.WorkspaceMemberFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "user_id",
			Filter: filter.UserID,
		},
		dbutil.MappedColumnFilter[workspaceservice.MemberRole, string]{
			Column: opts.columnPrefix + "role",
			Filter: filter.Role,
			Mapper: convertMemberRoleToDB,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildWorkspaceMemberFilterExpr(orFilter)
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
				expr, err := buildWorkspaceMemberFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}

			return clause.And(exprs...), nil
		}),
	)
}
func buildWorkspaceInviteFilterExpr(filter *workspaceservice.WorkspaceInviteFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "inviter_user_id",
			Filter: filter.InviterUserID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "email",
			Filter: filter.Email,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "token",
			Filter: filter.Token,
		},
		dbutil.MappedColumnFilter[workspaceservice.InviteStatus, string]{
			Column: opts.columnPrefix + "status",
			Filter: filter.Status,
			Mapper: convertInviteStatusToDB,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildWorkspaceInviteFilterExpr(orFilter)
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
				expr, err := buildWorkspaceInviteFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}

			return clause.And(exprs...), nil
		}),
	)
}
