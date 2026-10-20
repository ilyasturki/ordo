package app

import (
	"errors"
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/ui"
)

// SaveState is what the open project's copy holds that never reached disk.
type SaveState struct {
	Unsaved int
	Err     error
}

type leaveKind int

const (
	leaveQuit leaveKind = iota
	leaveSwitch
	leaveCreate
)

type ConfirmLeaveState struct {
	Kind leaveKind
	// The picker's status for the project it opens once the switch is confirmed.
	status string
}

func (s SaveState) edits() string {
	return fmt.Sprintf("%d edit%s", s.Unsaved, plural(s.Unsaved, "", "s"))
}

// mark is the header's standing note until a reload or a saved edit clears it.
func (s SaveState) mark() string {
	switch {
	case s.Err == nil:
		return ""
	case errors.Is(s.Err, data.ErrStaleProject):
		return "[changed on disk: r drops " + s.edits() + "]"
	case errors.Is(s.Err, data.ErrProjectNotFound):
		return "[deleted on disk: " + s.edits() + " not saved]"
	}
	return "[not saved: " + s.edits() + "]"
}

func (m *model) checkSaves() {
	if m.deps.Store == nil {
		return
	}
	n, err := m.deps.Store.Unsaved(m.project.ID)
	m.ui.Save = SaveState{Unsaved: n, Err: err}
}

// holdsUnsaved waits out the queued writes, then reports whether any of the
// open project's edits failed to reach disk.
func (m *model) holdsUnsaved() bool {
	if m.deps.Saver != nil {
		m.deps.Saver.Drain()
	}
	m.checkSaves()
	return m.ui.Save.Err != nil
}

func (m *model) quit() tea.Cmd {
	if m.holdsUnsaved() {
		m.ui.ConfirmLeave = ConfirmLeaveState{Kind: leaveQuit}
		m.ui.Modes.ToConfirmLeave()
		return nil
	}
	return tea.Quit
}

func (m model) handleConfirmLeaveKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "y", "enter":
		if m.ui.ConfirmLeave.Kind != leaveQuit {
			m.deps.Store.DropUnsaved(m.project.ID)
		}
		switch m.ui.ConfirmLeave.Kind {
		case leaveSwitch:
			m.ui.Modes.Uncover()
			m.switchToPicked(m.ui.ConfirmLeave.status)
			return m, nil
		case leaveCreate:
			m.ui.Modes.Uncover()
			m.createPickedProject(strings.TrimSpace(m.ui.Picker.input.Value()))
			return m, nil
		}
		return m, tea.Quit
	case "n", "esc":
		m.ui.Modes.Uncover()
	}
	return m, nil
}

func (m model) confirmLeaveView() string {
	save := m.ui.Save
	question := fmt.Sprintf("Quit and drop %s?", save.edits())
	confirm := "quit anyway"
	if m.ui.ConfirmLeave.Kind != leaveQuit {
		question = fmt.Sprintf("Switch projects and drop %s?", save.edits())
		confirm = "switch anyway"
	}
	name := truncateText(m.project.Name, 30)
	lost := plural(save.Unsaved, "it was", "they were")
	var why string
	switch {
	case errors.Is(save.Err, data.ErrStaleProject):
		why = fmt.Sprintf("%q changed on disk since it was loaded, so %s not saved.", name, lost)
	case errors.Is(save.Err, data.ErrProjectNotFound):
		why = fmt.Sprintf("%q was deleted on disk, so %s not saved.", name, lost)
	default:
		why = "Saving failed: " + save.Err.Error()
	}
	lines := []string{
		question,
		why,
		"",
		ui.RenderHints([]ui.Hint{{Key: "y/enter", Label: confirm}, {Key: "n/esc", Label: "cancel"}}),
	}
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}
