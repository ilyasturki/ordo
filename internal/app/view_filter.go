package app

import (
	"fmt"
	"strings"

	"charm.land/lipgloss/v2"

	"github.com/ilyasturki/ordo/internal/config"
	"github.com/ilyasturki/ordo/internal/ui"
)

var filterSubHints = []ui.Hint{
	{Key: "space", Label: "toggle"},
	{Key: "esc", Label: "back"},
	{Key: "f", Label: "close"},
}

func (m model) filterView() string {
	switch m.ui.Filter.View() {
	case FilterViewStatus:
		return m.filterStatusView()
	case FilterViewPriority:
		return m.filterPriorityView()
	case FilterViewCategory:
		return m.filterCategoryView()
	case FilterViewTag:
		return m.filterTagView()
	default:
		return m.filterHubView()
	}
}

func (m model) filterHubView() string {
	rows := []struct {
		label string
		count int
	}{
		{"Status", m.ui.Filter.StatusCount()},
		{"Priority", m.ui.Filter.PriorityCount()},
		{"Category", m.ui.Filter.CategoryCount()},
		{"Tag", m.ui.Filter.TagCount()},
	}

	title := ui.DialogTitleStyle.Render("Filter Tasks")
	if m.ui.Filter.IsOff() && m.ui.Filter.HasValues() {
		title += ui.MutedStyle.Render(" · off, a change turns it on")
	}
	lines := []string{title, ""}
	selected := m.ui.Filter.HubSelected()
	for i, row := range rows {
		marker := "  "
		if i == selected {
			marker = "> "
		}
		summary := ui.MutedStyle.Render("[—]")
		if row.count > 0 {
			summary = fmt.Sprintf("[%d active]", row.count)
		}
		line := fmt.Sprintf("%s%-10s %s", marker, row.label, summary)
		if i == selected {
			line = ui.SelectedStyle.Render(line)
		}
		lines = append(lines, line)
	}

	lines = append(lines, "")
	const clearAllLabel = "Clear all filters"
	var clearLine string
	switch {
	case selected == FilterHubClearAll:
		clearLine = ui.SelectedStyle.Render("> " + clearAllLabel)
	case !m.ui.Filter.HasValues():
		clearLine = "  " + ui.MutedStyle.Render(clearAllLabel)
	default:
		clearLine = "  " + clearAllLabel
	}
	lines = append(lines, clearLine)

	lines = append(lines, "", ui.RenderHints([]ui.Hint{{Key: "enter", Label: "open"}, {Key: "q/esc/f", Label: "close"}}))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

func (m model) filterStatusView() string {
	lines := []string{ui.DialogTitleStyle.Render("Filter · Status"), ""}
	for i, status := range filterStatuses {
		selected := i == m.ui.Filter.Selected()
		lines = append(lines, renderCheckboxLine(
			m.filterStatusLabel(status, selected),
			m.ui.Filter.IsStatusEnabled(status),
			selected,
		))
	}
	lines = append(lines, "", ui.RenderHints(filterSubHints))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

func (m model) filterPriorityView() string {
	lines := []string{ui.DialogTitleStyle.Render("Filter · Priority"), ""}
	for i, p := range filterPriorities {
		selected := i == m.ui.Filter.Selected()
		lines = append(lines, renderCheckboxLine(
			m.filterPriorityLabel(p, selected),
			m.ui.Filter.IsPriorityEnabled(p),
			selected,
		))
	}
	lines = append(lines, "", ui.RenderHints(filterSubHints))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

func (m model) filterCategoryView() string {
	lines := []string{ui.DialogTitleStyle.Render("Filter · Category"), ""}
	if len(m.project.Categories) == 0 {
		lines = append(lines, ui.MutedStyle.Render("  (no categories)"))
	} else {
		for i, cat := range m.project.Categories {
			label := cat.Name
			if cat.ParentID != "" {
				label = strings.Repeat(" ", nestIndent) + label
			}
			lines = append(lines, renderCheckboxLine(
				label,
				m.ui.Filter.IsCategoryEnabled(cat.ID),
				i == m.ui.Filter.Selected(),
			))
		}
	}
	lines = append(lines, "", ui.RenderHints(filterSubHints))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

func (m model) filterTagView() string {
	lines := []string{ui.DialogTitleStyle.Render("Filter · Tag"), ""}
	for i, color := range filterTagColors {
		lines = append(lines, renderCheckboxLine(
			formatTagFilterLabel(color),
			m.ui.Filter.IsTagEnabled(color),
			i == m.ui.Filter.Selected(),
		))
	}
	lines = append(lines, "", ui.RenderHints(filterSubHints))
	return m.dialogStyle().Render(strings.Join(lines, "\n"))
}

// filterStatusLabel leads with the mark a row draws for status; a selected row keeps it unstyled, as the list does.
func (m model) filterStatusLabel(status string, selected bool) string {
	label := formatStatusLabel(status)
	style := ui.StatusStyle(status)
	if selected {
		style = lipgloss.NewStyle()
	}
	if m.deps.CfgManager.Get().StatusDisplay != config.StatusDisplayIcons {
		return style.Render(label)
	}
	return style.Render(statusIcon(status)) + " " + label
}

func (m model) filterPriorityLabel(priority string, selected bool) string {
	icon := ui.PriorityIcon(priority)
	if icon == "" {
		icon = " "
	} else if !selected {
		icon = ui.PriorityIconStyle(priority, m.deps.CfgManager.Get().PriorityColor).Render(icon)
	}
	return icon + " " + formatPriorityLabel(priority)
}

// formatTagFilterLabel renders a tag filter row: a colored dot plus the color
// name, or a plain "untagged" for the empty bucket.
func formatTagFilterLabel(color string) string {
	if color == "" {
		return "untagged"
	}
	style, _ := ui.TagDotStyle(color, false)
	return style.Render(ui.TagGlyph) + " " + tagColorName(color)
}

func renderCheckboxLine(label string, enabled, selected bool) string {
	marker := "  "
	if selected {
		marker = "> "
	}
	checkbox := "[ ]"
	if enabled {
		checkbox = "[x]"
	}
	line := fmt.Sprintf("%s%s %s", marker, checkbox, label)
	if selected {
		line = ui.SelectedStyle.Render(line)
	}
	return line
}
