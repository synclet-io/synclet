package pipelinesources_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice/pipelinesources"
)

func TestCreateSourceExecutorGroup(t *testing.T) {
	cases := []struct {
		name  string
		group string
		want  string
	}{
		{name: "trimmed and kept verbatim", group: "  Prod Cluster ACME  ", want: "Prod Cluster ACME"},
		{name: "empty is default", group: "", want: pipelineservice.DefaultExecutorGroup},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			ws := uuid.New()
			store := newSrcStore()
			conn := store.addConnector(ws, specWithoutSecrets)

			created, err := pipelinesources.NewCreateSource(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinesources.CreateSourceParams{
				WorkspaceID:        ws,
				Name:               "x",
				ManagedConnectorID: conn.ID,
				Config:             json.RawMessage(`{}`),
				ExecutorGroup:      tc.group,
			})
			require.NoError(t, err)
			assert.Equal(t, tc.want, created.ExecutorGroup)
		})
	}

	t.Run("invalid group is a validation error", func(t *testing.T) {
		ws := uuid.New()
		store := newSrcStore()
		conn := store.addConnector(ws, specWithoutSecrets)

		_, err := pipelinesources.NewCreateSource(store, newFakeSecrets(), &captureAudit{}, newLogger()).Execute(context.Background(), pipelinesources.CreateSourceParams{
			WorkspaceID:        ws,
			Name:               "x",
			ManagedConnectorID: conn.ID,
			Config:             json.RawMessage(`{}`),
			ExecutorGroup:      "a\nb",
		})

		var validationErr *pipelineservice.ValidationError
		require.ErrorAs(t, err, &validationErr)
		assert.Equal(t, "executor_group", validationErr.Field)
	})
}
