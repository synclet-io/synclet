package notifyservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

// CreateNotificationRuleParams holds parameters for creating a notification rule.
type CreateNotificationRuleParams struct {
	WorkspaceID    uuid.UUID
	ChannelID      uuid.UUID
	ConnectionID   uuid.UUID // Zero value means workspace-level default.
	Condition      NotificationCondition
	ConditionValue int
	Enabled        bool
}

// CreateNotificationRule creates a new notification rule.
type CreateNotificationRule struct {
	storage Storage
	audit   AuditRecorder
}

// NewCreateNotificationRule creates a new CreateNotificationRule use case.
func NewCreateNotificationRule(storage Storage, audit AuditRecorder) *CreateNotificationRule {
	return &CreateNotificationRule{storage: storage, audit: audit}
}

// Execute creates a notification rule with the given parameters.
func (uc *CreateNotificationRule) Execute(ctx context.Context, params CreateNotificationRuleParams) (*NotificationRule, error) {
	if !params.Condition.IsValid() {
		return nil, ErrInvalidCondition
	}

	if params.Condition == NotificationConditionOnConsecutiveFailures && params.ConditionValue < 1 {
		return nil, ErrConditionValueRequired
	}

	now := time.Now()
	rule := &NotificationRule{
		ID:             uuid.New(),
		WorkspaceID:    params.WorkspaceID,
		ChannelID:      params.ChannelID,
		ConnectionID:   params.ConnectionID,
		Condition:      params.Condition,
		ConditionValue: params.ConditionValue,
		Enabled:        params.Enabled,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	created, err := uc.storage.NotificationRules().Create(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("creating notification rule: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:  params.WorkspaceID,
		Action:       "create",
		ResourceType: "notification_rule",
		ResourceID:   created.ID,
		After: map[string]any{
			"channel_id":      created.ChannelID.String(),
			"connection_id":   created.ConnectionID.String(),
			"condition":       created.Condition.String(),
			"condition_value": created.ConditionValue,
			"enabled":         created.Enabled,
		},
	})

	return created, nil
}
