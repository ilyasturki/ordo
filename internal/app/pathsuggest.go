package app

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"charm.land/bubbles/v2/textarea"
	tea "charm.land/bubbletea/v2"
	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/ui"
)

const (
	pathListRows     = 8
	pathListMaxWidth = 40
)

type pathEntry struct {
	name string
	dir  bool
}

type pathDirMsg struct {
	dir     string
	entries []pathEntry
}

// pathSuggest is the list of entries matching the path under a field's caret.
type pathSuggest struct {
	listings map[string][]pathEntry
	reading  map[string]bool
	// word names the path under the caret by its line and start, so a
	// dismissal holds until the caret leaves it.
	word      string
	start     int
	typed     string
	shown     string
	items     []pathEntry
	sel       int
	dismissed bool
	// completed is the file path tab put in, which keeps the list closed
	// until the path changes.
	completed string
}

func (s *pathSuggest) open() bool {
	return !s.dismissed && len(s.items) > 0 && s.typed != s.completed
}

// refresh follows the field's caret, returning a read of a directory not
// listed yet.
func (s *pathSuggest) refresh(buf fieldBuffer) tea.Cmd {
	line, caret := buf.currentLine()
	typed, start, ok := pathBeforeCaret(line, caret)
	if !ok {
		*s = pathSuggest{listings: s.listings, reading: s.reading}
		return nil
	}
	if word := fmt.Sprint(buf.snap().line, ":", start); word != s.word {
		s.word, s.dismissed, s.completed = word, false, ""
	}
	s.start, s.typed = start, typed
	cut := strings.LastIndex(typed, "/") + 1
	dir, ok := resolvePath(typed[:cut])
	if !ok {
		s.items, s.shown = nil, ""
		return nil
	}
	entries, listed := s.listings[dir]
	if !listed {
		s.items, s.shown = nil, ""
		if s.reading[dir] {
			return nil
		}
		if s.reading == nil {
			s.reading = map[string]bool{}
		}
		s.reading[dir] = true
		return readPathDir(dir)
	}
	if typed == s.shown {
		return nil
	}
	s.items, s.sel, s.shown = matchPathEntries(entries, typed[cut:]), 0, typed
	return nil
}

func matchPathEntries(entries []pathEntry, partial string) []pathEntry {
	hidden := strings.HasPrefix(partial, ".")
	var out []pathEntry
	for _, e := range entries {
		if strings.HasPrefix(e.name, partial) && (hidden || !strings.HasPrefix(e.name, ".")) {
			out = append(out, e)
		}
	}
	return out
}

func (s *pathSuggest) listed(msg pathDirMsg) {
	if !s.reading[msg.dir] {
		return
	}
	delete(s.reading, msg.dir)
	if s.listings == nil {
		s.listings = map[string][]pathEntry{}
	}
	s.listings[msg.dir] = msg.entries
	s.shown = ""
}

func readPathDir(dir string) tea.Cmd {
	return func() tea.Msg {
		des, _ := os.ReadDir(dir)
		entries := make([]pathEntry, 0, len(des))
		for _, de := range des {
			isDir := de.IsDir()
			if de.Type()&fs.ModeSymlink != 0 {
				if fi, err := os.Stat(filepath.Join(dir, de.Name())); err == nil {
					isDir = fi.IsDir()
				}
			}
			entries = append(entries, pathEntry{name: de.Name(), dir: isDir})
		}
		return pathDirMsg{dir: dir, entries: entries}
	}
}

// key runs the open list's keys and reports whether it took key.
func (s *pathSuggest) key(key string, buf fieldBuffer, edits *fieldEdits) bool {
	if !s.open() {
		return false
	}
	n := len(s.items)
	switch key {
	case "tab":
		s.complete(buf, edits)
	case "up", "ctrl+p":
		s.sel = (s.sel + n - 1) % n
	case "down", "ctrl+n":
		s.sel = (s.sel + 1) % n
	case "esc":
		s.dismissed = true
	default:
		return false
	}
	return true
}

// complete puts the selected entry in place of the typed name; a file ends
// the list, a directory opens its own.
func (s *pathSuggest) complete(buf fieldBuffer, edits *fieldEdits) {
	entry := s.items[s.sel]
	line, caret := buf.currentLine()
	cut := s.start + len([]rune(s.typed[:strings.LastIndex(s.typed, "/")+1]))
	name := []rune(entry.name)
	if entry.dir {
		name = append(name, '/')
	}
	edits.push(buf.snap())
	edits.moved()
	next := append(append(append([]rune{}, line[:cut]...), name...), line[caret:]...)
	buf.setCurrentLine(next, cut+len(name))
	if !entry.dir {
		s.completed = s.typed[:strings.LastIndex(s.typed, "/")+1] + entry.name
	}
}

func titlePathsOn(kind selection.FocusKind) bool {
	return kind == selection.FocusTask || kind == selection.FocusSeparator
}

