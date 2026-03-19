package notifyservice

import (
	fmt "fmt"
	// user code 'imports'
	// end user code 'imports'
)

type NotFoundError string

func (n NotFoundError) Error() string {
	return fmt.Sprintf("%s not found", string(n))
}

type AlreadyExistsError string

func (a AlreadyExistsError) Error() string {
	return fmt.Sprintf("%s already exists", string(a))
}

const (
	ErrWebhookNotFound      = NotFoundError("Webhook")
	ErrWebhookAlreadyExists = AlreadyExistsError("Webhook")
)
const (
	ErrNotificationChannelNotFound      = NotFoundError("NotificationChannel")
	ErrNotificationChannelAlreadyExists = AlreadyExistsError("NotificationChannel")
)
const (
	ErrNotificationRuleNotFound      = NotFoundError("NotificationRule")
	ErrNotificationRuleAlreadyExists = AlreadyExistsError("NotificationRule")
)
