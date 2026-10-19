package app

import (
	"fmt"
	"slices"

	tea "charm.land/bubbletea/v2"

	"github.com/ilyasturki/ordo/internal/app/components"
	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/operations"
)

func (m *model) deleteSelected() {
	if !m.ui.Modes.CanPerformAction(modes.ActionDeleteItem) {
		return
	}
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind == selection.FocusProject || pos.Kind == selection.FocusHidden {
		return
	}
	if pos.Kind == selection.FocusDescription {
		m.clearDescription(pos)
		return
	}
	// Separators delete instantly (no confirm dialog): they hold no data worth a
	// second thought, and undo covers an accidental removal.
	if pos.Kind == selection.FocusSeparator {
		m.deleteSeparator(pos)
		return
	}
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteSelection}
	m.ui.Modes.ToConfirmDelete()
}

func (m *model) deleteSeparator(pos selection.Position) {
	if pos.CategoryIndex < 0 || pos.CategoryIndex >= len(m.project.Categories) {
		return
	}
	cat := &m.project.Categories[pos.CategoryIndex]
	if pos.TaskIndex < 0 || pos.TaskIndex >= len(cat.Tasks) {
		return
	}
	m.recordHistory()
	if _, err := operations.DeleteTasks(&m.project, []string{cat.Tasks[pos.TaskIndex].ID}, false); !m.applied(err) {
		return
	}
	m.rebuildAndClamp()
	m.storeTaskUpdate()
}

func (m *model) clearDescription(pos selection.Position) {
	if pos.CategoryIndex < 0 || pos.CategoryIndex >= len(m.project.Categories) {
		return
	}
	cat := &m.project.Categories[pos.CategoryIndex]
	if pos.TaskIndex < 0 || pos.TaskIndex >= len(cat.Tasks) {
		return
	}
	task := &cat.Tasks[pos.TaskIndex]
	if task.Description == "" {
		return
	}
	m.recordHistory()
	task.Description = ""
	task.UpdatedAt = domain.NowTimestamp()
	m.storeTaskUpdate()
	m.rebuildPositions()
	m.ensureVisible()
	m.ui.Screen.StatusMsg = "Description cleared"
}

func (m *model) confirmDeleteAction() {
	m.confirmDelete(false)
}

func (m *model) confirmDeleteKeepingChildren() {
	m.confirmDelete(true)
}

func (m *model) confirmDelete(keepChildren bool) {
	m.ui.ConfirmDelete.reset()
	m.ui.Modes.ToNormal()
	position, ok := m.selectedPosition()
	if !ok {
		return
	}
	switch position.Kind {
	case selection.FocusTask:
		m.deleteTask(position, keepChildren)
	case selection.FocusCategory:
		m.deleteCategory(position, keepChildren)
	}
}

func (m model) deletedChildren(pos selection.Position) int {
	switch pos.Kind {
	case selection.FocusTask:
		return domain.ChildCount(m.project.Categories[pos.CategoryIndex].Tasks, pos.TaskIndex)
	case selection.FocusCategory:
		return domain.ChildCount(m.project.Categories, pos.CategoryIndex)
	}
	return 0
}

func (m *model) deleteTask(position selection.Position, keepChildren bool) {
	id := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex].ID
	// Snapshot BEFORE stashing the task in the clipboard, so that undo
	// restores both the project (task back) AND the clipboard (back to its
	// pre-delete contents). Otherwise `d`→`u`→`p` would paste a duplicate of
	// the just-restored task.
	m.recordHistory()
	removed, err := operations.DeleteTasks(&m.project, []string{id}, keepChildren)
	if !m.applied(err) {
		return
	}
	ids := make([]string, len(removed))
	for i, t := range removed {
		ids[i] = t.ID
	}
	m.ui.Clipboard = ClipboardState{Tasks: removed, TaskIDs: ids}
	m.ui.TagCopiedLast = false
	m.rebuildAndClamp()
	m.storeTaskUpdate()
}

func (m *model) deleteCategory(position selection.Position, keepChildren bool) {
	catIndex := position.CategoryIndex
	m.recordHistory()
	if keepChildren {
		if err := m.project.ReleaseSubCategories(catIndex); err != nil {
			m.discardLastHistory()
			m.ui.Screen.StatusMsg = "Can't keep the sub-categories: a top-level category has one of their names"
			return
		}
	}
	m.project.RemoveCategoryBlock(catIndex)
	m.rebuildAndClamp()
	m.storeTaskUpdate()
}

func (m *model) rebuildAndClamp() {
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) toggleSelectedTask() {
	if m.showHiddenTasks() {
		return
	}
	m.cycleSelectedTask(false)
}

func (m *model) toggleSelectedTaskReverse() {
	m.cycleSelectedTask(true)
}

func (m *model) cycleSelectedTask(reverse bool) {
	if !m.ui.Modes.CanPerformAction(modes.ActionToggleTask) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	id := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex].ID
	m.recordHistory()
	if _, _, err := operations.CycleStatus(&m.project, []string{id}, reverse); m.applied(err) {
		m.storeTaskUpdate()
	}
}

