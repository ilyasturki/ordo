package cli

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/ilyasturki/ordo/internal/config"
)

func newConfigCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "config",
		Aliases: []string{"cfg"},
		Short:   "Show or edit configuration",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := config.ResolveConfigPath(viper.GetString("config"))
			if err != nil {
				return err
			}
			cfgManager := config.NewManager(configPath)
			if err := cfgManager.Load(); err != nil {
				return err
			}

			cfg := cfgManager.Get()

			if getOutputFormat() == FormatJSON {
				return writeJSON(cmd.OutOrStdout(), cfg)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "Config file: %s\n\n", cfgManager.Path())
			data, err := json.MarshalIndent(cfg, "", "  ")
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), string(data))
			return nil
		},
	}

	cmd.AddCommand(newConfigPathCmd())
	cmd.AddCommand(newConfigSetCmd())

	return cmd
}

func newConfigPathCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "path",
		Short: "Show config file path",
		RunE: func(cmd *cobra.Command, args []string) error {
			configPath, err := config.ResolveConfigPath(viper.GetString("config"))
			if err != nil {
				return err
			}
			fmt.Fprintln(cmd.OutOrStdout(), configPath)
			return nil
		},
	}
}

func newConfigSetCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "set <key> <value>",
		Short: "Set a configuration value",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			key := args[0]
			value := args[1]

			configPath, err := config.ResolveConfigPath(viper.GetString("config"))
			if err != nil {
				return err
			}
			cfgManager := config.NewManager(configPath)
			if err := cfgManager.Load(); err != nil {
				return err
			}

			switch key {
			case "status_display":
				if value != config.StatusDisplayText && value != config.StatusDisplayIcons {
					return fmt.Errorf("invalid value for status_display: %s (use text or icons)", value)
				}
				err = cfgManager.Update(func(c *config.Config) {
					c.StatusDisplay = value
				})
			case "copy_via":
				if value != config.CopyViaOSC52 && value != config.CopyViaWlCopy {
					return fmt.Errorf("invalid value for copy_via: %s (use osc52 or wl-copy)", value)
				}
				err = cfgManager.Update(func(c *config.Config) {
					c.CopyVia = value
				})
			default:
				return fmt.Errorf("unknown config key: %s", key)
			}

			if err != nil {
				return err
			}

			writeSuccess(cmd.OutOrStdout(), fmt.Sprintf("Set %s = %s", key, value))
			return nil
		},
	}
}
