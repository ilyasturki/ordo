package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

func TestUndo_EmptyHistorySetsStatus(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.undo()
	assert.Equal(t, "Nothing to undo", m.ui.Screen.StatusMsg)
}

func TestRedo_EmptyHistorySetsStatus(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.redo()
	assert.Equal(t, "Nothing to redo", m.ui.Screen.StatusMsg)
}

func TestUndo_ToggleStatus(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1 (Todo)
	m.toggleSelectedTask()
	require.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)

	m.undo()
	assert.Equal(t, domain.StatusTodo, m.project.Categories[0].Tasks[0].Status)

	m.redo()
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
}

func TestUndo_Priority(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Priority = domain.PriorityMedium
	m := newTestModel(t, p)
	m.ui.Selection.MoveTo(2)
	m.increasePriority()
	require.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[0].Priority)

	m.undo()
	assert.Equal(t, domain.PriorityMedium, m.project.Categories[0].Tasks[0].Priority)

	m.redo()
	assert.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[0].Priority)
}

func TestUndo_DeleteTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.deleteTask(selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}, false)
	require.Len(t, m.project.Categories[0].Tasks, 2)

	m.undo()
	require.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[0].ID)

	m.redo()
	require.Len(t, m.project.Categories[0].Tasks, 2)
	assert.NotEqual(t, "t1", m.project.Categories[0].Tasks[0].ID)
}

func TestUndo_DeleteCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.deleteCategory(selection.Position{Kind: selection.FocusCategory, CategoryIndex: 0}, false)
	require.Len(t, m.project.Categories, 1)
	assert.Equal(t, "c2", m.project.Categories[0].ID)

	m.undo()
	require.Len(t, m.project.Categories, 2)
	assert.Equal(t, "c1", m.project.Categories[0].ID)
}

func TestUndo_MoveTaskDown(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.moveSelectedRow(1)
	require.Equal(t, "t2", m.project.Categories[0].Tasks[0].ID)

	m.undo()
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[0].ID)
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[1].ID)

	m.redo()
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[0].ID)
}

func TestUndo_MoveCategoryDown(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.moveSelectedRow(1)
	require.Equal(t, "c2", m.project.Categories[0].ID)

	m.undo()
	assert.Equal(t, "c1", m.project.Categories[0].ID)
}

func TestUndo_PasteCopy(t *testing.T) {
	m := newTestModel(t, sampleProject())
	src := m.project.Categories[0].Tasks[0]
	m.ui.Clipboard = ClipboardState{Tasks: []domain.Task{src}, TaskIDs: []string{src.ID}}
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})
	m.pasteInternal()
	require.Len(t, m.project.Categories[1].Tasks, 3)

	m.undo()
	assert.Len(t, m.project.Categories[1].Tasks, 2)

	m.redo()
	assert.Len(t, m.project.Categories[1].Tasks, 3)
}

func TestUndo_PasteCut(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.cutSelectedTask()
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})
	m.pasteInternal()
	require.Len(t, m.project.Categories[0].Tasks, 2)
	require.Len(t, m.project.Categories[1].Tasks, 3)

	m.undo()
	assert.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Len(t, m.project.Categories[1].Tasks, 2)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[0].ID)
}

func TestUndo_AddTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.startAddingTask()
	require.True(t, m.ui.Modes.IsEdit())
	require.Len(t, m.project.Categories[0].Tasks, 4)
	// Simulate user typing and committing.
	m.ui.Edit.input.SetValue("Brand new task")
	m.finishEditing()
	require.Equal(t, "Brand new task", m.project.Categories[0].Tasks[1].Title)

	// One undo should remove the added task entirely (single logical op).
	m.undo()
	assert.Len(t, m.project.Categories[0].Tasks, 3)
	for _, task := range m.project.Categories[0].Tasks {
		assert.NotEqual(t, "Brand new task", task.Title)
	}
}

func TestUndo_AddTask_CancelDoesNothing(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	m.startAddingTask()
	require.True(t, m.ui.Modes.IsEdit())
	m.cancelEditing()

	// Nothing changed; undo should report nothing to do.
	require.Len(t, m.project.Categories[0].Tasks, 3)
	m.undo()
	assert.Equal(t, "Nothing to undo", m.ui.Screen.StatusMsg)
}

func TestUndo_RenameTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.startEditing()
	m.ui.Edit.input.SetValue("Alpha-renamed")
	m.finishEditing()
	require.Equal(t, "Alpha-renamed", m.project.Categories[0].Tasks[0].Title)

	m.undo()
	assert.Equal(t, "Alpha", m.project.Categories[0].Tasks[0].Title)

	m.redo()
	assert.Equal(t, "Alpha-renamed", m.project.Categories[0].Tasks[0].Title)
}

