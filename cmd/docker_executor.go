package cmd

import (
	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/app"
)

func newDockerExecutorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "docker-executor",
		Short: "Run the Docker sync executor daemon",
		Long: `Runs the Docker sync executor. It claims work from the API server through
EXECUTOR_API_URL and EXECUTOR_API_TOKEN and never opens the database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			return app.RunDockerExecutor(app.ExecutorOptions{DotEnvFiles: dotEnvs})
		},
	}
}
