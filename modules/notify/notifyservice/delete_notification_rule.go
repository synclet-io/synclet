package notifyservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// DeleteNotificationRuleParams holds parameters for deleting a notification rule.
type DeleteNotificationRuleParams struct {
	ID          uuid.UUID
	WorkspaceID uuid.UUID
}

// DeleteNotificationRule deletes a notification rule by ID and workspace.
type DeleteNotificationRule struct {
	storage Storage
	audit   AuditRecorder
}

// NewDeleteNotificationRule creates a new DeleteNotificationRule use case.
func NewDeleteNotificationRule(storage Storage, audit AuditRecorder) *DeleteNotificationRule {
	return &DeleteNotificationRule{storage: storage, audit: audit}
}

// Execute deletes the notification rule matching the given ID and workspace.
// It returns a not-found error when no such rule exists.
func (uc *DeleteNotificationRule) Execute(ctx context.Context, params DeleteNotificationRuleParams) error {
	existing, err := uc.storage.NotificationRules().First(ctx, &NotificationRuleFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		return fmt.Errorf("getting notification rule: %w", err)
	}

	if err := uc.storage.NotificationRules().Delete(ctx, &NotificationRuleFilter{
		ID:          filter.Equals(existing.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}); err != nil {
		return fmt.Errorf("deleting notification rule: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:  params.WorkspaceID,
		Action:       "delete",
		ResourceType: "notification_rule",
		ResourceID:   existing.ID,
		Before: map[string]any{
			"channel_id":      existing.ChannelID.String(),
			"connection_id":   existing.ConnectionID.String(),
			"condition":       existing.Condition.String(),
			"condition_value": existing.ConditionValue,
			"enabled":         existing.Enabled,
		},
	})

	return nil
}
