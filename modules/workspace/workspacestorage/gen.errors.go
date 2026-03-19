package workspacestorage

import (
	errors1 "errors"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	errors "github.com/pkg/errors"
	gorm "gorm.io/gorm"

	workspacesvc "github.com/synclet-io/synclet/modules/workspace/workspaceservice"
	// user code 'imports'
	// end user code 'imports'
)

func wrapWorkspaceQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(workspacesvc.ErrWorkspaceNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(workspacesvc.ErrWorkspaceAlreadyExists, err))
		}
	}

	return err
}
func wrapWorkspaceMemberQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(workspacesvc.ErrWorkspaceMemberNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(workspacesvc.ErrWorkspaceMemberAlreadyExists, err))
		}
	}

	return err
}
func wrapWorkspaceInviteQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(workspacesvc.ErrWorkspaceInviteNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(workspacesvc.ErrWorkspaceInviteAlreadyExists, err))
		}
	}

	return err
}
