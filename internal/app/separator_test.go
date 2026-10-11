package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/app/modes"
	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/domain"
)

func separatorProject() domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{
				ID:   "c1",
				Name: "Cat A",
				Tasks: []domain.Task{
					{ID: "t1", Title: "Alpha", Status: domain.StatusTodo},
					{ID: "s1", Kind: domain.KindSeparator, Title: "mid"},
					{ID: "t2", Title: "Beta", Status: domain.StatusTodo},
				},
			},
		},
	}
}

func (m *model) selectSeparator(catIdx, taskIdx int) bool {
	return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusSeparator && p.CategoryIndex == catIdx && p.TaskIndex == taskIdx
	})
}

func TestRebuildPositionsEmitsSeparator(t *testing.T) {
	m := newTestModel(t, separatorProject())
	var found bool
	for _, p := range m.ui.Selection.Positions() {
		if p.Kind == selection.FocusSeparator && p.CategoryIndex == 0 && p.TaskIndex == 1 {
			found = true
		}
	}
	assert.True(t, found, "expected a FocusSeparator position for the separator row")
}

func TestSeparatorHiddenWhileFiltered(t *testing.T) {
	m := newTestModel(t, separatorProject())
	m.ui.Filter = NewFilterState()
	m.ui.Filter.statuses = map[string]bool{domain.StatusTodo: true}
	m.rebuildPositions()

	for _, p := range m.ui.Selection.Positions() {
		if p.Kind == selection.FocusSeparator {
			t.Fatal("separator should be hidden while a filter is active")
		}
	}
}

func TestStartAddingSeparatorInsertsBelow(t *testing.T) {
	m := newTestModel(t, separatorProject())
	// Select the first task (Alpha) then insert a separator below it.
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.TaskIndex == 0
	}))
	m.startAddingSeparator()

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 4)
	assert.True(t, tasks[1].IsSeparator(), "new separator should sit right below Alpha")
	assert.Equal(t, "", tasks[1].Title, "new separator should be bare")

	// Mode stays normal (no inline editor opened) and selection lands on it.
	assert.True(t, m.ui.Modes.IsNormal())
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusSeparator, pos.Kind)
	assert.Equal(t, 1, pos.TaskIndex)
}

func TestStartAddingTaskBelowSeparator(t *testing.T) {
	m := newTestModel(t, separatorProject())
	// Land on the separator (index 1), then press "a" to add a task.
	require.True(t, m.selectSeparator(0, 1))
	m.startAddingTask()

	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 4)
	// New task must sit directly below the separator, not jump to the top.
	assert.True(t, tasks[1].IsSeparator(), "separator stays at index 1")
	assert.False(t, tasks[2].IsSeparator(), "new task lands right after the separator")
	assert.Equal(t, "", tasks[2].Title, "new task starts blank")

	// Editor opens on the freshly inserted task.
	assert.True(t, m.ui.Modes.IsEdit())
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 2, pos.TaskIndex)
}

func TestStartAddingSeparatorBlockedWhileFiltered(t *testing.T) {
	m := newTestModel(t, separatorProject())
	m.ui.Filter = NewFilterState()
	m.ui.Filter.statuses = map[string]bool{domain.StatusTodo: true}
	m.rebuildPositions()
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask
	}))

	before := len(m.project.Categories[0].Tasks)
	m.startAddingSeparator()
	assert.Equal(t, before, len(m.project.Categories[0].Tasks), "insert must be blocked while filtered")
}

func TestDeleteSeparatorInstant(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectSeparator(0, 1))

	m.deleteSelected()

	// No confirm dialog: mode stays normal and the separator is already gone.
	assert.True(t, m.ui.Modes.IsNormal(), "separator delete must not open the confirm dialog")
	for _, tk := range m.project.Categories[0].Tasks {
		assert.False(t, tk.IsSeparator(), "separator should be deleted")
	}
	// Undo restores it.
	m.undo()
	require.Len(t, m.project.Categories[0].Tasks, 3)
	assert.True(t, m.project.Categories[0].Tasks[1].IsSeparator())
}

