package cmd

import (
	"github.com/spf13/cobra"
)

func Run() error {
	rootCmd := &cobra.Command{
		Use:   "synclet",
		Short: "Synclet — data synchronization platform",
	}

	rootCmd.PersistentFlags().StringArray("dotenv", []string{}, ".env file paths")
	rootCmd.Version = "0.1.0"

	rootCmd.AddCommand(newServerCommand())
	rootCmd.AddCommand(newJobsCommand())
	rootCmd.AddCommand(newMigrateCommand())
	rootCmd.AddCommand(newOrchestrateCommand())
	rootCmd.AddCommand(newDockerExecutorCommand())
	rootCmd.AddCommand(newK8sExecutorCommand())

	return rootCmd.Execute()
}
