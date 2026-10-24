package authservice

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegistrationOpen(t *testing.T) {
	ctx := context.Background()

	empty := newFakeStorage()
	withUser := newFakeStorage()
	withUser.seedUser(&User{ID: uuid.New(), Email: "admin@example.com"})

	tests := []struct {
		name    string
		storage *fakeStorage
		enabled bool
		want    bool
	}{
		{name: "enabled", storage: withUser, enabled: true, want: true},
		{name: "disabled, no user yet", storage: empty, want: true},
		{name: "disabled, a user exists", storage: withUser, want: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			open, err := NewRegistrationOpen(tt.storage, Config{RegistrationEnabled: tt.enabled}).Execute(ctx)
			require.NoError(t, err)
			assert.Equal(t, tt.want, open)
		})
	}
}
