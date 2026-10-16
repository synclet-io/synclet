package cmd

import (
	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/app"
)

func newK8sExecutorCommand() *cobra.Command {
	var standalone bool

	cmd := &cobra.Command{
		Use:   "k8s-executor",
		Short: "Run the Kubernetes sync executor daemon",
		Long: `Runs the K8s sync executor. In distributed mode (default), connects to the
API server via EXECUTOR_API_URL and EXECUTOR_API_TOKEN env vars. In standalone
mode (--standalone), runs in-process with direct DB access.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			opts := append([]app.RunOption{app.WithDotEnvFiles(dotEnvs...)}, app.ExecutorCommandOptions(app.WithK8sExecutor(), standalone)...)

			return app.RunServer(opts...)
		},
	}

	cmd.Flags().BoolVar(&standalone, "standalone", false, "Run in standalone mode with direct DB access (no API server needed)")

	return cmd
}
