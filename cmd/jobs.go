package cmd

import (
	"github.com/synclet-io/synclet/app"

	"github.com/spf13/cobra"
)

func newJobsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "jobs",
		Short: "Run background jobs",
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			return app.RunJobs(append([]app.RunOption{app.WithDotEnvFiles(dotEnvs...)}, app.JobsCommandOptions()...)...)
		},
	}
}
