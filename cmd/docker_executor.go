package cmd

import (
	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/app"
)

func newDockerExecutorCommand() *cobra.Command {
	var standalone bool

	cmd := &cobra.Command{
		Use:   "docker-executor",
		Short: "Run the Docker sync executor daemon",
		Long: `Runs the Docker sync executor. In distributed mode (default), connects to the
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
				app.WithDockerExecutor(),
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
