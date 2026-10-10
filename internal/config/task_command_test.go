package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func loadConfigFile(t *testing.T, body string) (*Manager, error) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.json")
	require.NoError(t, os.WriteFile(path, []byte(body), 0o600))
	m := NewManager(path)
	return m, m.Load()
}

func TestTaskCommands_LoadAndSurviveSave(t *testing.T) {
	m, err := loadConfigFile(t, `{"task_commands": [
		{"key": "gr", "label": "notify", "prompt": "when", "run": ["notify-send", "--app-name=x"]}
	]}`)
	require.NoError(t, err)
	want := []TaskCommand{{Key: "gr", Label: "notify", Prompt: "when", Run: []string{"notify-send", "--app-name=x"}}}
	assert.Equal(t, want, m.Get().TaskCommands)

	require.NoError(t, m.Update(func(c *Config) { c.HelpExpanded = true }))
	again := NewManager(m.Path())
	require.NoError(t, again.Load())
	assert.Equal(t, want, again.Get().TaskCommands)
}

func TestTaskCommands_DefaultHasNone(t *testing.T) {
	m := NewManager(filepath.Join(t.TempDir(), "config.json"))
	require.NoError(t, m.Load())
	assert.Empty(t, m.Get().TaskCommands)

	data, err := os.ReadFile(m.Path())
	require.NoError(t, err)
	assert.NotContains(t, string(data), "task_commands")
}

func TestTaskCommands_RejectInvalid(t *testing.T) {
	cases := map[string]struct{ body, want string }{
		"unknown field": {`{"task_commands": [{"key": "m", "label": "x", "args": ["echo"]}]}`, `unknown field "args"`},
		"missing key":   {`{"task_commands": [{"label": "x", "run": ["echo"]}]}`, `task_commands[0]: "key" is required`},
		"missing label": {`{"task_commands": [{"key": "m", "run": ["echo"]}]}`, `task_commands[0]: "label" is required`},
		"empty run":     {`{"task_commands": [{"key": "m", "label": "x", "run": []}]}`, `task_commands[0]: "run" needs a program`},
		"blank program": {`{"task_commands": [{"key": "m", "label": "x", "run": [" "]}]}`, `task_commands[0]: "run" needs a program`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := loadConfigFile(t, tc.body)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}