func (m *model) increasePriority() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangePriority) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	id := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex].ID
	m.recordHistory()
	if m.applied(operations.StepPriority(&m.project, id, true)) {
		m.storeTaskUpdate()
	}
}

func (m *model) decreasePriority() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangePriority) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	id := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex].ID
	m.recordHistory()
	if m.applied(operations.StepPriority(&m.project, id, false)) {
		m.storeTaskUpdate()
	}
}

func (m *model) openEstimatePicker() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangeEstimate) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind == selection.FocusProject || position.Kind == selection.FocusSeparator || position.Kind == selection.FocusHidden {
		return
	}

	var currentEstimate int
	if position.Kind == selection.FocusTask {
		currentEstimate = m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex].EstimateMinutes
	} else {
		currentEstimate = m.project.Categories[position.CategoryIndex].EstimateMinutes
	}

	m.ui.EstimatePicker = components.NewEstimatePickerState(currentEstimate)
	m.ui.Modes.ToEstimatePicker()
}

func (m *model) selectEstimate(minutes int) {
	position, ok := m.selectedPosition()
	if !ok || position.Kind == selection.FocusProject || position.Kind == selection.FocusSeparator || position.Kind == selection.FocusHidden {
		return
	}

	var prev int
	if position.Kind == selection.FocusTask {
		prev = m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex].EstimateMinutes
	} else {
		prev = m.project.Categories[position.CategoryIndex].EstimateMinutes
	}
	if prev == minutes {
		return
	}

	m.recordHistory()
	cat := m.project.Categories[position.CategoryIndex]
	var err error
	if position.Kind == selection.FocusTask {
		_, err = operations.SetEstimates(&m.project, []string{cat.Tasks[position.TaskIndex].ID}, minutes)
	} else {
		err = operations.SetCategoryEstimate(&m.project, cat.ID, minutes)
	}
	if m.applied(err) {
		m.storeTaskUpdate()
	}
}

func (m *model) reverseCategories() {
	if !m.ui.Modes.CanPerformAction(modes.ActionMoveItem) {
		return
	}
	cats := m.project.Categories
	if len(cats) == 0 || domain.BlockEnd(cats, 0) == len(cats) {
		return
	}
	m.recordHistory()
	m.project.ReverseCategories()
	m.rebuildPositions()
	m.ensureVisible()
	m.storeTaskUpdate()
}

func (m *model) cutSelectedTask() tea.Cmd {
	if !m.ui.Modes.CanPerformAction(modes.ActionDeleteItem) {
		return nil
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		m.ui.Screen.StatusMsg = "Can only cut tasks"
		return nil
	}

	task := m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
	text := m.copyTextForPosition(position)
	tasks, ids := m.taskBlock(position)
	m.ui.Clipboard = ClipboardState{Tasks: tasks, TaskIDs: ids, IsCut: true, Text: text, Written: true}
	m.ui.TagCopiedLast = false

	status := "Cut: " + truncateText(task.Title, 30) + ", p moves it here, esc cancels"
	if n := len(tasks) - 1; n > 0 {
		status = fmt.Sprintf("Cut: %s and %d subtask%s, p moves them here, esc cancels", truncateText(task.Title, 30), n, plural(n, "", "s"))
	}
	m.ui.Screen.StatusMsg = status
	return m.writeClipboard(clipboardResultMsg{text: text, status: status})
}

func (m *model) taskBlock(pos selection.Position) ([]domain.Task, []string) {
	cat := m.project.Categories[pos.CategoryIndex]
	tasks := slices.Clone(cat.Tasks[pos.TaskIndex:domain.BlockEnd(cat.Tasks, pos.TaskIndex)])
	ids := make([]string, len(tasks))
	for i, t := range tasks {
		ids[i] = t.ID
	}
	return tasks, ids
}

func (m *model) removeTaskByID(id string) {
	for catIdx := range m.project.Categories {
		for taskIdx, task := range m.project.Categories[catIdx].Tasks {
			if task.ID == id {
				_ = m.project.Categories[catIdx].RemoveTask(taskIdx)
				return
			}
		}
	}
}

func (m *model) findTaskByID(id string) (int, int) {
	for catIdx := range m.project.Categories {
		for taskIdx, task := range m.project.Categories[catIdx].Tasks {
			if task.ID == id {
				return catIdx, taskIdx
			}
		}
	}
	return -1, -1
}

// selectTaskByID moves the cursor onto the visible task row with the given ID.
// Reports whether a match was found. Call after rebuildPositions to keep the
// cursor on a task whose position may have shifted.
func (m *model) selectTaskByID(id string) bool {
	return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		if p.Kind != selection.FocusTask {
			return false
		}
		return m.project.Categories[p.CategoryIndex].Tasks[p.TaskIndex].ID == id
	})
}

func (m *model) focusTask(selector string) error {
	catIdx, taskIdx, err := m.project.FindTask(selector)
	if err != nil {
		return fmt.Errorf("task %q not found in project %q", selector, m.project.Name)
	}
	m.landOnRow(selection.Position{Kind: selection.FocusTask, CategoryIndex: catIdx, TaskIndex: taskIdx,
		ID: m.project.Categories[catIdx].Tasks[taskIdx].ID})
	return nil
}
