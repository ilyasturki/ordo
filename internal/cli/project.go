package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"phasionary/internal/app"
	"phasionary/internal/config"
	"phasionary/internal/data"
	"phasionary/internal/domain"
)

func newPickCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "pick",
		Aliases: []string{"switch"},
		Short:   "Launch the TUI and open the project picker",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := config.ResolveConfigPath(viper.GetString("config"))
			if err != nil {
				return err
			}
			cfgManager := config.NewManager(configPath)
			if err := cfgManager.Load(); err != nil {
				return err
			}
			dataDir, err := config.ResolveDataDir(viper.GetString("data"))
			if err != nil {
				return err
			}
			workingDir, err := os.Getwd()
			if err != nil {
				return err
			}
			return app.Run(dataDir, viper.GetString("project"), "", cfgManager, workingDir, true)
		},
	}
	return cmd
}

func newProjectsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "projects",
		Aliases: []string{"ps"},
		Short:   "List all projects",
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storeFromViper()
			if err != nil {
				return err
			}
			projects, err := store.ListProjects()
			if err != nil {
				return err
			}
			return writeProjects(cmd.OutOrStdout(), projects)
		},
	}
	return cmd
}

func newProjectCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "project",
		Short: "Manage projects",
	}

	cmd.AddCommand(newProjectShowCmd())
	cmd.AddCommand(newProjectAddCmd())
	cmd.AddCommand(newProjectEditCmd())
	cmd.AddCommand(newProjectDeleteCmd())
	cmd.AddCommand(newProjectLinkCmd())
	cmd.AddCommand(newProjectUnlinkCmd())

	return cmd
}

func newProjectShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "show [name-or-id]",
		Aliases:           []string{"p"},
		Short:             "Show project details",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			return viewProject(projectSelector(args), func(project domain.Project) error {
				return writeProjectDetail(cmd.OutOrStdout(), project)
			})
		},
	}
	return cmd
}

func newProjectAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "add <name>",
		Aliases: []string{"pa"},
		Short:   "Add a new project",
		Args:    exactArgs("name"),
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storeFromViper()
			if err != nil {
				return err
			}
			project, err := store.CreateProject(args[0])
			if err != nil {
				return err
			}
			if mgr, err := stateManagerFromViper(); err == nil {
				_ = mgr.SetProjectForDir(project.ID)
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Created project: %s (%s)", project.Name, project.ID))
			return nil
		},
	}
	return cmd
}

func newProjectEditCmd() *cobra.Command {
	var name string

	cmd := &cobra.Command{
		Use:               "edit [name-or-id]",
		Aliases:           []string{"pe"},
		Short:             "Edit project (rename)",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" {
				return fmt.Errorf("--name is required")
			}
			store, project, err := loadStoreAndProject(projectSelector(args))
			if err != nil {
				return err
			}
			// RenameProject takes the global lock so the duplicate-name
			// check + save are atomic across processes.
			if _, err := store.RenameProject(project.ID, name); err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Renamed project to: %s", name))
			return nil
		},
	}

	cmd.Flags().StringVarP(&name, "name", "n", "", "new project name")

	return cmd
}

func newProjectDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:               "delete [name-or-id]",
		Aliases:           []string{"pd"},
		Short:             "Delete a project",
		Args:              cobra.MaximumNArgs(1),
		ValidArgsFunction: completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, project, err := loadStoreAndProject(projectSelector(args))
			if err != nil {
				return err
			}

			if !force {
				fmt.Fprintf(cmd.OutOrStdout(), "Delete project %q? [y/N]: ", project.Name)
				var response string
				if _, err := fmt.Fscanln(cmd.InOrStdin(), &response); err != nil {
					return nil
				}
				if response != "y" && response != "Y" {
					fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
					return nil
				}
			}

			if err := store.DeleteProject(project.ID); err != nil {
				return err
			}

			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Deleted project: %s", project.Name))
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation prompt")

	return cmd
}

func newProjectLinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "link <project>",
		Aliases:           []string{"pl"},
		Short:             "Link the current directory to a project",
		Args:              exactArgs("project"),
		ValidArgsFunction: completeProjects,
		RunE: func(cmd *cobra.Command, args []string) error {
			store, err := storeFromViper()
			if err != nil {
				return err
			}
			project, err := store.LoadProject(args[0])
			if err != nil {
				return err
			}
			mgr, err := stateManagerFromViper()
			if err != nil {
				return err
			}
			if err := mgr.SetProjectForDir(project.ID); err != nil {
				return err
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Linked directory to project: %s", project.Name))
			return nil
		},
	}
	return cmd
}

func newProjectUnlinkCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "unlink",
		Aliases: []string{"pul"},
		Short:   "Remove the link between the current directory and its project",
		Args:    cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			mgr, err := stateManagerFromViper()
			if err != nil {
				return err
			}
			prevID, err := mgr.UnlinkDir()
			if err != nil {
				return err
			}
			if prevID == "" {
				fmt.Fprintln(cmd.OutOrStdout(), "No project linked to this directory.")
				return nil
			}
			store, err := storeFromViper()
			if err != nil {
				return err
			}
			name := prevID
			if p, err := store.LoadProject(prevID); err == nil {
				name = p.Name
			}
			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Unlinked directory from project: %s", name))
			return nil
		},
	}
	return cmd
}

func storeFromViper() (*data.Store, error) {
	dataDir, err := config.ResolveDataDir(viper.GetString("data"))
	if err != nil {
		return nil, err
	}
	return data.NewStore(dataDir), nil
}
