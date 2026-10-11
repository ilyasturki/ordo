package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/domain"
)

// typeSearch drives a search the way the key handler would: start, set the
// query, and run the incremental preview.
func typeSearch(m *model, query string) {
	m.startSearch()
	m.ui.Search.input.SetValue(query)
	m.previewSearch()
}

func assertOnTask(t *testing.T, m *model, catIdx, taskIdx int) {
	t.Helper()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, catIdx, pos.CategoryIndex)
	assert.Equal(t, taskIdx, pos.TaskIndex)
}

func TestSearch_PreviewJumpsToFirstMatch(t *testing.T) {
	m := newTestModel(t, sampleProject())
	typeSearch(m, "ta") // matches Beta (c0,t1) and Delta (c1,t0)
	assert.True(t, m.ui.Modes.IsSearch())
	assertOnTask(t, m, 0, 1) // Beta, the first match from the project row
}

func TestSearch_CommitActivatesAndKeepsCursor(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	typeSearch(m, "ta")
	m.commitSearch()

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "ta", m.ui.Search.query) // committed query stays in effect
	assert.Equal(t, "ta", m.searchQuery())
	assertOnTask(t, m, 0, 1) // still on Beta
}

func TestSearch_NextAndPrevCycleWithWrap(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	typeSearch(m, "ta")
	m.commitSearch()
	assertOnTask(t, m, 0, 1) // Beta

	m.searchNext()
	assertOnTask(t, m, 1, 0) // Delta

	m.searchNext() // wraps back to Beta
	assertOnTask(t, m, 0, 1)

	m.searchPrev() // back to Delta
	assertOnTask(t, m, 1, 0)
}

func TestSearch_NextIsNoopWithoutActiveSearch(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	m.searchNext()
	assert.Equal(t, 2, m.ui.Selection.Selected())
	m.searchPrev()
	assert.Equal(t, 2, m.ui.Selection.Selected())
}

func TestSearch_CancelRestoresCursorScrollAndFolds(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	// Fold Cat B up-front; it must come back folded after cancelling.
	m.ui.Fold.Toggle("c2")
	m.rebuildPositions()
	m.ui.Selection.MoveTo(2) // Alpha
	origin := m.ui.Selection.Selected()

	typeSearch(m, "Delta") // reveals folded Cat B during preview
	require.False(t, m.ui.Fold.IsFolded("c2"))

	m.cancelSearch()
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Empty(t, m.ui.Search.query, "search no longer in effect")
	assert.True(t, m.ui.Fold.IsFolded("c2"), "fold state should be restored")
	assert.Equal(t, origin, m.ui.Selection.Selected())
}

func TestSearch_CommitRevealsFoldedMatch(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	m.ui.Fold.Toggle("c2") // Delta/Epsilon now hidden
	m.rebuildPositions()

	typeSearch(m, "Epsilon")
	m.commitSearch()

	assert.False(t, m.ui.Fold.IsFolded("c2"), "category should be unfolded to reveal the match")
	assertOnTask(t, m, 1, 1) // Epsilon
	assert.ElementsMatch(t, []string{}, stub.folded["p1"], "revealed fold should be persisted")
}

func TestSearch_CommitNoMatchRestoresAndReports(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(3)
	origin := m.ui.Selection.Selected()

	typeSearch(m, "zzzznope")
	m.commitSearch()

	assert.Empty(t, m.ui.Search.query)
	assert.Equal(t, origin, m.ui.Selection.Selected())
	assert.Contains(t, m.ui.Screen.StatusMsg, "No matches")
}

func TestSearch_ClearDropsHighlightKeepsCursor(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	typeSearch(m, "ta")
	m.commitSearch()
	require.NotEmpty(t, m.ui.Search.query)
	sel := m.ui.Selection.Selected()

	m.clearSearch()
	assert.Empty(t, m.ui.Search.query)
	assert.Equal(t, "", m.searchQuery())
	assert.Equal(t, sel, m.ui.Selection.Selected())
}

