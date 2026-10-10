package app

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"charm.land/bubbles/v2/textinput"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
	"phasionary/internal/ui"
)

func (m model) dialogWidth() int {
	return ui.DialogContentWidth(m.ui.Screen.Width)
}

func (m model) dialogStyle() lipgloss.Style {
	return ui.HelpDialogStyle.Width(m.dialogWidth() + ui.DialogChromeWidth)
}

// filterPromptRow draws a type-to-filter prompt: "/<query>" with a live cursor
// on the left and the match count (or "no matches") right-aligned to width.
// query is what matches was counted for, so an all-whitespace one shows nothing.
func (m model) filterPromptRow(input textinput.Model, query string, matches, width int) string {
	status := ""
	if query != "" {
		text := "no matches"
		if matches > 0 {
			text = fmt.Sprintf("%d %s", matches, plural(matches, "match", "matches"))
		}
		status = ui.MutedStyle.Render(text)
	}
	return m.promptRow("/", input, status, width)
}

func (m model) promptRow(label string, input textinput.Model, status string, width int) string {
	split := splitAtCursor(input.Value(), input.Position())
	left := label + split.left +
		ui.GetCursorStyle(m.ui.Screen.WindowFocused).Render(split.cursorCh) + split.right
	return ui.SplitRow(left, status, width)
}

func (m model) positions() []selection.Position {
	return m.ui.Selection.Positions()
}

func (m model) selected() int {
	return m.ui.Selection.Selected()
}

func (m model) selectedPosition() (selection.Position, bool) {
	return m.ui.Selection.SelectedPosition()
}

func (m model) isTaskCut(taskID string) bool {
	if !m.ui.Clipboard.IsCut || taskID == "" {
		return false
	}
	for _, id := range m.ui.Clipboard.TaskIDs {
		if id == taskID {
			return true
		}
	}
	return false
}

func (m model) isCategoryCut(categoryID string) bool {
	if !m.ui.Clipboard.IsCut || categoryID == "" {
		return false
	}
	for _, id := range m.ui.Clipboard.CategoryIDs {
		if id == categoryID {
			return true
		}
	}
	return false
}

// A paste moves a cut parent with the children it has then, so one nested since the cut is cut too.
func (m model) taskRowCut(t domain.Task) bool {
	return m.isTaskCut(t.ID) || m.isTaskCut(t.ParentID)
}

func (m model) categoryRowCut(c domain.Category) bool {
	return m.isCategoryCut(c.ID) || m.isCategoryCut(c.ParentID)
}

const (
	prefixWidth     = 2
	footerHeight    = 0 // No bottom status area — status text appears on the project line.
	blankAfterProj  = 1
	blankBetweenCat = 1
	blankAfterCat   = 1
)

func (m model) footerHeight() int {
	return footerHeight + m.bottomBarHeight()
}

func (m model) layoutConfig() LayoutConfig {
	cfg := DefaultLayoutConfig()
	cfg.FooterHeight += m.bottomBarHeight()
	return cfg
}

// bubbles' own ctrl+v reads through atotto, whose wl-paste can't be killed; handleKeyMsg reads it through internal/clipboard.
func newTextInput() textinput.Model {
	ti := textinput.New()
	ti.KeyMap.Paste.SetEnabled(false)
	return ti
}

func sanitizeInput(input *textinput.Model) {
	val := input.Value()
	cleaned := strings.Map(func(r rune) rune {
		if r == '\n' || r == '\r' || r == '\t' {
			return ' '
		}
		return r
	}, val)
	if cleaned != val {
		pos := input.Position()
		input.SetValue(cleaned)
		input.SetCursor(pos)
	}
}

func safeWidth(totalWidth, overhead int) int {
	available := totalWidth - overhead
	if available < 1 {
		return 1
	}
	return available
}

