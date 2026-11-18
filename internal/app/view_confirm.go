package app

import (
	"fmt"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

type confirmPick int

const (
	confirmCancel confirmPick = iota
	confirmDelete
	confirmKeepChildren
)

var (
	deleteOption = ui.ChoiceOption{Label: "Delete", Key: "d", Aliases: []string{"y"}, Danger: true}
	cancelOption = ui.ChoiceOption{Label: "Cancel", Key: "c", Aliases: []string{"n"}}
)

func (m model) confirmDeleteChoice() (ui.Choice, []confirmPick) {
	c := ui.Choice{Selected: m.ui.ConfirmDelete.Pick, Options: []ui.ChoiceOption{deleteOption, cancelOption}}
	picks := []confirmPick{confirmDelete, confirmCancel}
	switch m.ui.ConfirmDelete.Kind {
	case ConfirmDeleteProject:
		// A project delete can't be undone, and d opened it: a doubled d must not delete.
		c.Options[0].Key = ""
		c.Title = m.confirmDeleteProjectTitle()
	case ConfirmDeleteVisualRange:
		c.Title = m.confirmDeleteVisualRangeTitle()
	default:
		var keep string
		c.Title, keep = m.confirmDeleteSelectionTitle()
		if keep != "" {
			c.Options = []ui.ChoiceOption{deleteOption, {Label: keep, Key: "s"}, cancelOption}
			picks = []confirmPick{confirmDelete, confirmKeepChildren, confirmCancel}
		}
	}
	return c, picks
}

func (m model) confirmDeleteView() string {
	c, _ := m.confirmDeleteChoice()
	if c.Title == "" {
		return ""
	}
	return m.choiceView(c)
}

func (m model) confirmDeleteSelectionTitle() (title, keep string) {
	position, ok := m.selectedPosition()
	if !ok || position.Kind == selection.FocusProject {
		return "", ""
	}
	children := m.deletedChildren(position)
	if position.Kind == selection.FocusTask {
		task := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
		if children > 0 {
			return fmt.Sprintf("Delete task %q and its %d subtask%s?", truncateText(task.Title, 30), children, plural(children, "", "s")), "Keep subtasks"
		}
		return fmt.Sprintf("Delete task %q?", truncateText(task.Title, 30)), ""
	}
	cat := m.project.Categories[position.CategoryIndex]
	tasks := m.project.TreeTaskCount(position.CategoryIndex)
	if children > 0 {
		return fmt.Sprintf("Delete category %q, its %d sub-categor%s and %d tasks?", truncateText(cat.Name, 30), children, plural(children, "y", "ies"), tasks), "Keep sub-categories"
	}
	return fmt.Sprintf("Delete category %q and %d tasks?", truncateText(cat.Name, 30), tasks), ""
}

func (m model) confirmDeleteVisualRangeTitle() string {
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
		tasks, subtasks, separators, notes := 0, 0, 0, 0
		for _, id := range m.ui.ConfirmDelete.TaskIDs {
			t := m.taskByID(id)
			switch {
			case t != nil && t.IsSeparator():
				separators++
			case t != nil && t.IsNote():
				notes++
			case t != nil && drop[t.ParentID]:
				subtasks++
			default:
				tasks++
			}
		}
		message = "Delete " + rowCountLabel(tasks, separators, notes) + "?"
		if subtasks > 0 {
			message = fmt.Sprintf("Delete %d task(s) and %d subtask(s)?", tasks, subtasks)
		}
	}
	return message
}

// mergeLines tell what merging the edited category into the one whose name it took does.
func (m model) mergeLines() []string {
	srcIdx, dst, ok := m.mergeOffer()
	if !ok {
		return nil
	}
	target := truncateText(m.project.Categories[dst].Name, 30)
	src := m.project.Categories[srcIdx]
	srcName := truncateText(src.Name, 30)
	moved := "Its " + rowsLabel(src.Tasks) + " move"
	subs := domain.ChildCount(m.project.Categories, srcIdx)
	switch {
	case m.ui.Edit.isAdding:
		return []string{fmt.Sprintf("The new category is dropped and the cursor moves to %q.", target)}
	case len(src.Tasks) == 0 && subs == 0:
		return []string{fmt.Sprintf("%q is empty, so it is deleted.", srcName)}
	}
	var lines []string
	its := "Its"
	if len(src.Tasks) > 0 {
		lines = append(lines, fmt.Sprintf("%s to the end of %q under a %q separator,", moved, target, srcName))
		its = "its"
	}
	if subs > 0 {
		lines = append(lines, fmt.Sprintf("%s %d sub-categor%s %s %q's, same names merging,",
			its, subs, plural(subs, "y", "ies"), plural(subs, "joins", "join"), target))
	}
	return append(lines, "then the category is deleted. u undoes it.")
}

func (m model) confirmDeleteProjectTitle() string {
	var projectName string
	for _, p := range m.ui.Picker.projects {
		if p.ID == m.ui.ConfirmDelete.ProjectID {
			projectName = p.Name
			break
		}
	}
	return fmt.Sprintf("Delete project %q?", truncateText(projectName, 30))
}
