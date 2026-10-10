package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

func TestToggleSelectedTask_CyclesStatus(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1 (Todo)
	m.toggleSelectedTask()
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
	m.toggleSelectedTask()
	assert.Equal(t, domain.StatusCompleted, m.project.Categories[0].Tasks[0].Status)
}

func TestToggleSelectedTask_NoopOnCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	// Should not panic / mutate any task
	m.toggleSelectedTask()
	assert.Equal(t, domain.StatusTodo, m.project.Categories[0].Tasks[0].Status)
}

func TestToggleSelectedTaskReverse_CyclesStatusBackward(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1 (Todo)
	m.toggleSelectedTaskReverse()
	assert.Equal(t, domain.StatusCancelled, m.project.Categories[0].Tasks[0].Status)
	m.toggleSelectedTaskReverse()
	assert.Equal(t, domain.StatusCompleted, m.project.Categories[0].Tasks[0].Status)
	m.toggleSelectedTaskReverse()
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
	m.toggleSelectedTaskReverse()
	assert.Equal(t, domain.StatusTodo, m.project.Categories[0].Tasks[0].Status)
}

func TestToggleSelectedTaskReverse_RecordsHistory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	require.Equal(t, 0, m.ui.History.UndoDepth())
	m.toggleSelectedTaskReverse()
	assert.Equal(t, 1, m.ui.History.UndoDepth())
	m.undo()
	assert.Equal(t, domain.StatusTodo, m.project.Categories[0].Tasks[0].Status)
}

func TestIncreasePriority_FromMediumToHigh(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Priority = domain.PriorityMedium
	m := newTestModel(t, p)
	m.ui.Selection.MoveTo(2)
	m.increasePriority()
	assert.Equal(t, domain.PriorityHigh, m.project.Categories[0].Tasks[0].Priority)
}

func TestDecreasePriority_FromMediumToLow(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Priority = domain.PriorityMedium
	m := newTestModel(t, p)
	m.ui.Selection.MoveTo(2)
	m.decreasePriority()
	assert.Equal(t, domain.PriorityLow, m.project.Categories[0].Tasks[0].Priority)
}

func TestIncreasePriority_NoopOnCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	originalPriority := m.project.Categories[0].Tasks[0].Priority
	m.increasePriority()
	assert.Equal(t, originalPriority, m.project.Categories[0].Tasks[0].Priority)
}

func TestReverseCategories_FlipsOrder(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.reverseCategories()
	assert.Equal(t, "c2", m.project.Categories[0].ID)
	assert.Equal(t, "c1", m.project.Categories[1].ID)
	// Tasks within a category keep their order.
	assert.Equal(t, []string{"t1", "t2", "t3"}, taskIDs(m.project.Categories[1]))
}

func TestReverseCategories_KeepsSelectedCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header (c1)
	m.reverseCategories()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	// c1 is now the last category; selection tracks it rather than the header slot.
	assert.Equal(t, 1, pos.CategoryIndex)
	assert.Equal(t, "c1", m.project.Categories[pos.CategoryIndex].ID)
}

func TestReverseCategories_KeepsSelectedTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1 in Cat A
	m.reverseCategories()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 1, pos.CategoryIndex)
	assert.Equal(t, 0, pos.TaskIndex)
	assert.Equal(t, "t1", m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID)
}

func TestReverseCategories_RecordsHistory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.Equal(t, 0, m.ui.History.UndoDepth())
	m.reverseCategories()
	assert.Equal(t, 1, m.ui.History.UndoDepth())
	m.undo()
	assert.Equal(t, "c1", m.project.Categories[0].ID)
	assert.Equal(t, "c2", m.project.Categories[1].ID)
}

func TestReverseCategories_KeepsProjectHeaderSelected(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(0) // project header, which never moves on a flip
	m.reverseCategories()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusProject, pos.Kind)
	// Order still flipped despite the header staying selected.
	assert.Equal(t, "c2", m.project.Categories[0].ID)
}

