package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ilyasturki/ordo"
)

func newChangelogCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "changelog",
		Short: "Print the changelog",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			_, err := fmt.Fprint(cmd.OutOrStdout(), ordo.Changelog)
			return err
		},
	}
}
