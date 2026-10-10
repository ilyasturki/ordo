package app

import (
	"fmt"

	tea "charm.land/bubbletea/v2"

	"phasionary/internal/app/components"
	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

func (m *model) deleteSelected() {
	if !m.ui.Modes.CanPerformAction(modes.ActionDeleteItem) {
		return
	}
	pos, ok := m.selectedPosition()
	if !ok || pos.Kind == selection.FocusProject {
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
	_ = cat.RemoveTask(pos.TaskIndex)
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
	m.ui.ConfirmDelete.reset()
	m.ui.Modes.ToNormal()
	position, ok := m.selectedPosition()
	if !ok {
		return
	}
	switch position.Kind {
	case selection.FocusTask:
		m.deleteTask(position)
	case selection.FocusCategory:
		m.deleteCategory(position)
	}
}

func (m *model) deleteTask(position selection.Position) {
	catIndex := position.CategoryIndex
	taskIndex := position.TaskIndex

	// Snapshot BEFORE stashing the task in the clipboard, so that undo
	// restores both the project (task back) AND the clipboard (back to its
	// pre-delete contents). Otherwise `d`→`u`→`p` would paste a duplicate of
	// the just-restored task.
	m.recordHistory()

	task := m.project.Categories[catIndex].Tasks[taskIndex]
	m.ui.Clipboard = ClipboardState{Tasks: []domain.Task{task}, TaskIDs: []string{task.ID}}
	m.ui.TagCopiedLast = false

	_ = m.project.Categories[catIndex].RemoveTask(taskIndex)
	m.rebuildAndClamp()
	m.storeTaskUpdate()
}

func (m *model) deleteCategory(position selection.Position) {
	catIndex := position.CategoryIndex
	m.recordHistory()
	_ = m.project.RemoveCategory(catIndex)
	m.rebuildAndClamp()
	m.storeTaskUpdate()
}

func (m *model) rebuildAndClamp() {
	m.rebuildPositions()
	m.ensureVisible()
}

func (m *model) toggleSelectedTask() {
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
	task := &m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
	m.recordHistory()
	var changed bool
	if reverse {
		changed = task.CycleStatusReverse()
	} else {
		changed = task.CycleStatus()
	}
	if changed {
		m.storeTaskUpdate()
		return
	}
	m.discardLastHistory()
}

func (m *model) increasePriority() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangePriority) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	task := &m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
	m.recordHistory()
	if task.IncreasePriority() {
		m.storeTaskUpdate()
		return
	}
	m.discardLastHistory()
}

func (m *model) decreasePriority() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangePriority) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind != selection.FocusTask {
		return
	}
	task := &m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
	m.recordHistory()
	if task.DecreasePriority() {
		m.storeTaskUpdate()
		return
	}
	m.discardLastHistory()
}

func (m *model) openEstimatePicker() {
	if !m.ui.Modes.CanPerformAction(modes.ActionChangeEstimate) {
		return
	}
	position, ok := m.selectedPosition()
	if !ok || position.Kind == selection.FocusProject || position.Kind == selection.FocusSeparator {
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
	if !ok || position.Kind == selection.FocusProject || position.Kind == selection.FocusSeparator {
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
	if position.Kind == selection.FocusTask {
		task := &m.project.Categories[position.CategoryIndex].Tasks[position.TaskIndex]
		task.SetEstimate(minutes)
	} else {
		category := &m.project.Categories[position.CategoryIndex]
		category.SetEstimate(minutes)
	}
	m.storeTaskUpdate()
}

func (m *model) reverseCategories() {
	if !m.ui.Modes.CanPerformAction(modes.ActionMoveItem) {
		return
	}
	if len(m.project.Categories) < 2 {
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
	m.ui.Clipboard = ClipboardState{Tasks: []domain.Task{task}, TaskIDs: []string{task.ID}, IsCut: true, Text: text, Written: true}
	m.ui.TagCopiedLast = false

	status := "Cut: " + truncateText(task.Title, 30) + ", p moves it here, esc cancels"
	m.ui.Screen.StatusMsg = status
	return m.writeClipboard(clipboardResultMsg{text: text, status: status})
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
