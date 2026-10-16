package cmd

import (
	"github.com/spf13/cobra"

	"github.com/synclet-io/synclet/app"
)

func newGenerateDotEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "dotenv",
		Aliases: []string{"env"},
		Short:   "Append every configuration variable missing from the .env file",
		Long: `Appends every configuration variable the binary reads, with its default, to the
first --dotenv file (.env when none is given). Variables already set in that file
are left alone; variables without a default are written commented out.`,
		RunE: func(cmd *cobra.Command, args []string) error {
			dotEnvs, err := cmd.Flags().GetStringArray("dotenv")
			if err != nil {
				return err
			}

			path := ".env"
			if len(dotEnvs) > 0 {
				path = dotEnvs[0]
			}

			return app.RunGenerateDotEnvFile(path)
		},
	}
}
