package cmd

import (
	"github.com/synclet-io/synclet/app"

	"github.com/spf13/cobra"
)

func newServerCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the Synclet server",
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			standalone, err := cmd.Flags().GetBool("standalone")
			if err != nil {
				return err
			}

			opts := append([]app.RunOption{app.WithDotEnvFiles(dotEnvs...)}, app.ServerCommandOptions(standalone)...)

			return app.RunServer(opts...)
		},
	}

	cmd.Flags().Bool("standalone", false, "Also run background jobs in the same process")

	return cmd
}