func TestReverseCategories_NoopWithSingleCategory(t *testing.T) {
	p := domain.Project{Categories: []domain.Category{
		{ID: "only", Tasks: []domain.Task{{ID: "x"}}},
	}}
	m := newTestModel(t, p)
	m.reverseCategories()
	assert.Equal(t, 0, m.ui.History.UndoDepth(), "no history recorded for a no-op")
	assert.Equal(t, "only", m.project.Categories[0].ID)
}

func TestMoveTaskDown_SwapsWithinCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.moveSelectedRow(1)
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[0].ID)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[1].ID)
	// Selection follows the moved task
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 0, pos.CategoryIndex)
	assert.Equal(t, 1, pos.TaskIndex)
}

func TestMoveTaskDown_CrossesIntoNextCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(4) // t3 (last task in Cat A)
	m.moveSelectedRow(1)
	// t3 should be inserted at the head of Cat B
	assert.Len(t, m.project.Categories[0].Tasks, 2)
	require.Len(t, m.project.Categories[1].Tasks, 3)
	assert.Equal(t, "t3", m.project.Categories[1].Tasks[0].ID)
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, 1, pos.CategoryIndex)
	assert.Equal(t, 0, pos.TaskIndex)
}

func TestMoveTaskUp_SwapsWithinCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(3) // t2
	m.moveSelectedRow(-1)
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[0].ID)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[1].ID)
}

func TestMoveTaskUp_CrossesIntoPrevCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(6) // t4 (first task in Cat B)
	m.moveSelectedRow(-1)
	// t4 should be appended to Cat A
	require.Len(t, m.project.Categories[0].Tasks, 4)
	assert.Equal(t, "t4", m.project.Categories[0].Tasks[3].ID)
	assert.Len(t, m.project.Categories[1].Tasks, 1)
}

func TestMoveTaskDown_FollowsTaskPastDescriptionRow(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[1].Description = "beta details" // t2 has a description row
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusTask && pp.CategoryIndex == 0 && pp.TaskIndex == 0
	}), "focus t1 (no description)")

	m.moveSelectedRow(1)

	assert.Equal(t, "t2", m.project.Categories[0].Tasks[0].ID)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[1].ID)
	// Selection must follow t1, not land on t2's description row.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 0, pos.CategoryIndex)
	assert.Equal(t, 1, pos.TaskIndex)
}

func TestMoveTaskUp_FollowsTaskPastDescriptionRow(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "alpha details" // t1 has a description row
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusTask && pp.CategoryIndex == 0 && pp.TaskIndex == 1
	}), "focus t2 (no description), below t1's description row")

	m.moveSelectedRow(-1)

	assert.Equal(t, "t2", m.project.Categories[0].Tasks[0].ID)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[1].ID)
	// Selection must follow t2 to index 0, not land on t1's row/description.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 0, pos.CategoryIndex)
	assert.Equal(t, 0, pos.TaskIndex)
}

func TestMoveCategoryDown_SwapsCategories(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.moveSelectedRow(1)
	assert.Equal(t, "c2", m.project.Categories[0].ID)
	assert.Equal(t, "c1", m.project.Categories[1].ID)
	// Selection follows the moved category
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, 1, pos.CategoryIndex)
}

func TestMoveCategoryDown_NoopAtEnd(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(5) // Cat B (last)
	m.moveSelectedRow(1)
	assert.Equal(t, "c1", m.project.Categories[0].ID)
	assert.Equal(t, "c2", m.project.Categories[1].ID)
}

func TestMoveCategoryUp_SwapsCategories(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(5) // Cat B
	m.moveSelectedRow(-1)
	assert.Equal(t, "c2", m.project.Categories[0].ID)
	assert.Equal(t, "c1", m.project.Categories[1].ID)
}

func TestMoveCategoryUp_NoopAtStart(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.moveSelectedRow(-1)
	assert.Equal(t, "c1", m.project.Categories[0].ID)
}

