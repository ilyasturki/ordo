package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"
	"github.com/charmbracelet/x/ansi"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

func renderProjectLine(name string, selected, focused, filtered, visualMode bool, statusText string, width int) string {
	prefix := "  "
	if selected {
		prefix = "> "
	}
	line := fmt.Sprintf("%s■ %s", prefix, name)
	if selected {
		if visualMode {
			line = ui.GetVisualSelectedStyle(focused).Render(line)
		} else {
			line = ui.GetSelectedStyle(focused).Render(line)
		}
	} else {
		line = ui.HeaderStyle.Render(line)
	}
	if filtered {
		line += " " + ui.FilterTagStyle.Render("[filtered]")
	}
	if visualMode {
		line += " " + ui.VisualTagStyle.Render("[visual]")
	}
	return withStatus(line, statusText, width)
}

// withStatus right-aligns statusText on line, truncating both to width.
func withStatus(line, statusText string, width int) string {
	if statusText == "" {
		if width > 0 {
			line = ansi.Truncate(line, width, "")
		}
		return line
	}

	suffix := ui.StatusLineStyle.Render(statusText)
	if width <= 0 {
		return line + " " + suffix
	}

	leftW := ansi.StringWidth(line)
	suffixW := ansi.StringWidth(suffix)
	if leftW+1+suffixW > width {
		return ansi.Truncate(line+" "+suffix, width, "")
	}
	padding := width - leftW - suffixW
	return line + strings.Repeat(" ", padding) + suffix
}

func (m model) renderEditProjectLine() string {
	prefix := "> " + ui.HeaderStyle.Render("■ ")
	cursorStyle := ui.GetCursorStyle(m.ui.Screen.WindowFocused)
	return m.withEditError(renderEditRows(m.editRows(projectPrefixWidth), projectPrefixWidth, prefix, ui.HeaderStyle, cursorStyle), projectPrefixWidth)
}

// withEditError adds the editor's error as one row under the edited text.
func (m model) withEditError(rows string, overhead int) string {
	if m.ui.Edit.err == "" {
		return rows
	}
	msg := truncateText(m.ui.Edit.err, safeWidth(m.ui.Screen.Width, overhead))
	return rows + "\n" + strings.Repeat(" ", overhead) + ui.ErrorStyle.Render(msg)
}

func renderCategoryLine(h categoryHeader, selected bool, width int, focused bool, visualMode bool, isCursor bool, searchQuery string, searchMatch lipgloss.Style) string {
	name, estimateMinutes, aggregateStatus, cut := h.name, h.estimate, h.status, h.cut
	prefix := "  " + strings.Repeat(" ", h.indent)
	if selected {
		prefix = "> " + strings.Repeat(" ", h.indent)
	}
	style := ui.CategoryStyle
	if selected {
		switch {
		case visualMode && isCursor:
			style = ui.GetVisualCursorStyle(focused)
		case visualMode:
			style = ui.GetVisualSelectedStyle(focused)
		default:
			style = ui.GetSelectedStyle(focused)
		}
	}
	if cut {
		style = ui.ApplyCut(style)
	}

	foldIndicator := "▼ "
	if h.folded {
		foldIndicator = "▶ "
	}

	statusBadge := ""
	statusBadgeText := ""
	if aggregateStatus != "" {
		statusBadgeText = " [" + statusIcon(aggregateStatus) + "]"
		switch {
		case selected && visualMode && isCursor:
			statusBadge = ui.GetVisualCursorStyle(focused).Render(statusBadgeText)
		case selected && visualMode:
			statusBadge = ui.GetVisualSelectedStyle(focused).Render(statusBadgeText)
		case selected:
			statusBadge = ui.GetSelectedStatusStyle(aggregateStatus, focused).Render(statusBadgeText)
		default:
			statusBadge = ui.StatusStyle(aggregateStatus).Render(statusBadgeText)
		}
	}

	progressBadge := ""
	if h.progress != "" {
		text := " " + h.progress
		switch {
		case selected && visualMode && isCursor:
			progressBadge = ui.GetVisualCursorStyle(focused).Render(text)
		case selected && visualMode:
			progressBadge = ui.GetVisualSelectedStyle(focused).Render(text)
		case selected:
			progressBadge = ui.GetSelectedStyle(focused).Render(text)
		default:
			progressBadge = ui.MutedStyle.Render(text)
		}
	}

	estimateBadge := ""
	estimateBadgeText := ""
	if estimateMinutes > 0 {
		estimateBadgeText = " ~" + FormatEstimate(estimateMinutes)
		switch {
		case selected && visualMode && isCursor:
			estimateBadge = ui.GetVisualCursorStyle(focused).Render(estimateBadgeText)
		case selected && visualMode:
			estimateBadge = ui.GetVisualSelectedStyle(focused).Render(estimateBadgeText)
		case selected:
			estimateBadge = ui.GetSelectedStyle(focused).Render(estimateBadgeText)
		default:
			estimateBadge = ui.MutedStyle.Render(estimateBadgeText)
		}
	}

	cutBadge := ""
	cutBadgeText := ""
	if cut {
		cutBadgeText = ui.CutMark
		if selected {
			cutBadge = style.Render(cutBadgeText)
		} else {
			cutBadge = ui.CutBadgeStyle.Render(cutBadgeText)
		}
	}

	suffix := cutBadge + statusBadge + progressBadge + estimateBadge

	if width <= 0 {
		return style.Render(prefix+foldIndicator) + ui.HighlightMatches(name, searchQuery, style, searchMatch) + suffix
	}

	available := categoryNameWidth(width, h)
	lines := ui.WrapClamped(name, available, ui.MaxLineRows, ui.FirstMatchIndex(name, searchQuery))
	indent := strings.Repeat(" ", prefixWidth+h.indent+categoryFoldWidth)

	var result []string
	for i, line := range lines {
		styledLine := ui.HighlightMatches(line, searchQuery, style, searchMatch)
		if i == 0 {
			result = append(result, style.Render(prefix+foldIndicator)+styledLine+suffix)
		} else {
			result = append(result, style.Render(indent)+styledLine)
		}
	}
	if selected && visualMode {
		for i, l := range result {
			result[i] = ui.PadToStyle(l, width, style)
		}
	}
	return strings.Join(result, "\n")
}

