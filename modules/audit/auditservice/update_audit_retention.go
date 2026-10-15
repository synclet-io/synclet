package auditservice

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
)

type UpdateAuditRetentionParams struct {
	WorkspaceID   uuid.UUID
	RetentionDays int
}

type UpdateAuditRetention struct {
	storage Storage
	config  Config
}

func NewUpdateAuditRetention(storage Storage, config Config) *UpdateAuditRetention {
	return &UpdateAuditRetention{storage: storage, config: config}
}

func (uc *UpdateAuditRetention) Execute(ctx context.Context, params UpdateAuditRetentionParams) (AuditRetention, error) {
	if params.RetentionDays < 1 || params.RetentionDays > uc.config.MaxRetentionDays {
		return AuditRetention{}, &ValidationError{
			Message: fmt.Sprintf("retention_days must be between 1 and %d", uc.config.MaxRetentionDays),
		}
	}

	setting := &AuditRetentionSetting{
		WorkspaceID:   params.WorkspaceID,
		RetentionDays: params.RetentionDays,
		UpdatedAt:     time.Now(),
	}

	if _, err := uc.storage.AuditRetentionSettings().Save(ctx, setting); err != nil {
		return AuditRetention{}, fmt.Errorf("saving audit retention setting: %w", err)
	}

	return AuditRetention{
		RetentionDays:    params.RetentionDays,
		MaxRetentionDays: uc.config.MaxRetentionDays,
	}, nil
}
