package authstorage

import (
	errors1 "errors"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	errors "github.com/pkg/errors"
	gorm "gorm.io/gorm"

	authsvc "github.com/synclet-io/synclet/modules/auth/authservice"
	// user code 'imports'
	// end user code 'imports'
)

func wrapUserQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(authsvc.ErrUserNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(authsvc.ErrUserAlreadyExists, err))
		}
	}

	return err
}
func wrapRefreshTokenQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(authsvc.ErrRefreshTokenNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(authsvc.ErrRefreshTokenAlreadyExists, err))
		}
	}

	return err
}
func wrapAPIKeyQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(authsvc.ErrAPIKeyNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(authsvc.ErrAPIKeyAlreadyExists, err))
		}
	}

	return err
}
func wrapOIDCIdentityQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(authsvc.ErrOIDCIdentityNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(authsvc.ErrOIDCIdentityAlreadyExists, err))
		}
	}

	return err
}
func wrapOIDCStateQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(authsvc.ErrOIDCStateNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(authsvc.ErrOIDCStateAlreadyExists, err))
		}
	}

	return err
}
