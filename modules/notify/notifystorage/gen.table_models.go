package notifystorage

import (
	time "time"

	uuid "github.com/google/uuid"

	notifyservice "github.com/synclet-io/synclet/modules/notify/notifyservice"
	// user code 'imports'
	// end user code 'imports'
)

type dbWebhook struct {
	ID          uuid.UUID `gorm:"column:id;"`
	WorkspaceID uuid.UUID `gorm:"column:workspace_id;"`
	URL         string    `gorm:"column:url;type:text;"`
	Events      jsonb     `gorm:"column:events;"`
	Secret      string    `gorm:"column:secret;type:text;"`
	Enabled     bool      `gorm:"column:enabled;"`
	CreatedAt   time.Time `gorm:"column:created_at;"`
	UpdatedAt   time.Time `gorm:"column:updated_at;"`
}

func convertWebhookToDB(src *notifyservice.Webhook) (*dbWebhook, error) {
	result := &dbWebhook{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.URL = src.URL
	result.Events = src.Events
	result.Secret = src.Secret
	result.Enabled = src.Enabled
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()

	return result, nil
}

func convertWebhookFromDB(src *dbWebhook) (*notifyservice.Webhook, error) {
	result := &notifyservice.Webhook{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.URL = src.URL
	result.Events = src.Events
	result.Secret = src.Secret
	result.Enabled = src.Enabled
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt

	return result, nil
}
func (a dbWebhook) TableName() string {
	return "notify.webhooks"
}

type dbNotificationChannel struct {
	ID          uuid.UUID `gorm:"column:id;"`
	WorkspaceID uuid.UUID `gorm:"column:workspace_id;"`
	Name        string    `gorm:"column:name;type:text;"`
	ChannelType string    `gorm:"column:channel_type;type:text;"`
	Config      jsonb     `gorm:"column:config;"`
	Enabled     bool      `gorm:"column:enabled;"`
	CreatedAt   time.Time `gorm:"column:created_at;"`
	UpdatedAt   time.Time `gorm:"column:updated_at;"`
}

func convertNotificationChannelToDB(src *notifyservice.NotificationChannel) (*dbNotificationChannel, error) {
	result := &dbNotificationChannel{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	tmp3, err := convertChannelTypeToDB(src.ChannelType)
	if err != nil {
		return nil, err
	}
	result.ChannelType = tmp3
	result.Config = src.Config
	result.Enabled = src.Enabled
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()

	return result, nil
}

func convertNotificationChannelFromDB(src *dbNotificationChannel) (*notifyservice.NotificationChannel, error) {
	result := &notifyservice.NotificationChannel{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.Name = src.Name
	tmp11, err := convertChannelTypeFromDB(src.ChannelType)
	if err != nil {
		return nil, err
	}
	result.ChannelType = tmp11
	result.Config = src.Config
	result.Enabled = src.Enabled
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt

	return result, nil
}
func (a dbNotificationChannel) TableName() string {
	return "notify.notification_channels"
}

type dbNotificationRule struct {
	ID             uuid.UUID `gorm:"column:id;"`
	WorkspaceID    uuid.UUID `gorm:"column:workspace_id;"`
	ChannelID      uuid.UUID `gorm:"column:channel_id;"`
	ConnectionID   uuid.UUID `gorm:"column:connection_id;"`
	Condition      string    `gorm:"column:condition;type:text;"`
	ConditionValue int       `gorm:"column:condition_value;"`
	Enabled        bool      `gorm:"column:enabled;"`
	CreatedAt      time.Time `gorm:"column:created_at;"`
	UpdatedAt      time.Time `gorm:"column:updated_at;"`
}

func convertNotificationRuleToDB(src *notifyservice.NotificationRule) (*dbNotificationRule, error) {
	result := &dbNotificationRule{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.ChannelID = src.ChannelID
	result.ConnectionID = src.ConnectionID
	tmp4, err := convertNotificationConditionToDB(src.Condition)
	if err != nil {
		return nil, err
	}
	result.Condition = tmp4
	result.ConditionValue = src.ConditionValue
	result.Enabled = src.Enabled
	result.CreatedAt = (src.CreatedAt).UTC()
	result.UpdatedAt = (src.UpdatedAt).UTC()

	return result, nil
}

func convertNotificationRuleFromDB(src *dbNotificationRule) (*notifyservice.NotificationRule, error) {
	result := &notifyservice.NotificationRule{}
	result.ID = src.ID
	result.WorkspaceID = src.WorkspaceID
	result.ChannelID = src.ChannelID
	result.ConnectionID = src.ConnectionID
	tmp13, err := convertNotificationConditionFromDB(src.Condition)
	if err != nil {
		return nil, err
	}
	result.Condition = tmp13
	result.ConditionValue = src.ConditionValue
	result.Enabled = src.Enabled
	result.CreatedAt = src.CreatedAt
	result.UpdatedAt = src.UpdatedAt

	return result, nil
}
func (a dbNotificationRule) TableName() string {
	return "notify.notification_rules"
}
