package config

import (
	"cmp"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

// Reader exposes read-only access to the active config. Implemented by *Manager.
type Reader interface {
	Get() Config
	Update(fn func(*Config)) error
}

// Manager handles loading and saving configuration.
type Manager struct {
	path string
	cfg  Config
}

var _ Reader = (*Manager)(nil)

// NewManager creates a new config manager for the given config file path.
func NewManager(configPath string) *Manager {
	return &Manager{
		path: configPath,
		cfg:  DefaultConfig(),
	}
}

// configFileMode is the permission the config file is kept at. Owner-only:
// past versions stored a credential here and a future sync client may again,
// so the file never opens wider than it needs to be.
const configFileMode fs.FileMode = 0o600

// Load reads the config from disk. Creates a default config file if missing.
func (m *Manager) Load() error {
	data, err := os.ReadFile(m.path)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return m.Save()
		}
		return fmt.Errorf("reading config %s: %w", m.path, err)
	}
	if err := json.Unmarshal(data, &m.cfg); err != nil {
		return fmt.Errorf("parsing config %s: %w", m.path, err)
	}
	if err := validateTaskCommands(m.cfg.TaskCommands); err != nil {
		return fmt.Errorf("config %s: %w", m.path, err)
	}
	d := DefaultConfig()
	m.cfg.StatusDisplay = cmp.Or(m.cfg.StatusDisplay, d.StatusDisplay)
	m.cfg.PriorityColor = cmp.Or(m.cfg.PriorityColor, d.PriorityColor)
	m.tightenPermissions()
	return nil
}

// tightenPermissions narrows an existing config file to configFileMode.
//
// Files created by early versions were written 0644, and Save alone would not
// fix them: it writes through to the existing inode, which keeps the old mode.
// Failure is ignored — a config that cannot be chmodded (unusual ownership,
// odd filesystem) should not stop the tool from running.
func (m *Manager) tightenPermissions() {
	info, err := os.Stat(m.path)
	if err != nil || info.Mode().Perm() == configFileMode {
		return
	}
	_ = os.Chmod(m.path, configFileMode)
}

// Save writes the current config to disk. Creates the directory if needed.
func (m *Manager) Save() error {
	dir := filepath.Dir(m.path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("creating config directory %s: %w", dir, err)
	}
	data, err := json.MarshalIndent(m.cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}
	if err := os.WriteFile(m.path, data, configFileMode); err != nil {
		return fmt.Errorf("writing config %s: %w", m.path, err)
	}
	// WriteFile only applies the mode when it creates the file, so an existing
	// 0644 config would keep its old permissions after a write. Chmod
	// unconditionally so a save always lands on an owner-only file.
	if err := os.Chmod(m.path, configFileMode); err != nil {
		return fmt.Errorf("securing config %s: %w", m.path, err)
	}
	return nil
}

// Get returns the current config.
func (m *Manager) Get() Config {
	return m.cfg
}

// Update applies a function to modify the config and saves it.
func (m *Manager) Update(fn func(*Config)) error {
	fn(&m.cfg)
	return m.Save()
}

// Path returns the path to the config file.
func (m *Manager) Path() string {
	return m.path
}
