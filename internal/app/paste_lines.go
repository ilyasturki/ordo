package app

import (
	"fmt"
	"regexp"
	"strings"

	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/selection"
	"phasionary/internal/clipboard"
	"phasionary/internal/domain"
)

// A separator with an empty title renders as a bare rule.
type pastedRow struct {
	title       string
	status      string
	separator   bool
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

// continuesDescription reports whether a line belongs to the description of
// the task above it. An indented list item stays a row of its own: this is the
// one place that decides what nesting means.
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
			continue
		}
		if m := headingPattern.FindString(raw); m != "" {
			label := strings.Clone(strings.TrimSpace(raw[len(m):]))
			rows = append(rows, pastedRow{title: label, separator: true})
			continue
		}

		body, status, marked := parseItem(raw)
		if body == "" {
			continue
		}
		body = strings.Clone(body)
		if !marked && hasMarked {
			rows = append(rows, pastedRow{title: body, separator: true})
			continue
		}
		rows = append(rows, pastedRow{title: body, status: status})
		parentIndent = indentOf(raw)
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

// dedentLines strips the indent the lines share and the blank lines that trail
// them.
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

// appendDescription puts text on the line below what the description holds.
func appendDescription(existing, text string) string {
	existing = strings.TrimRight(existing, "\n")
	if existing == "" || text == "" {
		return existing + text
	}
	return existing + "\n" + text
}

func pastedRowsSummary(rows []pastedRow) string {
	seps := 0
	for _, r := range rows {
		if r.separator {
			seps++
		}
	}
	tasks := len(rows) - seps
	var parts []string
	if tasks > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", tasks, plural(tasks, "task", "tasks")))
	}
	if seps > 0 {
		parts = append(parts, fmt.Sprintf("%d %s", seps, plural(seps, "separator", "separators")))
	}
	return "Added " + strings.Join(parts, ", ")
}

// pasteLinesInEdit turns a multi-line paste made in the inline task editor into
// a batch insert. Reports false so the paste falls back to the plain
// single-line input path.
func (m *model) pasteLinesInEdit(text string) bool {
	if !m.ui.Modes.IsEdit() || m.ui.Edit.itemType != selection.FocusTask {
		return false
	}
	rows := parsePastedRows(text)
	if len(rows) < 2 {
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
	edited := &cat.Tasks[pos.TaskIndex]
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
	for i, task := range newRows {
		cat.InsertTask(pos.TaskIndex+1+i, task)
	}

	m.ui.Modes.ToNormal()
	m.ui.Edit.reset()
	m.rebuildPositions()
	m.selectTaskLevelRow(newRows[len(newRows)-1].ID, false)
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
	text string
	err  error
}

func readClipboardLines() tea.Cmd {
	return func() tea.Msg {
		text, err := clipboard.Read()
		return clipboardLinesMsg{text: text, err: err}
	}
}

func (m *model) pasteClipboardLines(msg clipboardLinesMsg) {
	rows := parsePastedRows(msg.text)
	if msg.err != nil || len(rows) == 0 {
		m.ui.Screen.StatusMsg = "Nothing to paste"
		return
	}
	catIndex, taskIndex, ok := m.pasteDestination()
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
	for i, task := range newRows {
		m.project.Categories[catIndex].InsertTask(taskIndex+i, task)
	}

	m.rebuildPositions()
	m.selectTaskLevelRow(newRows[len(newRows)-1].ID, false)
	m.ensureVisible()
	m.storeTaskUpdate()
	m.ui.Screen.StatusMsg = pastedRowsSummary(rows)
}
