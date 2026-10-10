package app

import (
	"errors"
	"fmt"
	"strings"
	"unicode/utf8"

	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/data"
	"phasionary/internal/domain"
)

// editOrFocusDescription is the shift+enter handler. From a task with a
// description, it jumps the selection to the description row. From a task
// without one, it opens the external editor to create the description. From
// the description row itself it just opens the editor.
func (m *model) editOrFocusDescription() tea.Cmd {
	pos, ok := m.selectedPosition()
	if !ok {
		return nil
	}
	switch pos.Kind {
	case selection.FocusDescription:
		return m.startDescriptionInlineEdit(pos.CategoryIndex, pos.TaskIndex)
	case selection.FocusTask:
		task := m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
		if task.Description == "" {
			return m.startDescriptionInlineEdit(pos.CategoryIndex, pos.TaskIndex)
		}
		if !m.ui.Screen.ExpandDescriptions {
			m.ui.Screen.ExpandDescriptions = true
			m.rebuildPositions()
		}
		taskID := task.ID
		m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			if p.Kind != selection.FocusDescription || p.CategoryIndex != pos.CategoryIndex {
				return false
			}
			cat := m.project.Categories[p.CategoryIndex]
			return p.TaskIndex >= 0 && p.TaskIndex < len(cat.Tasks) && cat.Tasks[p.TaskIndex].ID == taskID
		})
		m.ensureVisible()
		return nil
	}
	return nil
}

func (m *model) startEditing() {
	position, ok := m.selectedPosition()
	if !ok {
		return
	}
	defer m.ensureVisible()
	switch position.Kind {
	case selection.FocusProject:
		m.ui.Modes.ToEdit()
		m.ui.Edit = newEditState(m.project.Name, false, "", selection.FocusProject)
	case selection.FocusTask:
		task := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
		m.ui.Modes.ToEdit()
		m.ui.Edit = newEditState(task.Title, false, "", selection.FocusTask)
	case selection.FocusCategory:
		category := m.project.Categories[position.CategoryIndex]
		m.ui.Modes.ToEdit()
		m.ui.Edit = newEditState(category.Name, false, "", selection.FocusCategory)
	case selection.FocusSeparator:
		sep := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
		m.ui.Modes.ToEdit()
		m.ui.Edit = newEditState(sep.Title, false, "", selection.FocusSeparator)
	}
}

func (m *model) startAddingTask() {
	position, ok := m.selectedPosition()
	if !ok {
		return
	}
	catIndex := position.CategoryIndex
	if catIndex < 0 || catIndex >= len(m.project.Categories) {
		return
	}
	newTask, err := domain.NewTask("")
	if err != nil {
		return
	}

	// Insert directly below the current row when it shares the category's
	// Tasks addressing (a task, its description line, or a separator); otherwise
	// (e.g. focus on the category header) land at the top of the category.
	var taskIndex int
	switch position.Kind {
	case selection.FocusTask, selection.FocusDescription, selection.FocusSeparator:
		taskIndex = position.TaskIndex + 1
	default:
		taskIndex = 0
	}
	m.recordHistory()
	m.project.Categories[catIndex].InsertTask(taskIndex, newTask)

	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == catIndex && p.TaskIndex == taskIndex
	})
	m.ui.Modes.ToEdit()
	m.ui.Edit = newEditState("", true, newTask.ID, selection.FocusTask)
	m.ensureVisible()
}

// startAddingSeparator inserts a bare (unlabeled) separator directly below the
// selection and lands on it. Unlike a task it opens no editor — Enter later
// edits its label. Blocked while a filter is active, since the new separator
// would be hidden (separators never survive a filter) and thus unreachable.
func (m *model) startAddingSeparator() {
	position, ok := m.selectedPosition()
	if !ok {
		return
	}
	catIndex := position.CategoryIndex
	if catIndex < 0 || catIndex >= len(m.project.Categories) {
		return
	}
	if m.ui.Filter.HasActiveFilter() {
		m.ui.Screen.StatusMsg = "Clear the filter to add a separator"
		return
	}
	sep, err := domain.NewSeparator()
	if err != nil {
		return
	}

	var taskIndex int
	switch position.Kind {
	case selection.FocusTask, selection.FocusSeparator, selection.FocusDescription:
		taskIndex = position.TaskIndex + 1
	default:
		taskIndex = 0
	}
	m.recordHistory()
	m.project.Categories[catIndex].InsertTask(taskIndex, sep)
	m.storeTaskUpdate()

	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusSeparator && p.CategoryIndex == catIndex && p.TaskIndex == taskIndex
	})
	m.ensureVisible()
}

