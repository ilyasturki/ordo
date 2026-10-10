package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

func TestCutPaste_KeepsEditsMadeAfterTheCut(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].CreatedAt = "2026-01-02T03:04:05Z"
	m := newTestModel(t, p)
	require.True(t, m.selectTaskIdx(0, 0))
	m.cutSelectedTask()

	m.toggleSelectedTask()
	m.startEditing()
	m.ui.Edit.input.SetValue("Alpha, renamed")
	m.finishEditing()

	require.True(t, m.selectTaskIdx(1, 0))
	m.pasteFromClipboard()

	assert.Equal(t, []string{"t2", "t3"}, taskIDs(m.project.Categories[0]))
	require.Equal(t, []string{"t4", "t1", "t5"}, taskIDs(m.project.Categories[1]))
	moved := m.project.Categories[1].Tasks[1]
	assert.Equal(t, "Alpha, renamed", moved.Title)
	assert.Equal(t, domain.StatusInProgress, moved.Status)
	assert.Equal(t, "2026-01-02T03:04:05Z", moved.CreatedAt)
	assert.Equal(t, "Moved!", m.ui.Screen.StatusMsg)
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, "t1", m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID)
}

func TestCutPaste_RangeKeepsIDsAndEdits(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.enterVisualMode()
	m.visualMoveCursor(1)
	m.visualCut()

	require.True(t, m.selectTaskIdx(0, 1))
	m.toggleSelectedTask()
	require.True(t, m.selectTaskIdx(1, 1))
	m.pasteFromClipboard()

	assert.Equal(t, []string{"t3"}, taskIDs(m.project.Categories[0]))
	require.Equal(t, []string{"t4", "t5", "t1", "t2"}, taskIDs(m.project.Categories[1]))
	assert.Equal(t, domain.StatusCompleted, m.project.Categories[1].Tasks[3].Status,
		"the status cycled after the cut travels with the row")
	assert.Equal(t, "Moved 2 task(s)", m.ui.Screen.StatusMsg)
}

func TestCutPaste_UndoAndRedoRestoreExactly(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.cutSelectedTask()
	require.True(t, m.selectTaskIdx(1, 0))
	before := cloneProject(m.project)

	m.pasteFromClipboard()
	after := cloneProject(m.project)

	m.undo()
	assert.Equal(t, before, m.project)
	assert.True(t, m.isTaskCut("t1"), "undo brings the pending cut back")

	m.redo()
	assert.Equal(t, after, m.project)
}

func TestCutPaste_WithinItsOwnCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.cutSelectedTask()
	require.True(t, m.selectTaskIdx(0, 2))

	m.pasteFromClipboard()

	assert.Equal(t, []string{"t2", "t3", "t1"}, taskIDs(m.project.Categories[0]))
}

func TestCutPaste_SourceGoneMovesNothing(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.cutSelectedTask()
	m.removeTaskByID("t1")
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(1, 0))
	depth := m.ui.History.UndoDepth()

	m.pasteFromClipboard()

	assert.Equal(t, []string{"t4", "t5"}, taskIDs(m.project.Categories[1]))
	assert.Equal(t, depth, m.ui.History.UndoDepth(), "a move of nothing leaves no undo step")
	assert.False(t, m.ui.Clipboard.IsCut)
	assert.Equal(t, "Nothing to move: the cut rows are gone", m.ui.Screen.StatusMsg)
}

func TestCutPaste_CategoryKeepsEditsToItsTasks(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusCategory && p.CategoryIndex == 1
	}))
	m.enterVisualMode()
	m.visualCut()

	require.True(t, m.selectTaskIdx(1, 0))
	m.toggleSelectedTask()
	m.ui.Selection.MoveTo(1) // Cat A
	m.pasteFromClipboard()

	require.Len(t, m.project.Categories, 2)
	assert.Equal(t, "c2", m.project.Categories[0].ID)
	assert.Equal(t, "c1", m.project.Categories[1].ID)
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
	assert.Equal(t, "Moved 1 category", m.ui.Screen.StatusMsg)
}
