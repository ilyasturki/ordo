package app

import (
	"fmt"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func moveProject() domain.Project {
	return domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{ID: "inbox", Name: "next", Tasks: []domain.Task{
				{ID: "n1", Title: "Idea", Status: domain.StatusTodo},
				{ID: "n1a", ParentID: "n1", Title: "Detail", Status: domain.StatusTodo},
				{ID: "n2", Title: "Other", Status: domain.StatusTodo},
				{ID: "n3", Title: "Third", Status: domain.StatusTodo},
			}},
			{ID: "v10", Name: "v0.10.0", Tasks: []domain.Task{}},
			{ID: "v91", Name: "v0.9.1", Tasks: []domain.Task{
				{ID: "a1", Title: "Menu", Status: domain.StatusTodo},
			}},
			{ID: "v91ui", Name: "ui", ParentID: "v91", Tasks: []domain.Task{}},
			{ID: "v90", Name: "v0.9.0", Tasks: []domain.Task{
				{ID: "b1", Title: "Left over", Status: domain.StatusTodo},
			}},
			{ID: "fut", Name: "MARKETING + SEO", Tasks: []domain.Task{
				{ID: "f1", Title: "Landing page", Status: domain.StatusTodo},
			}},
		},
	}
}

func newMoveModel(t *testing.T) *model {
	t.Helper()
	m := newTestModel(t, moveProject())
	withStubStateManager(m)
	m.rebuildPositions()
	return m
}

func targetLabels(m *model) []string {
	var out []string
	for _, t := range m.ui.MovePicker.targets {
		out = append(out, t.label)
	}
	return out
}

func selectedTarget(m *model) string {
	return m.ui.MovePicker.targets[m.ui.MovePicker.selected].categoryID
}

func movePickerKeys(m *model, keys ...string) *model {
	for _, k := range keys {
		next, _ := m.handleMovePickerKey(keyPress(k))
		m = &next
	}
	return m
}

func TestMovePickerListsEveryOtherCategory(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n1"))

	m.startMove()

	require.True(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"v0.10.0", "v0.9.1", "  ui", "v0.9.0", "MARKETING + SEO"}, targetLabels(m), "every category but the source, in project order")
	assert.Equal(t, "v10", selectedTarget(m), "the category right after the source")
	assert.Equal(t, "Move 2 task(s) to", m.ui.MovePicker.title, "a parent brings its subtask")
}

func TestMovePickerFromAParentNamesItsSubCategoriesByPath(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "a1"))

	m.startMove()

	assert.Equal(t, []string{"next", "v0.10.0", "v0.9.1 / ui", "v0.9.0", "MARKETING + SEO"}, targetLabels(m))
	assert.Equal(t, "v91ui", selectedTarget(m))
}

func TestMovePickerFromTheLastCategoryPreselectsTheOneBefore(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "f1"))

	m.startMove()

	assert.Equal(t, "v90", selectedTarget(m))
}

func TestMoveKeepsIDsLeavesTheCursorAndUndoesInOneStep(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n1"))
	m.startMove()

	m = movePickerKeys(m, "j", "enter")

	assert.False(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"n2", "n3"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, []string{"a1", "n1", "n1a"}, taskIDs(m.project.Categories[2]))
	assert.Equal(t, "n1", m.project.Categories[2].Tasks[2].ParentID, "the subtask keeps its parent")
	pos, _ := m.selectedPosition()
	assert.Equal(t, "n2", pos.ID, "the cursor stays in the source for the next triage")
	assert.Equal(t, "Moved 2 task(s) to v0.9.1", m.ui.Screen.StatusMsg)

	m.undo()
	assert.Equal(t, []string{"n1", "n1a", "n2", "n3"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, []string{"a1"}, taskIDs(m.project.Categories[2]))
	assert.False(t, m.ui.History.CanUndo())
}

func TestMoveToANonVersionCategory(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "b1"))
	m.startMove()

	m = movePickerKeys(m, "enter")

	assert.Equal(t, []string{"f1", "b1"}, taskIDs(m.project.Categories[5]), "tasks land at the end")
	assert.Equal(t, "Moved to MARKETING + SEO", m.ui.Screen.StatusMsg)
}

func TestMovePickerRemembersTheLastTarget(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMove()
	m = movePickerKeys(m, "G", "enter")
	assert.Equal(t, []string{"f1", "n2"}, taskIDs(m.project.Categories[5]))

	m.startMove()
	assert.Equal(t, "fut", selectedTarget(m))
}

func TestMovePickerForgetsALastTargetThatIsGone(t *testing.T) {
	m := newMoveModel(t)
	m.ui.MovePicker.last = "deleted"
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMove()
	assert.Equal(t, "v10", selectedTarget(m))
}

func TestMoveFromTheLastRowLandsOnTheRowAbove(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n3"))
	m.startMove()
	m = movePickerKeys(m, "enter")
	pos, _ := m.selectedPosition()
	assert.Equal(t, "n2", pos.ID)
}

