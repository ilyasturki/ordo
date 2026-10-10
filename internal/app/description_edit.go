package app

import (
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

// startDescriptionInlineEdit opens the in-app textarea editor for the task's
// description at (catIdx, taskIdx). Use this instead of an external editor.
func (m *model) startDescriptionInlineEdit(catIdx, taskIdx int) tea.Cmd {
	if catIdx < 0 || catIdx >= len(m.project.Categories) {
		return nil
	}
	cat := &m.project.Categories[catIdx]
	if taskIdx < 0 || taskIdx >= len(cat.Tasks) {
		return nil
	}
	task := cat.Tasks[taskIdx]

	ta := textarea.New()
	// Add ctrl-based bindings alongside the widget's alt-based defaults so the
	// description editor feels like a conventional text editor: ctrl+arrow jumps
	// by word, ctrl+backspace/ctrl+delete removes a word. The existing alt/emacs
	// bindings (alt+arrows, ctrl+w, ctrl+a/e, ctrl+k/u, ...) keep working.
	km := ta.KeyMap
	km.WordForward.SetKeys("alt+right", "alt+f", "ctrl+right")
	km.WordBackward.SetKeys("alt+left", "alt+b", "ctrl+left")
	km.DeleteWordForward.SetKeys("alt+delete", "alt+d", "ctrl+delete")
	km.DeleteWordBackward.SetKeys("alt+backspace", "ctrl+w", "ctrl+backspace")
	// Both run atotto, whose clipboard tools can't be killed.
	km.Paste.SetEnabled(false)
	km.CopySelection.SetEnabled(false)
	ta.KeyMap = km
	// No current-line highlight: the block cursor alone marks the position.
	// Instead a ">" gutter (via the prompt func below) points at the cursor's row.
	styles := ta.Styles()
	styles.Focused.CursorLine = lipgloss.NewStyle()
	// Clear the cursor's fixed gray so its reverse block swaps the terminal's
	// real fg/bg — a high-contrast, theme-adaptive cursor matching the app's
	// other edit cursors (ui.SelectedStyle) instead of a washed-out gray box.
	styles.Cursor.Color = nil
	ta.SetStyles(styles)
	ta.SetValue(task.Description)
	ta.ShowLineNumbers = false
	cursorRow := new(int)
	// A 2-wide gutter on every line keeps text aligned; the cursor's row shows
	// "> ", the rest "  ". *cursorRow is refreshed each render (descriptionEditView).
	ta.SetPromptFunc(2, func(info textarea.PromptInfo) string {
		if info.LineNumber == *cursorRow {
			return "> "
		}
		return "  "
	})
	ta.CharLimit = 0
	// SetWidth must follow SetPromptFunc so it reserves the gutter width.
	ta.SetWidth(m.dialogWidth())
	ta.SetHeight(descriptionEditorVisibleHeight(m.ui.Screen.Height))
	cmd := ta.Focus()
	ta.CursorEnd()
	*cursorRow = descriptionCursorDisplayRow(ta)

	m.ui.DescriptionEdit = DescriptionEditState{
		textarea:      ta,
		categoryIndex: catIdx,
		taskIndex:     taskIdx,
		original:      task.Description,
		creating:      task.Description == "",
		cursorRow:     cursorRow,
	}
	if !m.ui.Modes.ToDescriptionEdit() {
		return nil
	}
	return cmd
}

// descriptionCursorDisplayRow returns the display row the textarea cursor sits
// on, accounting for soft-wrapping of the logical lines above it. The prompt
// func keys the ">" gutter off this value so the marker follows the cursor even
// when earlier lines wrap.
func descriptionCursorDisplayRow(ta textarea.Model) int {
	width := ta.Width()
	row := ta.Line()
	lines := strings.Split(ta.Value(), "\n")
	display := 0
	for i := 0; i < row && i < len(lines); i++ {
		display += wrappedRowCount(lines[i], width)
	}
	return display + ta.LineInfo().RowOffset
}

// wrappedRowCount reports how many display rows a single logical line occupies
// when soft-wrapped to width.
func wrappedRowCount(line string, width int) int {
	if width <= 0 || line == "" {
		return 1
	}
	return strings.Count(ansi.Wrap(line, width, ""), "\n") + 1
}

func descriptionEditorVisibleHeight(screenHeight int) int {
	// Leave room for project line, headers, and surrounding context.
	const chrome = 8
	h := screenHeight - chrome
	switch {
	case h < 3:
		return 3
	case h > 12:
		return 12
	default:
		return h
	}
}

func (m *model) handleDescriptionEditKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.cancelDescriptionEdit()
		return m, nil
	case "enter", "ctrl+s", "ctrl+enter":
		m.finishDescriptionEdit()
		return m, nil
	case "ctrl+v":
		return m, readClipboardForEditor(modes.ModeDescriptionEdit)
	case "shift+enter", "alt+enter":
		// Shift+Enter inserts a newline. Terminals disagree on its wire
		// encoding: kitty/ghostty send ESC+CR ("alt+enter"), Kitty-protocol
		// terminals a genuine "shift+enter". Accept both and forward a plain
		// Enter, which the textarea maps to its InsertNewline binding.
		msg = tea.KeyPressMsg{Code: tea.KeyEnter}
	}
	var cmd tea.Cmd
	m.ui.DescriptionEdit.textarea, cmd = m.ui.DescriptionEdit.textarea.Update(msg)
	return m, cmd
}