func TestMoveSeparatorDown(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectSeparator(0, 1))

	m.moveSelectedRow(1)

	tasks := m.project.Categories[0].Tasks
	assert.Equal(t, "t1", tasks[0].ID)
	assert.Equal(t, "t2", tasks[1].ID)
	assert.Equal(t, "s1", tasks[2].ID, "separator should have moved down past Beta")

	// Selection follows the separator to its new index.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusSeparator, pos.Kind)
	assert.Equal(t, 2, pos.TaskIndex)
}

func TestSeparatorLabelEditAndClear(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectSeparator(0, 1))

	// Edit the label.
	m.startEditing()
	require.True(t, m.ui.Modes.IsEdit())
	m.ui.Edit.input.SetValue("Milestone")
	m.finishEditing()
	assert.Equal(t, "Milestone", m.project.Categories[0].Tasks[1].Title)

	// Clearing the label back to bare is allowed (unlike task titles).
	require.True(t, m.selectSeparator(0, 1))
	m.startEditing()
	m.ui.Edit.input.SetValue("")
	m.finishEditing()
	assert.Equal(t, "", m.project.Categories[0].Tasks[1].Title)
	assert.True(t, m.project.Categories[0].Tasks[1].IsSeparator())
}

func TestSeparatorTaskActionsAreNoOps(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectSeparator(0, 1))

	// Status cycle: no change.
	m.toggleSelectedTask()
	assert.Equal(t, "", m.project.Categories[0].Tasks[1].Status)

	// Estimate picker: does not open.
	m.openEstimatePicker()
	assert.False(t, m.ui.Modes.IsEstimatePicker())
	assert.True(t, m.ui.Modes.Current() == modes.ModeNormal)
}

// selectTaskIdx lands the cursor on the task at (catIdx, taskIdx).
func (m *model) selectTaskIdx(catIdx, taskIdx int) bool {
	return m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == catIdx && p.TaskIndex == taskIdx
	})
}

func TestEnterVisualMode_StartsOnSeparator(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectSeparator(0, 1))

	m.enterVisualMode()

	require.True(t, m.ui.Modes.IsVisual(), "visual mode should start on a separator")
	// A separator anchor normalizes to the task-level domain.
	assert.Equal(t, selection.FocusTask, m.ui.Visual.Kind)
	assert.Equal(t, "s1", m.ui.Visual.AnchorTaskID)
}

func TestVisualCursor_StopsOnSeparator(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectTaskIdx(0, 0)) // Alpha
	m.enterVisualMode()

	m.visualMoveCursor(1) // should land on the separator, not skip it

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusSeparator, pos.Kind)
	assert.Equal(t, 1, pos.TaskIndex)
}

func TestVisualRange_SweepsInteriorSeparator(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectTaskIdx(0, 0)) // Alpha
	m.enterVisualMode()
	m.visualMoveCursor(2) // Alpha .. Beta, crossing the separator

	sel := m.visualSelectedPositions()
	require.Len(t, sel, 3, "range covers Alpha, separator, Beta")
	assert.Equal(t, selection.FocusTask, sel[0].Kind)
	assert.Equal(t, selection.FocusSeparator, sel[1].Kind)
	assert.Equal(t, selection.FocusTask, sel[2].Kind)

	// The separator's own row reports as in-range so it renders inside the band.
	sepIdx := m.ui.Selection.FindPositionIndex(func(p selection.Position) bool {
		return p.Kind == selection.FocusSeparator
	})
	require.GreaterOrEqual(t, sepIdx, 0)
	assert.True(t, m.isInVisualRange(sepIdx))
}

func TestVisualCopyText_SkipsSeparatorButClipboardKeepsIt(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.enterVisualMode()
	m.visualMoveCursor(2) // Alpha .. Beta

	sel := m.visualSelectedPositions()
	// Copied text omits the divider row entirely.
	assert.Equal(t, "- Alpha\n- Beta", m.visualMarkdownText(sel, false))
	assert.Equal(t, "- [ ] Alpha\n- [ ] Beta", m.visualMarkdownText(sel, true))

	// The internal clipboard still carries the separator for a faithful paste.
	_ = m.visualCopyBullets()
	require.Len(t, m.ui.Clipboard.Tasks, 3)
	assert.True(t, m.ui.Clipboard.Tasks[1].IsSeparator())
}

