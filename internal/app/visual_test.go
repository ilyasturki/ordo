package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/phasionary/internal/app/modes"
	"github.com/ilyasturki/phasionary/internal/app/selection"
	"github.com/ilyasturki/phasionary/internal/config"
	"github.com/ilyasturki/phasionary/internal/domain"
)

type stubConfigReader struct{ cfg config.Config }

func (s *stubConfigReader) Get() config.Config                   { return s.cfg }
func (s *stubConfigReader) Update(fn func(*config.Config)) error { fn(&s.cfg); return nil }

func newTestModel(t *testing.T, project domain.Project) *model {
	t.Helper()
	positions := rebuildPositions(project.Categories, nil, nil, false, "")
	sel := selection.NewManager(positions, 0)
	mode := modes.NewMachine(modes.ModeNormal)
	ui := NewUIState(sel, mode)
	ui.Screen.Width = 80
	ui.Screen.Height = 40
	m := &model{
		project: project,
		ui:      ui,
		deps:    &Dependencies{CfgManager: &stubConfigReader{cfg: config.DefaultConfig()}},
	}
	return m
}

func taskIDs(cat domain.Category) []string {
	ids := make([]string, len(cat.Tasks))
	for i, t := range cat.Tasks {
		ids[i] = t.ID
	}
	return ids
}

func sampleProject() domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{
				ID:   "c1",
				Name: "Cat A",
				Tasks: []domain.Task{
					{ID: "t1", Title: "Alpha", Status: domain.StatusTodo},
					{ID: "t2", Title: "Beta", Status: domain.StatusInProgress},
					{ID: "t3", Title: "Gamma", Status: domain.StatusCompleted},
				},
			},
			{
				ID:   "c2",
				Name: "Cat B",
				Tasks: []domain.Task{
					{ID: "t4", Title: "Delta", Status: domain.StatusTodo},
					{ID: "t5", Title: "Epsilon", Status: domain.StatusTodo},
				},
			},
		},
	}
}

func TestNextVisualPosition_SkipsCategoryRows(t *testing.T) {
	positions := []selection.Position{
		{Kind: selection.FocusProject},
		{Kind: selection.FocusCategory, CategoryIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 0},
		{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1},
		{Kind: selection.FocusCategory, CategoryIndex: 1},
		{Kind: selection.FocusTask, CategoryIndex: 1, TaskIndex: 0},
	}

	// from a task, +1 with kind=Task should jump over the category row
	got := nextVisualPosition(positions, 3, 1, selection.FocusTask)
	assert.Equal(t, 5, got, "should skip category row at index 4")

	// from a category, +1 with kind=Category should jump over tasks
	got = nextVisualPosition(positions, 1, 1, selection.FocusCategory)
	assert.Equal(t, 4, got, "should skip task rows to next category")

	// At end, stays put
	got = nextVisualPosition(positions, 5, 1, selection.FocusTask)
	assert.Equal(t, 5, got)
}

func TestEnterVisualMode_FromTaskRow(t *testing.T) {
	m := newTestModel(t, sampleProject())
	// Position 0 is project, 1 is Cat A header, 2 is t1
	m.ui.Selection.MoveTo(2)
	m.enterVisualMode()
	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, "c1", m.ui.Visual.AnchorCategoryID)
	assert.Equal(t, "t1", m.ui.Visual.AnchorTaskID)
	assert.Equal(t, selection.FocusTask, m.ui.Visual.Kind)
}

func TestEnterVisualMode_RejectedOnProjectRow(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(0)
	m.enterVisualMode()
	assert.False(t, m.ui.Modes.IsVisual())
	assert.Contains(t, m.ui.Screen.StatusMsg, "Visual mode")
}

func TestVisualMoveCursor_ExtendsRangeAcrossCategories(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(3) // should land on t4 (skipping the Cat B header)

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 1, pos.CategoryIndex)
	assert.Equal(t, 0, pos.TaskIndex)

	selPositions := m.visualSelectedPositions()
	assert.Len(t, selPositions, 4, "anchor + 3 more tasks")
}

