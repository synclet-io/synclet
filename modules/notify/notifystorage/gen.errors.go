package notifystorage

import (
	errors1 "errors"

	pgconn "github.com/jackc/pgx/v5/pgconn"
	errors "github.com/pkg/errors"
	gorm "gorm.io/gorm"

	notifysvc "github.com/synclet-io/synclet/modules/notify/notifyservice"
	// user code 'imports'
	// end user code 'imports'
)

func wrapWebhookQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(notifysvc.ErrWebhookNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(notifysvc.ErrWebhookAlreadyExists, err))
		}
	}

	return err
}
func wrapNotificationChannelQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(notifysvc.ErrNotificationChannelNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(notifysvc.ErrNotificationChannelAlreadyExists, err))
		}
	}

	return err
}
func wrapNotificationRuleQueryError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, gorm.ErrRecordNotFound) {
		return errors.WithStack(errors1.Join(notifysvc.ErrNotificationRuleNotFound, err))
	}

	var pgErr *pgconn.PgError

	if errors.As(err, &pgErr) {
		if pgErr.Code == "23505" {
			return errors.WithStack(errors1.Join(notifysvc.ErrNotificationRuleAlreadyExists, err))
		}
	}

	return err
}
