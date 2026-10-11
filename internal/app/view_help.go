package app

import (
	"fmt"
	"slices"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ilyasturki/phasionary/internal/ui"
)

const (
	// The dialog's own rows around the body: title, the blank under it, the
	// blank above the footer, the footer.
	helpOwnRows = 4
	// helpCardBody is taller than the card itself, because it is also the
	// window the `/` filter scrolls its matches inside.
	helpCardBody = 11
	// Floor on a short terminal: overflow rather than collapse to nothing.
	helpMinBody = 5
	// The longest display string plus a gap.
	helpKeyCol     = 17
	helpCardKeyCol = 8
)

type helpEntry struct {
	keys         string
	desc         string
	bindingIndex int
	// command is set instead of bindingIndex for a configured task command.
	command *taskCommand
	// false for a shortcut Enter must not fire: one that only exists inside
	// editing or visual mode, or `?` itself.
	runnable bool
}

// Natural case, so smartcase applies.
func (e helpEntry) filterText() string { return e.keys + " " + e.desc }

// Exactly one of header, entry and note is set.
type helpCell struct {
	header string
	entry  *helpEntry
	note   string
}

// The full reference uses one cell per line; the essentials card lays two
// columns side by side.
type helpLine []helpCell

type helpCursor struct {
	line int
	col  int
}

type helpBody struct {
	lines []helpLine
	// The shortcut cells the cursor can land on, in the order j/k walks them.
	targets []helpCursor
}

func (b helpBody) entryAt(c helpCursor) *helpEntry {
	return b.lines[c.line][c.col].entry
}

type helpSection struct {
	title   string
	entries []helpEntry
}

// Reference only: pressing Enter on one of these would fire it in the wrong
// context, so every entry stays unrunnable.
var helpModeSections = []helpSection{
	{
		title: "While editing text",
		entries: []helpEntry{
			{keys: "enter", desc: "save"},
			{keys: "esc", desc: "cancel"},
			{keys: "← / →", desc: "move the cursor"},
			{keys: "ctrl+a / ctrl+e", desc: "start / end of line"},
			{keys: "ctrl+← / ctrl+→", desc: "move by word"},
			{keys: "ctrl+w", desc: "delete word back"},
			{keys: "ctrl+k / ctrl+u", desc: "delete to end / start"},
			{keys: "ctrl+v", desc: "paste (several lines ask how)"},
		},
	},
	{
		title: "In visual mode (v)",
		entries: []helpEntry{
			{keys: "j / k", desc: "extend the selection"},
			{keys: "J / K", desc: "move the selection down / up"},
			{keys: "> / <", desc: "nest / un-nest the selection"},
			{keys: "o", desc: "swap which end moves"},
			{keys: "space", desc: "cycle status of every row"},
			{keys: "y / Y", desc: "copy / copy as markdown"},
			{keys: "x", desc: "cut (p moves it, esc cancels)"},
			{keys: "d", desc: "delete (asks first)"},
			{keys: "s", desc: "ship / reopen the selected versions"},
			{keys: "M", desc: "move the selection to a version"},
			{keys: "esc", desc: "leave visual mode"},
		},
	},
}

var builtinHelpSections = buildHelpSections(nil)

// Sections come out in declaration order, so the groups in normalBindings are
// the one source of truth for how the reference reads.
func buildHelpSections(commands []taskCommand) []helpSection {
	var out []helpSection
	for i, b := range normalBindings {
		if b.desc == "" {
			continue
		}
		if len(out) == 0 || out[len(out)-1].title != b.section {
			out = append(out, helpSection{title: b.section})
		}
		display := b.display
		if display == "" {
			display = strings.Join(b.keys, " / ")
		}
		last := &out[len(out)-1]
		last.entries = append(last.entries, helpEntry{
			keys:         display,
			desc:         b.desc,
			bindingIndex: i,
			// Running `?` from the dialog would only reopen the dialog.
			runnable: !slices.Contains(b.keys, "?"),
		})
	}
	if len(commands) > 0 {
		section := helpSection{title: sectionCommands}
		for i := range commands {
			section.entries = append(section.entries, helpEntry{
				keys:     commands[i].Key,
				desc:     commands[i].Label,
				command:  &commands[i],
				runnable: true,
			})
		}
		out = append(out, section)
	}
	return append(out, helpModeSections...)
}

