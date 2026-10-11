package cli

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"phasionary/internal/app"
	"phasionary/internal/config"
	"phasionary/internal/version"
)

func Execute() error {
	return newRootCmd().Execute()
}

func newRootCmd() *cobra.Command {
	var task string
	cmd := &cobra.Command{
		Use:   "phasionary",
		Short: "Terminal-first project planning tool",
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
			return app.Run(dataDir, viper.GetString("project"), task, cfgManager, workingDir, false)
		},
	}

	cmd.Version = fmt.Sprintf("%s (commit: %s, built: %s)", version.Version, version.Commit, version.BuildDate)
	cmd.SetOut(os.Stdout)
	cmd.SetErr(os.Stderr)
	cmd.SilenceUsage = true

	cmd.Flags().BoolP("version", "v", false, "Print version information")
	cmd.Flags().StringVar(&task, "task", "", "open on this task (ID, 4+ char ID prefix or title)")

	cmd.PersistentFlags().StringP("config", "c", "", "override config directory path")
	cmd.PersistentFlags().StringP("data", "d", "", "override data directory path")
	cmd.PersistentFlags().StringP("project", "p", "", "target project for commands")
	cmd.PersistentFlags().BoolP("json", "j", false, "output in JSON format")
	cmd.PersistentFlags().BoolP("quiet", "q", false, "suppress non-essential output")
	cmd.PersistentFlags().BoolP("long", "l", false, "show additional details (IDs, timestamps) in list output")

	// Deliberately no viper.AutomaticEnv(): without a prefix it maps the key
	// "data" to the bare environment variable DATA and "config" to CONFIG,
	// names generic enough to already exist in a shell for unrelated reasons —
	// silently relocating where projects are read and written. Every key that
	// should be settable from the environment is bound explicitly below, under
	// a PHASIONARY_ name, and BindEnv works without AutomaticEnv.
	if err := viper.BindEnv("config", config.EnvConfigPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := viper.BindEnv("data", config.EnvDataPath); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := viper.BindPFlag("config", cmd.PersistentFlags().Lookup("config")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := viper.BindPFlag("data", cmd.PersistentFlags().Lookup("data")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := viper.BindPFlag("project", cmd.PersistentFlags().Lookup("project")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := viper.BindPFlag("json", cmd.PersistentFlags().Lookup("json")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := viper.BindPFlag("quiet", cmd.PersistentFlags().Lookup("quiet")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}
	if err := viper.BindPFlag("long", cmd.PersistentFlags().Lookup("long")); err != nil {
		fmt.Fprintln(os.Stderr, err)
	}

	cmd.AddCommand(newInitCmd())
	cmd.AddCommand(newProjectCmd())
	cmd.AddCommand(newProjectsCmd())
	cmd.AddCommand(newPickCmd())
	cmd.AddCommand(newTaskCmd())
	cmd.AddCommand(newTasksCmd())
	cmd.AddCommand(newCategoryCmd())
	cmd.AddCommand(newCategoriesCmd())
	cmd.AddCommand(newVersionsCmd())
	cmd.AddCommand(newExportCmd())
	cmd.AddCommand(newImportCmd())
	cmd.AddCommand(newConfigCmd())
	cmd.AddCommand(newVersionCmd())
	cmd.AddCommand(newCompletionCmd())

	return cmd
}