func TestVisualCut_StoresMultipleTasksOnClipboard(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t2
	m.visualCut()

	assert.False(t, m.ui.Modes.IsVisual())
	assert.True(t, m.ui.Clipboard.IsCut)
	assert.Len(t, m.ui.Clipboard.Tasks, 2)
	assert.Equal(t, []string{"t1", "t2"}, m.ui.Clipboard.TaskIDs)
}

func TestPasteMultiTasks_FlattensIntoCursorCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t2
	m.visualCut()

	// Move cursor into Cat B (position 5 = Cat B header, 6 = t4)
	// After visual cut, positions still valid. Land on t4.
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})

	m.pasteInternal()

	// Cat A should now have just t3
	require.Len(t, m.project.Categories[0].Tasks, 1)
	assert.Equal(t, "t3", m.project.Categories[0].Tasks[0].ID)

	// Cat B should now have t4, then pasted t1, t2 (after cursor), then t5
	require.Len(t, m.project.Categories[1].Tasks, 4)
	assert.Equal(t, "Delta", m.project.Categories[1].Tasks[0].Title)
	assert.Equal(t, "Alpha", m.project.Categories[1].Tasks[1].Title)
	assert.Equal(t, "Beta", m.project.Categories[1].Tasks[2].Title)
	assert.Equal(t, "Epsilon", m.project.Categories[1].Tasks[3].Title)

	assert.False(t, m.ui.Clipboard.IsCut, "the move used the cut up")
}

func TestVisualYank_PopulatesInternalClipboardForPaste(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t2

	_ = m.visualCopyBullets() // y

	// After yank: clipboard should hold the two tasks, NOT marked as cut.
	require.Len(t, m.ui.Clipboard.Tasks, 2)
	assert.False(t, m.ui.Clipboard.IsCut)

	// Move cursor into Cat B and paste — sources should remain in Cat A.
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})
	m.pasteInternal()

	// Cat A unchanged (still 3 tasks t1, t2, t3)
	require.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Equal(t, "t1", m.project.Categories[0].Tasks[0].ID)
	assert.Equal(t, "t2", m.project.Categories[0].Tasks[1].ID)

	// Cat B: cursor was on t4 (Delta); paste inserts AFTER cursor.
	require.Len(t, m.project.Categories[1].Tasks, 4)
	assert.Equal(t, "Delta", m.project.Categories[1].Tasks[0].Title)
	assert.Equal(t, "Alpha", m.project.Categories[1].Tasks[1].Title)
	assert.Equal(t, "Beta", m.project.Categories[1].Tasks[2].Title)
	assert.Equal(t, "Epsilon", m.project.Categories[1].Tasks[3].Title)
}

func TestVisualDelete_OpensConfirmThenRemovesAll(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(2) // t1..t3
	m.visualDelete()

	// Confirmation opened with all 3 IDs captured.
	assert.True(t, m.ui.Modes.IsConfirmDelete())
	assert.Equal(t, ConfirmDeleteVisualRange, m.ui.ConfirmDelete.Kind)
	assert.Equal(t, []string{"t1", "t2", "t3"}, m.ui.ConfirmDelete.TaskIDs)

	m.confirmDeleteVisualRange()

	// Cat A is now empty; Cat B untouched.
	assert.Empty(t, m.project.Categories[0].Tasks)
	require.Len(t, m.project.Categories[1].Tasks, 2)
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Contains(t, m.ui.Screen.StatusMsg, "Deleted 3")
}

func TestVisualDelete_CategoryRangeRemovesCategoriesAndTasks(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.enterVisualMode()
	m.visualMoveCursor(1) // Cat A + Cat B
	m.visualDelete()

	assert.Equal(t, []string{"c1", "c2"}, m.ui.ConfirmDelete.CategoryIDs)
	m.confirmDeleteVisualRange()

	assert.Empty(t, m.project.Categories)
	assert.Contains(t, m.ui.Screen.StatusMsg, "Deleted 2 categories")
}

func TestVisualMode_TaskAnchorDoesNotEscapeToCategories(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	// Try to go backwards into a category row — anchor on task, kind=Task
	m.visualMoveCursor(-1)
	// Should still be on t1 since the previous row is a category header (no earlier task)
	assert.Equal(t, 2, m.ui.Selection.Selected())
}