var builtinReferenceLines = referenceLines(builtinHelpSections)

func (m model) helpSections() []helpSection {
	if m.deps.helpSections == nil {
		return builtinHelpSections
	}
	return m.deps.helpSections
}

func (m model) helpReferenceLines() []helpLine {
	if m.deps.helpReferenceLines == nil {
		return builtinReferenceLines
	}
	return m.deps.helpReferenceLines
}

func (m model) helpShortcutCount() int {
	n := 0
	for _, s := range m.helpSections() {
		n += len(s.entries)
	}
	return n
}

func referenceLines(sections []helpSection) []helpLine {
	var lines []helpLine
	for i, s := range sections {
		if i > 0 {
			lines = append(lines, helpLine{{}})
		}
		lines = append(lines, helpLine{{header: s.title}})
		for _, e := range s.entries {
			lines = append(lines, helpLine{{entry: &e}})
		}
	}
	return lines
}

// cardShortcut names a binding by the key that triggers it, so a shortcut that
// no longer exists fails the card's test instead of teaching the wrong key.
type cardShortcut struct {
	key   string
	label string
	desc  string
}

type cardGroup struct {
	title     string
	shortcuts []cardShortcut
}

// The essentials card: the shortcuts that cover reading, creating, changing and
// finding. Everything else waits behind `a`.
var helpCardColumns = [2][]cardGroup{
	{
		{title: "Move", shortcuts: []cardShortcut{
			{key: "j", label: "j / k", desc: "move up / down"},
			{key: "tab", label: "tab", desc: "fold / unfold"},
			{key: "/", label: "/", desc: "search"},
		}},
		{title: "Do", shortcuts: []cardShortcut{
			{key: "space", label: "space", desc: "cycle status"},
			{key: "u", label: "u", desc: "undo"},
			{key: "f", label: "f", desc: "filter tasks"},
		}},
	},
	{
		{title: "Create & edit", shortcuts: []cardShortcut{
			{key: "a", label: "a", desc: "add task"},
			{key: "A", label: "A", desc: "add category"},
			{key: "enter", label: "enter", desc: "edit"},
			{key: "d", label: "d", desc: "delete"},
		}},
	},
}

var helpCardLines = cardLines()

func cardLines() []helpLine {
	columns := [2][]helpCell{}
	for col, groups := range helpCardColumns {
		for i, g := range groups {
			if i > 0 {
				columns[col] = append(columns[col], helpCell{})
			}
			columns[col] = append(columns[col], helpCell{header: g.title})
			for _, s := range g.shortcuts {
				e := s.entry()
				columns[col] = append(columns[col], helpCell{entry: &e})
			}
		}
	}

	height := max(len(columns[0]), len(columns[1]))
	lines := make([]helpLine, 0, height)
	for i := range height {
		line := make(helpLine, 2)
		for col := range columns {
			if i < len(columns[col]) {
				line[col] = columns[col][i]
			}
		}
		lines = append(lines, line)
	}
	return lines
}

// entry resolves the card shortcut against the real bindings, so Enter runs the
// same action the key would in normal mode.
func (s cardShortcut) entry() helpEntry {
	for i, b := range normalBindings {
		if b.prefix == 0 && slices.Contains(b.keys, s.key) {
			return helpEntry{keys: s.label, desc: s.desc, bindingIndex: i, runnable: true}
		}
	}
	return helpEntry{keys: s.label, desc: s.desc, bindingIndex: -1}
}

func (m model) helpExpanded() bool {
	return m.ui.Screen.HelpExpanded
}