func (m *model) startAddingCategory() {
	position, ok := m.selectedPosition()
	if !ok {
		return
	}
	insertIndex := position.CategoryIndex + 1
	newCat, err := domain.NewCategory("")
	if err != nil {
		return
	}
	m.recordHistory()
	m.project.InsertCategory(insertIndex, newCat)
	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusCategory && p.CategoryIndex == insertIndex
	})
	m.ui.Modes.ToEdit()
	m.ui.Edit = newEditState("", true, newCat.ID, selection.FocusCategory)
	m.ensureVisible()
}

func (m *model) removeNewCategory() {
	if m.ui.Edit.newItemID == "" {
		return
	}
	catIndex := -1
	for i, cat := range m.project.Categories {
		if cat.ID == m.ui.Edit.newItemID {
			catIndex = i
			break
		}
	}
	if catIndex < 0 {
		return
	}
	_ = m.project.RemoveCategory(catIndex)
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) handleEditKey(msg tea.KeyPressMsg) tea.Cmd {
	defer m.ensureVisible()
	if m.ui.Edit.mergeInto != "" {
		switch msg.String() {
		case "y", "enter":
			m.mergeEditedCategory()
		case "n", "esc":
			m.ui.Edit.mergeInto = ""
		}
		return nil
	}
	switch msg.String() {
	case "enter":
		m.finishEditing()
		return nil
	case "esc":
		m.cancelEditing()
		return nil
	case "up", "down":
		// Not forwarded: bubbles' textinput binds up/down to suggestion history.
		m.moveEditCursorRow(msg.String() == "down")
		return nil
	case "ctrl+v":
		// bubbles' own paste reads through atotto alone, missing the Wayland fallback.
		return readClipboardForEditor(modes.ModeEdit)
	}
	before := m.ui.Edit.input.Value()
	var cmd tea.Cmd
	m.ui.Edit.input, cmd = m.ui.Edit.input.Update(msg)
	sanitizeInput(&m.ui.Edit.input)
	if m.ui.Edit.input.Value() != before {
		m.ui.Edit.err = ""
	}
	return cmd
}

// moveEditCursorRow walks the caret one wrapped row, landing on the start or
// end of the buffer past either end.
func (m *model) moveEditCursorRow(down bool) {
	edit, ok := m.openEditRows()
	if !ok {
		return
	}
	target := edit.cursorRow - 1
	if down {
		target = edit.cursorRow + 1
	}
	if target < 0 {
		m.ui.Edit.input.SetCursor(0)
		return
	}
	value := m.ui.Edit.input.Value()
	if target >= len(edit.rows) {
		m.ui.Edit.input.SetCursor(utf8.RuneCountInString(value))
		return
	}
	row := edit.rows[target]
	runes := []rune(row.Text)
	offset := row.Start + len(string(runes[:min(edit.cursorCol, len(runes))]))
	m.ui.Edit.input.SetCursor(utf8.RuneCountInString(value[:min(offset, len(value))]))
}

func (m *model) finishEditing() {
	if !m.ui.Modes.IsEdit() {
		return
	}
	position, ok := m.selectedPosition()
	if !ok {
		m.cancelEditing()
		return
	}
	trimmed := strings.TrimSpace(m.ui.Edit.input.Value())
	// A separator label may legitimately be cleared back to bare, so it bypasses
	// the empty-title guard that cancels task/category edits.
	if position.Kind == selection.FocusSeparator {
		m.finishSeparatorEditing(position, trimmed)
		m.ui.Modes.ToNormal()
		m.ui.Edit.reset()
		return
	}
	if trimmed == "" {
		m.cancelEditing()
		return
	}
	switch position.Kind {
	case selection.FocusProject:
		if m.project.Name != trimmed {
			if err := m.renameProject(trimmed); errors.Is(err, data.ErrDuplicateProjectName) {
				m.ui.Edit.err = fmt.Sprintf("A project named %q already exists", trimmed)
				return
			} else if err != nil {
				m.ui.Screen.StatusMsg = fmt.Sprintf("Failed to save: %v", err)
			}
		}
	case selection.FocusTask:
		task := &m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
		if task.Title != trimmed || m.ui.Edit.isAdding {
			if !m.ui.Edit.isAdding {
				m.recordHistory()
			}
			task.Title = trimmed
			task.UpdatedAt = domain.NowTimestamp()
			m.storeTaskUpdate()
		}
	case selection.FocusCategory:
		if !m.finishCategoryEditing(position, trimmed) {
			return
		}
	default:
		m.cancelEditing()
		return
	}
	m.ui.Modes.ToNormal()
	m.ui.Edit.reset()
}

