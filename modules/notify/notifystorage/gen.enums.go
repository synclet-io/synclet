package notifystorage

import (
	fmt "fmt"

	notifyservice "github.com/synclet-io/synclet/modules/notify/notifyservice"
	// user code 'imports'
	// end user code 'imports'
)

const (
	channelTypeSlack    = "slack"
	channelTypeEmail    = "email"
	channelTypeTelegram = "telegram"
)

func convertChannelTypeToDB(channelTypeValue notifyservice.ChannelType) (string, error) {
	result, ok := map[notifyservice.ChannelType]string{
		notifyservice.ChannelTypeSlack:    channelTypeSlack,
		notifyservice.ChannelTypeEmail:    channelTypeEmail,
		notifyservice.ChannelTypeTelegram: channelTypeTelegram,
	}[channelTypeValue]
	if !ok {
		return "", fmt.Errorf("unknown ChannelType value: %d", channelTypeValue)
	}
	return result, nil
}

func convertChannelTypeFromDB(channelTypeValue string) (notifyservice.ChannelType, error) {
	result, ok := map[string]notifyservice.ChannelType{
		channelTypeSlack:    notifyservice.ChannelTypeSlack,
		channelTypeEmail:    notifyservice.ChannelTypeEmail,
		channelTypeTelegram: notifyservice.ChannelTypeTelegram,
	}[channelTypeValue]
	if !ok {
		return 0, fmt.Errorf("unknown ChannelType db value: %s", channelTypeValue)
	}
	return result, nil
}

const (
	notificationConditionOnFailure             = "on_failure"
	notificationConditionOnConsecutiveFailures = "on_consecutive_failures"
	notificationConditionOnZeroRecords         = "on_zero_records"
)

func convertNotificationConditionToDB(notificationConditionValue notifyservice.NotificationCondition) (string, error) {
	result, ok := map[notifyservice.NotificationCondition]string{
		notifyservice.NotificationConditionOnFailure:             notificationConditionOnFailure,
		notifyservice.NotificationConditionOnConsecutiveFailures: notificationConditionOnConsecutiveFailures,
		notifyservice.NotificationConditionOnZeroRecords:         notificationConditionOnZeroRecords,
	}[notificationConditionValue]
	if !ok {
		return "", fmt.Errorf("unknown NotificationCondition value: %d", notificationConditionValue)
	}
	return result, nil
}

func convertNotificationConditionFromDB(notificationConditionValue string) (notifyservice.NotificationCondition, error) {
	result, ok := map[string]notifyservice.NotificationCondition{
		notificationConditionOnFailure:             notifyservice.NotificationConditionOnFailure,
		notificationConditionOnConsecutiveFailures: notifyservice.NotificationConditionOnConsecutiveFailures,
		notificationConditionOnZeroRecords:         notifyservice.NotificationConditionOnZeroRecords,
	}[notificationConditionValue]
	if !ok {
		return 0, fmt.Errorf("unknown NotificationCondition db value: %s", notificationConditionValue)
	}
	return result, nil
}