// Depends only on the mode and the terminal, never on what the filter matched:
// the body is padded with blanks instead of shrinking, so the dialog keeps its
// size and — centered on the rendered box — its position.
func (m model) helpBodyHeight() int {
	room := ui.DialogBodyHeight(m.ui.Screen.Height, helpOwnRows, helpMinBody)
	if !m.helpExpanded() {
		return min(helpCardBody, room)
	}
	return min(len(m.helpReferenceLines()), room)
}

func (m model) helpQuery() string {
	if !m.ui.Help.Filtering {
		return ""
	}
	return m.ui.Help.Filter.Value()
}

// Filtering always searches the whole reference — including sections the
// current mode hides — so `/` finds a shortcut wherever it lives.
func (m model) helpBody() helpBody {
	var lines []helpLine
	switch {
	case strings.TrimSpace(m.helpQuery()) != "":
		lines = filteredReferenceLines(m.helpSections(), m.helpQuery())
	case m.helpExpanded():
		lines = m.helpReferenceLines()
	default:
		lines = helpCardLines
	}
	return helpBody{lines: lines, targets: helpTargets(lines)}
}

// Down a column before moving to the next one, so the card's two columns read
// as two lists rather than as rows.
func helpTargets(lines []helpLine) []helpCursor {
	var out []helpCursor
	for col := range 2 {
		for i, l := range lines {
			if col < len(l) && l[col].entry != nil {
				out = append(out, helpCursor{line: i, col: col})
			}
		}
	}
	return out
}

func filteredReferenceLines(all []helpSection, query string) []helpLine {
	var sections []helpSection
	for _, s := range all {
		var matched []helpEntry
		for _, e := range s.entries {
			if ui.Contains(e.filterText(), query) {
				matched = append(matched, e)
			}
		}
		if len(matched) > 0 {
			sections = append(sections, helpSection{title: s.title, entries: matched})
		}
	}
	if len(sections) == 0 {
		return []helpLine{{{note: "nothing matches that"}}}
	}
	return referenceLines(sections)
}

// ensureHelpVisible clamps the cursor into the current target list and scrolls
// the body so the cursor's line is on screen.
func (m *model) ensureHelpVisible() {
	body := m.helpBody()
	height := m.helpBodyHeight()

	if len(body.targets) == 0 {
		m.ui.Help.Focused = 0
	} else {
		m.ui.Help.Focused = min(max(m.ui.Help.Focused, 0), len(body.targets)-1)
		line := body.targets[m.ui.Help.Focused].line
		if line < m.ui.Help.ScrollOffset {
			m.ui.Help.ScrollOffset = line
		}
		if line >= m.ui.Help.ScrollOffset+height {
			m.ui.Help.ScrollOffset = line - height + 1
		}
	}
	m.ui.Help.ScrollOffset = min(max(m.ui.Help.ScrollOffset, 0), max(len(body.lines)-height, 0))
}

func (m *model) moveHelpFocus(delta int) {
	body := m.helpBody()
	if len(body.targets) == 0 {
		return
	}
	m.ui.Help.Focused = min(max(m.ui.Help.Focused+delta, 0), len(body.targets)-1)
	m.ensureHelpVisible()
}

func (m *model) moveHelpColumn(delta int) {
	body := m.helpBody()
	if len(body.targets) == 0 {
		return
	}
	cur := body.targets[min(max(m.ui.Help.Focused, 0), len(body.targets)-1)]
	want := cur.col + delta
	best := -1
	for i, t := range body.targets {
		if t.col == want && (best < 0 || abs(t.line-cur.line) < abs(body.targets[best].line-cur.line)) {
			best = i
		}
	}
	if best < 0 {
		return
	}
	m.ui.Help.Focused = best
	m.ensureHelpVisible()
}

// The cursor resets to the top: the two faces share no ordering, so carrying a
// position across would land anywhere.
func (m *model) toggleHelpExpanded() {
	m.ui.Screen.HelpExpanded = !m.ui.Screen.HelpExpanded
	m.ui.Help.Focused = 0
	m.ui.Help.ScrollOffset = 0
	m.ensureHelpVisible()
}