func TestVisualCounts_ExcludeSeparators(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.enterVisualMode()
	m.visualMoveCursor(2) // Alpha .. Beta, separator swept in

	// Header reports the two real tasks, not the 3-row span.
	assert.Equal(t, "-- VISUAL -- 2 tasks selected", m.statusText())

	m.visualCut()
	assert.Equal(t, "Cut: 2 task(s), p moves them here, esc cancels", m.ui.Screen.StatusMsg)
}

func TestVisualDelete_RemovesInteriorSeparator(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectTaskIdx(0, 0))
	m.enterVisualMode()
	m.visualMoveCursor(2) // Alpha .. Beta

	m.visualDelete()
	require.True(t, m.ui.Modes.IsConfirmDelete())
	assert.Equal(t, []string{"t1", "s1", "t2"}, m.ui.ConfirmDelete.TaskIDs)

	m.confirmDeleteVisualRange()
	assert.Empty(t, m.project.Categories[0].Tasks, "block including the separator is gone")
	// The message counts the two real tasks, not the swept-in separator.
	assert.Equal(t, "Deleted 2 task(s)", m.ui.Screen.StatusMsg)
}

func TestVisualCutPaste_PreservesSeparator(t *testing.T) {
	project := domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{
				ID:   "c1",
				Name: "Cat A",
				Tasks: []domain.Task{
					{ID: "t1", Title: "Alpha", Status: domain.StatusTodo},
					{ID: "s1", Kind: domain.KindSeparator, Title: "mid"},
					{ID: "t2", Title: "Beta", Status: domain.StatusTodo},
				},
			},
			{ID: "c2", Name: "Cat B", Tasks: []domain.Task{{ID: "t3", Title: "Gamma", Status: domain.StatusTodo}}},
		},
	}
	m := newTestModel(t, project)
	require.True(t, m.selectTaskIdx(0, 0))
	m.enterVisualMode()
	m.visualMoveCursor(2) // Alpha .. Beta
	m.visualCut()

	require.True(t, m.selectTaskIdx(1, 0)) // Gamma in Cat B
	m.pasteInternal()

	// Source category is emptied; the block lands after Gamma, divider intact.
	assert.Empty(t, m.project.Categories[0].Tasks)
	got := m.project.Categories[1].Tasks
	require.Len(t, got, 4)
	assert.Equal(t, "Gamma", got[0].Title)
	assert.Equal(t, "Alpha", got[1].Title)
	assert.True(t, got[2].IsSeparator(), "pasted separator keeps its Kind")
	assert.Equal(t, "mid", got[2].Title)
	assert.Equal(t, "Beta", got[3].Title)
	// Move message counts the two real tasks, not the separator.
	assert.Equal(t, "Moved 2 task(s)", m.ui.Screen.StatusMsg)
}

func TestVisualMoveDown_CarriesInteriorSeparator(t *testing.T) {
	project := domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{
				ID:   "c1",
				Name: "Cat A",
				Tasks: []domain.Task{
					{ID: "t1", Title: "Alpha", Status: domain.StatusTodo},
					{ID: "s1", Kind: domain.KindSeparator, Title: "mid"},
					{ID: "t2", Title: "Beta", Status: domain.StatusTodo},
					{ID: "t3", Title: "Gamma", Status: domain.StatusTodo},
				},
			},
		},
	}
	m := newTestModel(t, project)
	require.True(t, m.selectTaskIdx(0, 0)) // Alpha
	m.enterVisualMode()
	m.visualMoveCursor(1) // Alpha + separator → block [t1, s1]

	m.visualMoveDown() // Beta (the neighbor) hops above the block

	assert.Equal(t, []string{"t2", "t1", "s1", "t3"}, taskIDs(m.project.Categories[0]),
		"the separator travels with its block, staying adjacent to Alpha")
}

func TestSeparatorRendersCutBadge(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectTaskIdx(0, 0)) // Alpha
	m.enterVisualMode()
	m.visualMoveCursor(2) // Alpha .. Beta, sweeping the separator
	m.visualCut()

	layout := m.buildLayout()
	var rendered string
	for _, item := range layout.Items {
		if item.Kind == LayoutSeparator {
			rendered = m.renderLayoutItem(item)
		}
	}
	require.NotEmpty(t, rendered, "expected the separator row to render")
	assert.Contains(t, rendered, "✂", "a cut separator must carry the cut badge like a cut task")
}
