package notifyservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
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
	storage     Storage
	connections ConnectionChecker
	audit       AuditRecorder
}

// NewCreateNotificationRule creates a new CreateNotificationRule use case.
func NewCreateNotificationRule(storage Storage, connections ConnectionChecker, audit AuditRecorder) *CreateNotificationRule {
	return &CreateNotificationRule{storage: storage, connections: connections, audit: audit}
}

// Execute creates a notification rule with the given parameters. The channel
// and, when set, the connection must belong to the rule's workspace.
func (uc *CreateNotificationRule) Execute(ctx context.Context, params CreateNotificationRuleParams) (*NotificationRule, error) {
	if !params.Condition.IsValid() {
		return nil, ErrInvalidCondition
	}

	if params.Condition == NotificationConditionOnConsecutiveFailures && params.ConditionValue < 1 {
		return nil, ErrConditionValueRequired
	}

	if _, err := uc.storage.NotificationChannels().First(ctx, &NotificationChannelFilter{
		ID:          filter.Equals(params.ChannelID),
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}); err != nil {
		return nil, fmt.Errorf("getting notification channel: %w", err)
	}

	if params.ConnectionID != uuid.Nil {
		exists, err := uc.connections.ConnectionExists(ctx, params.WorkspaceID, params.ConnectionID)
		if err != nil {
			return nil, fmt.Errorf("checking connection: %w", err)
		}

		if !exists {
			return nil, ErrConnectionNotFound
		}
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
