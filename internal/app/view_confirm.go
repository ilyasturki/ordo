package app

import (
	"fmt"
	"strings"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
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
	hints := []ui.Hint{{Key: "y/enter", Label: "confirm"}, {Key: "n/esc", Label: "cancel"}}
	children := m.deletedChildren(position)
	if position.Kind == selection.FocusTask {
		task := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
		message = fmt.Sprintf("Delete task %q?", truncateText(task.Title, 30))
		if children > 0 {
			message = fmt.Sprintf("Delete task %q and its %d subtask%s?", truncateText(task.Title, 30), children, plural(children, "", "s"))
			hints = []ui.Hint{{Key: "y/enter", Label: "delete all"}, {Key: "k", Label: "keep the subtasks"}, {Key: "n/esc", Label: "cancel"}}
		}
	} else {
		cat := m.project.Categories[position.CategoryIndex]
		tasks := m.project.TreeTaskCount(position.CategoryIndex)
		message = fmt.Sprintf("Delete category %q and %d tasks?", truncateText(cat.Name, 30), tasks)
		if children > 0 {
			message = fmt.Sprintf("Delete category %q, its %d sub-categor%s and %d tasks?", truncateText(cat.Name, 30), children, plural(children, "y", "ies"), tasks)
			hints = []ui.Hint{{Key: "y/enter", Label: "delete all"}, {Key: "k", Label: "keep the sub-categories"}, {Key: "n/esc", Label: "cancel"}}
		}
	}
	lines := []string{
		message,
		"",
		ui.RenderHints(hints),
	}
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}

func (m model) confirmDeleteVisualRangeView() string {
	var message string
	if len(m.ui.ConfirmDelete.CategoryIDs) > 0 {
		drop := make(map[string]bool)
		for _, id := range m.ui.ConfirmDelete.CategoryIDs {
			drop[id] = true
		}
		cats, subs, tasks := 0, 0, 0
		for _, c := range m.project.Categories {
			if !drop[c.ID] {
				continue
			}
			if drop[c.ParentID] {
				subs++
			} else {
				cats++
			}
			tasks += c.TaskCount()
		}
		message = fmt.Sprintf("Delete %d categor%s and %d task(s)?", cats, plural(cats, "y", "ies"), tasks)
		if subs > 0 {
			message = fmt.Sprintf("Delete %d categor%s, %d sub-categor%s and %d task(s)?", cats, plural(cats, "y", "ies"), subs, plural(subs, "y", "ies"), tasks)
		}
	} else {
		drop := make(map[string]bool)
		for _, id := range m.ui.ConfirmDelete.TaskIDs {
			drop[id] = true
		}
		tasks, subtasks := 0, 0
		for _, id := range m.ui.ConfirmDelete.TaskIDs {
			if t := m.taskByID(id); t != nil && drop[t.ParentID] {
				subtasks++
			} else {
				tasks++
			}
		}
		message = fmt.Sprintf("Delete %d task(s)?", tasks)
		if subtasks > 0 {
			message = fmt.Sprintf("Delete %d task(s) and %d subtask(s)?", tasks, subtasks)
		}
	}
	lines := []string{
		message,
		"",
		ui.RenderHints([]ui.Hint{{Key: "y/enter", Label: "confirm"}, {Key: "n/esc", Label: "cancel"}}),
	}
	return ui.HelpDialogStyle.Render(strings.Join(lines, "\n"))
}

func (m model) confirmMergeView() string {
	srcIdx, dst, ok := m.mergeOffer()
	if !ok {
		return ""
	}
	target := truncateText(m.project.Categories[dst].Name, 30)
	src := m.project.Categories[srcIdx]
	srcName := truncateText(src.Name, 30)
	moved := fmt.Sprintf("Its %d task(s) move", src.TaskCount())
	if src.TaskCount() == 0 {
		moved = "Its separators move"
	}
	subs := domain.ChildCount(m.project.Categories, srcIdx)
	var lines []string
	switch {
	case m.ui.Edit.isAdding:
		lines = []string{
			fmt.Sprintf("Merge into %q?", target),
			fmt.Sprintf("The new category is dropped and the cursor moves to %q.", target),
		}
	case len(src.Tasks) == 0 && subs == 0:
		lines = []string{
			fmt.Sprintf("Merge %q into %q?", srcName, target),
			fmt.Sprintf("%q is empty, so it is deleted.", srcName),
		}
	default:
		lines = []string{fmt.Sprintf("Merge %q into %q?", srcName, target)}
		its := "Its"
		if len(src.Tasks) > 0 {
			lines = append(lines, fmt.Sprintf("%s to the end of %q under a %q separator,", moved, target, srcName))
			its = "its"
		}
		if subs > 0 {
			lines = append(lines, fmt.Sprintf("%s %d sub-categor%s %s %q's, same names merging,",
				its, subs, plural(subs, "y", "ies"), plural(subs, "joins", "join"), target))
		}
		lines = append(lines, "then the category is deleted. u undoes it.")
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
