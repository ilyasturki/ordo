package config

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const (
	EnvDataPath   = "PHASIONARY_DATA_PATH"
	EnvConfigPath = "PHASIONARY_CONFIG_PATH"

	StatusDisplayText  = "text"
	StatusDisplayIcons = "icons"

	PriorityColorFull = "full"
	PriorityColorIcon = "icon"
	PriorityColorNone = "none"

	CopyViaOSC52  = "osc52"
	CopyViaWlCopy = "wl-copy"
)

// Config holds user preferences.
type Config struct {
	StatusDisplay               string `json:"status_display,omitempty"`
	PriorityColor               string `json:"priority_color,omitempty"`
	CopyVia                     string `json:"copy_via,omitempty"`
	ShowShortcutBar             bool   `json:"show_shortcut_bar"`
	ExpandDescriptionsByDefault bool   `json:"expand_descriptions_by_default"`

	TaskCommands []TaskCommand `json:"task_commands,omitempty"`
}

// Run is executed directly, never through a shell; the task arrives as PHASIONARY_* variables.
type TaskCommand struct {
	Key    string   `json:"key"`
	Label  string   `json:"label"`
	Prompt string   `json:"prompt,omitempty"`
	Run    []string `json:"run"`
}

func (c *TaskCommand) UnmarshalJSON(data []byte) error {
	type plain TaskCommand
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var p plain
	if err := dec.Decode(&p); err != nil {
		return err
	}
	*c = TaskCommand(p)
	return nil
}

func (c TaskCommand) validate() error {
	switch {
	case strings.TrimSpace(c.Key) == "":
		return errors.New(`"key" is required`)
	case strings.TrimSpace(c.Label) == "":
		return errors.New(`"label" is required`)
	case len(c.Run) == 0 || strings.TrimSpace(c.Run[0]) == "":
		return errors.New(`"run" needs a program, e.g. ["notify-send"]`)
	}
	return nil
}

func validateTaskCommands(cmds []TaskCommand) error {
	for i, c := range cmds {
		if err := c.validate(); err != nil {
			return fmt.Errorf("task_commands[%d]: %w", i, err)
		}
	}
	return nil
}

// DefaultConfig returns a Config with default values.
func DefaultConfig() Config {
	return Config{
		StatusDisplay:               StatusDisplayIcons,
		PriorityColor:               PriorityColorFull,
		CopyVia:                     CopyViaOSC52,
		ShowShortcutBar:             true,
		ExpandDescriptionsByDefault: false,
	}
}

// input > $env > $xdg/sub > ~/home...; an empty xdg skips that branch.
func resolveDir(input, env, xdg, sub string, home ...string) (string, error) {
	if input != "" {
		return input, nil
	}
	if v := os.Getenv(env); v != "" {
		return v, nil
	}
	if xdg != "" {
		if v := os.Getenv(xdg); v != "" {
			return filepath.Join(v, sub), nil
		}
	}
	h, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(append([]string{h}, home...)...), nil
}

func ResolveDataDir(input string) (string, error) {
	dir, err := resolveDir(input, EnvDataPath, "", "", ".local", "share", "phasionary")
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "projects"), nil
}

// Accepts either a directory path or a path pointing at a `*.json` file.
func configDirFromPath(p string) string {
	if strings.HasSuffix(p, ".json") {
		return filepath.Dir(p)
	}
	return p
}

func ResolveConfigDir(input string) (string, error) {
	dir, err := resolveDir(input, EnvConfigPath, "XDG_CONFIG_HOME", "phasionary", ".config", "phasionary")
	if err != nil {
		return "", err
	}
	return configDirFromPath(dir), nil
}

// ResolveConfigPath returns the full path to config.json.
func ResolveConfigPath(input string) (string, error) {
	dir, err := ResolveConfigDir(input)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}
