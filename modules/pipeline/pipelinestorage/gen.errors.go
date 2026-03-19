package pipelinestorage

import (
	errors1 "errors"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	errors "github.com/pkg/errors"
	gorm "gorm.io/gorm"

	pipelinesvc "github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	// user code 'imports'
	// end user code 'imports'
)

func wrapManagedConnectorQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrManagedConnectorNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrManagedConnectorAlreadyExists, err))
		}
	}

	return err
}
func wrapRepositoryQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrRepositoryNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrRepositoryAlreadyExists, err))
		}
	}

	return err
}
func wrapRepositoryConnectorQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrRepositoryConnectorNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrRepositoryConnectorAlreadyExists, err))
		}
	}

	return err
}
func wrapSourceQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrSourceNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrSourceAlreadyExists, err))
		}
	}

	return err
}
func wrapDestinationQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrDestinationNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrDestinationAlreadyExists, err))
		}
	}

	return err
}
func wrapConnectionQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrConnectionNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrConnectionAlreadyExists, err))
		}
	}

	return err
}
func wrapJobQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrJobNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrJobAlreadyExists, err))
		}
	}

	return err
}
func wrapJobAttemptQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrJobAttemptNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrJobAttemptAlreadyExists, err))
		}
	}

	return err
}
func wrapCatalogDiscoveryQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrCatalogDiscoveryNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrCatalogDiscoveryAlreadyExists, err))
		}
	}

	return err
}
func wrapConfiguredCatalogQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrConfiguredCatalogNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrConfiguredCatalogAlreadyExists, err))
		}
	}

	return err
}
func wrapJobLogQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrJobLogNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrJobLogAlreadyExists, err))
		}
	}

	return err
}
func wrapConnectionStateQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrConnectionStateNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrConnectionStateAlreadyExists, err))
		}
	}

	return err
}

func wrapWorkspaceSettingsQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrWorkspaceSettingsNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrWorkspaceSettingsAlreadyExists, err))
		}
	}

	return err
}
func wrapConnectorTaskQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrConnectorTaskNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrConnectorTaskAlreadyExists, err))
		}
	}

	return err
}
func wrapStreamGenerationQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(pipelinesvc.ErrStreamGenerationNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(pipelinesvc.ErrStreamGenerationAlreadyExists, err))
		}
	}

	return err
}
