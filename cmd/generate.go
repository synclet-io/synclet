package cmd

import "github.com/spf13/cobra"

func newGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen", "g"},
		Short:   "Generate files from the application configuration",
	}
	cmd.AddCommand(newGenerateDotEnvCommand())

	return cmd
}
