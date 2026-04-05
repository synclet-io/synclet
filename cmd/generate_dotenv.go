package cmd

import (
	"github.com/spf13/cobra"
	"github.com/synclet-io/synclet/app"
)

func newGenerateDotEnvCommand() *cobra.Command {
	return &cobra.Command{
		Use:     "dotenv",
		Aliases: []string{"env"},
		Short:   "Generate .env file",
		RunE: func(cmd *cobra.Command, args []string) error {
			return app.RunGenerateDotEnvFile()
		},
	}
}
