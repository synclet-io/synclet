package notifyservice

import (
	strconv "strconv"
	// user code 'imports'
	// end user code 'imports'
)

type ChannelType byte

const (
	ChannelTypeSlack ChannelType = iota + 1
	ChannelTypeEmail
	ChannelTypeTelegram
)

// user code 'ChannelType methods'
// end user code 'ChannelType methods'
func (c ChannelType) IsValid() bool {
	return c > 0 && c < 4
}
func (c ChannelType) String() string {
	const names = "SlackEmailTelegram"

	var indexes = [...]int32{0, 5, 10, 18}
	if c < 1 || c > 3 {
		return "ChannelType(" + strconv.FormatInt(int64(c), 10) + ")"
	}

	return names[indexes[c-1]:indexes[c]]
}

type NotificationCondition byte

const (
	NotificationConditionOnFailure NotificationCondition = iota + 1
	NotificationConditionOnConsecutiveFailures
	NotificationConditionOnZeroRecords
)

// user code 'NotificationCondition methods'
// end user code 'NotificationCondition methods'
func (n NotificationCondition) IsValid() bool {
	return n > 0 && n < 4
}
func (n NotificationCondition) String() string {
	const names = "OnFailureOnConsecutiveFailuresOnZeroRecords"

	var indexes = [...]int32{0, 9, 30, 43}
	if n < 1 || n > 3 {
		return "NotificationCondition(" + strconv.FormatInt(int64(n), 10) + ")"
	}

	return names[indexes[n-1]:indexes[n]]
}
