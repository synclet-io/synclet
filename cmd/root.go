package cmd

import (
	"runtime/debug"

	"github.com/spf13/cobra"
)

// version is set at build time:
//
//	go build -ldflags "-X github.com/synclet-io/synclet/cmd.version=v0.4.0"
//
//nolint:gochecknoglobals
var version = ""

func Run() error {
	rootCmd := &cobra.Command{
		Use:     "synclet",
		Short:   "Synclet — data synchronization platform",
		Version: buildVersion(),
	}

	rootCmd.PersistentFlags().StringArray("dotenv", []string{}, ".env file paths")

	rootCmd.AddCommand(newServerCommand())
	rootCmd.AddCommand(newJobsCommand())
	rootCmd.AddCommand(newMigrateCommand())
	rootCmd.AddCommand(newOrchestrateCommand())
	rootCmd.AddCommand(newDockerExecutorCommand())
	rootCmd.AddCommand(newK8sExecutorCommand())
	rootCmd.AddCommand(newGenerateCommand())

	return rootCmd.Execute()
}

// buildVersion returns the -ldflags version, else the VCS revision Go stamped
// into the binary, else "dev".
func buildVersion() string {
	if version != "" {
		return version
	}

	info, ok := debug.ReadBuildInfo()
	if !ok {
		return "dev"
	}

	for _, setting := range info.Settings {
		if setting.Key == "vcs.revision" && len(setting.Value) >= 12 {
			return "dev-" + setting.Value[:12]
		}
	}

	return "dev"
}