func TestVisualSwap_ExchangesAnchorAndCursor(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // cursor at t2, anchor at t1

	require.Equal(t, 3, m.ui.Selection.Selected())
	require.Equal(t, "t1", m.ui.Visual.AnchorTaskID)

	m.visualSwap()

	// Anchor now stable on t2, cursor moved back to t1's index.
	assert.Equal(t, "t2", m.ui.Visual.AnchorTaskID)
	assert.Equal(t, 2, m.ui.Selection.Selected())

	// Range still covers the same two tasks.
	selPositions := m.visualSelectedPositions()
	require.Len(t, selPositions, 2)
	assert.Equal(t, 0, selPositions[0].TaskIndex)
	assert.Equal(t, 1, selPositions[1].TaskIndex)

	// Swapping again restores the original orientation.
	m.visualSwap()
	assert.Equal(t, "t1", m.ui.Visual.AnchorTaskID)
	assert.Equal(t, 3, m.ui.Selection.Selected())
}

func TestVisualSwap_NoopWhenAnchorEqualsCursor(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2)
	m.enterVisualMode()
	// Anchor and cursor are both on t1 — swap should leave state unchanged.
	m.visualSwap()
	assert.Equal(t, "t1", m.ui.Visual.AnchorTaskID)
	assert.Equal(t, 2, m.ui.Selection.Selected())
}

func TestVisualSwap_ExtendsFromSwappedEnd(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(2) // cursor at t3, anchor at t1, range = t1..t3

	m.visualSwap() // anchor now t3, cursor on t1
	m.visualMoveCursor(-1)
	// Cursor should have stayed at t1 since there's no earlier task in Cat A.
	assert.Equal(t, 2, m.ui.Selection.Selected())

	// Now extend from the anchor end downward by swapping back and growing.
	m.visualSwap()        // anchor t1, cursor t3
	m.visualMoveCursor(1) // cursor t4 (skips Cat B header)
	selPositions := m.visualSelectedPositions()
	assert.Len(t, selPositions, 4)
}

func TestVisualMoveDown_ShiftsTaskBlockWithinCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t2 → block [t1, t2], cursor on t2

	m.visualMoveDown()

	// Cat A order is now t3, t1, t2
	require.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Equal(t, []string{"t3", "t1", "t2"},
		[]string{
			m.project.Categories[0].Tasks[0].ID,
			m.project.Categories[0].Tasks[1].ID,
			m.project.Categories[0].Tasks[2].ID,
		})

	// Cursor follows t2 to its new position.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, 0, pos.CategoryIndex)
	assert.Equal(t, 2, pos.TaskIndex)

	// Anchor (t1) is still part of the range.
	assert.Equal(t, "t1", m.ui.Visual.AnchorTaskID)
	selPositions := m.visualSelectedPositions()
	require.Len(t, selPositions, 2)
	assert.Equal(t, 1, selPositions[0].TaskIndex)
	assert.Equal(t, 2, selPositions[1].TaskIndex)
}

func TestVisualMoveUp_ShiftsTaskBlockWithinCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(3) // t2
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t3 → block [t2, t3]

	m.visualMoveUp()

	require.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Equal(t, []string{"t2", "t3", "t1"},
		[]string{
			m.project.Categories[0].Tasks[0].ID,
			m.project.Categories[0].Tasks[1].ID,
			m.project.Categories[0].Tasks[2].ID,
		})
}

func TestVisualMoveDown_NoOpAtBottomOfLastCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	// Select t4, t5 — the whole of Cat B, the final category.
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t5 → block reaches end of last category

	m.visualMoveDown()

	// No category below to receive the block: order unchanged.
	assert.Equal(t, []string{"t4", "t5"},
		[]string{
			m.project.Categories[1].Tasks[0].ID,
			m.project.Categories[1].Tasks[1].ID,
		})
}

