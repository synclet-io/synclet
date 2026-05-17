package notifyservice

import (
	"context"

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
func (uc *DeleteNotificationRule) Execute(ctx context.Context, params DeleteNotificationRuleParams) error {
	if err := uc.storage.NotificationRules().Delete(ctx, &NotificationRuleFilter{
		ID:          filter.Equals(params.ID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}); err != nil {
		return err
	}

	uc.audit.Record(ctx, AuditEvent{
		WorkspaceID:  params.WorkspaceID,
		Action:       "delete",
		ResourceType: "notification_rule",
		ResourceID:   params.ID,
	})

	return nil
}
