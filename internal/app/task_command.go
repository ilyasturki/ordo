package app

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"unicode"

	"charm.land/bubbles/v2/textinput"
	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"

	"phasionary/internal/app/selection"
	"phasionary/internal/config"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

const sectionCommands = "Task commands"

type taskCommand struct {
	prefix rune
	key    string
	config.TaskCommand
}

type TaskCommandState struct {
	input   textinput.Model
	command taskCommand
	taskID  string
}

type taskCommandResultMsg struct {
	label  string
	stdout string
	stderr string
	err    error
}

func bindTaskCommands(cmds []config.TaskCommand) ([]taskCommand, error) {
	out := make([]taskCommand, 0, len(cmds))
	for i, c := range cmds {
		bound, err := bindTaskCommand(c, out)
		if err != nil {
			return nil, fmt.Errorf("task_commands[%d] (%s): %w", i, c.Label, err)
		}
		out = append(out, bound)
	}
	return out, nil
}

func bindTaskCommand(c config.TaskCommand, taken []taskCommand) (taskCommand, error) {
	runes := []rune(c.Key)
	for _, r := range runes {
		if unicode.IsSpace(r) || unicode.IsControl(r) {
			return taskCommand{}, fmt.Errorf("unknown key %q: use one character, or g or z followed by one", c.Key)
		}
	}
	bound := taskCommand{TaskCommand: c}
	switch len(runes) {
	case 1:
		bound.key = c.Key
		if _, isChord := chordPrefixes[runes[0]]; isChord {
			return taskCommand{}, fmt.Errorf("key %q starts built-in shortcuts such as %s%s", c.Key, c.Key, c.Key)
		}
	case 2:
		if _, isChord := chordPrefixes[runes[0]]; !isChord {
			return taskCommand{}, fmt.Errorf("unknown key %q: a two-key shortcut must start with g or z", c.Key)
		}
		bound.prefix, bound.key = runes[0], string(runes[1])
	default:
		return taskCommand{}, fmt.Errorf("unknown key %q: use one character, or g or z followed by one", c.Key)
	}

	if b := matchBinding(bound.prefix, bound.key); b != nil {
		return taskCommand{}, fmt.Errorf("key %q is already bound to %q", c.Key, builtinDesc(b))
	}
	for _, other := range taken {
		if other.prefix == bound.prefix && other.key == bound.key {
			return taskCommand{}, fmt.Errorf("key %q is already bound to the command %q", c.Key, other.Label)
		}
	}
	return bound, nil
}

// A partner binding such as `k` in "j / k" has no desc; it borrows the one above it.
func builtinDesc(b *keyBinding) string {
	for i := range normalBindings {
		if &normalBindings[i] != b {
			continue
		}
		for j := i; j >= 0; j-- {
			if normalBindings[j].desc != "" {
				return normalBindings[j].desc
			}
		}
	}
	return b.desc
}

func (m model) taskCommands() []taskCommand {
	return m.deps.TaskCommands
}

func (m model) matchTaskCommand(prefix rune, key string) *taskCommand {
	for i, c := range m.taskCommands() {
		if c.prefix == prefix && c.key == key {
			return &m.deps.TaskCommands[i]
		}
	}
	return nil
}

// A description row counts as its task.
func (m model) focusedTask() (int, domain.Task, bool) {
	pos, ok := m.selectedPosition()
	if !ok || (pos.Kind != selection.FocusTask && pos.Kind != selection.FocusDescription) {
		return -1, domain.Task{}, false
	}
	return pos.CategoryIndex, m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex], true
}

func (m *model) startTaskCommand(c taskCommand) tea.Cmd {
	ci, task, ok := m.focusedTask()
	if !ok {
		m.ui.Screen.StatusMsg = c.Label + ": focus a task first"
		return nil
	}
	if c.Prompt == "" {
		m.ui.Screen.StatusMsg = c.Label + "…"
		return runTaskCommand(c.TaskCommand, taskCommandEnv(m.project, ci, task, ""))
	}
	ti := newTextInput()
	ti.Prompt = ""
	cmd := ti.Focus()
	m.ui.TaskCommand = TaskCommandState{input: ti, command: c, taskID: task.ID}
	if !m.ui.Modes.ToCommandPrompt() {
		return nil
	}
	m.ensureVisible()
	return cmd
}

func (m model) handleCommandPromptKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	s := &m.ui.TaskCommand
	switch msg.String() {
	case "esc":
		m.ui.Modes.ToNormal()
		return m, nil
	case "enter":
		m.ui.Modes.ToNormal()
		input := strings.TrimSpace(s.input.Value())
		if input == "" {
			return m, nil
		}
		ci, ti := m.findTaskByID(s.taskID)
		if ci < 0 {
			m.ui.Screen.StatusMsg = s.command.Label + ": the task is gone"
			return m, nil
		}
		m.ui.Screen.StatusMsg = s.command.Label + "…"
		return m, runTaskCommand(s.command.TaskCommand, taskCommandEnv(m.project, ci, m.project.Categories[ci].Tasks[ti], input))
	}
	var cmd tea.Cmd
	s.input, cmd = s.input.Update(msg)
	sanitizeInput(&s.input)
	return m, cmd
}

func (m model) renderCommandPrompt() string {
	s := m.ui.TaskCommand
	return m.promptRow(s.command.Prompt+": ", s.input, ui.MutedStyle.Render(s.command.Label), m.ui.Screen.Width)
}

// A sub-category's name repeats across parents, so the category goes by its path, which -C resolves.
func taskCommandEnv(p domain.Project, ci int, t domain.Task, input string) []string {
	return append(os.Environ(),
		"PHASIONARY_PROJECT_ID="+p.ID,
		"PHASIONARY_PROJECT_NAME="+p.Name,
		"PHASIONARY_TASK_ID="+t.ID,
		"PHASIONARY_TASK_TITLE="+t.Title,
		"PHASIONARY_TASK_DESCRIPTION="+t.Description,
		"PHASIONARY_CATEGORY="+p.CategoryPath(ci),
		"PHASIONARY_CATEGORY_ID="+p.Categories[ci].ID,
		"PHASIONARY_INPUT="+input,
	)
}

// Stdin stays empty: the terminal belongs to the TUI.
func runTaskCommand(c config.TaskCommand, env []string) tea.Cmd {
	return func() tea.Msg {
		cmd := exec.Command(c.Run[0], c.Run[1:]...)
		cmd.Env = env
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		err := cmd.Run()
		return taskCommandResultMsg{label: c.Label, stdout: stdout.String(), stderr: stderr.String(), err: err}
	}
}

func (m *model) handleTaskCommandResult(msg taskCommandResultMsg) {
	if msg.err != nil {
		detail := firstOutputLine(msg.stderr)
		if detail == "" {
			detail = firstOutputLine(msg.stdout)
		}
		if detail == "" {
			detail = msg.err.Error()
		}
		m.ui.Screen.StatusMsg = msg.label + " failed: " + detail
		return
	}
	if line := firstOutputLine(msg.stdout); line != "" {
		m.ui.Screen.StatusMsg = line
		return
	}
	m.ui.Screen.StatusMsg = msg.label + ": done"
}

// Escape sequences are stripped so command output cannot drive the terminal.
func firstOutputLine(out string) string {
	for line := range strings.SplitSeq(out, "\n") {
		line = domain.StripControl(ansi.Strip(line), false)
		if line = strings.TrimSpace(line); line != "" {
			return line
		}
	}
	return ""
}