func (m *model) finishSeparatorEditing(position selection.Position, label string) {
	sep := &m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
	if sep.Title == label {
		return
	}
	m.recordHistory()
	sep.Title = label
	sep.UpdatedAt = domain.NowTimestamp()
	m.storeTaskUpdate()
}

// finishCategoryEditing applies the name and reports whether the editor can
// close. A name another category holds keeps the editor open with an error and
// offers to merge into that category instead.
func (m *model) finishCategoryEditing(position selection.Position, name string) bool {
	currentName := m.project.Categories[position.CategoryIndex].Name
	if !m.ui.Edit.isAdding && currentName == name {
		return true
	}
	if i := domain.CategoryNameConflict(m.project.Categories, name, position.CategoryIndex); i >= 0 {
		target := m.project.Categories[i]
		m.ui.Edit.err = fmt.Sprintf("A category named %q already exists", target.Name)
		m.ui.Edit.mergeInto = target.ID
		return false
	}
	if !m.ui.Edit.isAdding {
		m.recordHistory()
	}
	if err := m.project.RenameCategory(position.CategoryIndex, name); err != nil {
		if m.ui.Edit.isAdding {
			m.removeNewCategory()
		}
		m.discardLastHistory()
		m.ui.Screen.StatusMsg = fmt.Sprintf("Rename failed: %v", err)
		return true
	}
	m.storeTaskUpdate()
	return true
}

// mergeEditedCategory accepts the merge offer: the edited category goes into
// the one whose name it took, as one undo step, and the editor closes. A new
// category was only ever a name, so it is dropped and the cursor moves to the
// existing one.
func (m *model) mergeEditedCategory() {
	srcIdx, dst, ok := m.mergeOffer()
	if !ok {
		m.cancelEditing()
		return
	}
	target := m.project.Categories[dst]
	src := m.project.Categories[srcIdx]
	if m.ui.Edit.isAdding {
		m.cancelEditing()
	} else {
		m.recordHistory()
		if err := m.project.MergeCategory(srcIdx, dst); err != nil {
			m.discardLastHistory()
			m.ui.Screen.StatusMsg = fmt.Sprintf("Merge failed: %v", err)
		} else {
			m.storeTaskUpdate()
			m.ui.Screen.StatusMsg = fmt.Sprintf("Merged %q into %q", src.Name, target.Name)
		}
		m.ui.Modes.ToNormal()
		m.ui.Edit.reset()
	}
	m.rebuildPositions()

	// Land on the separator that now heads the moved tasks, else the header.
	onTarget := func(p selection.Position) bool {
		return p.Kind != selection.FocusProject && m.project.Categories[p.CategoryIndex].ID == target.ID
	}
	if len(src.Tasks) == 0 || !m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return onTarget(p) && p.Kind == selection.FocusSeparator && p.TaskIndex == len(target.Tasks)
	}) {
		m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
			return onTarget(p) && p.Kind == selection.FocusCategory
		})
	}
	m.ensureVisible()
}

// mergeOffer returns the edited category and the one the open merge offer
// would merge it into.
func (m model) mergeOffer() (src, dst int, ok bool) {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusCategory {
		return 0, 0, false
	}
	dst, err := m.project.FindCategoryByID(m.ui.Edit.mergeInto)
	if err != nil {
		return 0, 0, false
	}
	return pos.CategoryIndex, dst, true
}

// renameProject checks the name against every other project, then renames the
// in-memory project and saves it like any other edit, so unsaved edits survive.
func (m *model) renameProject(name string) error {
	if err := m.deps.Store.CheckProjectName(name, m.project.ID); err != nil {
		return err
	}
	m.recordHistory()
	m.project.Name = name
	m.project.UpdatedAt = domain.NowTimestamp()
	m.storeTaskUpdate()
	return nil
}

func (m *model) cancelEditing() {
	if m.ui.Edit.isAdding {
		switch m.ui.Edit.itemType {
		case selection.FocusTask:
			m.removeNewTask()
		case selection.FocusCategory:
			m.removeNewCategory()
		}
		// startAddingTask/startAddingCategory pushed a history entry; the cancel
		// rolls back the addition, so drop that snapshot to avoid a no-op undo.
		m.discardLastHistory()
	}
	m.ui.Modes.ToNormal()
	m.ui.Edit.reset()
}

