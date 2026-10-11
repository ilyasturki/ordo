package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/config"
)

func newCommandModel(t *testing.T, cmds ...config.TaskCommand) *model {
	t.Helper()
	bound, err := bindTaskCommands(cmds)
	require.NoError(t, err)
	m := newTestModel(t, sampleProject())
	m.deps.setTaskCommands(bound)
	m.ui.Selection.SetSelected(findFirstTaskIndex(m.positions()))
	return m
}

// runCmd executes a command synchronously and feeds its message back, the way
// the Bubble Tea runtime would.
func runCmd(t *testing.T, m *model, cmd tea.Cmd) *model {
	t.Helper()
	require.NotNil(t, cmd, "expected a command to run")
	next, _ := m.Update(cmd())
	updated := next.(model)
	return &updated
}

func pressKeys(m *model, keys ...string) (*model, tea.Cmd) {
	var cmd tea.Cmd
	for _, k := range keys {
		var next tea.Model
		next, cmd = m.Update(keyPress(k))
		updated := next.(model)
		m = &updated
	}
	return m, cmd
}

func keyPress(k string) tea.KeyPressMsg {
	switch k {
	case "enter":
		return tea.KeyPressMsg{Code: tea.KeyEnter}
	case "esc":
		return tea.KeyPressMsg{Code: tea.KeyEscape}
	}
	r := []rune(k)[0]
	return tea.KeyPressMsg{Code: r, Text: k}
}

func shCommand(key, prompt, script string) config.TaskCommand {
	return config.TaskCommand{Key: key, Label: "probe", Prompt: prompt, Run: []string{"sh", "-c", script}}
}

func TestBindTaskCommands_RejectsClashesAndUnknownKeys(t *testing.T) {
	cases := map[string]struct {
		cmds []config.TaskCommand
		want string
	}{
		"built-in single key":  {[]config.TaskCommand{shCommand("d", "", "true")}, `key "d" is already bound to "delete"`},
		"move key":             {[]config.TaskCommand{shCommand("m", "", "true")}, `key "m" is already bound to "move to a category"`},
		"partner key":          {[]config.TaskCommand{shCommand("k", "", "true")}, `key "k" is already bound to "move up / down"`},
		"built-in chord":       {[]config.TaskCommand{shCommand("gx", "", "true")}, `key "gx" is already bound to "open a link or image"`},
		"chord prefix alone":   {[]config.TaskCommand{shCommand("g", "", "true")}, `key "g" starts built-in shortcuts`},
		"foreign chord prefix": {[]config.TaskCommand{shCommand("cr", "", "true")}, `unknown key "cr"`},
		"too long":             {[]config.TaskCommand{shCommand("ctrl+o", "", "true")}, `unknown key "ctrl+o"`},
		"space":                {[]config.TaskCommand{shCommand(" ", "", "true")}, `unknown key " "`},
		"two commands":         {[]config.TaskCommand{shCommand("gr", "", "true"), shCommand("gr", "", "true")}, `task_commands[1] (probe): key "gr" is already bound to the command "probe"`},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := bindTaskCommands(tc.cmds)
			assert.ErrorContains(t, err, tc.want)
		})
	}
}

func TestBindTaskCommands_AcceptsFreeKeys(t *testing.T) {
	bound, err := bindTaskCommands([]config.TaskCommand{shCommand("gr", "", "true"), shCommand("w", "", "true")})
	require.NoError(t, err)
	assert.Equal(t, 'g', bound[0].prefix)
	assert.Equal(t, "r", bound[0].key)
	assert.Equal(t, rune(0), bound[1].prefix)
	assert.Equal(t, "w", bound[1].key)
}

func TestTaskCommand_PromptRunsWithTaskEnvironment(t *testing.T) {
	m := newCommandModel(t, shCommand("gr", "when",
		`printf '%s|%s|%s|%s|%s\n' "$ORDO_INPUT" "$ORDO_TASK_ID" "$ORDO_TASK_TITLE" "$ORDO_CATEGORY" "$ORDO_PROJECT_NAME"; echo second line`))

	m, cmd := pressKeys(m, "g", "r")
	require.True(t, m.ui.Modes.IsCommandPrompt())
	assert.Contains(t, m.renderBottomBar(), "when: ")

	m, cmd = pressKeys(m, "n", "e", "x", "t", " ", "f", "r", "i", "enter")
	require.True(t, m.ui.Modes.IsNormal())
	m = runCmd(t, m, cmd)
	assert.Equal(t, "next fri|t1|Alpha|Cat A|P", m.ui.Screen.StatusMsg)
}

