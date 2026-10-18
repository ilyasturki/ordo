package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/x/ansi"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/attachments"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

func (m model) infoView() string {
	lines := m.infoLines()
	if len(lines) == 0 {
		return ""
	}

	// The dialog holds the full text of fields the list only shows clamped, so
	// its body is unbounded: a long title alone can outgrow the terminal.
	// Without a window, placeOverlay silently drops whatever falls past the last
	// screen row — including the hint that says how to close the dialog.
	height := m.infoViewportHeight()
	start := min(m.ui.Info.ScrollOffset, m.infoMaxScroll())
	end := min(start+height, len(lines))

	var out []string
	if start > 0 {
		out = append(out, ui.MutedStyle.Render(scrollMoreAbove))
	}
	out = append(out, lines[start:end]...)
	if end < len(lines) {
		out = append(out, ui.MutedStyle.Render(scrollMoreBelow))
	}
	out = append(out, "", ui.RenderHints([]ui.Hint{{Key: "esc/q", Label: "close"}}))
	return m.dialogStyle().Render(strings.Join(out, "\n"))
}

// infoLines builds the dialog body for whatever the cursor is on. Reported
// separately from infoView so scrolling can measure the body without rendering.
func (m model) infoLines() []string {
	pos, ok := m.selectedPosition()
	if !ok {
		return nil
	}
	switch pos.Kind {
	case selection.FocusProject:
		return m.projectInfoLines()
	case selection.FocusCategory:
		return m.categoryInfoLines(pos.CategoryIndex)
	case selection.FocusSeparator:
		return m.separatorInfoLines(pos.CategoryIndex, pos.TaskIndex)
	case selection.FocusTask, selection.FocusDescription:
		return m.taskInfoLines(pos.CategoryIndex, pos.TaskIndex)
	}
	return nil
}

// infoViewportHeight is how many body rows the dialog can show. Its own rows
// are the two scroll indicators, the blank above the hints, and the hints.
func (m model) infoViewportHeight() int {
	return ui.DialogBodyHeight(m.ui.Screen.Height, 4, 5)
}

// infoMaxScroll is the largest offset that still fills the window.
func (m model) infoMaxScroll() int {
	return max(len(m.infoLines())-m.infoViewportHeight(), 0)
}

func (m model) taskInfoLines(catIdx, taskIdx int) []string {
	task := m.project.Categories[catIdx].Tasks[taskIdx]
	category := m.project.Categories[catIdx]

	lines := []string{
		ui.DialogTitleStyle.Render("Task Info"),
		"",
	}

	infoMaxWidth := m.dialogWidth()
	const titleLabel = "Title:    "
	labelWidth := len(titleLabel)
	available := infoMaxWidth - labelWidth
	wrapped := ansi.Wrap(task.Title, available, "")
	titleLines := strings.Split(wrapped, "\n")
	indent := strings.Repeat(" ", labelWidth)
	for i, line := range titleLines {
		if i == 0 {
			lines = append(lines, titleLabel+line)
		} else {
			lines = append(lines, indent+line)
		}
	}

	lines = append(lines,
		fmt.Sprintf("Status:   %s", formatStatusLabel(task.Status)),
		fmt.Sprintf("Priority: %s", formatPriorityLabel(task.Priority)),
		fmt.Sprintf("Tag:      %s", formatTagInfo(task)),
		fmt.Sprintf("Estimate: %s", domain.EstimateLabel(task.EstimateMinutes)),
		fmt.Sprintf("Category: %s", m.infoValue(m.project.CategoryPath(catIdx))),
	)
	if parent := domain.ParentIndex(category.Tasks, taskIdx); parent >= 0 {
		lines = append(lines, fmt.Sprintf("Parent:   %s", m.infoValue(category.Tasks[parent].Title)))
	}
	if done, total := category.ChildProgress(taskIdx); total > 0 {
		lines = append(lines, fmt.Sprintf("Subtasks: %d of %d done", done, total))
	}
	lines = append(lines,
		"",
		fmt.Sprintf("Created:  %s", FormatDateWithRelative(task.CreatedAt)),
		fmt.Sprintf("Updated:  %s", FormatDateWithRelative(task.UpdatedAt)),
	)

	if task.CompletionDate != "" {
		lines = append(lines, fmt.Sprintf("Completed: %s", FormatDateWithRelative(task.CompletionDate)))
	}

	if task.Description != "" {
		lines = append(lines, "", ui.DialogTitleStyle.Render("Description"), "")
		descIndent := strings.Repeat(" ", 2)
		for _, srcLine := range strings.Split(attachments.Display(task.Description), "\n") {
			if strings.TrimSpace(srcLine) == "" {
				lines = append(lines, "")
				continue
			}
			wrapped := ansi.Wrap(srcLine, infoMaxWidth-len(descIndent), "")
			for _, wline := range strings.Split(wrapped, "\n") {
				lines = append(lines, descIndent+wline)
			}
		}
	}

	if images := m.taskImages(task); len(images) > 0 {
		lines = append(lines, "", ui.DialogTitleStyle.Render("Images"), "")
		for _, img := range images {
			lines = append(lines, "  "+truncateText(img.ref.Label(), max(infoMaxWidth-2, 1)))
			lines = append(lines, imageInfoLines(img, infoMaxWidth-4)...)
		}
	}

	return lines
}

