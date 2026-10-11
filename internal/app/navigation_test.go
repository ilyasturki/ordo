package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
)

func TestMoveSelection_AdvancesByDelta(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(0)
	m.moveSelection(2)
	assert.Equal(t, 2, m.ui.Selection.Selected())
}

func TestMoveSelection_ClampsAtBounds(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(0)
	m.moveSelection(-5)
	assert.Equal(t, 0, m.ui.Selection.Selected())

	m.moveSelection(999)
	last := m.ui.Selection.Count() - 1
	assert.Equal(t, last, m.ui.Selection.Selected())
}

func TestMoveSelection_NoopInBlockedMode(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	// In Edit mode, ActionNavigate is blocked.
	require.True(t, m.ui.Modes.ToEdit())
	m.moveSelection(3)
	assert.Equal(t, 2, m.ui.Selection.Selected())
}

func TestMoveSelectionByPage_UsesAvailableHeight(t *testing.T) {
	m := newTestModel(t, sampleProject())
	// Screen.Height = 40 in helper; FooterHeight is small, so availableHeight is ~37.
	// factor 0.5 ⇒ ~18, clamped to last position.
	m.ui.Selection.MoveTo(0)
	m.moveSelectionByPage(0.5)
	last := m.ui.Selection.Count() - 1
	assert.Equal(t, last, m.ui.Selection.Selected())
}

func TestMoveSelectionByPage_ZeroFactorStillMovesOne(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	// pageSize would round to 0; positive factor forces at least +1.
	m.moveSelectionByPage(0.0001)
	assert.Equal(t, 3, m.ui.Selection.Selected())

	m.ui.Selection.MoveTo(2)
	m.moveSelectionByPage(-0.0001)
	assert.Equal(t, 1, m.ui.Selection.Selected())
}

func TestMoveSelectionByPage_MovesByRowsNotItems(t *testing.T) {
	p := longProject()
	for i := range p.Categories[0].Tasks {
		p.Categories[0].Tasks[i].Description = "first line\nsecond line\nthird line"
	}
	m := newTestModel(t, p)
	m.ui.Screen.Height = 24
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusTask, "t00"))
	rowOf := func() int {
		start, _, ok := m.buildLayout().rowRange(m.selected())
		require.True(t, ok)
		return start
	}
	half := m.availableHeight() / 2

	from := rowOf()
	m.moveSelectionByPage(0.5)
	assert.InDelta(t, from+half, rowOf(), 3, "half a screen of rows, not of tasks")

	from = rowOf()
	m.moveSelectionByPage(-0.5)
	assert.InDelta(t, from-half, rowOf(), 3)
}

func TestJumpToFirst(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(5)
	m.jumpToFirst()
	assert.Equal(t, 0, m.ui.Selection.Selected())
}

func TestJumpToLast(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(0)
	m.jumpToLast()
	assert.Equal(t, m.ui.Selection.Count()-1, m.ui.Selection.Selected())
}

func TestJumpToNextCategory_FromTaskInFirstCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.jumpToNextCategory()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, 1, pos.CategoryIndex)
}

func TestJumpToNextCategory_NoopWhenNoFollowingCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(7) // t5 in last category
	m.jumpToNextCategory()
	assert.Equal(t, 7, m.ui.Selection.Selected())
}

func TestJumpToPrevCategory_FromTaskInSecondCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(7) // t5
	m.jumpToPrevCategory()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, 1, pos.CategoryIndex) // Cat B is "prev category" header from t5
}

func TestJumpToPrevCategory_NoopAtProjectRow(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(0) // project row
	m.jumpToPrevCategory()
	assert.Equal(t, 0, m.ui.Selection.Selected())
}

func TestNavigation_NoopOnEmptySelection(t *testing.T) {
	m := newTestModel(t, sampleProject())
	// Replace with an empty manager.
	m.ui.Selection = selection.NewManager(nil, 0)
	require.True(t, m.ui.Selection.IsEmpty())
	// None of these should panic.
	m.moveSelection(1)
	m.moveSelectionByPage(0.5)
	m.jumpToFirst()
	m.jumpToLast()
	m.jumpToNextCategory()
	m.jumpToPrevCategory()
}

func TestJumpToFirst_NoopInBlockedMode(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(5)
	m.ui.Modes = modes.NewMachine(modes.ModeEdit)
	m.jumpToFirst()
	assert.Equal(t, 5, m.ui.Selection.Selected())
}
