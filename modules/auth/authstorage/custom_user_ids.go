package authstorage

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/synclet-io/synclet/modules/auth/authservice"
)

// UserIDStorage answers id-only questions about user accounts.
type UserIDStorage struct {
	db *gorm.DB
}

func NewUserIDStorage(db *gorm.DB) *UserIDStorage {
	return &UserIDStorage{db: db}
}

var _ authservice.UserIDStorage = (*UserIDStorage)(nil)

func (s *UserIDStorage) ExistingUserIDs(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	if len(ids) == 0 {
		return nil, nil
	}

	values := make([]any, len(ids))
	for i, id := range ids {
		values[i] = id
	}

	var existing []uuid.UUID
	if err := s.db.WithContext(ctx).Model(&dbUser{}).
		Clauses(clause.IN{Column: clause.Column{Name: "id"}, Values: values}).
		Pluck("id", &existing).Error; err != nil {
		return nil, fmt.Errorf("finding existing user ids: %w", wrapUserQueryError(err))
	}

	return existing, nil
}
