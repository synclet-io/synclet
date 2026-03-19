package notifyservice

import (
	"context"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// ListNotificationRulesParams holds parameters for listing notification rules.
type ListNotificationRulesParams struct {
	WorkspaceID  uuid.UUID
	ChannelID    *uuid.UUID
	ConnectionID *uuid.UUID
}

// ListNotificationRules returns all notification rules for a workspace.
type ListNotificationRules struct {
	storage Storage
}

// NewListNotificationRules creates a new ListNotificationRules use case.
func NewListNotificationRules(storage Storage) *ListNotificationRules {
	return &ListNotificationRules{storage: storage}
}

// Execute returns all notification rules for the given workspace.
func (uc *ListNotificationRules) Execute(ctx context.Context, params ListNotificationRulesParams) ([]*NotificationRule, error) {
	f := &NotificationRuleFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	}

	if params.ChannelID != nil {
		f.ChannelID = filter.Equals(*params.ChannelID)
	}
	if params.ConnectionID != nil {
		f.ConnectionID = filter.Equals(*params.ConnectionID)
	}

	return uc.storage.NotificationRules().Find(ctx, f)
}
