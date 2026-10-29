package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

const (
	// title, the blank under it, the pinned New Project row, the blank under
	// that, the blank above the hints.
	pickerOwnRows = 5
	// Rows kept free below the list for the up/down scroll indicators, one each.
	pickerScrollReserve = 2
	// pickerMinVisible keeps the list usable on short terminals even when the
	// height budget would otherwise round down to nothing.
	pickerMinVisible = 3
	// pickerFallbackVisible is used before the first window-size message, when
	// the terminal height isn't known yet.
	pickerFallbackVisible = 10
	// On a very narrow terminal, overflow beats collapsing to nothing.
	pickerFullMinWidth = 20
	// "9999 open" + "999 ▸" + "12mo", plus a gap each.
	pickerFullMetaWidth = 24
	pickerMinNameWidth  = 16
	pickerColumnGap     = 2
)

// pickerNormalHints are the footer hints shown while browsing the picker. Plain
// navigation (j/k, paging, g/G) is intentionally omitted per dialog-footer
// convention; only picker-specific actions are listed. link is the label of l,
// "unlink" on the linked row.
func (m model) pickerNormalHints(link string) []ui.Hint {
	back := "back"
	if m.project.ID == "" {
		back = "quit"
	}
	return []ui.Hint{
		{Key: "⏎", Label: "select"},
		{Key: "a", Label: "new"},
		{Key: "/", Label: "search"},
		{Key: "l", Label: link},
		{Key: "J/K", Label: "move"},
		{Key: "d", Label: "delete"},
		{Key: "esc", Label: back},
	}
}

func (m model) pickerHints() []ui.Hint {
	switch {
	case m.ui.Picker.isAdding:
		return []ui.Hint{{Key: "⏎", Label: "create"}, {Key: "esc", Label: "cancel"}}
	case m.ui.Picker.filtering:
		return []ui.Hint{{Key: "⏎", Label: "select"}, {Key: "esc", Label: "clear"}}
	}
	link := "link"
	if m.onLinkedRow() {
		link = "unlink"
	}
	return m.pickerNormalHints(link)
}

func (m model) pickerContentWidth() int {
	if !m.ui.Picker.fullscreen {
		return m.dialogWidth()
	}
	// Wider than a dialog, but never the whole terminal: the metadata stays
	// beside the names instead of stranded a screen's width away.
	widest := ui.DialogContentWidth(0) + pickerFullMetaWidth
	if m.ui.Screen.Width <= 0 {
		return widest
	}
	return max(min(m.ui.Screen.Width-ui.FullPanelChromeWidth, widest), pickerFullMinWidth)
}

func (m model) pickerHintRows() int {
	// Measured with the longer label so the list keeps its height as the cursor
	// crosses the linked row.
	rendered := lipgloss.NewStyle().Width(m.pickerContentWidth()).Render(ui.RenderHints(m.pickerNormalHints("unlink")))
	return lipgloss.Height(rendered)
}

func (m model) pickerVisibleCount() int {
	total := len(m.ui.Picker.projects)
	if m.ui.Screen.Height <= 0 {
		return min(total, pickerFallbackVisible)
	}
	chrome := ui.PanelChromeHeight
	if m.ui.Picker.fullscreen {
		chrome = ui.FullPanelChromeHeight
	}
	ownRows := chrome + pickerOwnRows + m.pickerHintRows() + pickerScrollReserve
	return min(max(m.ui.Screen.Height-ownRows, pickerMinVisible), total)
}

