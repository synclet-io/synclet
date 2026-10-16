package coordinator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConnectorConfigRedactor(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "source-config"), []byte(`{"password":"src-secret-1"}`), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "dest-config"), []byte(`{"api_key":"dst-secret-2"}`), 0o600))

	redactor := connectorConfigRedactor(dir)
	assert.Equal(t, "*** and ***", redactor.Redact("src-secret-1 and dst-secret-2"))

	assert.Equal(t, "src-secret-1", connectorConfigRedactor(t.TempDir()).Redact("src-secret-1"), "missing configs redact nothing")
}