func TestVisualMoveDown_CrossesBlockIntoNextCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(3) // t2
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t3 → block [t2, t3] at end of Cat A

	m.visualMoveDown()

	// Block lands at the top of Cat B, in order.
	assert.Equal(t, []string{"t1"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, []string{"t2", "t3", "t4", "t5"}, taskIDs(m.project.Categories[1]))

	// Cursor follows t3; anchor (t2) still anchors the range.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusTask, pos.Kind)
	assert.Equal(t, "t3", m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].ID)
	assert.Equal(t, "t2", m.ui.Visual.AnchorTaskID)
	selPositions := m.visualSelectedPositions()
	require.Len(t, selPositions, 2)
	assert.Equal(t, "t2", m.project.Categories[selPositions[0].CategoryIndex].Tasks[selPositions[0].TaskIndex].ID)
	assert.Equal(t, "t3", m.project.Categories[selPositions[1].CategoryIndex].Tasks[selPositions[1].TaskIndex].ID)
}

func TestVisualMoveUp_CrossesBlockIntoPrevCategory(t *testing.T) {
	m := newTestModel(t, sampleProject())
	// Select t4, t5 — the whole of Cat B, anchored at its top.
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusTask && p.CategoryIndex == 1 && p.TaskIndex == 0
	})
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t5

	m.visualMoveUp()

	// Block lands at the bottom of Cat A, in order; Cat B is now empty.
	assert.Equal(t, []string{"t1", "t2", "t3", "t4", "t5"}, taskIDs(m.project.Categories[0]))
	assert.Empty(t, m.project.Categories[1].Tasks)
}

func TestVisualMoveDown_NoOpAcrossCategories(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(4) // t3 (last in Cat A)
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t4 (first in Cat B)

	m.visualMoveDown()

	// No mutation — block spans categories.
	assert.Equal(t, []string{"t1", "t2", "t3"},
		[]string{
			m.project.Categories[0].Tasks[0].ID,
			m.project.Categories[0].Tasks[1].ID,
			m.project.Categories[0].Tasks[2].ID,
		})
	assert.Equal(t, []string{"t4", "t5"},
		[]string{
			m.project.Categories[1].Tasks[0].ID,
			m.project.Categories[1].Tasks[1].ID,
		})
}

func TestVisualMoveDown_ShiftsCategoryBlock(t *testing.T) {
	project := sampleProject()
	project.Categories = append(project.Categories, domain.Category{ID: "c3", Name: "Cat C"})
	m := newTestModel(t, project)
	m.ui.Selection.MoveTo(1) // Cat A header
	m.enterVisualMode()
	m.visualMoveCursor(1) // extend to Cat B → block [Cat A, Cat B]

	m.visualMoveDown()

	require.Len(t, m.project.Categories, 3)
	assert.Equal(t, []string{"c3", "c1", "c2"},
		[]string{
			m.project.Categories[0].ID,
			m.project.Categories[1].ID,
			m.project.Categories[2].ID,
		})
	// Anchor stable on Cat A.
	assert.Equal(t, "c1", m.ui.Visual.AnchorCategoryID)
	// Range still covers Cat A + Cat B.
	selPositions := m.visualSelectedPositions()
	require.Len(t, selPositions, 2)
	assert.Equal(t, "c1", m.project.Categories[selPositions[0].CategoryIndex].ID)
	assert.Equal(t, "c2", m.project.Categories[selPositions[1].CategoryIndex].ID)
}

func TestVisualMoveUp_ShiftsCategoryBlock(t *testing.T) {
	project := sampleProject()
	project.Categories = append(project.Categories, domain.Category{ID: "c3", Name: "Cat C"})
	m := newTestModel(t, project)
	// Start on Cat B (position 5 = Cat B header in [Project, CatA, t1, t2, t3, CatB, t4, t5, CatC])
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == selection.FocusCategory && p.CategoryIndex == 1
	})
	m.enterVisualMode()
	m.visualMoveCursor(1) // extend to Cat C

	m.visualMoveUp()

	require.Len(t, m.project.Categories, 3)
	assert.Equal(t, []string{"c2", "c3", "c1"},
		[]string{
			m.project.Categories[0].ID,
			m.project.Categories[1].ID,
			m.project.Categories[2].ID,
		})
}

