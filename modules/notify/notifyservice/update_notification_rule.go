package notifyservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// UpdateNotificationRuleParams holds parameters for updating a notification rule.
type UpdateNotificationRuleParams struct {
	ID             uuid.UUID
	WorkspaceID    uuid.UUID
	Condition      *NotificationCondition
	ConditionValue *int
	Enabled        *bool
}

// UpdateNotificationRule updates an existing notification rule.
type UpdateNotificationRule struct {
	storage Storage
	audit   AuditRecorder
}

// NewUpdateNotificationRule creates a new UpdateNotificationRule use case.
func NewUpdateNotificationRule(storage Storage, audit AuditRecorder) *UpdateNotificationRule {
	return &UpdateNotificationRule{storage: storage, audit: audit}
}

// Execute updates the notification rule matching the given ID and workspace.
func (uc *UpdateNotificationRule) Execute(ctx context.Context, params UpdateNotificationRuleParams) (*NotificationRule, error) {
	rule, err := uc.storage.NotificationRules().First(ctx, &NotificationRuleFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return nil, fmt.Errorf("getting notification rule: %w", err)
	}

	before := map[string]any{
		"condition":       rule.Condition.String(),
		"condition_value": rule.ConditionValue,
		"enabled":         rule.Enabled,
	}

	if params.Condition != nil {
		if !params.Condition.IsValid() {
			return nil, ErrInvalidCondition
		}

		rule.Condition = *params.Condition
	}

	if params.ConditionValue != nil {
		rule.ConditionValue = *params.ConditionValue
	}

	if params.Enabled != nil {
		rule.Enabled = *params.Enabled
	}

	// Validate condition_value after potential updates.
	if rule.Condition == NotificationConditionOnConsecutiveFailures && rule.ConditionValue < 1 {
		return nil, ErrConditionValueRequired
	}

	rule.UpdatedAt = time.Now()

	updated, err := uc.storage.NotificationRules().Update(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("updating notification rule: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:  params.WorkspaceID,
		Action:       "update",
		ResourceType: "notification_rule",
		ResourceID:   updated.ID,
		Before:       before,
		After: map[string]any{
			"condition":       updated.Condition.String(),
			"condition_value": updated.ConditionValue,
			"enabled":         updated.Enabled,
		},
	})

	return updated, nil
}
