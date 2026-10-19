package coordinator

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/synclet-io/synclet/pkg/protocol"
)

func TestConnectorConfigRedactor(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "source-config"), []byte(`{"password":"src-secret-1"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dest-config"), []byte(`{"api_key":"dst-secret-2"}`), 0o600))

	redactor := connectorConfigRedactor(dir)
	assert.Equal(t, "*** and ***", redactor.Redact("src-secret-1 and dst-secret-2"))

	assert.Equal(t, "src-secret-1", connectorConfigRedactor(t.TempDir()).Redact("src-secret-1"), "missing configs redact nothing")
}

func TestLoadRecordTransformer(t *testing.T) {
	dir := t.TempDir()
	catalog := `{"streams":[{"stream":{"name":"orders","namespace":"public","json_schema":{"type":"object","properties":{"id":{"type":"integer"},"email":{"type":"string"},"secret":{"type":"string"}}}},"sync_mode":"full_refresh","destination_sync_mode":"append","generation_id":0,"minimum_generation_id":0,"sync_id":0,"selected_fields":[{"field_path":["id"]},{"field_path":["email"]}],"mappers":[{"hash":["email"]}]}]}`
	require.NoError(t, os.WriteFile(filepath.Join(dir, "router-catalog"), []byte(catalog), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "mapper-salt"), []byte("salt-1"), 0o600))

	transformer, err := loadRecordTransformer(Config{SecretsDir: dir, JobID: "job-1", ConnectionID: "conn-1"}, time.Unix(0, 0).UTC())
	require.NoError(t, err)
	require.NotNil(t, transformer)

	rec := &protocol.AirbyteRecordMessage{Stream: "orders", Namespace: "public", Data: json.RawMessage(`{"id":1,"email":"a@b.c","secret":"x"}`)}
	keep, err := transformer.Apply(rec)
	require.NoError(t, err)
	assert.True(t, keep)

	sum := sha256.Sum256([]byte("salt-1" + "a@b.c"))
	assert.JSONEq(t, `{"id":1,"email":"`+hex.EncodeToString(sum[:])+`"}`, string(rec.Data))

	t.Run("no router catalog leaves records unchanged", func(t *testing.T) {
		transformer, err := loadRecordTransformer(Config{SecretsDir: t.TempDir()}, time.Now())
		require.NoError(t, err)
		assert.Nil(t, transformer)
	})

	t.Run("mappers that do not compile fail", func(t *testing.T) {
		bad := t.TempDir()
		require.NoError(t, os.WriteFile(filepath.Join(bad, "router-catalog"), []byte(`{"streams":[{"stream":{"name":"orders","json_schema":{"type":"object"}},"sync_mode":"full_refresh","destination_sync_mode":"append","generation_id":0,"minimum_generation_id":0,"sync_id":0,"mappers":[{"hash":["missing"]}]}]}`), 0o600))

		_, err := loadRecordTransformer(Config{SecretsDir: bad}, time.Now())
		require.Error(t, err)
		assert.Contains(t, err.Error(), `stream "orders"`)
	})
}