func TestVisualMoveUp_NoOpAtTop(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	m.enterVisualMode()

	m.visualMoveUp()

	assert.Equal(t, []string{"c1", "c2"},
		[]string{
			m.project.Categories[0].ID,
			m.project.Categories[1].ID,
		})
}

func TestVisualMode_CategoryAnchorSelectsOnlyCategories(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	m.enterVisualMode()
	assert.Equal(t, selection.FocusCategory, m.ui.Visual.Kind)

	m.visualMoveCursor(1) // should jump to Cat B header
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, 1, pos.CategoryIndex)

	selPositions := m.visualSelectedPositions()
	assert.Len(t, selPositions, 2)
	for _, p := range selPositions {
		assert.Equal(t, selection.FocusCategory, p.Kind)
	}
}

// newTestModelWithDescription builds a model on sampleProject with t1 carrying
// an expanded description, so the position list is:
// 0 project, 1 Cat A, 2 t1, 3 t1-description, 4 t2, 5 t3, 6 Cat B, 7 t4, 8 t5.
func newTestModelWithDescription(t *testing.T) *model {
	t.Helper()
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "alpha details"
	m := newTestModel(t, p)
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()
	return m
}

func TestEnterVisualMode_FromDescriptionRow(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(3) // t1's description row
	m.enterVisualMode()

	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, selection.FocusTask, m.ui.Visual.Kind)
	assert.Equal(t, "t1", m.ui.Visual.AnchorTaskID)
	assert.True(t, m.ui.Visual.AnchorOnDescription)
	// The anchor resolves to the description row, not the task row above it.
	assert.Equal(t, 3, m.resolveVisualAnchor())
}

func TestVisualMoveCursor_LandsOnDescriptionRows(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1)

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusDescription, pos.Kind)
	assert.True(t, m.isInVisualRange(3))
}

func TestVisualCopyBullets_NestsSelectedDescriptionBody(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(2) // t1, t1-description, t2

	text := m.visualMarkdownText(m.visualSelectedPositions(), false)
	assert.Equal(t, "- Alpha\n  alpha details\n- Beta", text)
}

func TestVisualCopyBullets_LoneDescriptionYieldsIndentedBody(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(3) // t1's description row
	m.enterVisualMode()

	text := m.visualMarkdownText(m.visualSelectedPositions(), false)
	assert.Equal(t, "  alpha details", text)
}

func TestVisualChecklist_NestsDescriptionUnderItem(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t1's description

	text := m.visualMarkdownText(m.visualSelectedPositions(), true)
	assert.Equal(t, "- [ ] Alpha\n  alpha details", text)
}

func TestVisualDelete_RoundsUpOrphanDescription(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(3) // t1's description row, t1 itself stays above the range
	m.enterVisualMode()
	m.visualMoveCursor(1) // extend down to t2
	m.visualDelete()

	// The orphan description pulls its parent in, without duplicates.
	assert.Equal(t, []string{"t1", "t2"}, m.ui.ConfirmDelete.TaskIDs)
	m.confirmDeleteVisualRange()

	require.Len(t, m.project.Categories[0].Tasks, 1)
	assert.Equal(t, "t3", m.project.Categories[0].Tasks[0].ID)
	assert.Contains(t, m.ui.Screen.StatusMsg, "Deleted 2")
}

func TestVisualCut_DedupesTaskWithItsDescription(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t1's description
	m.visualCut()

	assert.Equal(t, []string{"t1"}, m.ui.Clipboard.TaskIDs)
	assert.Len(t, m.ui.Clipboard.Tasks, 1)
	assert.Contains(t, m.ui.Screen.StatusMsg, "1 task(s)")
}

func TestVisualCycleTag_AppliesOncePerTask(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t1's description
	m.visualCycleTag()

	// Task + its description row count as one task: the tag advances one step.
	assert.Equal(t, domain.NextTagColor(""), m.project.Categories[0].Tasks[0].TagColor)
}