func TestMoveToAFoldedCategorySaysSo(t *testing.T) {
	m := newMoveModel(t)
	m.ui.Fold.Toggle("v10")
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMove()

	m = movePickerKeys(m, "enter")

	assert.Equal(t, []string{"n2"}, taskIDs(m.project.Categories[1]))
	assert.Equal(t, "Moved to v0.10.0 (folded)", m.ui.Screen.StatusMsg)
	assert.True(t, m.ui.Fold.IsFolded("v10"), "the move doesn't unfold it")
}

func TestMovePickerFilter(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMove()

	m = movePickerKeys(m, "/", "u", "i")
	require.True(t, m.ui.MovePicker.filtering)
	require.Len(t, m.ui.MovePicker.targets, 1)
	assert.Equal(t, "v91ui", selectedTarget(m))
	view := ansi.Strip(m.movePickerView())
	assert.Contains(t, view, "/ui")
	assert.Contains(t, view, "> v0.9.1 / ui", "a filtered sub-category shows its path")

	m = movePickerKeys(m, "esc")
	assert.False(t, m.ui.MovePicker.filtering)
	assert.True(t, m.ui.Modes.IsMovePicker(), "esc clears the filter first")
	assert.Len(t, m.ui.MovePicker.targets, 5)
	assert.Equal(t, "v91ui", selectedTarget(m), "the highlighted target survives the clear")

	m = movePickerKeys(m, "/", "M", "A", "R", "enter")
	assert.False(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"f1", "n2"}, taskIDs(m.project.Categories[5]))
}

func TestMovePickerFilterTakesJAsText(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMove()

	m = movePickerKeys(m, "/", "j")
	assert.Equal(t, "j", m.ui.MovePicker.query)
	assert.Empty(t, m.ui.MovePicker.targets)
	assert.Contains(t, ansi.Strip(m.movePickerView()), "no matches")

	m = movePickerKeys(m, "enter")
	assert.True(t, m.ui.Modes.IsMovePicker(), "enter with no match does nothing")
	assert.Equal(t, []string{"n1", "n1a", "n2", "n3"}, taskIDs(m.project.Categories[0]))
}

func TestVisualMoveSendsTheRange(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.enterVisualMode()
	m.visualMoveCursor(1)

	m.visualMove()
	require.True(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"n2", "n3"}, m.ui.MovePicker.taskIDs)
	m = movePickerKeys(m, "j", "enter")

	assert.Equal(t, []string{"a1", "n2", "n3"}, taskIDs(m.project.Categories[2]))
	pos, _ := m.selectedPosition()
	assert.Equal(t, "n1a", pos.ID)
	m.undo()
	assert.Equal(t, []string{"n1", "n1a", "n2", "n3"}, taskIDs(m.project.Categories[0]))
}

func TestVisualMoveRefusesCategories(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v10"))
	m.enterVisualMode()
	m.visualMoveCursor(1)

	m.visualMove()

	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, "Select tasks to move them", m.ui.Screen.StatusMsg)
}

func TestMoveRefusals(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "inbox"))
	m.startMove()
	assert.False(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, "Put the cursor on a task to move it", m.ui.Screen.StatusMsg)

	lone := newTestModel(t, domain.Project{ID: "p1", Categories: []domain.Category{
		{ID: "c1", Name: "todo", Tasks: []domain.Task{{ID: "t1", Title: "Alpha", Status: domain.StatusTodo}}},
	}})
	lone.rebuildPositions()
	require.True(t, lone.selectItem(selection.FocusTask, "t1"))
	lone.startMove()
	assert.Equal(t, modes.ModeNormal, lone.ui.Modes.Current())
	assert.Equal(t, "No other category to move to", lone.ui.Screen.StatusMsg)
}

func TestMKeyOpensThePicker(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))

	m, _ = pressKeys(m, "M")
	assert.Equal(t, modes.ModeNormal, m.ui.Modes.Current(), "M is free")

	m, _ = pressKeys(m, "m")
	assert.True(t, m.ui.Modes.IsMovePicker())
	m, _ = pressKeys(m, "esc")
	assert.Equal(t, modes.ModeNormal, m.ui.Modes.Current())
}

func TestMovePickerScrollsALongList(t *testing.T) {
	p := domain.Project{ID: "p1", Categories: []domain.Category{{ID: "src", Name: "inbox", Tasks: []domain.Task{{ID: "x", Title: "x", Status: domain.StatusTodo}}}}}
	for i := range 30 {
		p.Categories = append(p.Categories, domain.Category{ID: fmt.Sprintf("c%02d", i+1), Name: fmt.Sprintf("cat %02d", i+1)})
	}
	m := newTestModel(t, p)
	m.ui.Screen.Height = 20
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusTask, "x"))
	m.startMove()
	m = movePickerKeys(m, "G")
	view := ansi.Strip(m.movePickerView())
	assert.Contains(t, view, "> cat 30")
	assert.Contains(t, view, "↑ 22 more")
	assert.NotContains(t, view, "cat 01")

	m = sendMsg(m, tea.WindowSizeMsg{Width: m.ui.Screen.Width, Height: 16})
	assert.Contains(t, ansi.Strip(m.movePickerView()), "> cat 30", "a shrink keeps the selection on screen")
}
