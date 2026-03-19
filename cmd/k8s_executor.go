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
		Run: func(cmd *cobra.Command, args []string) {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				panic(err)
			}

			opts := []app.RunOption{
				app.WithDotEnvFiles(dotEnvs...),
				app.WithRunJobs(),
				app.WithK8sExecutor(),
			}
			if standalone {
				opts = append(opts, app.WithStandalone())
			}
			app.RunServer(opts...)
		},
	}

	cmd.Flags().BoolVar(&standalone, "standalone", false, "Run in standalone mode with direct DB access (no API server needed)")

	return cmd
}