func TestVisualMoveDown_CursorFollowsDescriptionRow(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // cursor on t1's description
	m.visualMoveDown()

	// t2 hopped above the block.
	assert.Equal(t, []string{"t2", "t1", "t3"}, taskIDs(m.project.Categories[0]))
	// The cursor re-attaches to t1's description row at its new index.
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusDescription, pos.Kind)
	assert.Equal(t, 1, pos.TaskIndex)
	// The range still covers t1 and its description.
	assert.Len(t, m.visualSelectedPositions(), 2)
}

func TestVisualStatusText_CountsRoundedUpTasks(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(3) // t1's description row
	m.enterVisualMode()

	// A lone description row still selects its parent task for every op.
	assert.Contains(t, m.statusText(), "1 task selected")

	m.visualMoveCursor(-1) // pull in t1's own row: still one task
	assert.Contains(t, m.statusText(), "1 task selected")
}

func TestVisualSwap_PreservesDescriptionAnchor(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // cursor on t1's description
	m.visualSwap()

	assert.True(t, m.ui.Visual.AnchorOnDescription)
	assert.Equal(t, 3, m.resolveVisualAnchor())
	assert.Equal(t, 2, m.ui.Selection.Selected())
}

func TestVisualCycleStatus_AppliesSharedNextStatusToWholeRange(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1 (todo)
	m.enterVisualMode()
	m.visualMoveCursor(2) // extend through t2 (in_progress) and t3 (completed)

	m.visualCycleStatus(false)

	// The whole block steps from the first task's status, not each its own.
	cat := m.project.Categories[0]
	assert.Equal(t, domain.StatusInProgress, cat.Tasks[0].Status)
	assert.Equal(t, domain.StatusInProgress, cat.Tasks[1].Status)
	assert.Equal(t, domain.StatusInProgress, cat.Tasks[2].Status)
	assert.True(t, m.ui.Modes.IsVisual(), "stays in visual mode for repeated presses")

	// A second press walks the block on together.
	m.visualCycleStatus(false)
	cat = m.project.Categories[0]
	assert.Equal(t, domain.StatusCompleted, cat.Tasks[0].Status)
	assert.Equal(t, domain.StatusCompleted, cat.Tasks[2].Status)
}

func TestVisualCycleStatus_ReverseWalksBackwards(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1 (todo)
	m.enterVisualMode()
	m.visualMoveCursor(1)

	m.visualCycleStatus(true)

	cat := m.project.Categories[0]
	assert.Equal(t, domain.StatusCancelled, cat.Tasks[0].Status)
	assert.Equal(t, domain.StatusCancelled, cat.Tasks[1].Status)
}

func TestVisualCycleStatus_SkipsSeparatorsInRange(t *testing.T) {
	m := newTestModel(t, separatorProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(2) // sweep the separator and land on t2

	m.visualCycleStatus(false)

	cat := m.project.Categories[0]
	assert.Equal(t, domain.StatusInProgress, cat.Tasks[0].Status)
	assert.Empty(t, cat.Tasks[1].Status, "separator carries no status")
	assert.Equal(t, domain.StatusInProgress, cat.Tasks[2].Status)
}

func TestVisualCycleStatus_NoOpOnCategorySelection(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	m.enterVisualMode()
	require.Equal(t, selection.FocusCategory, m.ui.Visual.Kind)

	m.visualCycleStatus(false)

	assert.Equal(t, domain.StatusTodo, m.project.Categories[0].Tasks[0].Status)
}

func TestVisualCycleStatus_RoundsDescriptionUpToParentOnce(t *testing.T) {
	m := newTestModelWithDescription(t)
	m.ui.Selection.MoveTo(3) // t1's description row only
	m.enterVisualMode()

	m.visualCycleStatus(false)

	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[0].Status)
}

func TestEscCancelsVisualCut(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(2) // t1
	m.enterVisualMode()
	m.visualMoveCursor(1) // include t2
	m.visualCut()
	require.True(t, m.isTaskCut("t1"))

	m.dispatchNormalKey("esc")
	assert.False(t, m.ui.Clipboard.IsCut)
	assert.Empty(t, m.ui.Clipboard.TaskIDs)
	assert.False(t, m.isTaskCut("t1"))
	assert.False(t, m.isTaskCut("t2"))
	assert.Equal(t, "Cut cancelled", m.ui.Screen.StatusMsg)
}
