package cmd

import (
	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/pkg/buildinfo"
)

func Run() error {
	rootCmd := &cobra.Command{
		Use:     "synclet",
		Short:   "Synclet — data synchronization platform",
		Version: buildinfo.Version(),
	}

	rootCmd.PersistentFlags().StringArray("dotenv", []string{}, ".env file paths")

	rootCmd.AddCommand(newServerCommand())
	rootCmd.AddCommand(newJobsCommand())
	rootCmd.AddCommand(newMigrateCommand())
	rootCmd.AddCommand(newOrchestrateCommand())
	rootCmd.AddCommand(newDockerExecutorCommand())
	rootCmd.AddCommand(newK8sExecutorCommand())
	rootCmd.AddCommand(newGenerateCommand())
	rootCmd.AddCommand(newUsersCommand())

	return rootCmd.Execute()
}
