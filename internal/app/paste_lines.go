package app

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/clipboard"
	"github.com/ilyasturki/ordo/internal/domain"
)

// An untitled separator is a bare rule; a subtask joins the nearest non-subtask row above.
type pastedRow struct {
	title       string
	status      string
	separator   bool
	subtask     bool
	description string
}

var (
	bulletPattern = regexp.MustCompile(`^\s*(?:[-*+•]|\d+[.)])\s+`)
	// The markers are the ones the app's own markdown copy writes.
	checkboxPattern = regexp.MustCompile(`^\s*\[([ xX~-])\]\s*`)
	headingPattern  = regexp.MustCompile(`^\s*#{1,6}\s+`)
	rulePattern     = regexp.MustCompile(`^\s*(?:-{3,}|\*{3,}|_{3,})\s*$`)
)

func splitPasteLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	return strings.Split(strings.ReplaceAll(text, "\r", "\n"), "\n")
}

func isListItem(line string) bool {
	return bulletPattern.MatchString(line) || checkboxPattern.MatchString(line)
}

func indentOf(line string) int {
	return len(line) - len(strings.TrimLeft(line, " \t"))
}

// An indented list item stays a row of its own, never description text.
func continuesDescription(line string, parentIndent int) bool {
	return indentOf(line) > parentIndent && !isListItem(line)
}

func parsePastedRows(text string) []pastedRow {
	lines := splitPasteLines(text)
	// Bare lines standing beside list items are the separators that group them;
	// headings and rules say nothing about the shape of the rest.
	hasMarked := false
	for _, line := range lines {
		if rulePattern.MatchString(line) || headingPattern.MatchString(line) {
			continue
		}
		if body, _, marked := parseItem(line); marked && body != "" {
			hasMarked = true
			break
		}
	}

	var rows []pastedRow
	var desc []string
	parentIndent := -1
	// Last top-level task's indent, -1 after a separator: a deeper item is its subtask.
	topIndent := -1
	flush := func() {
		if len(desc) > 0 {
			rows[len(rows)-1].description = dedentLines(desc)
		}
		desc = nil
	}
	for _, raw := range lines {
		if strings.TrimSpace(raw) == "" {
			if desc != nil {
				desc = append(desc, "")
			}
			continue
		}
		if parentIndent >= 0 && continuesDescription(raw, parentIndent) {
			desc = append(desc, raw)
			continue
		}
		flush()
		parentIndent = -1

		if rulePattern.MatchString(raw) {
			rows = append(rows, pastedRow{separator: true})
			topIndent = -1
			continue
		}
		if m := headingPattern.FindString(raw); m != "" {
			label := strings.Clone(strings.TrimSpace(raw[len(m):]))
			rows = append(rows, pastedRow{title: label, separator: true})
			topIndent = -1
			continue
		}

		body, status, marked := parseItem(raw)
		if body == "" {
			continue
		}
		body = strings.Clone(body)
		if !marked && hasMarked {
			rows = append(rows, pastedRow{title: body, separator: true})
			topIndent = -1
			continue
		}
		indent := indentOf(raw)
		subtask := topIndent >= 0 && indent > topIndent
		if !subtask {
			topIndent = indent
		}
		rows = append(rows, pastedRow{title: body, status: status, subtask: subtask})
		parentIndent = indent
	}
	flush()
	return rows
}

func parseItem(raw string) (body, status string, marked bool) {
	body = raw
	if m := bulletPattern.FindString(body); m != "" {
		body = body[len(m):]
		marked = true
	}
	status = domain.StatusTodo
	if m := checkboxPattern.FindStringSubmatch(body); m != nil {
		body = body[len(m[0]):]
		marked = true
		switch m[1] {
		case "x", "X":
			status = domain.StatusCompleted
		case "~":
			status = domain.StatusInProgress
		case "-":
			status = domain.StatusCancelled
		}
	}
	return strings.TrimSpace(body), status, marked
}

func dedentLines(lines []string) string {
	for len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	common := -1
	for _, l := range lines {
		if l != "" && (common < 0 || indentOf(l) < common) {
			common = indentOf(l)
		}
	}
	out := make([]string, len(lines))
	for i, l := range lines {
		if l != "" {
			out[i] = strings.TrimRight(l[common:], " \t")
		}
	}
	return strings.Clone(strings.Join(out, "\n"))
}