const categoryFoldWidth = 2 // "▼ "

// categoryNameWidth is shared by the renderer and the layout so both count the same rows.
func categoryNameWidth(width int, h categoryHeader) int {
	badges := 0
	if h.status != "" {
		badges += ansi.StringWidth(" [" + statusIcon(h.status) + "]")
	}
	if h.progress != "" {
		badges += ansi.StringWidth(" " + h.progress)
	}
	if h.estimate > 0 {
		badges += ansi.StringWidth(" ~" + FormatEstimate(h.estimate))
	}
	if h.cut {
		badges += ansi.StringWidth(ui.CutMark)
	}
	return safeWidth(width, prefixWidth+h.indent+categoryFoldWidth+badges)
}

func (m model) renderTaskLine(catIdx, taskIdx int, selected bool, width int, focused bool, visualMode bool, isCursor bool, cut bool) string {
	cfg := m.deps.CfgManager.Get()
	cat := m.project.Categories[catIdx]
	task := cat.Tasks[taskIdx]
	renderer := components.NewTaskLineRenderer(width, cfg.StatusDisplay, cfg.PriorityColor, focused).
		WithNesting(taskIndent(cat, taskIdx), taskProgress(cat, taskIdx, &m.ui.Fold)).
		WithVisualMode(visualMode).
		WithCursor(isCursor).
		WithCut(cut).
		WithDescriptionExpanded(m.ui.Screen.ExpandDescriptions).
		WithRowLimits(m.rowLimits().titleRows(task.ID), components.DescriptionPreviewLines)
	if q := m.searchQuery(); q != "" {
		renderer = renderer.WithSearch(q, m.searchMatchStyle(isCursor))
	}
	return renderer.Render(task, selected)
}

func (m model) renderTaskDescription(catIdx, taskIdx int, descriptionRows int, selected bool, width int, focused bool, visualMode bool, isCursor bool, cut bool) string {
	cfg := m.deps.CfgManager.Get()
	task := m.project.Categories[catIdx].Tasks[taskIdx]
	renderer := components.NewTaskLineRenderer(width, cfg.StatusDisplay, cfg.PriorityColor, focused).
		WithVisualMode(visualMode).
		WithCursor(isCursor).
		WithCut(cut).
		WithRowLimits(ui.MaxLineRows, descriptionRows)
	if q := m.searchQuery(); q != "" {
		renderer = renderer.WithSearch(q, m.searchMatchStyle(selected))
	}
	indent := taskTitleColumn(task, cfg.StatusDisplay, m.rowIndent(catIdx, taskIdx))
	return renderer.RenderDescription(task, indent, selected)
}

