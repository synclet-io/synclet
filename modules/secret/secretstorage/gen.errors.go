package secretstorage

import (
	errors1 "errors"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	errors "github.com/pkg/errors"
	gorm "gorm.io/gorm"

	secretsvc "github.com/synclet-io/synclet/modules/secret/secretservice"
	// user code 'imports'
	// end user code 'imports'
)

func wrapSecretQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(secretsvc.ErrSecretNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(secretsvc.ErrSecretAlreadyExists, err))
		}
	}

	return err
}
