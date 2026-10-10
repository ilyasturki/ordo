package cli

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"phasionary/internal/config"
	"phasionary/internal/data"
	"phasionary/internal/domain"
)

func newInitCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "init [name]",
		Short: "Initialize the data directory with a project",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dataDir, err := config.ResolveDataDir(viper.GetString("data"))
			if err != nil {
				return err
			}
			store := data.NewStore(dataDir)

			name := ""
			if len(args) == 1 {
				name = strings.TrimSpace(args[0])
			}

			var (
				project domain.Project
				created bool
			)
			if name == "" {
				project, err = store.InitDefault()
				if err != nil {
					return err
				}
			} else {
				if err := store.Ensure(); err != nil {
					return err
				}
				if existing, lookupErr := store.LoadProject(name); lookupErr == nil {
					project = existing
				} else if errors.Is(lookupErr, data.ErrProjectNotFound) {
					project, err = store.CreateProject(name)
					if err != nil {
						return err
					}
					created = true
				} else {
					return lookupErr
				}
			}

			if mgr, err := stateManagerFromViper(); err == nil {
				_ = mgr.SetProjectForDir(project.ID)
			}
			fmt.Fprintf(cmd.OutOrStdout(), "Initialized data directory: %s\n", dataDir)
			label := "Project"
			if created {
				label = "Created project"
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s: %s (%s)\n", label, project.Name, project.ID)
			return nil
		},
	}
	return cmd
}