func imageInfoLines(img taskImage, width int) []string {
	const indent = "    "
	info, err := attachments.Describe(img.path)
	if err != nil {
		missing := "missing"
		if img.ref.Internal() {
			missing = "missing: not synced yet"
		}
		return []string{indent + ui.MutedStyle.Render(missing)}
	}
	var lines []string
	for _, l := range strings.Split(ansi.Wrap(img.path, max(width, 1), "/"), "\n") {
		lines = append(lines, indent+ui.MutedStyle.Render(l))
	}
	detail := attachments.FormatSize(info.Size)
	if info.Width > 0 {
		detail += fmt.Sprintf(" · %d×%d", info.Width, info.Height)
	}
	return append(lines, indent+detail)
}

// infoValue keeps a value behind a 10-column label on one row: the dialog's window counts rows, not wraps.
func (m model) infoValue(s string) string {
	return truncateText(s, max(m.dialogWidth()-len("Category: ")-3, 1))
}

func (m model) separatorInfoLines(catIdx, taskIdx int) []string {
	sep := m.project.Categories[catIdx].Tasks[taskIdx]

	label := sep.Title
	if label == "" {
		label = ui.MutedStyle.Render("(none)")
	}

	return []string{
		ui.DialogTitleStyle.Render("Separator Info"),
		"",
		fmt.Sprintf("Label:    %s", label),
		fmt.Sprintf("Category: %s", m.infoValue(m.project.CategoryPath(catIdx))),
		"",
		fmt.Sprintf("Created:  %s", FormatDateWithRelative(sep.CreatedAt)),
		fmt.Sprintf("Updated:  %s", FormatDateWithRelative(sep.UpdatedAt)),
	}
}

func (m model) categoryInfoLines(catIdx int) []string {
	category := m.project.Categories[catIdx]
	counts := m.project.TreeStatusCounts(catIdx)

	lines := []string{
		ui.DialogTitleStyle.Render("Category Info"),
		"",
		fmt.Sprintf("Name:     %s", category.Name),
	}
	if parent := domain.ParentIndex(m.project.Categories, catIdx); parent >= 0 {
		lines = append(lines, fmt.Sprintf("Parent:   %s", m.infoValue(m.project.Categories[parent].Name)))
	}
	if subs := domain.ChildCount(m.project.Categories, catIdx); subs > 0 {
		lines = append(lines, fmt.Sprintf("Sub-categories: %d", subs))
	}
	lines = append(lines,
		fmt.Sprintf("Estimate: %s", domain.EstimateLabel(category.EstimateMinutes)),
		fmt.Sprintf("Created:  %s", FormatDateWithRelative(category.CreatedAt)),
	)

	if category.UpdatedAt != "" {
		lines = append(lines, fmt.Sprintf("Updated:  %s", FormatDateWithRelative(category.UpdatedAt)))
	}

	lines = append(lines,
		"",
		fmt.Sprintf("Total Tasks: %d", m.project.TreeTaskCount(catIdx)),
		"",
	)
	lines = append(lines, statusBreakdownLines(counts)...)

	return lines
}

func (m model) projectInfoLines() []string {
	counts := m.project.StatusCounts()
	lines := []string{
		ui.DialogTitleStyle.Render("Project Info"),
		"",
		fmt.Sprintf("Name:       %s", m.project.Name),
		fmt.Sprintf("Created:    %s", FormatDateWithRelative(m.project.CreatedAt)),
		fmt.Sprintf("Updated:    %s", FormatDateWithRelative(m.project.UpdatedAt)),
		"",
		fmt.Sprintf("Categories: %d", len(m.project.Categories)),
		fmt.Sprintf("Total Tasks: %d", counts.Total()),
		"",
	}
	lines = append(lines, statusBreakdownLines(counts)...)
	return lines
}

func statusBreakdownLines(counts domain.StatusCounts) []string {
	return []string{
		"Task Breakdown:",
		fmt.Sprintf("  Todo:        %d", counts.Todo),
		fmt.Sprintf("  In Progress: %d", counts.InProgress),
		fmt.Sprintf("  Completed:   %d", counts.Completed),
		fmt.Sprintf("  Cancelled:   %d", counts.Cancelled),
	}
}

func formatStatusLabel(status string) string {
	switch status {
	case domain.StatusTodo:
		return "Todo"
	case domain.StatusInProgress:
		return "In Progress"
	case domain.StatusCompleted:
		return "Completed"
	case domain.StatusCancelled:
		return "Cancelled"
	default:
		return status
	}
}

// formatTagInfo renders a task's tag for the info modal: "None" when untagged,
// otherwise the colored mark and its name.
func formatTagInfo(task domain.Task) string {
	style, ok := ui.TagDotStyle(task.TagColor, false)
	if !ok {
		return "None"
	}
	return style.Render(ui.TagGlyph(task.TagColor)) + " " + tagColorName(task.TagColor)
}

func formatPriorityLabel(priority string) string {
	switch priority {
	case domain.PriorityCritical:
		return "Critical"
	case domain.PriorityHigh:
		return "High"
	case domain.PriorityMedium:
		return "Medium"
	case domain.PriorityLow:
		return "Low"
	case domain.PriorityTrivial:
		return "Trivial"
	case "":
		return "None"
	default:
		return priority
	}
}
