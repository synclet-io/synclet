package authstorage

import (
	time "time"

	uuid "github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	clause "gorm.io/gorm/clause"

	authservice "github.com/synclet-io/synclet/modules/auth/authservice"
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
func buildUserFilterExpr(filter *authservice.UserFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "email",
			Filter: filter.Email,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildUserFilterExpr(orFilter)
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
				expr, err := buildUserFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildRefreshTokenFilterExpr(filter *authservice.RefreshTokenFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "user_id",
			Filter: filter.UserID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "token_hash",
			Filter: filter.TokenHash,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildRefreshTokenFilterExpr(orFilter)
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
				expr, err := buildRefreshTokenFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildPasswordResetTokenFilterExpr(filter *authservice.PasswordResetTokenFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "user_id",
			Filter: filter.UserID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "token_hash",
			Filter: filter.TokenHash,
		},
		dbutil.ColumnFilter[time.Time]{
			Column: opts.columnPrefix + "expires_at",
			Filter: filter.ExpiresAt,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildPasswordResetTokenFilterExpr(orFilter)
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
				expr, err := buildPasswordResetTokenFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildAPIKeyFilterExpr(filter *authservice.APIKeyFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "key_hash",
			Filter: filter.KeyHash,
		},
		dbutil.ColumnFilter[*uuid.UUID]{
			Column: opts.columnPrefix + "oauth_grant_id",
			Filter: filter.OAuthGrantID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildAPIKeyFilterExpr(orFilter)
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
				expr, err := buildAPIKeyFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildOAuthClientFilterExpr(filter *authservice.OAuthClientFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "client_id",
			Filter: filter.ClientID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildOAuthClientFilterExpr(orFilter)
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
				expr, err := buildOAuthClientFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildOAuthCodeFilterExpr(filter *authservice.OAuthCodeFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "code_hash",
			Filter: filter.CodeHash,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "client_id",
			Filter: filter.ClientID,
		},
		dbutil.ColumnFilter[time.Time]{
			Column: opts.columnPrefix + "expires_at",
			Filter: filter.ExpiresAt,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildOAuthCodeFilterExpr(orFilter)
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
				expr, err := buildOAuthCodeFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildOAuthGrantFilterExpr(filter *authservice.OAuthGrantFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "client_id",
			Filter: filter.ClientID,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "user_id",
			Filter: filter.UserID,
		},
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
				expr, err := buildOAuthGrantFilterExpr(orFilter)
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
				expr, err := buildOAuthGrantFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildOAuthRefreshTokenFilterExpr(filter *authservice.OAuthRefreshTokenFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "grant_id",
			Filter: filter.GrantID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "token_hash",
			Filter: filter.TokenHash,
		},
		dbutil.ColumnFilter[time.Time]{
			Column: opts.columnPrefix + "expires_at",
			Filter: filter.ExpiresAt,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildOAuthRefreshTokenFilterExpr(orFilter)
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
				expr, err := buildOAuthRefreshTokenFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildOIDCIdentityFilterExpr(filter *authservice.OIDCIdentityFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "user_id",
			Filter: filter.UserID,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "provider_slug",
			Filter: filter.ProviderSlug,
		},
		dbutil.ColumnFilter[string]{
			Column: opts.columnPrefix + "subject",
			Filter: filter.Subject,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildOIDCIdentityFilterExpr(orFilter)
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
				expr, err := buildOIDCIdentityFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildOIDCStateFilterExpr(filter *authservice.OIDCStateFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "state",
			Filter: filter.State,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildOIDCStateFilterExpr(orFilter)
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
				expr, err := buildOIDCStateFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
func buildRateLimitFilterExpr(filter *authservice.RateLimitFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "key",
			Filter: filter.Key,
		},
		dbutil.ColumnFilter[time.Time]{
			Column: opts.columnPrefix + "expires_at",
			Filter: filter.ExpiresAt,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildRateLimitFilterExpr(orFilter)
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
				expr, err := buildRateLimitFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}
			return clause.And(exprs...), nil
		}),
	)
}