func TestCutSelectedTask_PopulatesClipboardAndSetsStatus(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.cutSelectedTask()
	require.Len(t, m.ui.Clipboard.Tasks, 1)
	assert.True(t, m.ui.Clipboard.IsCut)
	assert.Equal(t, []string{"t1"}, m.ui.Clipboard.TaskIDs)
	assert.Equal(t, "Cut: Alpha, p moves it here, esc cancels", m.ui.Screen.StatusMsg)
}

func TestCutSelectedTask_RejectsCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	m.cutSelectedTask()
	assert.Empty(t, m.ui.Clipboard.Tasks)
	assert.Equal(t, "Can only cut tasks", m.ui.Screen.StatusMsg)
}

func TestPasteTask_AfterCut_MovesTaskToTarget(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.cutSelectedTask()
	// Move to t4 in Cat B
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})
	m.pasteInternal()

	// Cat A lost t1
	assert.Len(t, m.project.Categories[0].Tasks, 2)
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[0].ID)
	// Cat B: cursor was on t4 (Delta); paste lands AFTER it.
	require.Len(t, m.project.Categories[1].Tasks, 3)
	assert.Equal(t, "Delta", m.project.Categories[1].Tasks[0].Title)
	assert.Equal(t, "Alpha", m.project.Categories[1].Tasks[1].Title)
	assert.False(t, m.ui.Clipboard.IsCut, "the move used the cut up")
	assert.Equal(t, "Moved!", m.ui.Screen.StatusMsg)
}

func TestPasteTask_AfterCopy_KeepsOriginal(t *testing.T) {
	m := newTestModel(t, sampleProject())
	// Simulate a copied (not cut) task via clipboard.
	src := m.project.Categories[0].Tasks[0]
	m.ui.Clipboard = ClipboardState{Tasks: []domain.Task{src}, TaskIDs: []string{src.ID}}
	// Paste onto t4 in Cat B
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})
	m.pasteInternal()

	// Original survived
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[0].ID)
	// Cat B got a clone AFTER cursor (cursor was on t4 / Delta at index 0)
	require.Len(t, m.project.Categories[1].Tasks, 3)
	assert.Equal(t, "Delta", m.project.Categories[1].Tasks[0].Title)
	clone := m.project.Categories[1].Tasks[1]
	assert.NotEqual(t, "t1", clone.ID)
	assert.Equal(t, "Alpha", clone.Title)
	assert.Equal(t, "Pasted!", m.ui.Screen.StatusMsg)
}

func TestPasteTask_EmptyClipboardFallsBackToSystemClipboard(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	cmd := m.paste()
	require.NotNil(t, cmd)
	assert.Empty(t, m.ui.Screen.StatusMsg)
}

func TestRemoveTaskByID(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.removeTaskByID("t2")
	assert.Len(t, m.project.Categories[0].Tasks, 2)
	for _, task := range m.project.Categories[0].Tasks {
		assert.NotEqual(t, "t2", task.ID)
	}
}

func TestRemoveTaskByID_NonexistentIsNoop(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.removeTaskByID("does-not-exist")
	assert.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Len(t, m.project.Categories[1].Tasks, 2)
}

func TestDeleteTask_RemovesAndCopiesToClipboard(t *testing.T) {
	m := newTestModel(t, sampleProject())
	pos := selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}
	m.ui.Selection.MoveTo(2)
	m.deleteTask(pos, false)
	assert.Len(t, m.project.Categories[0].Tasks, 2)
	require.Len(t, m.ui.Clipboard.Tasks, 1)
	assert.Equal(t, "Alpha", m.ui.Clipboard.Tasks[0].Title)
	assert.False(t, m.ui.Clipboard.IsCut)
}

func TestEditOrFocusDescription_TaskWithDescription_JumpsToDescriptionRow(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "Some details"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	// Select t1 (Alpha) — index after expansion still places task before its description.
	m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusTask && pp.CategoryIndex == 0 && pp.TaskIndex == 0
	})

	cmd := m.editOrFocusDescription()
	assert.Nil(t, cmd, "should not launch editor when description already exists")

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusDescription, pos.Kind, "selection should land on description row")
	assert.Equal(t, 0, pos.CategoryIndex)
	assert.Equal(t, 0, pos.TaskIndex)
}

