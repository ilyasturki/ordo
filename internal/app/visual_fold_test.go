package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

// foldedRunProject is open Alpha/Beta, then folded B (with a separator) and
// folded C (with a sub-category), then open D.
func foldedRunProject(withOpenBelow bool) domain.Project {
	p := domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{ID: "ca", Name: "A", Tasks: []domain.Task{
				{ID: "t1", Title: "Alpha", Status: domain.StatusTodo},
				{ID: "t2", Title: "Beta", Status: domain.StatusTodo},
			}},
			{ID: "cb", Name: "B", Tasks: []domain.Task{
				{ID: "b1", Title: "B1", Status: domain.StatusTodo},
				{ID: "bs", Kind: domain.KindSeparator, Title: "later"},
				{ID: "b2", Title: "B2", Status: domain.StatusTodo},
			}},
			{ID: "cc", Name: "C", Tasks: []domain.Task{
				{ID: "c1", Title: "C1", Status: domain.StatusTodo},
			}},
			{ID: "cc1", Name: "C sub", ParentID: "cc", Tasks: []domain.Task{
				{ID: "c11", Title: "C11", Status: domain.StatusTodo},
			}},
		},
	}
	if withOpenBelow {
		p.Categories = append(p.Categories, domain.Category{ID: "cd", Name: "D", Tasks: []domain.Task{
			{ID: "d1", Title: "Delta", Status: domain.StatusTodo},
		}})
	}
	return p
}

func newFoldedRunModel(t *testing.T, withOpenBelow bool) *model {
	t.Helper()
	m := newTestModel(t, foldedRunProject(withOpenBelow))
	withStubStateManager(m)
	m.ui.Fold = NewFoldStateFrom([]string{"cb", "cc"})
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 1)) // Beta
	return m
}

func (m *model) selectedRow() selection.Position {
	pos, _ := m.selectedPosition()
	return pos
}

func (m *model) visualRowIDs() []string {
	rows := m.visualTaskRows(m.visualSelectedPositions())
	ids := make([]string, len(rows))
	for i, t := range rows {
		ids[i] = t.ID
	}
	return ids
}

func TestVisualFold_JStepsOntoAFoldedHeader(t *testing.T) {
	m := newFoldedRunModel(t, false)
	send(t, m, key("v"))
	send(t, m, key("j"))

	assert.Equal(t, selection.Position{Kind: selection.FocusCategory, CategoryIndex: 1, TaskIndex: -1, ID: "cb"}, m.selectedRow())
	assert.True(t, m.isInVisualRange(m.selected()), "the header draws inside the band")
	assert.Equal(t, "-- VISUAL -- 1 task + 1 folded category (2 tasks) selected", m.statusText())
	assert.True(t, m.ui.Fold.IsFolded("cb"), "moving never unfolds")

	send(t, m, key("j"))
	assert.Equal(t, "cc", m.selectedRow().ID)
	assert.Equal(t, "-- VISUAL -- 1 task + 2 folded categories (4 tasks) selected", m.statusText(),
		"a folded top-level category stands for its sub-categories too")

	before := m.selected()
	send(t, m, key("j"))
	assert.Equal(t, before, m.selected(), "the last folded header is the end")
}

func TestVisualFold_EveryJumpReachesTheFoldedRun(t *testing.T) {
	for _, k := range []string{"G", "ctrl+d", "ctrl+f"} {
		t.Run(k, func(t *testing.T) {
			m := newFoldedRunModel(t, false)
			send(t, m, key("v"))
			send(t, m, key(k))
			assert.Equal(t, "cc", m.selectedRow().ID)
		})
	}
}

func TestVisualFold_RangeOverAFoldedRunKeepsItsTasks(t *testing.T) {
	m := newFoldedRunModel(t, true)
	send(t, m, key("v"))
	send(t, m, key("G"))

	require.Equal(t, "d1", m.selectedRow().ID)
	assert.Equal(t, []string{"t2", "b1", "bs", "b2", "c1", "c11", "d1"}, m.visualRowIDs())
	assert.Equal(t, "-- VISUAL -- 2 tasks + 2 folded categories (4 tasks) selected", m.statusText())
}

func TestVisualFold_CopyCarriesTheFoldedRows(t *testing.T) {
	m := newFoldedRunModel(t, false)
	send(t, m, key("v"))
	send(t, m, key("j"))
	send(t, m, key("Y"))

	assert.Equal(t, "- [ ] Beta\n- [ ] B1\n### later\n- [ ] B2", m.ui.Clipboard.Text)
	assert.Equal(t, []string{"t2", "b1", "bs", "b2"}, m.ui.Clipboard.TaskIDs)
}

func TestVisualFold_CutMovesTheFoldedRows(t *testing.T) {
	m := newFoldedRunModel(t, true)
	send(t, m, key("v"))
	send(t, m, key("j"))
	send(t, m, key("x"))
	landWrite(m)

	require.True(t, m.selectTaskIdx(4, 0)) // Delta
	m.resolvePaste(clipboardLinesMsg{text: m.ui.Clipboard.Text})

	assert.Equal(t, []string{"d1", "t2", "b1", "bs", "b2"}, taskIDs(m.project.Categories[4]))
	assert.Empty(t, m.project.Categories[1].Tasks, "folded B is left empty")
	assert.True(t, m.ui.Fold.IsFolded("cb"))
}

