package cmd

import (
	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/app"
)

func newK8sExecutorCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "k8s-executor",
		Short: "Run the Kubernetes sync executor daemon",
		Long: `Runs the K8s sync executor. It claims work from the API server through
EXECUTOR_API_URL and EXECUTOR_API_TOKEN and never opens the database.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			return app.RunK8sExecutor(app.ExecutorOptions{DotEnvFiles: dotEnvs})
		},
	}
}
