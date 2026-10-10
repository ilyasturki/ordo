package app

import (
	"fmt"
	"strings"

	"phasionary/internal/app/selection"
	"phasionary/internal/ui"
)

func (m model) confirmDeleteView() string {
	switch m.ui.ConfirmDelete.Kind {
	case ConfirmDeleteProject:
		return m.confirmDeleteProjectView()
	case ConfirmDeleteVisualRange:
		return m.confirmDeleteVisualRangeView()
	}

	position, ok := m.selectedPosition()
	if !ok || position.Kind == selection.FocusProject {
		return ""
	}
	var message string
	if position.Kind == selection.FocusTask {
		task := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
		message = fmt.Sprintf("Delete task %q?", truncateText(task.Title, 30))
	} else {
		cat := m.project.Categories[position.CategoryIndex]
		message = fmt.Sprintf("Delete category %q and %d tasks?", truncateText(cat.Name, 30), len(cat.Tasks))
	}
	lines := []string{
		message,
		"",
		ui.RenderHints([]ui.Hint{{Key: "y/enter", Label: "confirm"}, {Key: "n/esc", Label: "cancel"}}),
	}
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}

func (m model) confirmDeleteVisualRangeView() string {
	var message string
	if n := len(m.ui.ConfirmDelete.CategoryIDs); n > 0 {
		nested := 0
		idSet := make(map[string]struct{}, n)
		for _, id := range m.ui.ConfirmDelete.CategoryIDs {
			idSet[id] = struct{}{}
		}
		for _, c := range m.project.Categories {
			if _, drop := idSet[c.ID]; drop {
				nested += len(c.Tasks)
			}
		}
		message = fmt.Sprintf("Delete %d categor%s and %d task(s)?", n, plural(n, "y", "ies"), nested)
	} else {
		n := len(m.ui.ConfirmDelete.TaskIDs)
		message = fmt.Sprintf("Delete %d task(s)?", n)
	}
	lines := []string{
		message,
		"",
		ui.RenderHints([]ui.Hint{{Key: "y/enter", Label: "confirm"}, {Key: "n/esc", Label: "cancel"}}),
	}
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}

func (m model) confirmMergeView() string {
	pos, ok := m.selectedPosition()
	dst, err := m.project.FindCategoryByID(m.ui.Edit.mergeInto)
	if !ok || err != nil || pos.Kind != selection.FocusCategory {
		return ""
	}
	target := truncateText(m.project.Categories[dst].Name, 30)
	src := m.project.Categories[pos.CategoryIndex]
	var lines []string
	switch n := src.TaskCount(); {
	case m.ui.Edit.isAdding:
		lines = []string{
			fmt.Sprintf("Merge into %q?", target),
			fmt.Sprintf("The new category is dropped and the cursor moves to %q.", target),
		}
	case len(src.Tasks) == 0:
		lines = []string{
			fmt.Sprintf("Merge %q into %q?", truncateText(src.Name, 30), target),
			fmt.Sprintf("%q is empty, so it is deleted.", truncateText(src.Name, 30)),
		}
	default:
		lines = []string{
			fmt.Sprintf("Merge %q into %q?", truncateText(src.Name, 30), target),
			fmt.Sprintf("Its %d task(s) move to the end of %q under a %q separator,", n, target, truncateText(src.Name, 30)),
			"then the category is deleted. u undoes it.",
		}
	}
	lines = append(lines, "", ui.RenderHints([]ui.Hint{
		{Key: "y/enter", Label: "merge"}, {Key: "n/esc", Label: "keep editing"},
	}))
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}

func (m model) confirmDeleteProjectView() string {
	var projectName string
	for _, p := range m.ui.Picker.projects {
		if p.ID == m.ui.ConfirmDelete.ProjectID {
			projectName = p.Name
			break
		}
	}
	lines := []string{
		fmt.Sprintf("Delete project %q?", truncateText(projectName, 30)),
		"",
		ui.RenderHints([]ui.Hint{{Key: "y/enter", Label: "confirm"}, {Key: "n/esc", Label: "cancel"}}),
	}
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}