func TestUndo_RenameProject(t *testing.T) {
	m := newStoredModel(t, sampleProject())
	m.ui.Selection.MoveTo(0) // project line
	m.startEditing()
	m.ui.Edit.input.SetValue("Renamed Project")
	m.finishEditing()
	require.Equal(t, "Renamed Project", m.project.Name)

	m.undo()
	assert.Equal(t, "P", m.project.Name)
}

func TestUndo_RenameCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.startEditing()
	m.ui.Edit.input.SetValue("Cat A Renamed")
	m.finishEditing()
	require.Equal(t, "Cat A Renamed", m.project.Categories[0].Name)

	m.undo()
	assert.Equal(t, "Cat A", m.project.Categories[0].Name)
}

func TestUndo_EstimateChange(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.selectEstimate(60)
	require.Equal(t, 60, m.project.Categories[0].Tasks[0].EstimateMinutes)

	m.undo()
	assert.Equal(t, 0, m.project.Categories[0].Tasks[0].EstimateMinutes)
}

func TestUndo_ClearDescription(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "Old desc"
	m := newTestModel(t, p)
	m.ui.Selection.MoveTo(2) // t1
	m.clearDescription(selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0})
	require.Equal(t, "", m.project.Categories[0].Tasks[0].Description)

	m.undo()
	assert.Equal(t, "Old desc", m.project.Categories[0].Tasks[0].Description)
}

func TestRedo_InvalidatedByNewMutation(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.toggleSelectedTask()
	require.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)

	m.undo()
	require.Equal(t, domain.StatusTodo, m.project.Categories[0].Tasks[0].Status)
	require.True(t, m.ui.History.CanRedo())

	// A new mutation should clear the redo stack.
	m.increasePriority()
	assert.False(t, m.ui.History.CanRedo())
	m.redo()
	assert.Equal(t, "Nothing to redo", m.ui.Screen.StatusMsg)
}

func TestUndo_NoopActionDoesNotPushHistory(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Priority = domain.PriorityCritical
	m := newTestModel(t, p)
	m.ui.Selection.MoveTo(2)
	// IncreasePriority is a no-op when already at the ceiling (critical).
	m.increasePriority()
	assert.False(t, m.ui.History.CanUndo(), "no-op should not record history")
}

func TestUndo_RestoresSelection(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.toggleSelectedTask()
	// Move selection elsewhere after the mutation
	m.ui.Selection.MoveTo(5)

	m.undo()
	// Selection should restore back to t1
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 0, pos.CategoryIndex)
	assert.Equal(t, 0, pos.TaskIndex)
}

func TestUndo_HistoryLimitCaps(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	// Trigger many mutations to exceed historyLimit.
	for i := 0; i < historyLimit+10; i++ {
		m.toggleSelectedTask()
	}
	assert.LessOrEqual(t, m.ui.History.UndoDepth(), historyLimit)
}

func TestUndo_VisualDelete(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // extend to t2
	m.visualDelete()
	// confirmDeleteVisualRange runs from the confirm-delete handler; call directly.
	m.confirmDeleteVisualRange()
	require.Len(t, m.project.Categories[0].Tasks, 1)
	assert.Equal(t, "t3", m.project.Categories[0].Tasks[0].ID)

	m.undo()
	require.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[0].ID)
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[1].ID)
}

func TestUndoRedo_TapeSequenceMirror(t *testing.T) {
	// Mirrors testdata/vhs/12_undo_redo.tape: j, l, J, u, u, ctrl+r, ctrl+r.
	p := sampleProject()
	p.Categories[0].Tasks[1].Priority = domain.PriorityMedium
	m := newTestModel(t, p)
	// j: initial cursor at first task t1, j moves to t2.
	m.ui.Selection.MoveTo(2) // = t1
	m.moveSelection(1)       // = t2
	// l: increase priority
	m.increasePriority()
	require.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[1].Priority)
	// J: move down
	m.moveSelectedRow(1)
	require.Equal(t, "t2", m.project.Categories[0].Tasks[2].ID)
	require.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[2].Priority)
	// u: undo move (priority survives)
	m.undo()
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[1].ID, "t2 should be back at index 1")
	assert.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[1].Priority,
		"priority HIGH must survive the move undo")
	// u: undo priority
	m.undo()
	assert.Equal(t, domain.PriorityMedium, m.project.Categories[0].Tasks[1].Priority)
	// ctrl+r: redo priority
	m.redo()
	assert.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[1].Priority)
	// ctrl+r: redo move
	m.redo()
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[2].ID)
}

