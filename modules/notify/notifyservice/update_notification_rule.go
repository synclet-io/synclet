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
}

// NewUpdateNotificationRule creates a new UpdateNotificationRule use case.
func NewUpdateNotificationRule(storage Storage) *UpdateNotificationRule {
	return &UpdateNotificationRule{storage: storage}
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

	if params.Condition != nil {
		if !params.Condition.IsValid() {
			return nil, fmt.Errorf("invalid condition: must be one of on_failure, on_consecutive_failures, on_zero_records")
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
		return nil, fmt.Errorf("condition_value must be >= 1 for on_consecutive_failures")
	}

	rule.UpdatedAt = time.Now()

	updated, err := uc.storage.NotificationRules().Update(ctx, rule)
	if err != nil {
		return nil, fmt.Errorf("updating notification rule: %w", err)
	}

	return updated, nil
}
