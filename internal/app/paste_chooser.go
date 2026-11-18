package app

import (
	"fmt"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
	"github.com/ilyasturki/ordo/internal/ui"
)

type pasteMode int

const (
	pasteAsRows pasteMode = iota
	pasteAsDescription
	pasteAsOneLine
	pasteAppend
)

type pasteOption struct {
	key   string
	mode  pasteMode
	label string
}

type PasteChooserState struct {
	text     string
	rows     []pastedRow
	options  []pasteOption
	selected int
	// A paste made in the title editor, which the chooser returns to.
	fromEdit    bool
	retiresCopy bool
}

func pasteNeedsChooser(rows []pastedRow) bool {
	return len(rows) > 1 || (len(rows) == 1 && rows[0].description != "")
}

func (m *model) openPasteChooser(text string, rows []pastedRow, retiresCopy bool) {
	fromEdit := m.ui.Modes.IsEdit()
	oneLine := "One task, every line joined into the title"
	if fromEdit {
		oneLine = "Every line joined, at the cursor"
	}
	options := []pasteOption{
		{key: "t", mode: pasteAsRows, label: "One task per line"},
		{key: "d", mode: pasteAsDescription, label: "One task, the rest as its description"},
		{key: "o", mode: pasteAsOneLine, label: oneLine},
	}
	if title, ok := m.appendTarget(fromEdit); ok {
		label := "Append to this task's description"
		if title != "" {
			label = fmt.Sprintf("Append to the description of “%s”", truncateText(title, 30))
		}
		options = append(options, pasteOption{key: "a", mode: pasteAppend, label: label})
	}
	if !m.ui.Modes.ToPasteChooser() {
		return
	}
	m.ui.PasteChooser = PasteChooserState{text: text, rows: rows, options: options, fromEdit: fromEdit, retiresCopy: retiresCopy}
}

func (m *model) appendTarget(fromEdit bool) (string, bool) {
	if fromEdit {
		return strings.TrimSpace(m.ui.Edit.input.Value()), true
	}
	pos, ok := m.selectedPosition()
	if !ok || (pos.Kind != selection.FocusTask && pos.Kind != selection.FocusDescription) {
		return "", false
	}
	return m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].Title, true
}

func (m model) handlePasteChooserKey(msg tea.KeyPressMsg) (tea.Model, tea.Cmd) {
	c := m.pasteChoice()
	act, i := c.Handle(msg.String())
	m.ui.PasteChooser.selected = c.Selected
	switch act {
	case ui.ChoiceCancelled:
		m.closePasteChooser()
		m.ui.Screen.StatusMsg = "Paste cancelled"
	case ui.ChoicePicked:
		m.applyPaste(m.ui.PasteChooser.options[i].mode)
	}
	return m, nil
}

func (m model) pasteChoice() ui.Choice {
	st := m.ui.PasteChooser
	c := ui.Choice{Title: "Paste: " + describeRows(st.rows, true), Selected: st.selected}
	for _, o := range st.options {
		c.Options = append(c.Options, ui.ChoiceOption{Label: o.label, Key: o.key})
	}
	return c
}

func (m *model) closePasteChooser() {
	if m.ui.PasteChooser.fromEdit {
		m.ui.Modes.ToEdit()
	} else {
		m.ui.Modes.ToNormal()
	}
	m.ui.PasteChooser = PasteChooserState{}
}

func (m *model) applyPaste(mode pasteMode) {
	st := m.ui.PasteChooser
	m.closePasteChooser()
	defer m.ensureVisible()
	if st.fromEdit {
		m.applyPasteInEditor(mode, st)
		return
	}
	switch mode {
	case pasteAsRows:
		m.insertPastedRows(st.rows, st.retiresCopy)
	case pasteAsDescription:
		row := titleAndDescription(st.text)
		if row.title == "" {
			m.ui.Screen.StatusMsg = "Nothing to paste as a title"
			return
		}
		m.insertPastedRows([]pastedRow{row}, st.retiresCopy)
	case pasteAsOneLine:
		m.insertPastedRows([]pastedRow{{title: flattenPaste(st.text), status: domain.StatusTodo}}, st.retiresCopy)
	case pasteAppend:
		m.appendToFocusedDescription(st.text, st.retiresCopy)
	}
}

