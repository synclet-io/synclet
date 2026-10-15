package authservice

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/saturn4er/boilerplate-go/lib/filter"
)

// UpdateProfile updates a user's name.
type UpdateProfile struct {
	storage Storage
	audit   AuditRecorder
}

// NewUpdateProfile creates a new UpdateProfile use case.
func NewUpdateProfile(storage Storage, audit AuditRecorder) *UpdateProfile {
	return &UpdateProfile{storage: storage, audit: audit}
}

// Execute updates the name of the user with the given ID.
func (uc *UpdateProfile) Execute(ctx context.Context, userID uuid.UUID, name string) (*User, error) {
	user, err := uc.storage.Users().First(ctx, &UserFilter{
		ID: filter.Equals(userID),
	})
	if err != nil {
		return nil, fmt.Errorf("fetching user: %w", err)
	}

	before := map[string]any{"name": user.Name, "email": user.Email}

	user.Name = name

	updated, err := uc.storage.Users().Update(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("updating user: %w", err)
	}

	uc.audit.Record(ctx, AuditEvent{
		ActorID:       updated.ID,
		ActorLabel:    updated.Email,
		Action:        "update",
		ResourceType:  "user",
		ResourceID:    updated.ID,
		ResourceLabel: updated.Email,
		Before:        before,
		After:         map[string]any{"name": updated.Name, "email": updated.Email},
	})

	return updated, nil
}