func (m model) projectPickerView() string {
	contentWidth := m.pickerContentWidth()
	lines := m.pickerBody(contentWidth)
	if !m.ui.Picker.fullscreen {
		lines = append(lines, "", ui.RenderHints(m.pickerHints()))
		// Width or lipgloss leaves short rows unpadded and the list shows through.
		return ui.PanelStyle.Width(contentWidth + ui.PanelChromeWidth).Render(strings.Join(lines, "\n"))
	}

	// Nothing of the main view renders under a full-screen picker, so the status
	// line it normally shares takes the reserved blank row above the footer.
	status := ""
	if m.ui.Screen.StatusMsg != "" {
		status = ui.MutedStyle.Render(ansi.Truncate(m.ui.Screen.StatusMsg, contentWidth, ui.Ellipsis))
	}
	// With no frame nothing else wraps the hints, and a footer that soft-wraps in
	// the terminal takes rows the height budget never counted.
	footer := strings.Split(lipgloss.NewStyle().Width(contentWidth).Render(ui.RenderHints(m.pickerHints())), "\n")
	for ui.FullPanelChromeHeight+len(lines)+1+len(footer) < m.ui.Screen.Height {
		lines = append(lines, "")
	}
	lines = append(append(lines, status), footer...)
	return ui.FullPanelStyle.Render(strings.Join(lines, "\n"))
}

func (m model) pickerBody(contentWidth int) []string {
	p := &m.ui.Picker
	title := fmt.Sprintf("Projects (%d)", len(p.projects))
	// While filtering, the query prompt takes the pinned top row's place: New
	// Project isn't a filter target, so it's hidden until the filter is cleared.
	topLine := m.renderNewProjectLine(contentWidth)
	if p.filtering {
		topLine = m.renderPickerFilterLine(contentWidth)
	}
	lines := []string{ui.DialogTitleStyle.Render(title), "", topLine, ""}

	n := len(p.projects)
	start := p.scrollOffset
	end := min(start+m.pickerVisibleCount(), n)

	if p.filtering && n == 0 {
		lines = append(lines, ui.MutedStyle.Render("  no matches"))
	}
	if start > 0 {
		lines = append(lines, ui.DialogHintStyle.Render(scrollMoreAbove))
	}
	// Measured over the whole list, and over the unfiltered set while filtering,
	// so the columns hold still as it scrolls and as the query narrows it.
	source := p.projects
	if p.filtering {
		source = p.allProjects
	}
	cols := measurePickerColumns(source, contentWidth)
	linkedID := m.linkedProjectID()
	for i := start; i < end; i++ {
		isSelected := !p.onNew && !p.isAdding && i == p.selected
		lines = append(lines, m.renderPickerRow(i, isSelected, linkedID, cols, contentWidth))
	}
	if end < n {
		lines = append(lines, ui.DialogHintStyle.Render(scrollMoreBelow))
	}
	return lines
}

// 0 = column not drawn: nothing to put in it, or too narrow to keep.
type pickerColumns struct {
	open       int
	inProgress int
	edited     int
}

func (c pickerColumns) width() int {
	total := 0
	for _, w := range []int{c.open, c.inProgress, c.edited} {
		if w > 0 {
			total += pickerColumnGap + w
		}
	}
	return total
}

func measurePickerColumns(projects []domain.Project, contentWidth int) pickerColumns {
	var cols pickerColumns
	for _, p := range projects {
		open, inProgress := projectStats(p)
		cols.open = max(cols.open, ansi.StringWidth(openCell(open)))
		cols.inProgress = max(cols.inProgress, ansi.StringWidth(inProgressCell(inProgress)))
		cols.edited = max(cols.edited, ansi.StringWidth(FormatRelativeShort(p.UpdatedAt)))
	}
	for _, drop := range []*int{&cols.edited, &cols.inProgress, &cols.open} {
		if pickerRowPrefixWidth+pickerMinNameWidth+cols.width() <= contentWidth {
			break
		}
		*drop = 0
	}
	return cols
}

// Width of the "  " / "> " cursor gutter.
const pickerRowPrefixWidth = 2

func projectStats(p domain.Project) (open, inProgress int) {
	for _, c := range p.Categories {
		for _, t := range c.Tasks {
			if t.IsSeparator() {
				continue
			}
			switch t.Status {
			case domain.StatusCompleted, domain.StatusCancelled:
			case domain.StatusInProgress:
				open++
				inProgress++
			default:
				open++
			}
		}
	}
	return open, inProgress
}

