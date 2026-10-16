package auditservice

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"

	"github.com/synclet-io/synclet/pkg/connectutil"
)

type UpdateAuditRetentionParams struct {
	WorkspaceID   uuid.UUID
	RetentionDays int
}

// auditRecorder records the configuration change this use case makes.
type auditRecorder interface {
	Record(ctx context.Context, params RecordAuditEventParams)
}

type UpdateAuditRetention struct {
	storage Storage
	config  Config
	audit   auditRecorder
}

func NewUpdateAuditRetention(storage Storage, config Config, audit *RecordAuditEvent) *UpdateAuditRetention {
	return &UpdateAuditRetention{storage: storage, config: config, audit: audit}
}

func (uc *UpdateAuditRetention) Execute(ctx context.Context, params UpdateAuditRetentionParams) (AuditRetention, error) {
	if params.RetentionDays < 1 || params.RetentionDays > uc.config.MaxRetentionDays {
		return AuditRetention{}, &ValidationError{
			Message: fmt.Sprintf("retention_days must be between 1 and %d", uc.config.MaxRetentionDays),
		}
	}

	previous, err := uc.storage.AuditRetentionSettings().First(ctx, &AuditRetentionSettingFilter{
		WorkspaceID: filter.Equals(params.WorkspaceID),
	})
	if err != nil {
		if !errors.Is(err, ErrAuditRetentionSettingNotFound) {
			return AuditRetention{}, fmt.Errorf("getting audit retention setting: %w", err)
		}

		previous = nil
	}

	setting := &AuditRetentionSetting{
		WorkspaceID:   params.WorkspaceID,
		RetentionDays: params.RetentionDays,
		UpdatedAt:     time.Now(),
	}

	if _, err := uc.storage.AuditRetentionSettings().Save(ctx, setting); err != nil {
		return AuditRetention{}, fmt.Errorf("saving audit retention setting: %w", err)
	}

	actorType := ActorTypeUser

	actorID, err := connectutil.UserIDFromContext(ctx)
	if err != nil {
		actorType = ActorTypeSystem
		actorID = uuid.Nil
	}

	uc.audit.Record(ctx, RecordAuditEventParams{
		WorkspaceID:   params.WorkspaceID,
		ActorType:     actorType,
		ActorID:       actorID,
		Action:        ActionUpdate,
		ResourceType:  ResourceTypeWorkspace,
		ResourceID:    params.WorkspaceID,
		ResourceLabel: "audit retention",
		Before:        map[string]any{"audit_retention_days": uc.config.EffectiveRetentionDays(previous)},
		After:         map[string]any{"audit_retention_days": params.RetentionDays},
	})

	return AuditRetention{
		RetentionDays:    params.RetentionDays,
		MaxRetentionDays: uc.config.MaxRetentionDays,
	}, nil
}
