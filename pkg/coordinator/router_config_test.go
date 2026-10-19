package coordinator

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeRouterInputs(t *testing.T, sourceExitCode string) Config {
	t.Helper()

	cfg := Config{SecretsDir: t.TempDir(), DataDir: t.TempDir()}
	require.NoError(t, os.WriteFile(filepath.Join(cfg.SecretsDir, "source-catalog"),
		[]byte(`{"streams":[{"stream":{"name":"users","namespace":"public","json_schema":{}},"sync_mode":"full_refresh","destination_sync_mode":"overwrite"}]}`), 0o600))

	if sourceExitCode != "" {
		require.NoError(t, os.WriteFile(filepath.Join(cfg.DataDir, "source.exitcode"), []byte(sourceExitCode), 0o600))
	}

	return cfg
}

func TestRouterConfig_CarriesCatalogWithoutNamespaceDefinition(t *testing.T) {
	routeCfg := routerConfig(writeRouterInputs(t, "0"))

	require.NotNil(t, routeCfg.Catalog)
	require.Len(t, routeCfg.Catalog.Streams, 1)
	assert.Equal(t, "users", routeCfg.Catalog.Streams[0].Stream.Name)
	assert.Equal(t, "public", routeCfg.Catalog.Streams[0].Stream.Namespace)
	assert.Nil(t, routeCfg.Rewriter)
}

func TestRouterConfig_SourceSucceededReadsSourceExitCode(t *testing.T) {
	t.Run("exit 0", func(t *testing.T) {
		routeCfg := routerConfig(writeRouterInputs(t, "0\n"))
		require.NotNil(t, routeCfg.SourceSucceeded)
		assert.True(t, routeCfg.SourceSucceeded())
	})

	t.Run("exit 1", func(t *testing.T) {
		routeCfg := routerConfig(writeRouterInputs(t, "1\n"))
		require.NotNil(t, routeCfg.SourceSucceeded)
		assert.False(t, routeCfg.SourceSucceeded())
	})
}

func TestRouterConfig_BuildsRewriterWithNamespaceDefinition(t *testing.T) {
	cfg := writeRouterInputs(t, "0")
	cfg.NamespaceDefinition = "source"
	cfg.StreamPrefix = "dst_"

	routeCfg := routerConfig(cfg)
	require.NotNil(t, routeCfg.Rewriter)
	require.NotNil(t, routeCfg.Catalog)
}