func newTaskForRow(row pastedRow) (domain.Task, error) {
	if row.separator {
		sep, err := domain.NewSeparator()
		sep.Title = row.title
		return sep, err
	}
	task, err := domain.NewTask(row.title)
	if err != nil {
		return domain.Task{}, err
	}
	_ = task.SetStatus(row.status)
	task.Description = row.description
	return task, nil
}

// tasks[i] is built from rows[i]; parentID is the parent of any leading subtasks.
func linkSubtasks(tasks []domain.Task, rows []pastedRow, parentID string) {
	for i := range tasks {
		switch {
		case rows[i].subtask:
			tasks[i].ParentID = parentID
		case rows[i].separator:
			parentID = ""
		default:
			parentID = tasks[i].ID
		}
	}
}

func appendDescription(existing, text string) string {
	existing = strings.TrimRight(existing, "\n")
	if existing == "" || text == "" {
		return existing + text
	}
	return existing + "\n" + text
}

func describeRows(rows []pastedRow, withDescriptions bool) string {
	tasks, subtasks, seps, descs := 0, 0, 0, 0
	for _, r := range rows {
		switch {
		case r.separator:
			seps++
		case r.subtask:
			subtasks++
		default:
			tasks++
		}
		if !r.separator && r.description != "" {
			descs++
		}
	}
	var parts []string
	if tasks > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", tasks, plural(tasks, "task", "tasks")))
	}
	if subtasks > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", subtasks, plural(subtasks, "subtask", "subtasks")))
	}
	if seps > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", seps, plural(seps, "separator", "separators")))
	}
	if withDescriptions && descs > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", descs, plural(descs, "description", "descriptions")))
	}
	return strings.Join(parts, ", ")
}

func pastedRowsSummary(rows []pastedRow) string {
	if len(rows) == 0 {
		return "Pasted"
	}
	return "Added " + describeRows(rows, false)
}

// pasteLinesInEdit turns a multi-line paste made in the inline task editor into
// a batch insert. Reports false so the paste falls back to the plain
// single-line input path.
func (m *model) pasteLinesInEdit(text string) bool {
	if !m.ui.Modes.IsEdit() || m.ui.Edit.itemType != selection.FocusTask {
		return false
	}
	rows := parsePastedRows(text)
	if !pasteNeedsChooser(rows) {
		return false
	}
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind != selection.FocusTask {
		return false
	}

	adding := m.ui.Edit.isAdding
	typed := strings.TrimSpace(m.ui.Edit.input.Value())
	fold := adding || !rows[0].separator
	rest := rows
	if fold {
		if typed != "" {
			rows[0].title = strings.TrimSpace(typed + " " + rows[0].title)
		}
		rest = rows[1:]
	}

	// Mint up front: a mid-loop ID failure must not leave a half-inserted batch.
	newRows := make([]domain.Task, 0, len(rest))
	for _, row := range rest {
		task, err := newTaskForRow(row)
		if err != nil {
			m.ui.Screen.StatusMsg = "Failed to create task ID"
			return true
		}
		newRows = append(newRows, task)
	}

	// Adding already recorded its snapshot; editing records its own here.
	if !adding {
		m.recordHistory()
	}

	cat := &m.project.Categories[pos.CategoryIndex]
	idx := pos.TaskIndex
	if adding && rows[0].separator && cat.Tasks[idx].ParentID != "" {
		// A separator never nests, so the new row first leaves its parent's block.
		idx, _ = cat.OutdentTasks(idx, idx+1)
	}
	edited := &cat.Tasks[idx]
	switch {
	case adding:
		edited.Title = rows[0].title
		if rows[0].separator {
			edited.Kind = domain.KindSeparator
			edited.Status = ""
		} else {
			_ = edited.SetStatus(rows[0].status)
		}
	case fold:
		edited.Title = rows[0].title
	case typed != "":
		edited.Title = typed
	}
	if fold {
		edited.Description = appendDescription(edited.Description, rows[0].description)
	}
	edited.UpdatedAt = domain.NowTimestamp()
	editedID := edited.ID
	last := editedID
	at := domain.BlockEnd(cat.Tasks, idx)
	if fold && !edited.IsSeparator() {
		linkSubtasks(newRows, rest, editedID)
	} else {
		linkSubtasks(newRows, rest, "")
	}
	if edited.ParentID != "" || edited.IsSeparator() {
		at = settleTasks(cat.Tasks, newRows, at, edited.ParentID)
	}
	tookSubtasks := slices.ContainsFunc(newRows, func(t domain.Task) bool { return t.ParentID == editedID })
	cat.Tasks = slices.Insert(cat.Tasks, at, newRows...)
	cat.UpdatedAt = domain.NowTimestamp()
	if tookSubtasks {
		m.revealChildrenOf(editedID)
	}

	if len(newRows) > 0 {
		last = newRows[len(newRows)-1].ID
	}
	m.ui.Modes.ToNormal()
	m.ui.Edit.reset()
	m.rebuildPositions()
	m.selectTaskLevelRow(last)
	m.ensureVisible()
	m.storeTaskUpdate()
	added := rest
	if adding {
		added = rows
	}
	m.ui.Screen.StatusMsg = pastedRowsSummary(added)
	return true
}