func (m *model) refreshPathSuggest() tea.Cmd {
	switch {
	case m.ui.Modes.IsEdit() && titlePathsOn(m.ui.Edit.itemType):
		return m.ui.Edit.paths.refresh(&m.ui.Edit.input)
	case m.ui.Modes.IsDescriptionEdit():
		return m.ui.DescriptionEdit.paths.refresh(&m.ui.DescriptionEdit.textarea)
	}
	return nil
}

func (m *model) pathListed(msg pathDirMsg) {
	m.ui.Edit.paths.listed(msg)
	m.ui.DescriptionEdit.paths.listed(msg)
}

// renderPathList draws at most rows entries around the selection.
func renderPathList(s *pathSuggest, rows, width int) string {
	inner := max(min(pathListMaxWidth, width-4), 1)
	n := min(len(s.items), max(rows, 1))
	top := max(0, s.sel-n+1)
	labels := make([]string, 0, n+1)
	widest := 0
	for _, e := range s.items[top : top+n] {
		label := e.name
		if e.dir {
			label += "/"
		}
		label = ansi.Truncate(label, inner, "…")
		widest = max(widest, ansi.StringWidth(label))
		labels = append(labels, label)
	}
	count := ""
	if len(s.items) > n {
		count = fmt.Sprintf("%d of %d", s.sel+1, len(s.items))
		widest = max(widest, len(count))
	}
	lines := make([]string, 0, n+1)
	for i, label := range labels {
		e := s.items[top+i]
		padded := label + strings.Repeat(" ", widest-ansi.StringWidth(label))
		switch {
		case top+i == s.sel:
			lines = append(lines, ui.SelectedStyle.Render(padded))
		case e.dir:
			lines = append(lines, ui.CategoryStyle.Render(padded))
		default:
			lines = append(lines, padded)
		}
	}
	if count != "" {
		lines = append(lines, ui.MutedStyle.Render(count+strings.Repeat(" ", widest-len(count))))
	}
	return pathListStyle.Render(strings.Join(lines, "\n"))
}

func lineCount(lines []string) int {
	if len(lines) == 0 {
		return 0
	}
	return strings.Count(strings.Join(lines, "\n"), "\n") + 1
}

// withTitlePathList paints the open list under the edited row that starts at
// line editLine of content, entered skip rows down.
func (m model) withTitlePathList(content string, editLine, skip int) string {
	if !m.ui.Modes.IsEdit() || m.ui.Edit.clash != "" {
		return content
	}
	pos, ok := m.selectedPosition()
	if !ok {
		return content
	}
	overhead := m.editOverhead(pos)
	el := m.editRows(overhead)
	rows := len(el.rows)
	if m.ui.Edit.err != "" {
		rows++
	}
	n := strings.Count(content, "\n") + 1
	if n < m.ui.Screen.Height {
		content += strings.Repeat("\n", m.ui.Screen.Height-n)
	}
	caret := []rune(el.rows[el.cursorRow].Text)
	x := overhead + max(0, ansi.StringWidth(string(caret[:el.cursorCol]))-ansi.StringWidth(m.ui.Edit.paths.typed))
	painted := strings.TrimRight(placePathList(content, &m.ui.Edit.paths, editLine, editLine+rows-skip-1, x, m.ui.Screen.Height-1, m.ui.Screen.Width), "\n")
	if short := n - strings.Count(painted, "\n") - 1; short > 0 {
		painted += strings.Repeat("\n", short)
	}
	return painted
}

// descriptionPathList paints the open list over the description dialog's
// content, whose textarea starts on its third line behind a 2-wide gutter.
func descriptionPathList(content string, s *pathSuggest, ta textarea.Model) string {
	row := 2 + descriptionCursorDisplayRow(ta) - ta.ScrollYOffset()
	x := 2 + max(0, ta.LineInfo().CharOffset-ansi.StringWidth(s.typed))
	return placePathList(content, s, row, row, x, strings.Count(content, "\n")+1, lipgloss.Width(content))
}

var pathListStyle = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).Padding(0, 1)

// pathListHeight is the box's rows for n entries of a list of total.
func pathListHeight(n, total int) int {
	if total > n {
		return n + 3
	}
	return n + 2
}

// placePathList paints the open list over the first limit lines of bg, under
// the lines from top to bottom, or above them when there is more room there.
func placePathList(bg string, s *pathSuggest, top, bottom, x, limit, width int) string {
	below, above := limit-bottom-1, top
	room := below
	y := bottom + 1
	if pathListHeight(min(len(s.items), pathListRows), len(s.items)) > below && above > below {
		room = above
	}
	rows := min(len(s.items), pathListRows)
	for rows > 1 && pathListHeight(rows, len(s.items)) > room {
		rows--
	}
	box := renderPathList(s, rows, width)
	h := strings.Count(box, "\n") + 1
	if room == above {
		y = top - h
	}
	y = max(y, 0)
	x = max(min(x, width-lipgloss.Width(box)), 0)
	return components.OverlayAt(bg, box, x, y)
}