func (m *model) applyPasteInEditor(mode pasteMode, st PasteChooserState) {
	switch mode {
	case pasteAsRows:
		if !m.pasteLinesInEdit(st.text) {
			m.insertAtEditCaret(flattenPaste(st.text))
		}
	case pasteAsDescription:
		m.pasteTitleAndDescriptionInEdit(st.text)
	case pasteAsOneLine:
		m.insertAtEditCaret(flattenPaste(st.text))
	case pasteAppend:
		m.appendToEditedDescription(st.text)
	}
}

func (m *model) appendToFocusedDescription(text string, retiresCopy bool) {
	pos, ok := m.selectedPosition()
	if !ok || (pos.Kind != selection.FocusTask && pos.Kind != selection.FocusDescription) {
		return
	}
	m.recordHistory()
	if retiresCopy {
		m.ui.Clipboard = ClipboardState{}
	}
	task := &m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
	task.Description = operations.JoinDescription(task.Description, pasteBlock(text))
	task.UpdatedAt = domain.NowTimestamp()
	m.rebuildPositions()
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = "Appended to the description"
}

// An add's own undo step already covers the new task.
func (m *model) appendToEditedDescription(text string) {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		return
	}
	adding := m.ui.Edit.isAdding
	if !adding {
		m.recordHistory()
	}
	task := &m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
	task.Description = operations.JoinDescription(task.Description, pasteBlock(text))
	if !adding {
		task.UpdatedAt = domain.NowTimestamp()
		m.storeTaskUpdate()
	}
	m.rebuildPositions()
	m.ui.Screen.StatusMsg = "Appended to the description"
}

func (m *model) pasteTitleAndDescriptionInEdit(text string) {
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		return
	}
	row := titleAndDescription(text)
	title := strings.TrimSpace(strings.TrimSpace(m.ui.Edit.input.Value()) + " " + row.title)
	if title == "" {
		m.ui.Screen.StatusMsg = "Nothing to paste as a title"
		return
	}
	adding := m.ui.Edit.isAdding
	if !adding {
		m.recordHistory()
	}
	task := &m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex]
	task.Title = title
	if adding {
		_ = task.SetStatus(row.status)
	}
	task.Description = operations.JoinDescription(task.Description, row.description)
	task.UpdatedAt = domain.NowTimestamp()
	id := task.ID
	m.ui.Modes.ToNormal()
	m.ui.Edit.reset()
	m.rebuildPositions()
	m.selectTaskByID(id)
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = "Pasted into the title and description"
	if adding {
		m.ui.Screen.StatusMsg = "Added 1 task with a description"
	}
}

func (m *model) insertAtEditCaret(text string) {
	m.ui.Edit.input, _ = m.ui.Edit.input.Update(tea.PasteMsg{Content: text})
	sanitizeInput(&m.ui.Edit.input)
}

func titleAndDescription(text string) pastedRow {
	lines := splitPasteLines(text)
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			continue
		}
		title, status := strings.TrimSpace(l), domain.StatusTodo
		if m := headingPattern.FindString(l); m != "" {
			title = strings.TrimSpace(l[len(m):])
		} else if body, s, marked := parseItem(l); marked && body != "" {
			title, status = body, s
		}
		if title == "" {
			continue
		}
		return pastedRow{
			title:       title,
			status:      status,
			description: pasteBlock(strings.Join(lines[i+1:], "\n")),
		}
	}
	return pastedRow{status: domain.StatusTodo}
}

func pasteBlock(text string) string {
	lines := splitPasteLines(text)
	for i, l := range lines {
		if strings.TrimSpace(l) == "" {
			lines[i] = ""
		}
	}
	for len(lines) > 0 && lines[0] == "" {
		lines = lines[1:]
	}
	return dedentLines(lines)
}

func flattenPaste(text string) string {
	var parts []string
	for _, l := range splitPasteLines(text) {
		if strings.TrimSpace(l) != "" {
			parts = append(parts, l)
		}
	}
	if len(parts) == 1 {
		return parts[0]
	}
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return strings.Join(parts, " ")
}

func (m model) pasteChooserView() string {
	return m.choiceView(m.pasteChoice())
}