func TestVisualFold_DeleteTakesTheFoldedRows(t *testing.T) {
	m := newFoldedRunModel(t, false)
	send(t, m, key("v"))
	send(t, m, key("j"))
	send(t, m, key("d"))

	assert.Equal(t, []string{"t2", "b1", "bs", "b2"}, m.ui.ConfirmDelete.TaskIDs)
	assert.Contains(t, m.confirmDeleteView(), "Delete 3 task(s)?", "the folded separator isn't counted as a task")

	send(t, m, key("y"))
	assert.Equal(t, "Deleted 3 task(s)", m.ui.Screen.StatusMsg)
	assert.Equal(t, []string{"t1"}, taskIDs(m.project.Categories[0]))
	assert.Empty(t, m.project.Categories[1].Tasks)
}

func TestVisualDelete_PromptCountsSeparatorsApart(t *testing.T) {
	m := newTestModel(t, separatorProject())
	require.True(t, m.selectSeparator(0, 1))
	send(t, m, key("v"))
	send(t, m, key("d"))
	assert.Contains(t, m.confirmDeleteView(), "Delete 1 separator(s)?")
}

func TestVisualFold_MoveTakesTheFoldedRows(t *testing.T) {
	m := newFoldedRunModel(t, true)
	send(t, m, key("v"))
	send(t, m, key("j"))
	send(t, m, key("m"))

	require.True(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"t2", "b1", "bs", "b2"}, m.ui.MovePicker.taskIDs)
}

func TestVisualFold_PerRowEditsSkipTheFoldedRows(t *testing.T) {
	m := newFoldedRunModel(t, false)
	send(t, m, key("v"))
	send(t, m, key("j"))
	send(t, m, key("space"))

	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[1].Status)
	assert.Equal(t, domain.StatusTodo, m.project.Categories[1].Tasks[0].Status, "B1 stays as it was")
	assert.Equal(t, "Set 1 task(s) to In Progress", m.ui.Screen.StatusMsg)
}

func TestVisualFold_ShiftAndNestRefuse(t *testing.T) {
	for _, k := range []string{"J", "K", ">", "<"} {
		t.Run(k, func(t *testing.T) {
			m := newFoldedRunModel(t, false)
			send(t, m, key("v"))
			send(t, m, key("j"))
			send(t, m, key(k))

			assert.Equal(t, "Can't: the selection holds a folded category", m.ui.Screen.StatusMsg)
			assert.Equal(t, []string{"t1", "t2"}, taskIDs(m.project.Categories[0]))
			assert.Equal(t, []string{"b1", "bs", "b2"}, taskIDs(m.project.Categories[1]))
		})
	}
}

func TestVisualFold_SwapAnchorsOnTheHeader(t *testing.T) {
	m := newFoldedRunModel(t, false)
	beta := m.selected()
	send(t, m, key("v"))
	send(t, m, key("j"))
	header := m.selected()

	send(t, m, key("o"))
	assert.Equal(t, beta, m.selected())
	lo, hi, ok := m.visualRange()
	require.True(t, ok)
	assert.Equal(t, []int{beta, header}, []int{lo, hi})

	send(t, m, key("o"))
	assert.Equal(t, header, m.selected())
	assert.Equal(t, "t2", m.ui.Visual.AnchorTaskID)
}

func TestVisualFold_FoldedSubCategoryStandsForItselfOnly(t *testing.T) {
	m := newTestModel(t, foldedRunProject(false))
	withStubStateManager(m)
	m.ui.Fold = NewFoldStateFrom([]string{"cb", "cc1"})
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(2, 0)) // C1, inside open C
	send(t, m, key("v"))
	send(t, m, key("j"))

	assert.Equal(t, "cc1", m.selectedRow().ID)
	assert.Equal(t, []string{"c1", "c11"}, m.visualRowIDs())
}

func TestVisualFold_FilterHiddenRowsStayOut(t *testing.T) {
	p := foldedRunProject(false)
	p.Categories[1].Tasks[2].Status = domain.StatusCompleted
	m := newTestModel(t, p)
	withStubStateManager(m)
	m.ui.Fold = NewFoldStateFrom([]string{"cb", "cc"})
	m.ui.Filter = NewFilterState()
	m.ui.Filter.statuses = map[string]bool{domain.StatusTodo: true}
	m.rebuildPositions()
	require.True(t, m.selectTaskIdx(0, 1))
	send(t, m, key("v"))
	send(t, m, key("j"))

	assert.Equal(t, []string{"t2", "b1"}, m.visualRowIDs(),
		"the done task and the separator the filter hides don't travel")
}

func TestVisualFold_OpenHeadersStaySkippedAndHeaderAnchorsStayCategoryLevel(t *testing.T) {
	m := newFoldedRunModel(t, true)
	require.True(t, m.selectTaskIdx(0, 0))
	send(t, m, key("v"))
	send(t, m, key("k"))
	assert.Equal(t, "t1", m.selectedRow().ID, "the open header above is no stop")
	send(t, m, key("esc"))

	require.True(t, m.selectItem(selection.FocusCategory, "cb"))
	send(t, m, key("v"))
	assert.Equal(t, selection.FocusCategory, m.ui.Visual.Kind)
	send(t, m, key("j"))
	assert.Equal(t, "cc", m.selectedRow().ID)
	send(t, m, key("j"))
	assert.Equal(t, "cd", m.selectedRow().ID, "a category-level range still steps onto open headers")
}