func openCell(open int) string {
	return fmt.Sprintf("%d open", open)
}

func inProgressCell(inProgress int) string {
	if inProgress == 0 {
		return ""
	}
	return fmt.Sprintf("%d ▸", inProgress)
}

func pickerMetaCells(p domain.Project, cols pickerColumns, base lipgloss.Style, focused bool) string {
	open, inProgress := projectStats(p)

	type cell struct {
		text  string
		width int
		style lipgloss.Style
	}
	cells := []cell{
		{openCell(open), cols.open, ui.DialogHintStyle},
		{inProgressCell(inProgress), cols.inProgress, ui.StatusStyle(domain.StatusInProgress)},
		{FormatRelativeShort(p.UpdatedAt), cols.edited, ui.MutedStyle},
	}

	var out strings.Builder
	for _, c := range cells {
		if c.width == 0 {
			continue
		}
		style := c.style
		if focused {
			style = base
		}
		pad := c.width - ansi.StringWidth(c.text)
		out.WriteString(base.Render(strings.Repeat(" ", pickerColumnGap+max(pad, 0))))
		out.WriteString(style.Render(c.text))
	}
	return out.String()
}

func (m model) renderPickerRow(i int, isSelected bool, linkedID string, cols pickerColumns, contentWidth int) string {
	p := m.ui.Picker.projects[i]

	base, prefix, suffixStyle := lipgloss.NewStyle(), "  ", ui.DialogHintStyle
	if isSelected {
		base = ui.GetSelectedStyle(m.ui.Screen.WindowFocused)
		prefix, suffixStyle = "> ", base
	}
	var marks []string
	if p.ID == m.project.ID {
		marks = append(marks, "current")
	}
	if p.ID == linkedID {
		marks = append(marks, "linked")
	}
	suffix := ""
	if len(marks) > 0 {
		suffix = " (" + strings.Join(marks, ", ") + ")"
	}

	name, gap := pickerRowLayout(p.Name, suffix, cols.width(), contentWidth)
	displayName := base.Render(name)
	if !isSelected && m.ui.Picker.filtering && m.ui.Picker.query != "" {
		displayName = ui.HighlightMatches(name, m.ui.Picker.query, lipgloss.NewStyle(), ui.SearchMatchStyle)
	}
	return base.Render(prefix) + displayName + suffixStyle.Render(suffix) +
		base.Render(strings.Repeat(" ", gap)) + pickerMetaCells(p, cols, base, isSelected)
}

func (m model) renderNewProjectLine(contentWidth int) string {
	p := m.ui.Picker
	if p.isAdding {
		split := splitAtCursor(p.input.Value(), p.input.Position())
		return fmt.Sprintf("  %s %s%s%s",
			ui.SuccessStyle.Render("+"),
			split.left,
			ui.GetCursorStyle(m.ui.Screen.WindowFocused).Render(split.cursorCh),
			split.right,
		)
	}
	if p.onNew {
		return ui.GetSelectedStyle(m.ui.Screen.WindowFocused).Render(ui.PadTo("> + New Project", contentWidth))
	}
	// The green "+" reads as an affordance to act, not a disabled row.
	return "  " + ui.SuccessStyle.Render("+") + " New Project"
}

// renderPickerFilterLine draws the type-to-filter prompt that replaces the New
// Project row while filtering.
func (m model) renderPickerFilterLine(contentWidth int) string {
	return m.filterPromptRow(m.ui.Picker.filter, m.ui.Picker.query, len(m.ui.Picker.projects), contentWidth)
}

func pickerRowLayout(name, suffix string, metaWidth, contentWidth int) (string, int) {
	avail := max(contentWidth-pickerRowPrefixWidth-lipgloss.Width(suffix)-metaWidth, 1)
	name = ansi.Truncate(name, avail, ui.Ellipsis)
	return name, max(avail-lipgloss.Width(name), 0)
}