func TestEditOrFocusDescription_TaskWithoutDescription_OpensEditor(t *testing.T) {
	// t1 has no description here; shift+enter should produce a non-nil tea.Cmd
	// (the external editor launch) instead of a focus jump.
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1

	cmd := m.editOrFocusDescription()
	assert.NotNil(t, cmd, "should launch editor when description is empty")
	// Selection unchanged.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
}

func TestClearDescription_FocusDescription_ClearsAndReselectsTask(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "Some details"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	m.ui.Selection.SelectByPredicate(func(pp selection.Position) bool {
		return pp.Kind == selection.FocusDescription && pp.CategoryIndex == 0 && pp.TaskIndex == 0
	})

	m.clearDescription(selection.Position{Kind: selection.FocusDescription, CategoryIndex: 0, TaskIndex: 0})

	assert.Equal(t, "", m.project.Categories[0].Tasks[0].Description)
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind, "selection should snap back to parent task")
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[pos.TaskIndex].ID)
	assert.Equal(t, "Description cleared", m.ui.Screen.StatusMsg)
}

func TestRebuildPositions_ExpandedDescriptions_AddsRowOnlyForNonEmptyDesc(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "Some details"

	positions := rebuildPositions(p.Categories, nil, nil, true, "")

	descCount := 0
	for _, pos := range positions {
		if pos.Kind == selection.FocusDescription {
			descCount++
			assert.Equal(t, 0, pos.CategoryIndex)
			assert.Equal(t, 0, pos.TaskIndex, "only t1 has a description")
		}
	}
	assert.Equal(t, 1, descCount)
}

func TestDeleteCategory_RemovesEntireCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	pos := selection.Position{Kind: selection.FocusCategory, CategoryIndex: 0}
	m.ui.Selection.MoveTo(1)
	m.deleteCategory(pos, false)
	require.Len(t, m.project.Categories, 1)
	assert.Equal(t, "c2", m.project.Categories[0].ID)
}

func TestEscCancelsPendingCut(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.cutSelectedTask()
	require.True(t, m.isTaskCut("t1"))

	m.dispatchNormalKey("esc")
	assert.False(t, m.ui.Clipboard.IsCut)
	assert.Empty(t, m.ui.Clipboard.Tasks)
	assert.False(t, m.isTaskCut("t1"))
	assert.Equal(t, "Cut cancelled", m.ui.Screen.StatusMsg)
}

func TestEscKeepsCopiedTaskOnClipboard(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.copySelected()
	require.Len(t, m.ui.Clipboard.Tasks, 1)

	m.dispatchNormalKey("esc")
	assert.NotEmpty(t, m.ui.Clipboard.Tasks, "a plain copy survives esc — nothing on screen marks it")
}

func TestEscClearsSearchBeforeCut(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	m.ui.Selection.MoveTo(2) // t1
	m.cutSelectedTask()
	typeSearch(m, "ta")
	m.commitSearch()
	require.NotEmpty(t, m.ui.Search.query)

	// The first esc only drops the highlight, so hunting for the paste target
	// with `/` cannot cost the cut.
	m.dispatchNormalKey("esc")
	assert.Empty(t, m.ui.Search.query)
	assert.True(t, m.isTaskCut("t1"), "cut survives the first esc")

	m.dispatchNormalKey("esc")
	assert.False(t, m.isTaskCut("t1"))
}

func TestEscClearsPendingChordBeforeCut(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.cutSelectedTask()

	m.dispatchNormalKey("g") // chord prefix, awaiting its second key
	require.NotZero(t, m.ui.Screen.PendingKey)

	m.dispatchNormalKey("esc")
	assert.Zero(t, m.ui.Screen.PendingKey)
	assert.True(t, m.isTaskCut("t1"), "backing out of a chord doesn't cost the cut")

	m.dispatchNormalKey("esc")
	assert.False(t, m.isTaskCut("t1"))
}
