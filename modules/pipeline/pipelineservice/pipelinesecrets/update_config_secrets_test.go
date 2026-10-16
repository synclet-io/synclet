package pipelinesecrets

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/modules/pipeline/pipelineservice"
	"github.com/synclet-io/synclet/pkg/secretutil"
)

const (
	storedRef = "$secret:00000000-0000-0000-0000-000000000001"
	otherRef  = "$secret:00000000-0000-0000-0000-000000000002"
)

const testSpec = `{"connectionSpecification":{"properties":{
	"host":{"type":"string"},
	"password":{"type":"string","airbyte_secret":true},
	"tunnel":{"type":"object","properties":{"ssh_key":{"type":"string","airbyte_secret":true}}}
}}}`

type fakeSecrets struct {
	pipelineservice.SecretsProvider
	stored  map[string]string
	deleted []string
}

func newFakeSecrets() *fakeSecrets {
	return &fakeSecrets{stored: map[string]string{}}
}

func (f *fakeSecrets) StoreSecret(_ context.Context, _ string, _ uuid.UUID, plaintext string) (string, error) {
	ref := secretutil.MakeSecretRef(uuid.New())
	f.stored[ref] = plaintext

	return ref, nil
}

func (f *fakeSecrets) DeleteSecret(_ context.Context, ref string) error {
	f.deleted = append(f.deleted, ref)

	return nil
}

func TestMergeMaskedSecrets(t *testing.T) {
	stored := `{"host":"db","password":"` + storedRef + `","tunnel":{"ssh_key":"` + otherRef + `"}}`

	tests := []struct {
		name     string
		incoming string
		want     string
	}{
		{
			name:     "mask keeps stored ref",
			incoming: `{"host":"db2","password":"********","tunnel":{"ssh_key":"********"}}`,
			want:     `{"host":"db2","password":"` + storedRef + `","tunnel":{"ssh_key":"` + otherRef + `"}}`,
		},
		{
			name:     "removed field stays removed",
			incoming: `{"host":"db","tunnel":{}}`,
			want:     `{"host":"db","tunnel":{}}`,
		},
		{
			name:     "new plaintext replaces stored ref",
			incoming: `{"host":"db","password":"hunter2","tunnel":{"ssh_key":"********"}}`,
			want:     `{"host":"db","password":"hunter2","tunnel":{"ssh_key":"` + otherRef + `"}}`,
		},
		{
			name:     "mask without stored value is dropped",
			incoming: `{"host":"db","api_key":"********"}`,
			want:     `{"host":"db"}`,
		},
		{
			name:     "matching ref is kept",
			incoming: `{"password":"` + storedRef + `"}`,
			want:     `{"password":"` + storedRef + `"}`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := MergeMaskedSecrets(stored, tt.incoming)
			require.NoError(t, err)
			assert.JSONEq(t, tt.want, got)
		})
	}
}

func TestMergeMaskedSecrets_RejectsForeignRef(t *testing.T) {
	stored := `{"password":"` + storedRef + `"}`

	for name, incoming := range map[string]string{
		"different ref at path": `{"password":"` + otherRef + `"}`,
		"ref at new path":       `{"password":"********","token":"` + storedRef + `"}`,
	} {
		t.Run(name, func(t *testing.T) {
			_, err := MergeMaskedSecrets(stored, incoming)

			var validationErr *pipelineservice.ValidationError
			require.ErrorAs(t, err, &validationErr)
		})
	}
}

func TestUpdateConfigSecrets_MaskKeepsSecret(t *testing.T) {
	secrets := newFakeSecrets()
	existing := `{"host":"db","password":"` + storedRef + `"}`

	got, err := UpdateConfigSecrets(context.Background(), secrets, "source", uuid.New(), existing,
		`{"host":"db2","password":"********"}`, testSpec)
	require.NoError(t, err)

	assert.JSONEq(t, `{"host":"db2","password":"`+storedRef+`"}`, got)
	assert.Empty(t, secrets.stored)
	assert.Empty(t, secrets.deleted)
}

func TestUpdateConfigSecrets_NewPlaintextReplacesOld(t *testing.T) {
	secrets := newFakeSecrets()
	existing := `{"host":"db","password":"` + storedRef + `"}`

	got, err := UpdateConfigSecrets(context.Background(), secrets, "source", uuid.New(), existing,
		`{"host":"db","password":"hunter2"}`, testSpec)
	require.NoError(t, err)

	require.Len(t, secrets.stored, 1)

	var newRef string
	for ref, plaintext := range secrets.stored {
		newRef = ref

		assert.Equal(t, "hunter2", plaintext)
	}

	assert.JSONEq(t, `{"host":"db","password":"`+newRef+`"}`, got)
	assert.Equal(t, []string{storedRef}, secrets.deleted)
}

func TestUpdateConfigSecrets_RemovedFieldDeletesSecret(t *testing.T) {
	secrets := newFakeSecrets()
	existing := `{"host":"db","password":"` + storedRef + `","tunnel":{"ssh_key":"` + otherRef + `"}}`

	got, err := UpdateConfigSecrets(context.Background(), secrets, "source", uuid.New(), existing,
		`{"host":"db","tunnel":{"ssh_key":"********"}}`, testSpec)
	require.NoError(t, err)

	assert.JSONEq(t, `{"host":"db","tunnel":{"ssh_key":"`+otherRef+`"}}`, got)
	assert.Empty(t, secrets.stored)
	assert.Equal(t, []string{storedRef}, secrets.deleted)
}

func TestUpdateConfigSecrets_ForeignRefRejected(t *testing.T) {
	secrets := newFakeSecrets()

	_, err := UpdateConfigSecrets(context.Background(), secrets, "source", uuid.New(),
		`{"password":"`+storedRef+`"}`, `{"password":"`+otherRef+`"}`, testSpec)

	var validationErr *pipelineservice.ValidationError
	require.ErrorAs(t, err, &validationErr)
	assert.Empty(t, secrets.deleted)
}