// taskTitleColumn is where the title starts, so the description's bar can sit flush with it.
func taskTitleColumn(task domain.Task, statusDisplay string, indent int) int {
	prefix := "  " + strings.Repeat(" ", indent)
	dotText := ui.TagSegmentText(task.TagColor)
	priorityIcon := ui.PriorityIcon(task.Priority)
	iconText := ""
	if priorityIcon != "" {
		iconText = priorityIcon + " "
	}
	statusText := statusLabel(task.Status, statusDisplay)
	return ansi.StringWidth(prefix + "[" + statusText + "] " + dotText + iconText)
}

func statusLabel(status, displayMode string) string {
	if displayMode == config.StatusDisplayIcons {
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

func formatStatus(status, displayMode string) string {
	return ui.StatusStyle(status).Render(statusLabel(status, displayMode))
}

func (m model) renderEditCategoryLine(indent int) string {
	prefix := "> " + strings.Repeat(" ", indent)
	cursorStyle := ui.GetCursorStyle(m.ui.Screen.WindowFocused)
	if m.ui.Edit.isAdding && m.ui.Edit.input.Value() == "" {
		placeholder := ui.MutedStyle.Render("Enter category name...")
		styledText := cursorStyle.Render(" ") + placeholder
		return prefixLine(styledText, m.ui.Screen.Width, prefixWidth+indent, prefix)
	}
	return m.withEditError(renderEditRows(m.editRows(prefixWidth+indent), prefixWidth+indent, prefix, ui.CategoryStyle, cursorStyle), prefixWidth+indent)
}

func (m model) renderEditTaskLine(task domain.Task, indent int) string {
	prefix := "> " + strings.Repeat(" ", indent)
	cfg := m.deps.CfgManager.Get()
	statusText := formatStatus(task.Status, cfg.StatusDisplay)
	titleStyle := ui.PriorityStyle(task.Priority, cfg.PriorityColor)
	iconStyle := ui.PriorityIconStyle(task.Priority, cfg.PriorityColor)
	icon := ui.PriorityIcon(task.Priority)
	iconPrefix := ""
	if icon != "" {
		iconPrefix = iconStyle.Render(icon) + " "
	}
	dotPrefix := ""
	dotPlain := ui.TagSegmentText(task.TagColor)
	if dotPlain != "" {
		dotStyle, _ := ui.TagDotStyle(task.TagColor, task.Status == domain.StatusCompleted || task.Status == domain.StatusCancelled)
		dotPrefix = dotStyle.Render(dotPlain)
	}
	prefixPart := fmt.Sprintf("%s[%s] %s%s", prefix, statusText, dotPrefix, iconPrefix)
	overhead := taskTitleColumn(task, cfg.StatusDisplay, indent)
	cursorStyle := ui.GetCursorStyle(m.ui.Screen.WindowFocused)
	if m.ui.Edit.isAdding && m.ui.Edit.input.Value() == "" {
		placeholder := ui.MutedStyle.Render("Enter task title...")
		styledText := cursorStyle.Render(" ") + placeholder
		return prefixLine(styledText, m.ui.Screen.Width, overhead, prefixPart)
	}
	return renderEditRows(m.editRows(overhead), overhead, prefixPart, titleStyle, cursorStyle)
}

// separatorRule is the heavy box-drawing glyph the divider is built from.
const separatorRule = "━"

// separatorLead is the short rule that precedes a labeled separator's text.
const separatorLead = separatorRule + separatorRule

// Columns left of the edited text: "> ■ " and "> ━━ ".
const (
	projectPrefixWidth   = 4
	separatorPrefixWidth = 5
)

// sepContentCap caps the divider to the content column so it stops short of the
// empty right side of the screen instead of spanning the whole terminal.
const sepContentCap = 44

// sepContentWidth caps the rule to the content column. Fixed rather than
// computed per-category so the divider reads at a consistent width everywhere.
// The result is always >= 1 (safeWidth floors at 1, and the avail<=0 path caps).
func sepContentWidth(avail int) int {
	if avail <= 0 || avail > sepContentCap {
		return sepContentCap
	}
	return avail
}

// separatorLayout is a divider row split into styleable segments. A bare or
// over-wide label collapses the whole row into left (label/right empty);
// otherwise left is the leading rule, label the text, right the trailing rule.
// It is the single source of the divider geometry, shared by the selected and
// unselected render paths. width is assumed >= 1 (see sepContentWidth).
type separatorLayout struct {
	left  string
	label string
	right string
}

func layoutSeparator(prefix, label string, width int) separatorLayout {
	if label == "" {
		return separatorLayout{left: prefix + strings.Repeat(separatorRule, width)}
	}
	used := ansi.StringWidth(separatorLead) + ansi.StringWidth(" "+label+" ")
	if used >= width {
		return separatorLayout{left: prefix + ansi.Truncate(separatorLead+" "+label, width, "…")}
	}
	trailing := strings.Repeat(separatorRule, width-used)
	return separatorLayout{left: prefix + separatorLead + " ", label: label, right: " " + trailing}
}

// renderSeparatorLine draws an in-category divider: a heavy rule capped to the
// content column. A labeled separator reads "━━ label ━━━━" with the label set
// off from the rule. When selected the whole row takes the selection band; in
// visual mode it takes the visual cursor or visual range band so a divider
// swept into a task selection reads as part of the block.
func (m model) renderSeparatorLine(label string, indent int, selected, focused, visualMode, isCursor, cut bool, width int, searchQuery string, searchMatch lipgloss.Style) string {
	prefix := "  " + strings.Repeat(" ", indent)
	if selected {
		prefix = "> " + strings.Repeat(" ", indent)
	}
	avail := 0
	if width > 0 {
		avail = safeWidth(width, prefixWidth+indent)
	}
	layout := layoutSeparator(prefix, strings.TrimSpace(label), sepContentWidth(avail))

	if selected {
		style := ui.GetSelectedStyle(focused)
		switch {
		case visualMode && isCursor:
			style = ui.GetVisualCursorStyle(focused)
		case visualMode:
			style = ui.GetVisualSelectedStyle(focused)
		}
		if cut {
			style = ui.ApplyCut(style)
			return style.Render(layout.left+layout.label+layout.right) + style.Render(ui.CutMark)
		}
		return style.Render(layout.left + layout.label + layout.right)
	}

	ruleStyle := ui.SeparatorStyle
	labelStyle := ui.HeaderStyle
	cutBadge := ""
	if cut {
		ruleStyle = ui.ApplyCut(ruleStyle)
		labelStyle = ui.ApplyCut(labelStyle)
		cutBadge = ui.CutBadgeStyle.Render(ui.CutMark)
	}
	if layout.label == "" {
		return ruleStyle.Render(layout.left) + cutBadge
	}
	// Labeled: heavy rule with the label emphasized so it anchors the row,
	// honoring any active search highlight (a no-op when the query is empty).
	styledLabel := ui.HighlightMatches(layout.label, searchQuery, labelStyle, searchMatch)
	return ruleStyle.Render(layout.left) + styledLabel + ruleStyle.Render(layout.right) + cutBadge
}

// renderEditSeparatorLine draws the inline editor for a separator's label,
// entered with Enter. An empty value shows a placeholder rather than a rule.
func (m model) renderEditSeparatorLine(indent int) string {
	styledPrefix := ui.SeparatorStyle.Render("> " + strings.Repeat(" ", indent) + separatorLead + " ")
	overhead := separatorPrefixWidth + indent
	cursorStyle := ui.GetCursorStyle(m.ui.Screen.WindowFocused)

	if m.ui.Edit.input.Value() == "" {
		placeholder := ui.MutedStyle.Render("Enter separator label...")
		styledText := cursorStyle.Render(" ") + placeholder
		return prefixLine(styledText, m.ui.Screen.Width, overhead, styledPrefix)
	}
	return renderEditRows(m.editRows(overhead), overhead, styledPrefix, ui.HeaderStyle, cursorStyle)
}

func (m model) statusText() string {
	if m.ui.Screen.StatusMsg != "" {
		return m.ui.Screen.StatusMsg
	}
	if m.ui.Modes.IsVisual() {
		positions := m.visualSelectedPositions()
		count := len(positions)
		noun := plural(count, "category", "categories")
		if m.ui.Visual.Kind != selection.FocusCategory {
			// Task-level: count the tasks operations will act on — descriptions
			// round up to their parent (a lone description still selects its
			// task) and a swept-in divider never inflates the number.
			count = countRealTasks(visualRoundUpPositions(positions))
			noun = plural(count, "task", "tasks")
		}
		return fmt.Sprintf("-- VISUAL -- %d %s selected", count, noun)
	}
	return ""
}

func truncateText(s string, max int) string {
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}