type clipboardLinesMsg struct {
	text  string
	image []byte
	err   error
}

func readClipboardLines() tea.Cmd {
	return func() tea.Msg {
		c, err := clipboard.Read()
		return clipboardLinesMsg{text: c.Text, image: c.Image, err: err}
	}
}

// editorClipboardMsg answers a ctrl+v made in the editor of mode.
type editorClipboardMsg struct {
	mode  modes.Mode
	text  string
	image []byte
	err   error
}

func readClipboardForEditor(mode modes.Mode) tea.Cmd {
	return func() tea.Msg {
		c, err := clipboard.Read()
		return editorClipboardMsg{mode: mode, text: c.Text, image: c.Image, err: err}
	}
}

const clipboardHoldsImage = "The clipboard holds an image"

// clipboardReadFailure tells a clipboard nothing can read from one that holds
// nothing: the tools exit non-zero on an empty clipboard too.
func clipboardReadFailure(err error) string {
	switch {
	case errors.Is(err, clipboard.ErrNoBackend) || errors.Is(err, clipboard.ErrTimeout):
		return "Can't read the clipboard: paste with the terminal instead"
	case errors.Is(err, clipboard.ErrNotText):
		return "The clipboard holds no text"
	}
	return "Nothing to paste"
}

func (m *model) handleEditorClipboard(msg editorClipboardMsg) tea.Cmd {
	if m.ui.Modes.Current() != msg.mode {
		return nil
	}
	if msg.image != nil {
		m.ui.Screen.StatusMsg = clipboardHoldsImage
		return nil
	}
	if msg.err != nil {
		m.ui.Screen.StatusMsg = clipboardReadFailure(msg.err)
		return nil
	}
	return m.handlePaste(msg.text)
}

// retiresCopy: the clipboard held newer text than the app's copy, which the
// paste drops in its own undo step.
func (m *model) pasteText(text string, retiresCopy bool) {
	rows := parsePastedRows(text)
	if len(rows) == 0 {
		m.ui.Screen.StatusMsg = "Nothing to paste"
		return
	}
	if pasteNeedsChooser(rows) {
		m.openPasteChooser(text, rows, retiresCopy)
		return
	}
	m.insertPastedRows(rows, retiresCopy)
}

func (m *model) insertPastedRows(rows []pastedRow, retiresCopy bool) {
	catIndex, taskIndex, parentID, ok := m.pasteDestination()
	if !ok {
		m.ui.Screen.StatusMsg = "No category to paste into"
		return
	}

	newRows := make([]domain.Task, 0, len(rows))
	for _, row := range rows {
		task, err := newTaskForRow(row)
		if err != nil {
			m.ui.Screen.StatusMsg = "Failed to create task ID"
			return
		}
		newRows = append(newRows, task)
	}

	m.recordHistory()
	if retiresCopy {
		m.ui.Clipboard = ClipboardState{}
	}
	linkSubtasks(newRows, rows, "")
	cat := &m.project.Categories[catIndex]
	taskIndex = settleTasks(cat.Tasks, newRows, taskIndex, parentID)
	cat.Tasks = slices.Insert(cat.Tasks, taskIndex, newRows...)
	cat.UpdatedAt = domain.NowTimestamp()

	m.rebuildPositions()
	m.selectTaskLevelRow(newRows[len(newRows)-1].ID)
	m.ensureVisible()
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = pastedRowsSummary(rows)
}
