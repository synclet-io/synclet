package cmd

import "github.com/spf13/cobra"

func newGenerateCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "generate",
		Aliases: []string{"gen", "g"},
	}
	cmd.AddCommand(newGenerateDotEnvCommand())

	return cmd
}