// prefixLine renders already-styled text on a single row behind prefix,
// truncating rather than wrapping: an editor's placeholder is always one row.
func prefixLine(text string, width, overhead int, prefix string) string {
	if width <= 0 {
		return prefix + text
	}
	return prefix + ansi.Truncate(text, safeWidth(width, overhead), ui.Ellipsis)
}

type cursorSplit struct {
	left     string
	cursorCh string
	right    string
}

func splitAtCursor(text string, cursor int) cursorSplit {
	if text == "" {
		text = " "
	}
	runes := []rune(text)
	pos := min(max(cursor, 0), len(runes))
	left := string(runes[:pos])
	right := string(runes[pos:])
	cursorCh := " "
	if pos < len(runes) {
		cursorCh = string(runes[pos])
		right = string(runes[pos+1:])
	}
	return cursorSplit{left: left, cursorCh: cursorCh, right: right}
}

// rows can hold one row more than the wrap: a caret past the end of a full row
// would make it one column too wide, which the terminal soft-wraps.
type editLayout struct {
	rows      []ui.LineSpan
	cursorRow int
	cursorCol int // rune index into rows[cursorRow].Text
}

func layoutEdit(text string, cursor, available int) editLayout {
	available = max(available, 1)
	return locateEditCursor(text, ui.WrapSpans(text, available), cursor, available)
}

// locateEditCursor places cursor, a rune index into text, on rows.
func locateEditCursor(text string, rows []ui.LineSpan, cursor, available int) editLayout {
	offset := len(text)
	for i, n := 0, 0; i < len(text); n++ {
		if n == cursor {
			offset = i
			break
		}
		_, size := utf8.DecodeRuneInString(text[i:])
		i += size
	}
	if cursor <= 0 {
		offset = 0
	}

	row := 0
	for i, span := range rows {
		if span.Start > offset {
			break
		}
		row = i
	}
	col := min(offset-rows[row].Start, len(rows[row].Text))
	if col == len(rows[row].Text) && ansi.StringWidth(rows[row].Text) >= available {
		row++
		col = 0
		if row == len(rows) {
			// Capped so the phantom row can't land in a cached row slice's
			// spare capacity.
			rows = append(rows[:len(rows):len(rows)], ui.LineSpan{Start: len(text)})
		}
	}
	return editLayout{
		rows:      rows,
		cursorRow: row,
		cursorCol: utf8.RuneCountInString(rows[row].Text[:col]),
	}
}

func (m *model) editRows(overhead int) editLayout {
	available := safeWidth(m.ui.Screen.Width, overhead)
	value := m.ui.Edit.input.Value()
	return locateEditCursor(value, m.ui.Edit.wrapFor(value, available), m.ui.Edit.input.Position(), available)
}

// The paste chooser keeps the editor it opened from on screen.
func (m model) editorOpen() bool {
	return m.ui.Modes.IsEdit() || (m.ui.Modes.IsPasteChooser() && m.ui.PasteChooser.fromEdit)
}

func (m *model) openEditRows() (editLayout, bool) {
	if !m.editorOpen() {
		return editLayout{}, false
	}
	pos, ok := m.selectedPosition()
	if !ok {
		return editLayout{}, false
	}
	return m.editRows(m.editOverhead(pos)), true
}

func renderEditRows(el editLayout, overhead int, prefix string, textStyle, cursorStyle lipgloss.Style) string {
	indent := strings.Repeat(" ", overhead)

	lines := make([]string, len(el.rows))
	for i, span := range el.rows {
		var body string
		if i == el.cursorRow {
			split := splitAtCursor(span.Text, el.cursorCol)
			body = textStyle.Render(split.left) + cursorStyle.Render(split.cursorCh) + textStyle.Render(split.right)
		} else {
			body = textStyle.Render(span.Text)
		}
		if i == 0 {
			lines[i] = prefix + body
			continue
		}
		lines[i] = indent + body
	}
	return strings.Join(lines, "\n")
}
