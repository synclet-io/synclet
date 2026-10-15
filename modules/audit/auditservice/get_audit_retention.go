package auditservice

import (
	"context"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// AuditRetention is the retention window in effect for a workspace.
type AuditRetention struct {
	RetentionDays    int
	MaxRetentionDays int
}

type GetAuditRetention struct {
	storage Storage
	config  Config
}

func NewGetAuditRetention(storage Storage, config Config) *GetAuditRetention {
	return &GetAuditRetention{storage: storage, config: config}
}

func (uc *GetAuditRetention) Execute(ctx context.Context, workspaceID uuid.UUID) (AuditRetention, error) {
	setting, err := uc.storage.AuditRetentionSettings().First(ctx, &AuditRetentionSettingFilter{
		WorkspaceID: filter.Equals(workspaceID),
	})
	if err != nil {
		if !errors.Is(err, ErrAuditRetentionSettingNotFound) {
			return AuditRetention{}, fmt.Errorf("getting audit retention setting: %w", err)
		}

		setting = nil
	}

	return AuditRetention{
		RetentionDays:    uc.config.EffectiveRetentionDays(setting),
		MaxRetentionDays: uc.config.MaxRetentionDays,
	}, nil
}
