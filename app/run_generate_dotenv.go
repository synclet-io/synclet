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

// Roles name the processes that read a configuration key.
const (
	roleServer   = "server"
	roleExecutor = "executor"
	roleBoth     = "server, executor"
)

// configField is one configuration key of the binary and the processes that
// read it.
type configField struct {
	Key          string
	DefaultValue string
	Required     bool
	Role         string
}

// configGroup holds the keys read into one configuration struct.
type configGroup struct {
	Name   string
	Fields []configField
}

// RunGenerateDotEnvFile appends to path every configuration key the binary
// reads that path does not set yet, with the default the process applies and
// the processes that read it. Keys without a default are written commented
// out: an empty assignment would still count as set, defeating development
// secret generation and failing not-empty validation.
func RunGenerateDotEnvFile(path string) (rerr error) {
	groups, err := configInventory()
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

	for _, group := range groups {
		lines := make([]string, 0, len(group.Fields))

		for _, field := range group.Fields {
			if _, ok := existing[field.Key]; ok {
				continue
			}

			defaultValue := field.DefaultValue
			if value, ok := envDefaults[field.Key]; ok {
				defaultValue = value
			}

			lines = append(lines, dotEnvLine(field, defaultValue))
		}

		if len(lines) > 0 {
			out.WriteString("# " + group.Name + "\n")
			out.WriteString(strings.Join(lines, "\n") + "\n\n")
		}
	}

	_, err = dotEnvFile.WriteString(out.String())

	return err
}

// dotEnvLine renders one key with its default, or commented out without one,
// and ends with whether it is required and who reads it.
func dotEnvLine(field configField, defaultValue string) string {
	switch {
	case defaultValue != "":
		return fmt.Sprintf("%s=%q # %s", field.Key, defaultValue, field.Role)
	case field.Required:
		return fmt.Sprintf("# %s= # required, %s", field.Key, field.Role)
	default:
		return fmt.Sprintf("# %s= # %s", field.Key, field.Role)
	}
}

// configInventory lists the configuration of every process, grouped by
// configuration struct and ordered by the first key of each group. A key the
// server and the executors both read is listed once, with both roles.
func configInventory() ([]configGroup, error) {
	server, err := collectConfigInfo(
		NewFxAppOptions(&RunAppOptions{}),
		k8sExecutorConfigModule(),
		fx.Provide(configutil.NewPrefixedConfigInfoProvider[pnphttpservercors.Config](publicCORSConfigPrefix)),
	)
	if err != nil {
		return nil, err
	}

	var executors []configutil.ConfigInfo

	for _, graph := range []executorGraph{dockerExecutorGraph(), k8sExecutorGraph()} {
		infos, err := collectConfigInfo(graph.configOptions())
		if err != nil {
			return nil, err
		}

		executors = append(executors, infos...)
	}

	return mergeConfigRoles(server, executors), nil
}

func collectConfigInfo(options ...fx.Option) ([]configutil.ConfigInfo, error) {
	var inventory configutil.ConfigsInfoIn

	app := fx.New(fx.Options(options...), fx.NopLogger, fx.Populate(&inventory))
	if err := app.Err(); err != nil {
		return nil, err
	}

	return inventory.ConfigsInfo, nil
}

func mergeConfigRoles(server, executors []configutil.ConfigInfo) []configGroup {
	executorKeys := map[string]bool{}

	for _, info := range executors {
		for _, field := range info.Fields {
			executorKeys[field.Key] = true
		}
	}

	listed := map[string]bool{}

	var groups []configGroup

	add := func(infos []configutil.ConfigInfo, role func(key string) string) {
		for _, info := range infos {
			group := configGroup{Name: info.ConfigType.String()}

			for _, field := range info.Fields {
				if listed[field.Key] {
					continue
				}

				listed[field.Key] = true

				group.Fields = append(group.Fields, configField{
					Key:          field.Key,
					DefaultValue: field.DefaultValue,
					Required:     field.Required || field.NotEmpty,
					Role:         role(field.Key),
				})
			}

			if len(group.Fields) > 0 {
				groups = append(groups, group)
			}
		}
	}

	add(server, func(key string) string {
		if executorKeys[key] {
			return roleBoth
		}

		return roleServer
	})
	add(executors, func(string) string { return roleExecutor })

	sort.Slice(groups, func(i, j int) bool {
		return groups[i].Fields[0].Key < groups[j].Fields[0].Key
	})

	return groups
}
