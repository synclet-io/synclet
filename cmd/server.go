package cmd

import (
	"github.com/synclet-io/synclet/app"

	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the Synclet server",
		Run: func(cmd *cobra.Command, args []string) {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				panic(err)
			}

			standalone, err := cmd.Flags().GetBool("standalone")
			if err != nil {
				panic(err)
			}

			opts := []app.RunOption{
				app.WithDotEnvFiles(dotEnvs...),
			}
			if standalone {
				opts = append(opts, app.WithRunJobs(), app.WithStandalone(), app.WithAutoExecutor())
			}

			app.RunServer(opts...)
		},
	}

	cmd.Flags().Bool("standalone", false, "Also run background jobs in the same process")

	return cmd
}