func (m *model) cancelDescriptionEdit() {
	wasCreating := m.ui.DescriptionEdit.creating
	taskID := m.descriptionEditTaskID()
	m.ui.DescriptionEdit = DescriptionEditState{}
	m.ui.Modes.ToNormal()
	m.ui.Screen.StatusMsg = ""
	// If we opened the editor for a task that had no description, the
	// FocusDescription row may not exist; restore selection to the parent task.
	if wasCreating && taskID != "" {
		m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			if p.Kind != selection.FocusTask || p.CategoryIndex < 0 || p.CategoryIndex >= len(m.project.Categories) {
				return false
			}
			c := m.project.Categories[p.CategoryIndex]
			return p.TaskIndex >= 0 && p.TaskIndex < len(c.Tasks) && c.Tasks[p.TaskIndex].ID == taskID
		})
	}
}

func (m *model) finishDescriptionEdit() {
	state := m.ui.DescriptionEdit
	if state.categoryIndex < 0 || state.categoryIndex >= len(m.project.Categories) {
		m.cancelDescriptionEdit()
		return
	}
	cat := &m.project.Categories[state.categoryIndex]
	if state.taskIndex < 0 || state.taskIndex >= len(cat.Tasks) {
		m.cancelDescriptionEdit()
		return
	}
	task := &cat.Tasks[state.taskIndex]
	taskID := task.ID

	newDesc := strings.Trim(strings.ReplaceAll(state.textarea.Value(), "\r\n", "\n"), "\n")
	if strings.TrimSpace(newDesc) == "" {
		newDesc = ""
	}

	if newDesc != state.original {
		m.recordHistory()
		task.Description = newDesc
		task.UpdatedAt = domain.NowTimestamp()
		m.storeTaskUpdate()
		switch {
		case state.original != "" && newDesc == "":
			m.ui.Screen.StatusMsg = "Description cleared"
		case state.original == "" && newDesc != "":
			m.ui.Screen.StatusMsg = "Description added"
		default:
			m.ui.Screen.StatusMsg = "Description updated"
		}
	} else {
		m.ui.Screen.StatusMsg = ""
	}

	m.ui.DescriptionEdit = DescriptionEditState{}
	m.ui.Modes.ToNormal()
	m.rebuildPositions()

	// Restore selection: prefer the description row if it still exists, else the task row.
	if newDesc != "" && m.ui.Screen.ExpandDescriptions {
		m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			if p.Kind != selection.FocusDescription || p.CategoryIndex < 0 || p.CategoryIndex >= len(m.project.Categories) {
				return false
			}
			c := m.project.Categories[p.CategoryIndex]
			return p.TaskIndex >= 0 && p.TaskIndex < len(c.Tasks) && c.Tasks[p.TaskIndex].ID == taskID
		})
	} else {
		m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			if p.Kind != selection.FocusTask || p.CategoryIndex < 0 || p.CategoryIndex >= len(m.project.Categories) {
				return false
			}
			c := m.project.Categories[p.CategoryIndex]
			return p.TaskIndex >= 0 && p.TaskIndex < len(c.Tasks) && c.Tasks[p.TaskIndex].ID == taskID
		})
	}
	m.ensureVisible()
}

func (m model) descriptionEditView() string {
	state := m.ui.DescriptionEdit
	// Refresh the cursor row so the prompt func's ">" gutter tracks the cursor.
	if state.cursorRow != nil {
		*state.cursorRow = descriptionCursorDisplayRow(state.textarea)
	}
	title := "Edit Description"
	if state.creating {
		title = "Add Description"
	}
	body := state.textarea.View()
	lines := []string{
		ui.DialogTitleStyle.Render(title),
		"",
		body,
		"",
		ui.RenderHints([]ui.Hint{
			{Key: "enter", Label: "save"},
			{Key: "shift+enter", Label: "newline"},
			{Key: "esc", Label: "cancel"},
		}),
	}
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

func (m *model) descriptionEditTaskID() string {
	s := m.ui.DescriptionEdit
	if s.categoryIndex < 0 || s.categoryIndex >= len(m.project.Categories) {
		return ""
	}
	cat := m.project.Categories[s.categoryIndex]
	if s.taskIndex < 0 || s.taskIndex >= len(cat.Tasks) {
		return ""
	}
	return cat.Tasks[s.taskIndex].ID
}
