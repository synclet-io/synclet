package secretstorage

import (
	uuid "github.com/google/uuid"
	dbutil "github.com/saturn4er/boilerplate-go/lib/dbutil"
	clause "gorm.io/gorm/clause"

	secretservice "github.com/synclet-io/synclet/modules/secret/secretservice"
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
func buildSecretFilterExpr(filter *secretservice.SecretFilter, options ...func(*filterOptions)) (clause.Expression, error) {
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
			Column: opts.columnPrefix + "owner_type",
			Filter: filter.OwnerType,
		},
		dbutil.ColumnFilter[uuid.UUID]{
			Column: opts.columnPrefix + "owner_id",
			Filter: filter.OwnerID,
		},
		dbutil.ExpressionBuilderFunc(func() (clause.Expression, error) {
			if filter.Or == nil {
				return nil, nil
			}
			exprs := make([]clause.Expression, 0, len(filter.Or))
			for _, orFilter := range filter.Or {
				expr, err := buildSecretFilterExpr(orFilter)
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
				expr, err := buildSecretFilterExpr(andFilter)
				if err != nil {
					return nil, err
				}
				exprs = append(exprs, expr)
			}

			return clause.And(exprs...), nil
		}),
	)
}