func TestUndoRedo_PriorityThenMove_UndoMoveOnlyRevertsMove(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[1].Priority = domain.PriorityMedium
	m := newTestModel(t, p)
	// Select t2 (Beta) at category 0, index 1.
	m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusTask && pp.CategoryIndex == 0 && pp.TaskIndex == 1
	})

	m.increasePriority()
	require.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[1].Priority)

	m.moveSelectedRow(1)
	require.Equal(t, "t2", m.project.Categories[0].Tasks[2].ID, "t2 should have moved down")
	require.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[2].Priority)

	// Single undo: only the move should revert; priority must remain HIGH.
	m.undo()
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[1].ID, "t2 should be back at index 1")
	assert.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[1].Priority,
		"priority bump must survive the move undo")

	// Second undo: priority returns to medium.
	m.undo()
	assert.Equal(t, domain.PriorityMedium, m.project.Categories[0].Tasks[1].Priority)
}

// Regression: a record-then-discard sequence (e.g. cancelling an add) must
// NOT destroy the redo stack that the user built up via undo.
func TestDiscardLastHistory_PreservesRedoStack(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1

	// Build a redo entry: mutate, then undo.
	m.toggleSelectedTask()
	m.undo()
	require.True(t, m.ui.History.CanRedo(), "precondition: redo should hold the undone toggle")

	// Now perform a no-op: start adding a task and immediately cancel.
	m.startAddingTask()
	require.True(t, m.ui.Modes.IsEdit())
	m.cancelEditing()

	// The cancel must have rolled back, AND the redo stack must still be intact.
	require.Len(t, m.project.Categories[0].Tasks, 3, "blank add should have been rolled back")
	assert.True(t, m.ui.History.CanRedo(),
		"cancelled add must not destroy the redo stack (record-then-discard bug)")

	// Sanity: redo restores the original toggle.
	m.redo()
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
}

// Regression: deleting a task stashes it in the clipboard. After undo restores
// the task, the clipboard must NOT still be primed with the deleted copy —
// otherwise `p` would insert a duplicate.
func TestUndo_DeleteThenPasteDoesNotDuplicate(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.deleteTask(selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}, false)
	require.Len(t, m.project.Categories[0].Tasks, 2)
	require.NotEmpty(t, m.ui.Clipboard.Tasks, "delete stages the task in clipboard")

	m.undo()
	require.Len(t, m.project.Categories[0].Tasks, 3, "task is restored by undo")
	assert.Empty(t, m.ui.Clipboard.Tasks,
		"clipboard must be restored to its pre-delete state (nil) by undo, "+
			"otherwise paste would duplicate the just-restored task")

	// pasteInternal should be a no-op now.
	m.pasteInternal()
	assert.Len(t, m.project.Categories[0].Tasks, 3, "no paste should happen with empty clipboard")
}

// Regression: pasting on a description row used to fall through the switch
// and land the new task in category 0 at index 0.
func TestPaste_FocusDescriptionTargetsParentTaskCategory(t *testing.T) {
	p := sampleProject()
	p.Categories[1].Tasks[0].Description = "has desc"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	// Park the cursor on the description row of c2/t4.
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusDescription && p.CategoryIndex == 1 && p.TaskIndex == 0
	}), "expected to land on the description row")

	src := m.project.Categories[0].Tasks[0]
	m.ui.Clipboard = ClipboardState{Tasks: []domain.Task{src}, TaskIDs: []string{src.ID}}
	m.pasteInternal()

	// The new task must be inserted in c2 right after t4 — not in c1 at index 0.
	require.Len(t, m.project.Categories[1].Tasks, 3, "c2 should have grown by one")
	assert.Equal(t, "Alpha", m.project.Categories[1].Tasks[1].Title,
		"new task should sit at c2 index 1 (after the parent task), not in c1")
	assert.Len(t, m.project.Categories[0].Tasks, 3, "c1 must be unchanged")
}

func TestUndoRedo_MultipleStepsInOrder(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.toggleSelectedTask()   // Todo → InProgress
	m.toggleSelectedTask()   // InProgress → Completed
	require.Equal(t, domain.StatusCompleted, m.project.Categories[0].Tasks[0].Status)

	m.undo()
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
	m.undo()
	assert.Equal(t, domain.StatusTodo, m.project.Categories[0].Tasks[0].Status)

	m.redo()
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
	m.redo()
	assert.Equal(t, domain.StatusCompleted, m.project.Categories[0].Tasks[0].Status)
}