func (m model) forwardToInput(msg tea.Msg) (tea.Model, tea.Cmd) {
	if m.ui.Modes.IsDescriptionEdit() {
		var cmd tea.Cmd
		m.ui.DescriptionEdit.textarea, cmd = m.ui.DescriptionEdit.textarea.Update(msg)
		return m, cmd
	}
	if m.ui.Modes.IsEdit() {
		if m.ui.Edit.mergeInto != "" {
			return m, nil
		}
		before := m.ui.Edit.input.Value()
		var cmd tea.Cmd
		m.ui.Edit.input, cmd = m.ui.Edit.input.Update(msg)
		sanitizeInput(&m.ui.Edit.input)
		if m.ui.Edit.input.Value() != before {
			m.ui.Edit.err = ""
		}
		m.ensureVisible()
		return m, cmd
	}
	if m.ui.Modes.IsSearch() {
		var cmd tea.Cmd
		m.ui.Search.input, cmd = m.ui.Search.input.Update(msg)
		sanitizeInput(&m.ui.Search.input)
		return m, cmd
	}
	if m.ui.Modes.IsCommandPrompt() {
		var cmd tea.Cmd
		m.ui.TaskCommand.input, cmd = m.ui.TaskCommand.input.Update(msg)
		sanitizeInput(&m.ui.TaskCommand.input)
		return m, cmd
	}
	if m.ui.Picker.isAdding {
		var cmd tea.Cmd
		m.ui.Picker.input, cmd = m.ui.Picker.input.Update(msg)
		sanitizeInput(&m.ui.Picker.input)
		return m, cmd
	}
	return m, nil
}

func (m *model) removeNewTask() {
	if m.ui.Edit.newItemID == "" {
		return
	}
	for cIndex := range m.project.Categories {
		for tIndex, task := range m.project.Categories[cIndex].Tasks {
			if task.ID == m.ui.Edit.newItemID {
				_ = m.project.Categories[cIndex].RemoveTask(tIndex)
				break
			}
		}
	}
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) handlePaste(text string) tea.Cmd {
	flat := tea.PasteMsg{Content: flattenPaste(text)}
	var cmd tea.Cmd
	switch m.ui.Modes.Current() {
	case modes.ModeNormal:
		m.pasteText(text, false)
	case modes.ModeEdit:
		m.pasteIntoEditor(text)
	case modes.ModeDescriptionEdit:
		m.ui.DescriptionEdit.textarea, cmd = m.ui.DescriptionEdit.textarea.Update(tea.PasteMsg{Content: text})
	case modes.ModeSearch:
		m.ui.Search.input, cmd = m.ui.Search.input.Update(flat)
		sanitizeInput(&m.ui.Search.input)
		m.previewSearch()
	case modes.ModeCommandPrompt:
		m.ui.TaskCommand.input, cmd = m.ui.TaskCommand.input.Update(flat)
		sanitizeInput(&m.ui.TaskCommand.input)
	case modes.ModeHelp:
		if !m.ui.Help.Filtering {
			cmd = m.startHelpFilter()
		}
		cmd = tea.Batch(cmd, m.updateHelpFilter(flat))
	case modes.ModeProjectPicker:
		switch {
		case m.ui.Picker.isAdding:
			m.ui.Picker.input, cmd = m.ui.Picker.input.Update(flat)
			sanitizeInput(&m.ui.Picker.input)
		case m.ui.Picker.filtering:
			cmd = m.updatePickerFilter(flat)
		default:
			m.ui.Screen.StatusMsg = "Press / to filter, then paste"
		}
	case modes.ModeVisual:
		m.ui.Screen.StatusMsg = "Leave visual mode to paste"
	default:
		m.ui.Screen.StatusMsg = "Nothing here takes a paste"
	}
	return cmd
}

func (m *model) pasteIntoEditor(text string) {
	if m.ui.Edit.mergeInto != "" {
		return
	}
	if m.ui.Edit.itemType == selection.FocusTask {
		if rows := parsePastedRows(text); pasteNeedsChooser(rows) {
			m.openPasteChooser(text, rows, false)
			return
		}
	}
	m.insertAtEditCaret(flattenPaste(text))
	m.ui.Edit.err = ""
	m.ensureVisible()
}
