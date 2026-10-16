package app

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-pnp/go-pnp/config/configutil"
	"github.com/go-pnp/go-pnp/http/pnphttpservercors"
	"github.com/joho/godotenv"
	"go.uber.org/fx"
	"go.uber.org/multierr"
)

// RunGenerateDotEnvFile appends to path every configuration key the binary
// reads that path does not set yet, with the default the process applies.
// Keys without a default are written
// commented out: an empty assignment would still count as set, defeating
// development secret generation and failing not-empty validation.
func RunGenerateDotEnvFile(path string) (rerr error) {
	configs, err := configInventory()
	if err != nil {
		return err
	}

	existing, err := godotenv.Read(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read existing %s: %w", path, err)
	}

	dotEnvFile, err := os.OpenFile(filepath.Clean(path), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
	if err != nil {
		return err
	}
	defer multierr.AppendInvoke(&rerr, multierr.Close(dotEnvFile))

	var out strings.Builder

	for _, configInfo := range configs {
		lines := make([]string, 0, len(configInfo.Fields))

		for _, field := range configInfo.Fields {
			if _, ok := existing[field.Key]; ok {
				continue
			}

			defaultValue := field.DefaultValue
			if value, ok := envDefaults[field.Key]; ok {
				defaultValue = value
			}

			lines = append(lines, dotEnvLine(field.Key, defaultValue, field.Required || field.NotEmpty))
		}

		if len(lines) > 0 {
			out.WriteString("# " + configInfo.ConfigType.String() + "\n")
			out.WriteString(strings.Join(lines, "\n") + "\n\n")
		}
	}

	_, err = dotEnvFile.WriteString(out.String())

	return err
}

func dotEnvLine(key, defaultValue string, required bool) string {
	switch {
	case defaultValue != "":
		return fmt.Sprintf("%s=%q", key, defaultValue)
	case required:
		return fmt.Sprintf("# %s= # required", key)
	default:
		return fmt.Sprintf("# %s=", key)
	}
}

// configInventory lists the configuration of every run mode, ordered by the
// first key of each config struct.
func configInventory() ([]configutil.ConfigInfo, error) {
	var inventory configutil.ConfigsInfoIn

	app := fx.New(
		NewFxAppOptions(&RunAppOptions{}),
		fx.Provide(
			configutil.NewPrefixedConfigInfoProvider[pnphttpservercors.Config](publicCORSConfigPrefix),
			configutil.NewPrefixedConfigInfoProvider[k8sExecutorConfig]("K8S_EXECUTOR_"),
			configutil.NewPrefixedConfigInfoProvider[executorBackendConfig](""),
		),
		fx.Populate(&inventory),
	)
	if err := app.Err(); err != nil {
		return nil, err
	}

	configs := make([]configutil.ConfigInfo, 0, len(inventory.ConfigsInfo))
	for _, configInfo := range inventory.ConfigsInfo {
		if len(configInfo.Fields) > 0 {
			configs = append(configs, configInfo)
		}
	}

	sort.Slice(configs, func(i, j int) bool {
		return configs[i].Fields[0].Key < configs[j].Fields[0].Key
	})

	return configs, nil
}
