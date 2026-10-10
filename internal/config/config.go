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
	EnvStatePath  = "PHASIONARY_STATE_PATH"

	EnvServerDataPath = "PHASIONARY_SERVER_DATA_PATH"

	StatusDisplayText  = "text"
	StatusDisplayIcons = "icons"

	PriorityColorFull = "full"
	PriorityColorIcon = "icon"
	PriorityColorNone = "none"
)

// Config holds user preferences.
type Config struct {
	StatusDisplay               string `json:"status_display,omitempty"`
	PriorityColor               string `json:"priority_color,omitempty"`
	ShowShortcutBar             bool   `json:"show_shortcut_bar"`
	ExpandDescriptionsByDefault bool   `json:"expand_descriptions_by_default"`

	// HelpExpanded remembers whether the help dialog opens on the full
	// shortcut reference rather than the essentials card.
	HelpExpanded bool `json:"help_expanded"`

	TaskCommands []TaskCommand `json:"task_commands,omitempty"`
}

// TaskCommand runs an external program on the focused task. The task reaches
// it through PHASIONARY_* environment variables, and Run is executed directly,
// never through a shell.
type TaskCommand struct {
	Key   string `json:"key"`
	Label string `json:"label"`
	// Prompt, when set, asks for one line of input first; the answer is passed
	// as PHASIONARY_INPUT.
	Prompt string   `json:"prompt,omitempty"`
	Run    []string `json:"run"`
}

// UnmarshalJSON rejects unknown fields, so a misspelt "run" fails loudly
// instead of leaving a command that does nothing.
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
		ShowShortcutBar:             true,
		ExpandDescriptionsByDefault: false,
		HelpExpanded:                false,
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

// The per-device state directory must never be carried between machines by a
// file syncer: it holds the device identity.
func ResolveStateDir() (string, error) {
	return resolveDir("", EnvStatePath, "XDG_STATE_HOME", "phasionary", ".local", "state", "phasionary")
}

func ResolveServerDataDir(input string) (string, error) {
	return resolveDir(input, EnvServerDataPath, "XDG_DATA_HOME", "phasionary-server", ".local", "share", "phasionary-server")
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
