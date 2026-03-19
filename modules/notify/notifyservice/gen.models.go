package notifyservice

import (
	time "time"

	uuid "github.com/google/uuid"
	filter "github.com/saturn4er/boilerplate-go/lib/filter"
	order "github.com/saturn4er/boilerplate-go/lib/order"
	// user code 'imports'
	// end user code 'imports'
)

type WebhookField byte

const (
	WebhookFieldID WebhookField = iota + 1
	WebhookFieldWorkspaceID
	WebhookFieldURL
	WebhookFieldEvents
	WebhookFieldSecret
	WebhookFieldEnabled
	WebhookFieldCreatedAt
	WebhookFieldUpdatedAt
)

type WebhookFilter struct {
	ID          filter.Filter[uuid.UUID]
	WorkspaceID filter.Filter[uuid.UUID]
	Enabled     filter.Filter[bool]
	Or          []*WebhookFilter
	And         []*WebhookFilter
}
type WebhookOrder order.Order[WebhookField]

type Webhook struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	URL         string
	Events      jsonb
	Secret      string
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// user code 'Webhook methods'
// end user code 'Webhook methods'

func (w *Webhook) Copy() Webhook {
	var result Webhook
	result.ID = w.ID
	result.WorkspaceID = w.WorkspaceID
	result.URL = w.URL
	result.Events = w.Events
	result.Secret = w.Secret
	result.Enabled = w.Enabled
	result.CreatedAt = w.CreatedAt
	result.UpdatedAt = w.UpdatedAt

	return result
}
func (w *Webhook) Equals(to *Webhook) bool {
	if (w == nil) != (to == nil) {
		return false
	}
	if w == nil && to == nil {
		return true
	}
	if w.ID != to.ID {
		return false
	}
	if w.WorkspaceID != to.WorkspaceID {
		return false
	}
	if w.URL != to.URL {
		return false
	}
	if w.Events != to.Events {
		return false
	}
	if w.Secret != to.Secret {
		return false
	}
	if w.Enabled != to.Enabled {
		return false
	}
	if w.CreatedAt != to.CreatedAt {
		return false
	}
	if w.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type NotificationChannelField byte

const (
	NotificationChannelFieldID NotificationChannelField = iota + 1
	NotificationChannelFieldWorkspaceID
	NotificationChannelFieldName
	NotificationChannelFieldChannelType
	NotificationChannelFieldConfig
	NotificationChannelFieldEnabled
	NotificationChannelFieldCreatedAt
	NotificationChannelFieldUpdatedAt
)

type NotificationChannelFilter struct {
	ID          filter.Filter[uuid.UUID]
	WorkspaceID filter.Filter[uuid.UUID]
	ChannelType filter.Filter[ChannelType]
	Enabled     filter.Filter[bool]
	Or          []*NotificationChannelFilter
	And         []*NotificationChannelFilter
}
type NotificationChannelOrder order.Order[NotificationChannelField]

type NotificationChannel struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
	Name        string
	ChannelType ChannelType
	Config      jsonb
	Enabled     bool
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// user code 'NotificationChannel methods'
// end user code 'NotificationChannel methods'

func (n *NotificationChannel) Copy() NotificationChannel {
	var result NotificationChannel
	result.ID = n.ID
	result.WorkspaceID = n.WorkspaceID
	result.Name = n.Name
	result.ChannelType = n.ChannelType // enum
	result.Config = n.Config
	result.Enabled = n.Enabled
	result.CreatedAt = n.CreatedAt
	result.UpdatedAt = n.UpdatedAt

	return result
}
func (n *NotificationChannel) Equals(to *NotificationChannel) bool {
	if (n == nil) != (to == nil) {
		return false
	}
	if n == nil && to == nil {
		return true
	}
	if n.ID != to.ID {
		return false
	}
	if n.WorkspaceID != to.WorkspaceID {
		return false
	}
	if n.Name != to.Name {
		return false
	}
	if n.ChannelType != to.ChannelType {
		return false
	}
	if n.Config != to.Config {
		return false
	}
	if n.Enabled != to.Enabled {
		return false
	}
	if n.CreatedAt != to.CreatedAt {
		return false
	}
	if n.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}

type NotificationRuleField byte

const (
	NotificationRuleFieldID NotificationRuleField = iota + 1
	NotificationRuleFieldWorkspaceID
	NotificationRuleFieldChannelID
	NotificationRuleFieldConnectionID
	NotificationRuleFieldCondition
	NotificationRuleFieldConditionValue
	NotificationRuleFieldEnabled
	NotificationRuleFieldCreatedAt
	NotificationRuleFieldUpdatedAt
)

type NotificationRuleFilter struct {
	ID           filter.Filter[uuid.UUID]
	WorkspaceID  filter.Filter[uuid.UUID]
	ChannelID    filter.Filter[uuid.UUID]
	ConnectionID filter.Filter[uuid.UUID]
	Enabled      filter.Filter[bool]
	Or           []*NotificationRuleFilter
	And          []*NotificationRuleFilter
}
type NotificationRuleOrder order.Order[NotificationRuleField]

type NotificationRule struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	ChannelID      uuid.UUID
	ConnectionID   uuid.UUID
	Condition      NotificationCondition
	ConditionValue int
	Enabled        bool
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

// user code 'NotificationRule methods'
// end user code 'NotificationRule methods'

func (n *NotificationRule) Copy() NotificationRule {
	var result NotificationRule
	result.ID = n.ID
	result.WorkspaceID = n.WorkspaceID
	result.ChannelID = n.ChannelID
	result.ConnectionID = n.ConnectionID
	result.Condition = n.Condition // enum
	result.ConditionValue = n.ConditionValue
	result.Enabled = n.Enabled
	result.CreatedAt = n.CreatedAt
	result.UpdatedAt = n.UpdatedAt

	return result
}
func (n *NotificationRule) Equals(to *NotificationRule) bool {
	if (n == nil) != (to == nil) {
		return false
	}
	if n == nil && to == nil {
		return true
	}
	if n.ID != to.ID {
		return false
	}
	if n.WorkspaceID != to.WorkspaceID {
		return false
	}
	if n.ChannelID != to.ChannelID {
		return false
	}
	if n.ConnectionID != to.ConnectionID {
		return false
	}
	if n.Condition != to.Condition {
		return false
	}
	if n.ConditionValue != to.ConditionValue {
		return false
	}
	if n.Enabled != to.Enabled {
		return false
	}
	if n.CreatedAt != to.CreatedAt {
		return false
	}
	if n.UpdatedAt != to.UpdatedAt {
		return false
	}

	return true
}
