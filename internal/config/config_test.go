package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveConfigDir(t *testing.T) {
	t.Run("input takes priority", func(t *testing.T) {
		dir, err := ResolveConfigDir("/custom/path")
		require.NoError(t, err)
		assert.Equal(t, "/custom/path", dir)
	})

	t.Run("env var takes second priority", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "/env/path")
		dir, err := ResolveConfigDir("")
		require.NoError(t, err)
		assert.Equal(t, "/env/path", dir)
	})

	t.Run("XDG_CONFIG_HOME takes third priority", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "")
		t.Setenv("XDG_CONFIG_HOME", "/xdg/config")
		dir, err := ResolveConfigDir("")
		require.NoError(t, err)
		assert.Equal(t, "/xdg/config/phasionary", dir)
	})

	t.Run("falls back to ~/.config/phasionary", func(t *testing.T) {
		t.Setenv(EnvConfigPath, "")
		t.Setenv("XDG_CONFIG_HOME", "")
		home, err := os.UserHomeDir()
		require.NoError(t, err)

		dir, err := ResolveConfigDir("")
		require.NoError(t, err)
		assert.Equal(t, filepath.Join(home, ".config", "phasionary"), dir)
	})
}

func TestResolveConfigPath(t *testing.T) {
	t.Run("returns config.json in resolved directory", func(t *testing.T) {
		path, err := ResolveConfigPath("/custom/dir")
		require.NoError(t, err)
		assert.Equal(t, "/custom/dir/config.json", path)
	})
}

func TestManager(t *testing.T) {
	t.Run("creates config file on first load", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "subdir", "config.json")

		m := NewManager(configPath)
		err := m.Load()
		require.NoError(t, err)

		// File should exist now
		_, err = os.Stat(configPath)
		require.NoError(t, err)

		// Should contain default config with status_display
		data, err := os.ReadFile(configPath)
		require.NoError(t, err)
		assert.JSONEq(t, `{"status_display":"icons","priority_color":"full","show_shortcut_bar":true,"expand_descriptions_by_default":false}`, string(data))
	})

	t.Run("a hand-edited empty field loads as the default", func(t *testing.T) {
		configPath := filepath.Join(t.TempDir(), "config.json")
		require.NoError(t, os.WriteFile(configPath, []byte(`{"status_display":"","priority_color":""}`), 0o600))

		m := NewManager(configPath)
		require.NoError(t, m.Load())
		assert.Equal(t, StatusDisplayIcons, m.Get().StatusDisplay)
		assert.Equal(t, PriorityColorFull, m.Get().PriorityColor)
	})

	t.Run("loads existing config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		// Write a config file
		err := os.WriteFile(configPath, []byte("{}"), 0o644)
		require.NoError(t, err)

		m := NewManager(configPath)
		err = m.Load()
		require.NoError(t, err)

		cfg := m.Get()
		assert.Equal(t, DefaultConfig(), cfg)
	})

	t.Run("returns error for invalid JSON", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		err := os.WriteFile(configPath, []byte("not json"), 0o644)
		require.NoError(t, err)

		m := NewManager(configPath)
		err = m.Load()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "parsing config")
	})

	t.Run("save creates directory", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "nested", "dir", "config.json")

		m := NewManager(configPath)
		err := m.Save()
		require.NoError(t, err)

		_, err = os.Stat(configPath)
		require.NoError(t, err)
	})

	t.Run("update modifies and saves config", func(t *testing.T) {
		tmpDir := t.TempDir()
		configPath := filepath.Join(tmpDir, "config.json")

		m := NewManager(configPath)
		err := m.Update(func(cfg *Config) {
			// No fields to modify yet, but the save should still happen
		})
		require.NoError(t, err)

		// File should exist
		_, err = os.Stat(configPath)
		require.NoError(t, err)
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	assert.Equal(t, Config{StatusDisplay: StatusDisplayIcons, PriorityColor: PriorityColorFull, ShowShortcutBar: true}, cfg)
}
