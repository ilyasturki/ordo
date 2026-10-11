package cli

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ilyasturki/phasionary/internal/version"
)

type VersionOutput struct {
	Version string `json:"version"`
	Commit  string `json:"commit"`
	Built   string `json:"built"`
}

func newVersionCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "version",
		Aliases: []string{"v"},
		Short:   "Print version information",
		RunE: func(cmd *cobra.Command, args []string) error {
			if getOutputFormat() == FormatJSON {
				return writeJSON(cmd.OutOrStdout(), VersionOutput{
					Version: version.Version,
					Commit:  version.Commit,
					Built:   version.BuildDate,
				})
			}
			fmt.Fprintf(cmd.OutOrStdout(), "phasionary %s (commit: %s, built: %s)\n",
				version.Version, version.Commit, version.BuildDate)
			return nil
		},
	}
}
