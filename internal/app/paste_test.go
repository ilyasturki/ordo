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
	m.pasteInternal()

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
	m.pasteInternal()

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

	m.pasteInternal()
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

	m.pasteInternal()

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

	m.pasteInternal()

	assert.Equal(t, []string{"t4", "t5"}, taskIDs(m.project.Categories[1]))
	assert.Equal(t, depth, m.ui.History.UndoDepth(), "a move of nothing leaves no undo step")
	assert.True(t, m.ui.Clipboard.empty())
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
	m.pasteInternal()

	require.Len(t, m.project.Categories, 2)
	assert.Equal(t, "c2", m.project.Categories[0].ID)
	assert.Equal(t, "c1", m.project.Categories[1].ID)
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
	assert.Equal(t, "Moved 1 category", m.ui.Screen.StatusMsg)
}

func TestPaste_RepeatsTheCopyWhileTheClipboardHoldsIt(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "body"
	m := newTestModel(t, p)
	require.True(t, m.selectTaskIdx(0, 0))
	require.NotNil(t, m.copySelected())
	require.Equal(t, "Alpha\n\nbody", m.ui.Clipboard.Text)

	require.True(t, m.selectTaskIdx(1, 0))
	require.NotNil(t, m.paste(), "a written copy is checked against the clipboard")
	m.resolvePaste(clipboardLinesMsg{text: "Alpha\n\nbody\n"})
	m.resolvePaste(clipboardLinesMsg{text: "Alpha\n\nbody"})

	tasks := m.project.Categories[1].Tasks
	require.Len(t, tasks, 4, "two full copies, not the text split into tasks")
	for _, copied := range tasks[1:3] {
		assert.Equal(t, "Alpha", copied.Title)
		assert.Equal(t, "body", copied.Description)
		assert.NotEqual(t, "t1", copied.ID)
	}
	assert.NotEqual(t, tasks[1].ID, tasks[2].ID)
}

func TestPaste_ClipboardCopiedLaterWins(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.copySelected()

	require.True(t, m.selectTaskIdx(1, 0))
	m.resolvePaste(clipboardLinesMsg{text: "from the browser"})

	assert.Equal(t, []string{"Delta", "from the browser", "Epsilon"}, taskTitles(m.project.Categories[1]))
	assert.True(t, m.ui.Clipboard.empty(), "the stale copy is dropped")
}

func TestPaste_UnreadableClipboardFallsBackToTheCopy(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.copySelected()

	require.True(t, m.selectTaskIdx(1, 0))
	m.resolvePaste(clipboardLinesMsg{err: assert.AnError})

	assert.Equal(t, []string{"Delta", "Alpha", "Epsilon"}, taskTitles(m.project.Categories[1]))
}

func TestPaste_DeletedTaskSkipsTheClipboard(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.deleteTask(selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0})
	assert.False(t, m.ui.Clipboard.Written, "d writes nothing to the system clipboard")

	require.True(t, m.selectTaskIdx(1, 0))
	assert.Nil(t, m.paste())

	assert.Equal(t, []string{"Delta", "Alpha", "Epsilon"}, taskTitles(m.project.Categories[1]))
}

func TestPaste_FailedWriteTrustsTheCopy(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.copySelected()
	m.handleClipboardResult(clipboardResultMsg{text: "Alpha", err: assert.AnError})

	require.True(t, m.selectTaskIdx(1, 0))
	assert.Nil(t, m.paste(), "a clipboard that never got the copy can't vouch for it")
	assert.Equal(t, []string{"Delta", "Alpha", "Epsilon"}, taskTitles(m.project.Categories[1]))
}

func TestCut_WritesWhatYWould(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "body"
	m := newTestModel(t, p)
	require.True(t, m.selectTaskIdx(0, 0))
	pos, _ := m.selectedPosition()
	want := m.copyTextForPosition(pos)

	require.NotNil(t, m.cutSelectedTask())

	assert.Equal(t, want, m.ui.Clipboard.Text)
	assert.True(t, m.ui.Clipboard.Written)
	assert.Equal(t, "Cut: Alpha, p moves it here, esc cancels", m.ui.Screen.StatusMsg)

	m.handleClipboardResult(clipboardResultMsg{text: want, status: m.ui.Screen.StatusMsg, viaTerminal: true})
	assert.Equal(t, "Cut: Alpha, p moves it here, esc cancels (via terminal)", m.ui.Screen.StatusMsg,
		"the write's report keeps the cut's wording")
}

func TestVisualCut_WritesWhatVisualYWould(t *testing.T) {
	m := newTestModelWithDescription(t)
	require.True(t, m.selectTaskIdx(0, 0))
	m.enterVisualMode()
	m.visualMoveCursor(2)
	want := m.visualMarkdownText(m.visualSelectedPositions(), false)
	require.Contains(t, want, "\n  ", "the range takes a description row")

	require.NotNil(t, m.visualCut())

	assert.Equal(t, want, m.ui.Clipboard.Text)
	assert.True(t, m.ui.Clipboard.IsCut)
	assert.Equal(t, "Cut: 2 task(s), p moves them here, esc cancels", m.ui.Screen.StatusMsg)
}

func TestCut_PasteMovesThenPasteCopies(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.cutSelectedTask()
	text := m.ui.Clipboard.Text

	require.True(t, m.selectTaskIdx(1, 0))
	m.resolvePaste(clipboardLinesMsg{text: text})
	m.resolvePaste(clipboardLinesMsg{text: text})

	ids := taskIDs(m.project.Categories[1])
	require.Len(t, ids, 4)
	assert.Equal(t, "t1", ids[1], "the first p moves the task itself")
	assert.Equal(t, []string{"Delta", "Alpha", "Alpha", "Epsilon"}, taskTitles(m.project.Categories[1]))
	assert.NotContains(t, []string{"t1", "t2", "t3"}, ids[2], "the second p pastes a copy")
	assert.Equal(t, []string{"t2", "t3"}, taskIDs(m.project.Categories[0]))
}

func taskTitles(cat domain.Category) []string {
	out := make([]string, len(cat.Tasks))
	for i, t := range cat.Tasks {
		out[i] = t.Title
	}
	return out
}
