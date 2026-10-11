package components

import (
	"fmt"
	"slices"
	"strings"
	"unicode/utf8"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

type TaskLineRenderer struct {
	width               int
	statusDisplay       string
	priorityColor       string
	focused             bool
	visualMode          bool
	isCursor            bool
	cut                 bool
	descriptionExpanded bool
	searchQuery         string
	searchMatch         lipgloss.Style
	titleRows           int
	descriptionRows     int
	indent              string
	progress            string
}

// Uncapped as a row limit lets a field take every row it wraps to.
const Uncapped = 0

func NewTaskLineRenderer(width int, statusDisplay, priorityColor string, focused bool) *TaskLineRenderer {
	return &TaskLineRenderer{
		width:           width,
		statusDisplay:   statusDisplay,
		priorityColor:   priorityColor,
		focused:         focused,
		titleRows:       ui.MaxLineRows,
		descriptionRows: DescriptionPreviewLines,
	}
}

func (r *TaskLineRenderer) WithRowLimits(title, description int) *TaskLineRenderer {
	r.titleRows, r.descriptionRows = title, description
	return r
}

func (r *TaskLineRenderer) WithNesting(indent int, progress string) *TaskLineRenderer {
	r.indent, r.progress = strings.Repeat(" ", indent), progress
	return r
}

func (r *TaskLineRenderer) WithVisualMode(v bool) *TaskLineRenderer {
	r.visualMode = v
	return r
}

func (r *TaskLineRenderer) WithCursor(c bool) *TaskLineRenderer {
	r.isCursor = c
	return r
}

func (r *TaskLineRenderer) WithCut(c bool) *TaskLineRenderer {
	r.cut = c
	return r
}

func (r *TaskLineRenderer) WithDescriptionExpanded(v bool) *TaskLineRenderer {
	r.descriptionExpanded = v
	return r
}

// WithSearch enables search-match highlighting: occurrences of query within the
// title/description are rendered with match instead of the base text style.
func (r *TaskLineRenderer) WithSearch(query string, match lipgloss.Style) *TaskLineRenderer {
	r.searchQuery = query
	r.searchMatch = match
	return r
}

// highlightLine styles one already-wrapped plain line with base, emphasizing any
// active search match within it. With no active query it is just
// base.Render(line).
func (r *TaskLineRenderer) highlightLine(line string, base lipgloss.Style) string {
	if r.searchQuery == "" {
		return base.Render(line)
	}
	return ui.HighlightMatches(line, r.searchQuery, base, r.searchMatch)
}

func (r *TaskLineRenderer) maybeCut(s lipgloss.Style) lipgloss.Style {
	if r.cut {
		return ui.ApplyCut(s)
	}
	return s
}

func (r *TaskLineRenderer) selectedStyle() lipgloss.Style {
	if r.visualMode {
		if r.isCursor {
			return r.maybeCut(ui.GetVisualCursorStyle(r.focused))
		}
		return r.maybeCut(ui.GetVisualSelectedStyle(r.focused))
	}
	return r.maybeCut(ui.GetSelectedStyle(r.focused))
}

func (r *TaskLineRenderer) Render(task domain.Task, selected bool) string {
	prefix := "  " + r.indent
	if selected {
		prefix = "> " + r.indent
	}
	priorityIcon := ui.PriorityIcon(task.Priority)

	if selected {
		return r.padToWidth(r.renderSelected(task, prefix, priorityIcon))
	}
	return r.renderUnselected(task, prefix, priorityIcon)
}

// padToWidth extends each rendered line to the full row width using the
// active selection style, so a visual-mode highlight reads as a solid band
// rather than stopping at the end of the text. Only active when the row
// participates in the visual range (r.visualMode); normal-mode selection
// keeps its current "text-width" highlight.
func (r *TaskLineRenderer) padToWidth(rendered string) string {
	if r.width <= 0 || !r.visualMode {
		return rendered
	}
	style := r.selectedStyle()
	lines := strings.Split(rendered, "\n")
	for i, l := range lines {
		lines[i] = ui.PadToStyle(l, r.width, style)
	}
	return strings.Join(lines, "\n")
}

const descriptionBar = "▎ "

// DescriptionPreviewLines is how many description lines an unfocused row shows
// before truncating to a "… +N more lines" tail row.
const DescriptionPreviewLines = 3

// wrapDescriptionLines returns the description body as the plain wrapped lines
// it renders as with `available` content columns (<= 0 disables wrapping, so
// each \n-separated paragraph stays one line).
func wrapDescriptionLines(description string, available int) []string {
	if description == "" {
		return nil
	}
	var lines []string
	for _, paragraph := range strings.Split(description, "\n") {
		if paragraph == "" || available <= 0 {
			lines = append(lines, paragraph)
			continue
		}
		lines = append(lines, strings.Split(ansi.Wrap(paragraph, available, ""), "\n")...)
	}
	return lines
}

// descriptionWindow caps only when that saves a row, since the tail takes one; focus is -1 or a line to keep in view.
func descriptionWindow(lines []string, maxRows, focus int) (int, int) {
	if maxRows == Uncapped || len(lines) <= maxRows+1 {
		return 0, len(lines)
	}
	start := 0
	if focus >= maxRows {
		start = min(focus-1, len(lines)-maxRows)
	}
	return start, start + maxRows
}

// descriptionAvailable returns the content columns left for description text
// after the title-column indent and the blockquote bar (0 = no wrapping).
func descriptionAvailable(width, indent int) int {
	available := width - indent - ansi.StringWidth(descriptionBar)
	if width <= 0 || available < 1 {
		return 0
	}
	return available
}

// DescriptionHeight must match RenderDescription, whose window size a search match never changes.
func DescriptionHeight(description string, width, indent, maxRows int) int {
	lines := wrapDescriptionLines(description, descriptionAvailable(width, indent))
	start, end := descriptionWindow(lines, maxRows, -1)
	if end-start < len(lines) {
		return end - start + 1
	}
	return len(lines)
}

// RenderDescription renders a task description as a standalone, focusable block
// with a blockquote-style left bar. The bar sits at `indent` columns (aligned
// with the title on the task row) and text follows after it. The text mirrors
// the title's color treatment (priority color, faint when completed/cancelled)
// with italic on top so descriptions still read as a distinct element.
func (r *TaskLineRenderer) RenderDescription(task domain.Task, indent int, selected bool) string {
	description := task.Description
	if description == "" {
		return ""
	}
	indentStr := strings.Repeat(" ", indent)
	available := descriptionAvailable(r.width, indent)

	textStyle := ui.TaskTitleStyle(task.Priority, task.Status, r.priorityColor).Italic(true)
	barStyle := ui.DescriptionBarStyle
	tailStyle := ui.MutedStyle.Italic(true)
	if r.cut {
		textStyle = ui.ApplyCut(textStyle)
		barStyle = ui.ApplyCut(barStyle)
		tailStyle = ui.ApplyCut(tailStyle)
	}

	// When selected, every visual line gets padded to the full row width so the
	// highlight rectangle is uniform — otherwise blank-line indents and short
	// paragraph tails produce a ragged selection band.
	shouldPad := selected && r.width > 0

	lines := wrapDescriptionLines(description, available)
	start, end := descriptionWindow(lines, r.descriptionRows, r.firstMatchLine(lines))
	shown := slices.Clone(lines[start:end])
	if start > 0 {
		if available > 0 {
			shown[0] = ansi.Truncate(shown[0], available-1, "")
		}
		shown[0] = ui.Ellipsis + shown[0]
	}
	out := make([]string, 0, len(shown)+1)
	for _, l := range shown {
		out = append(out, r.styleDescriptionLine(indentStr, l, textStyle, barStyle, selected, shouldPad))
	}
	if len(shown) < len(lines) {
		tail := fmt.Sprintf("… +%d more lines", len(lines)-end)
		if end == len(lines) {
			tail = fmt.Sprintf("… %d lines above", start)
		}
		if r.isCursor {
			tail += " · gi full view"
		}
		out = append(out, r.styleDescriptionLine(indentStr, fitWidth(tail, available), tailStyle, barStyle, selected, shouldPad))
	}
	return strings.Join(out, "\n")
}

// firstMatchLine is the index of the first line holding a search match, or -1.
func (r *TaskLineRenderer) firstMatchLine(lines []string) int {
	if r.searchQuery == "" {
		return -1
	}
	for i, l := range lines {
		if ui.FirstMatchIndex(l, r.searchQuery) >= 0 {
			return i
		}
	}
	return -1
}

// fitWidth treats width <= 0 as no wrapping, so no limit.
func fitWidth(s string, width int) string {
	if width <= 0 {
		return s
	}
	return ansi.Truncate(s, width, "")
}

func (r *TaskLineRenderer) styleDescriptionLine(indentStr, text string, textStyle, barStyle lipgloss.Style, selected, shouldPad bool) string {
	if selected {
		sel := r.selectedStyle()
		prefix := indentStr + descriptionBar
		// Render the bar and text separately so a search match keeps its own
		// styling inside the selection band; pad with the band style to width.
		rendered := sel.Render(prefix) + r.highlightLine(text, sel)
		if shouldPad {
			rendered = ui.PadToStyle(rendered, r.width, sel)
		}
		return rendered
	}
	return indentStr + barStyle.Render(descriptionBar) + r.highlightLine(text, textStyle)
}

func (r *TaskLineRenderer) renderUnselected(task domain.Task, prefix, priorityIcon string) string {
	status := r.formatStatus(task.Status, false)
	finished := task.Status == domain.StatusCompleted || task.Status == domain.StatusCancelled
	iconStyled := ""
	if priorityIcon != "" {
		iconStyle := ui.PriorityIconStyle(task.Priority, r.priorityColor)
		if finished {
			iconStyle = iconStyle.Faint(true)
		}
		iconStyle = r.maybeCut(iconStyle)
		iconStyled = iconStyle.Render(priorityIcon) + " "
	}
	tagStyled := r.tagSegment(task, finished, false)
	leading := tagStyled + iconStyled
	descMarker := r.formatDescriptionBadge(task.Description, false)
	estimate := r.formatEstimateBadge(task.EstimateMinutes, false)
	cutBadge, _ := r.formatCutBadge(false)
	suffix := cutBadge + r.formatProgressBadge(false) + descMarker + estimate
	titleStyle := r.maybeCut(ui.TaskTitleStyle(task.Priority, task.Status, r.priorityColor))
	prefixPart := fmt.Sprintf("%s[%s] ", prefix, status)

	if r.width <= 0 {
		return prefixPart + leading + r.highlightLine(task.Title, titleStyle) + suffix
	}

	indent := strings.Repeat(" ", ansi.StringWidth(prefixPart))
	return r.wrapContentWithSuffix(task, prefixPart, leading, indent, titleStyle, suffix)
}

// tagSegment renders the leading tag mark and its trailing space, or "" when
// untagged. On a highlighted row the mark becomes a reverse color block so it
// joins the selection bar; finished dims it on unselected rows only.
func (r *TaskLineRenderer) tagSegment(task domain.Task, finished, selected bool) string {
	seg := ui.TagSegmentText(task.TagColor)
	if seg == "" {
		return ""
	}
	if selected {
		style, _ := ui.TagBlockStyle(task.TagColor)
		return r.maybeCut(style).Render(seg)
	}
	style, _ := ui.TagDotStyle(task.TagColor, finished)
	return r.maybeCut(style).Render(seg)
}

func (r *TaskLineRenderer) renderSelected(task domain.Task, prefix, priorityIcon string) string {
	statusText := r.statusLabel(task.Status)
	selectedStyle := r.selectedStyle()
	priorityStyle := selectedStyle
	statusStyle := selectedStyle

	iconStyled := ""
	if priorityIcon != "" {
		// Keep the priority color visible on the selection bar by rendering the
		// icon as a reverse color block, the same treatment the tag dot gets in
		// tagSegment; fall back to the plain selection style when icon color is
		// off so the icon still sits in the band.
		iconStyle, ok := ui.PriorityIconBlockStyle(task.Priority, r.priorityColor)
		if !ok {
			iconStyle = selectedStyle
		}
		iconStyled = r.maybeCut(iconStyle).Render(priorityIcon + " ")
	}
	finished := task.Status == domain.StatusCompleted || task.Status == domain.StatusCancelled
	tagStyled := r.tagSegment(task, finished, true)
	leading := tagStyled + iconStyled

	descMarker := r.formatDescriptionBadge(task.Description, true)
	estimate := r.formatEstimateBadge(task.EstimateMinutes, true)
	cutBadge, _ := r.formatCutBadge(true)
	suffix := cutBadge + r.formatProgressBadge(true) + descMarker + estimate

	prefixPart := selectedStyle.Render(prefix+"[") +
		statusStyle.Render(statusText) +
		selectedStyle.Render("] ")

	if r.width <= 0 {
		return prefixPart + leading + r.highlightLine(task.Title, priorityStyle) + suffix
	}

	indent := r.selectedStyle().Render(strings.Repeat(" ", ansi.StringWidth(prefix+"["+statusText+"] ")))
	return r.wrapContentWithSuffix(task, prefixPart, leading, indent, priorityStyle, suffix)
}

// titleWidth reserves the trailing badges on every row although only the last carries them.
func (r *TaskLineRenderer) titleWidth(task domain.Task) int {
	lead := "  " + r.indent + "[" + r.statusLabel(task.Status) + "] " + ui.TagSegmentText(task.TagColor)
	if icon := ui.PriorityIcon(task.Priority); icon != "" {
		lead += icon + " "
	}
	_, cutText := r.formatCutBadge(false)
	badges := cutText + r.progressBadgeText() + r.descriptionBadgeText(task.Description) + r.estimateBadgeText(task.EstimateMinutes)
	return safeWidth(r.width, ansi.StringWidth(lead+badges))
}

// TitleRows must match the rows Render draws.
func (r *TaskLineRenderer) TitleRows(task domain.Task) int {
	if r.width <= 0 {
		return 1
	}
	return ui.CountClamped(task.Title, r.titleWidth(task), r.maxTitleRows(task.Title))
}

func (r *TaskLineRenderer) maxTitleRows(title string) int {
	if r.titleRows == Uncapped {
		return utf8.RuneCountInString(title) + 1 // every row holds a rune at least
	}
	return r.titleRows
}

// wrapContentWithSuffix lays the title out behind prefixPart, continuation rows
// behind indent, with suffix on the last row.
func (r *TaskLineRenderer) wrapContentWithSuffix(task domain.Task, prefixPart, leading, indent string, titleStyle lipgloss.Style, suffix string) string {
	wrapLines := r.clampTitle(task.Title, r.titleWidth(task))
	last := len(wrapLines) - 1

	result := make([]string, len(wrapLines))
	for i, line := range wrapLines {
		rendered := indent + r.highlightLine(line, titleStyle)
		if i == 0 {
			rendered = prefixPart + leading + r.highlightLine(line, titleStyle)
		}
		if i == last {
			rendered += suffix
		}
		result[i] = rendered
	}
	return strings.Join(result, "\n")
}

func (r *TaskLineRenderer) formatStatus(status string, selected bool) string {
	label := r.statusLabel(status)
	if selected {
		return label
	}
	return ui.StatusStyle(status).Render(label)
}

func (r *TaskLineRenderer) statusLabel(status string) string {
	if r.statusDisplay == "icons" {
		return statusIcon(status)
	}
	switch status {
	case domain.StatusInProgress:
		return " progress"
	case domain.StatusCompleted:
		return "completed"
	case domain.StatusCancelled:
		return "cancelled"
	default:
		return "  todo   "
	}
}

func statusIcon(status string) string {
	switch status {
	case domain.StatusInProgress:
		return "/"
	case domain.StatusCompleted:
		return "x"
	case domain.StatusCancelled:
		return "-"
	default:
		return " "
	}
}

// clampTitle wraps a title into the rows a list row may occupy, with the window
// aimed at the search match so a hit past the cap is still visible.
func (r *TaskLineRenderer) clampTitle(title string, available int) []string {
	return ui.WrapClamped(title, available, r.maxTitleRows(title), ui.FirstMatchIndex(title, r.searchQuery))
}

func safeWidth(totalWidth, overhead int) int {
	available := totalWidth - overhead
	if available < 1 {
		return 1
	}
	return available
}

func (r *TaskLineRenderer) formatEstimateBadge(minutes int, selected bool) string {
	text := r.estimateBadgeText(minutes)
	if text == "" {
		return ""
	}
	if selected {
		return r.selectedStyle().Render(text)
	}
	return ui.MutedStyle.Render(text)
}

func (r *TaskLineRenderer) estimateBadgeText(minutes int) string {
	if minutes == 0 {
		return ""
	}
	return " ~" + formatEstimateShort(minutes)
}

func (r *TaskLineRenderer) progressBadgeText() string {
	if r.progress == "" {
		return ""
	}
	return " " + r.progress
}

func (r *TaskLineRenderer) formatProgressBadge(selected bool) string {
	text := r.progressBadgeText()
	if text == "" {
		return ""
	}
	if selected {
		return r.selectedStyle().Render(text)
	}
	return ui.MutedStyle.Render(text)
}

func (r *TaskLineRenderer) formatDescriptionBadge(description string, selected bool) string {
	text := r.descriptionBadgeText(description)
	if text == "" {
		return ""
	}
	if selected {
		return r.selectedStyle().Render(text)
	}
	return ui.MutedStyle.Render(text)
}

func (r *TaskLineRenderer) descriptionBadgeText(description string) string {
	if description == "" || r.descriptionExpanded {
		return ""
	}
	return " ¶"
}

func (r *TaskLineRenderer) formatCutBadge(selected bool) (string, string) {
	if !r.cut {
		return "", ""
	}
	text := ui.CutMark
	if selected {
		return r.selectedStyle().Render(text), text
	}
	return ui.CutBadgeStyle.Render(text), text
}

func formatEstimateShort(minutes int) string {
	if minutes < 60 {
		return fmt.Sprintf("%dm", minutes)
	}
	hours := minutes / 60
	if hours < 8 {
		return fmt.Sprintf("%dh", hours)
	}
	days := hours / 8
	return fmt.Sprintf("%dd", days)
}