var helpFilterHint = []ui.Hint{
	{Key: "enter", Label: "run"},
	{Key: "esc", Label: "clear"},
}

func (m model) helpHints() []ui.Hint {
	if m.ui.Help.Filtering {
		return helpFilterHint
	}
	toggle := fmt.Sprintf("all %d shortcuts", m.helpShortcutCount())
	if m.helpExpanded() {
		toggle = "essentials"
	}
	return []ui.Hint{
		{Key: "a", Label: toggle},
		{Key: "/", Label: "find"},
		{Key: "enter", Label: "run"},
		{Key: "?/esc", Label: "close"},
	}
}

// The title and the filter prompt share the first row, so entering the filter
// cannot change the dialog's height.
func (m model) helpTitleRow(body helpBody, width int) string {
	if m.ui.Help.Filtering {
		return m.filterPromptRow(m.ui.Help.Filter, strings.TrimSpace(m.helpQuery()), len(body.targets), width)
	}

	title := ui.DialogTitleStyle.Render("Keyboard Shortcuts")
	height := m.helpBodyHeight()
	if len(body.lines) <= height {
		return title
	}
	first := m.ui.Help.ScrollOffset + 1
	last := min(m.ui.Help.ScrollOffset+height, len(body.lines))
	pos := fmt.Sprintf("%d–%d of %d", first, last, len(body.lines))
	return ui.SplitRow(title, ui.MutedStyle.Render(pos), width)
}

func (m model) helpView() string {
	body := m.helpBody()
	width := m.dialogWidth()
	height := m.helpBodyHeight()
	query := strings.TrimSpace(m.helpQuery())

	cursor := helpCursor{line: -1}
	if n := len(body.targets); n > 0 {
		cursor = body.targets[min(max(m.ui.Help.Focused, 0), n-1)]
	}

	start := min(max(m.ui.Help.ScrollOffset, 0), max(len(body.lines)-height, 0))
	lines := []string{m.helpTitleRow(body, width), ""}
	for i := start; i < start+height; i++ {
		if i >= len(body.lines) {
			lines = append(lines, "")
			continue
		}
		col := -1
		if cursor.line == i {
			col = cursor.col
		}
		lines = append(lines, m.renderBodyLine(body.lines[i], col, width, query))
	}
	lines = append(lines, "", ui.RenderHintsToWidth(m.helpHints(), width))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

// cursorCol is the column holding the cursor, or -1 when the cursor is on
// another line.
func (m model) renderBodyLine(line helpLine, cursorCol, width int, query string) string {
	n := len(line)
	keyCol := helpKeyCol
	if n > 1 {
		keyCol = helpCardKeyCol
	}
	cellWidth := width / n
	var b strings.Builder
	for col, cell := range line {
		w := cellWidth
		if col == n-1 {
			w = width - cellWidth*(n-1)
		}
		b.WriteString(m.renderHelpCell(cell, w, keyCol, col == cursorCol, query))
	}
	return strings.TrimRight(b.String(), " ")
}

func (m model) renderHelpCell(cell helpCell, width, keyCol int, focused bool, query string) string {
	switch {
	case cell.header != "":
		return ui.PadTo(ui.HeaderStyle.Render(cell.header), width)
	case cell.note != "":
		return ui.PadTo(ui.MutedStyle.Render("  "+cell.note), width)
	case cell.entry == nil:
		return strings.Repeat(" ", width)
	}

	e := *cell.entry
	base := lipgloss.NewStyle()
	match := ui.SearchMatchStyle
	if focused {
		base = ui.GetSelectedStyle(m.ui.Screen.WindowFocused)
		match = ui.SearchCurrentMatchStyle
	}

	text := base.Render("  ") +
		ui.HighlightMatches(ui.PadTo(e.keys, keyCol), query, ui.DialogKey(base, focused), match) +
		ui.HighlightMatches(e.desc, query, base, match)
	if !focused {
		return ui.PadTo(text, width)
	}
	return ui.PadToStyle(text, width, base)
}
