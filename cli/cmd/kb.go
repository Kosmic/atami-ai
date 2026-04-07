package cmd

import "github.com/spf13/cobra"

func newKbCmd(opts Options) *cobra.Command {
	kbCmd := &cobra.Command{
		Use:   "kb",
		Short: "Manage project knowledge base scaffolding",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Help()
		},
	}

	kbCmd.AddCommand(newKBInitCmd(opts))

	return kbCmd
}