func TestSearch_MatchesCategoryName(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	typeSearch(m, "Cat B")
	m.commitSearch()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, 1, pos.CategoryIndex)
}

func TestRowMatches_TitleCategoryAndCollapsedDescription(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "secret sauce"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = false

	catRow := selection.Position{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1}
	taskRow := selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}

	assert.True(t, m.rowMatches(catRow, "Cat A"))
	assert.True(t, m.rowMatches(taskRow, "Alpha"))  // title
	assert.True(t, m.rowMatches(taskRow, "secret")) // collapsed description folds into the task row
	assert.False(t, m.rowMatches(taskRow, "nonexistent term"))
}

func TestRowMatches_ExpandedDescriptionOwnsItsMatch(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "secret sauce"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true

	taskRow := selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0}
	descRow := selection.Position{Kind: selection.FocusDescription, CategoryIndex: 0, TaskIndex: 0}

	// When expanded, the task row matches only its title; the description row
	// owns the description match.
	assert.False(t, m.rowMatches(taskRow, "secret"))
	assert.True(t, m.rowMatches(descRow, "secret"))
}

func TestSearch_RespectsActiveFilter(t *testing.T) {
	m := newTestModel(t, sampleProject())
	// Hide everything that isn't in-progress: only Beta (c0,t1) survives.
	m.ui.Filter.statuses[domain.StatusInProgress] = true
	m.rebuildPositions()

	rows := m.searchRows()
	// Alpha (todo) is filtered out, so "Alpha" has no reachable match.
	assert.Equal(t, 0, m.matchCount("Alpha"))
	assert.Equal(t, 1, m.matchCount("Beta"))
	// Category rows are always present.
	assert.Equal(t, 1, m.matchCount("Cat A"))
	_ = rows
}

func TestNextMatchingRow_DirectionsWrapAndIncludeFrom(t *testing.T) {
	rows := []selection.Position{
		{Kind: selection.FocusCategory, CategoryIndex: 0, TaskIndex: -1},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1},
	}
	// Match only the task rows.
	match := func(r selection.Position) bool { return r.Kind == selection.FocusTask }

	// Forward from -1 (no current row) finds the first task.
	assert.Equal(t, 1, nextMatchingRow(rows, -1, 1, true, match))
	// Forward from index 1, excluding self, finds index 2.
	assert.Equal(t, 2, nextMatchingRow(rows, 1, 1, false, match))
	// Forward from index 2, excluding self, wraps to index 1.
	assert.Equal(t, 1, nextMatchingRow(rows, 2, 1, false, match))
	// Backward from -1 starts at the end.
	assert.Equal(t, 2, nextMatchingRow(rows, -1, -1, true, match))
	// No match returns -1.
	assert.Equal(t, -1, nextMatchingRow(rows, 0, 1, true, func(selection.Position) bool { return false }))
}

func TestCurrentLogicalIndex_ProjectRowMapsToMinusOne(t *testing.T) {
	m := newTestModel(t, sampleProject())
	rows := m.searchRows()
	m.ui.Selection.MoveTo(0) // project row
	assert.Equal(t, -1, m.currentLogicalIndex(rows))

	m.ui.Selection.MoveTo(3) // Beta
	idx := m.currentLogicalIndex(rows)
	require.GreaterOrEqual(t, idx, 0)
	assert.Equal(t, selection.FocusTask, rows[idx].Kind)
	assert.Equal(t, 0, rows[idx].CategoryIndex)
	assert.Equal(t, 1, rows[idx].TaskIndex)
}

func TestSearchQuery_ReflectsModeAndActive(t *testing.T) {
	m := newTestModel(t, sampleProject())
	assert.Equal(t, "", m.searchQuery())

	m.startSearch()
	m.ui.Search.input.SetValue("typed")
	assert.Equal(t, "typed", m.searchQuery()) // live while in search mode

	m.ui.Modes.ToNormal()
	m.ui.Search.query = "committed"
	assert.Equal(t, "committed", m.searchQuery())
}
