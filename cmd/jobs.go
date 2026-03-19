package cmd

import (
	"github.com/synclet-io/synclet/app"

	"github.com/spf13/cobra"
)

func newJobsCommand() *cobra.Command {
	return &cobra.Command{
		Use:   "jobs",
		Short: "Run background jobs",
		Run: func(cmd *cobra.Command, args []string) {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				panic(err)
			}

			app.RunJobs(app.WithDotEnvFiles(dotEnvs...))
		},
	}
}