func TestTaskCommand_EmptyInputCancels(t *testing.T) {
	m := newCommandModel(t, shCommand("gr", "when", "echo ran"))
	m, _ = pressKeys(m, "g", "r")
	m, cmd := pressKeys(m, "enter")
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Nil(t, cmd)
}

func TestTaskCommand_EscCancels(t *testing.T) {
	m := newCommandModel(t, shCommand("gr", "when", "echo ran"))
	m, _ = pressKeys(m, "g", "r", "x")
	m, cmd := pressKeys(m, "esc")
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Nil(t, cmd)
}

func TestTaskCommand_NoPromptRunsAtOnce(t *testing.T) {
	m := newCommandModel(t, shCommand("w", "", `printf '\033[1mqueued\033[0m %s\n' "$ORDO_TASK_TITLE"`))
	m, cmd := pressKeys(m, "w")
	m = runCmd(t, m, cmd)
	assert.Equal(t, "queued Alpha", m.ui.Screen.StatusMsg)
}

func TestTaskCommand_FailureShowsStderr(t *testing.T) {
	m := newCommandModel(t, shCommand("w", "", "echo out; echo 'already exists' >&2; exit 1"))
	m, cmd := pressKeys(m, "w")
	m = runCmd(t, m, cmd)
	assert.Equal(t, "probe failed: already exists", m.ui.Screen.StatusMsg)
}

func TestTaskCommand_SilentFailureShowsExitStatus(t *testing.T) {
	m := newCommandModel(t, shCommand("w", "", "exit 3"))
	m, cmd := pressKeys(m, "w")
	m = runCmd(t, m, cmd)
	assert.Equal(t, "probe failed: exit status 3", m.ui.Screen.StatusMsg)
}

func TestTaskCommand_MissingProgram(t *testing.T) {
	m := newCommandModel(t, config.TaskCommand{Key: "w", Label: "probe", Run: []string{"ordo-no-such-program"}})
	m, cmd := pressKeys(m, "w")
	m = runCmd(t, m, cmd)
	assert.True(t, strings.HasPrefix(m.ui.Screen.StatusMsg, "probe failed: exec: "), m.ui.Screen.StatusMsg)
}

func TestTaskCommand_SilentSuccess(t *testing.T) {
	m := newCommandModel(t, shCommand("w", "", "true"))
	m, cmd := pressKeys(m, "w")
	assert.Equal(t, "probe…", m.ui.Screen.StatusMsg, "a running command says so until it finishes")
	m = runCmd(t, m, cmd)
	assert.Equal(t, "probe: done", m.ui.Screen.StatusMsg)
}

func TestTaskCommand_NeedsATask(t *testing.T) {
	m := newCommandModel(t, shCommand("w", "", "echo ran"))
	m.ui.Selection.SetSelected(0) // the project row
	m, cmd := pressKeys(m, "w")
	assert.Nil(t, cmd)
	assert.Equal(t, "probe: focus a task first", m.ui.Screen.StatusMsg)
	assert.Equal(t, modes.ModeNormal, m.ui.Modes.Current())
}

func TestTaskCommand_ListedInHelpAndBarOnlyWhenConfigured(t *testing.T) {
	plain := newTestModel(t, sampleProject())
	plain.deps.CfgManager.(*stubConfigReader).cfg.ShowShortcutBar = true
	assert.Equal(t, builtinReferenceLines, plain.helpReferenceLines())
	assert.NotContains(t, plain.renderShortcutBar(), "probe")

	m := newCommandModel(t, shCommand("gr", "when", "true"))
	m.deps.CfgManager.(*stubConfigReader).cfg.ShowShortcutBar = true
	m.ui.Screen.Width = 200
	assert.Contains(t, m.renderShortcutBar(), "probe")
	assert.Equal(t, plain.helpShortcutCount()+1, m.helpShortcutCount())

	m.ui.Modes.ToHelp()
	m.ui.Help = HelpState{}
	m.ui.Screen.HelpExpanded = true
	body := m.helpBody()
	for i, target := range body.targets {
		if e := body.entryAt(target); e.command != nil {
			m.ui.Help.Focused = i
		}
	}
	next, _ := m.runFocusedHelpBinding()
	got := next.(model)
	assert.True(t, got.ui.Modes.IsCommandPrompt(), "enter on the help row should open the command's prompt")
}

func TestTaskCommandEnvNamesASubCategoryByPath(t *testing.T) {
	p := nestedProject()
	env := taskCommandEnv(p, 1, p.Categories[1].Tasks[0], "")
	assert.Contains(t, env, "ORDO_CATEGORY=v0.2.0 / ui")
	assert.Contains(t, env, "ORDO_CATEGORY_ID=ui")
}
