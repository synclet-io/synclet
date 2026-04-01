package notifystorage

import (
	driver "database/sql/driver"
	json "encoding/json"
	time "time"

	uuid "github.com/google/uuid"

	notifyservice "github.com/synclet-io/synclet/modules/notify/notifyservice"
	// user code 'imports'
	// end user code 'imports'
)

type jsonWebhook struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	URL         string    `json:"url"`
	Events      jsonb     `json:"events"`
	Secret      string    `json:"secret"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (w *jsonWebhook) Scan(value any) error {
	return json.Unmarshal(value.([]byte), w)
}

func (w jsonWebhook) Value() (driver.Value, error) {
	return json.Marshal(w)
}

func convertWebhookToJsonModel(src *notifyservice.Webhook) (*jsonWebhook, error) {
	result := &jsonWebhook{}
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

func convertWebhookFromJsonModel(src *jsonWebhook) (*notifyservice.Webhook, error) {
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

type jsonNotificationChannel struct {
	ID          uuid.UUID `json:"id"`
	WorkspaceID uuid.UUID `json:"workspace_id"`
	Name        string    `json:"name"`
	ChannelType string    `json:"channel_type"`
	Config      jsonb     `json:"config"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (n *jsonNotificationChannel) Scan(value any) error {
	return json.Unmarshal(value.([]byte), n)
}

func (n jsonNotificationChannel) Value() (driver.Value, error) {
	return json.Marshal(n)
}

func convertNotificationChannelToJsonModel(src *notifyservice.NotificationChannel) (*jsonNotificationChannel, error) {
	result := &jsonNotificationChannel{}
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

func convertNotificationChannelFromJsonModel(src *jsonNotificationChannel) (*notifyservice.NotificationChannel, error) {
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

type jsonNotificationRule struct {
	ID             uuid.UUID `json:"id"`
	WorkspaceID    uuid.UUID `json:"workspace_id"`
	ChannelID      uuid.UUID `json:"channel_id"`
	ConnectionID   uuid.UUID `json:"connection_id"`
	Condition      string    `json:"condition"`
	ConditionValue int       `json:"condition_value"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func (n *jsonNotificationRule) Scan(value any) error {
	return json.Unmarshal(value.([]byte), n)
}

func (n jsonNotificationRule) Value() (driver.Value, error) {
	return json.Marshal(n)
}

func convertNotificationRuleToJsonModel(src *notifyservice.NotificationRule) (*jsonNotificationRule, error) {
	result := &jsonNotificationRule{}
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

func convertNotificationRuleFromJsonModel(src *jsonNotificationRule) (*notifyservice.NotificationRule, error) {
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
