package app

import (
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/modes"
	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
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
			{ID: "v90", Name: "v0.9.0", ReleasedAt: shippedAt, Tasks: []domain.Task{
				{ID: "b1", Title: "Left over", Status: domain.StatusTodo},
			}},
			{ID: "fut", Name: "FUTURE", Tasks: []domain.Task{}},
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

func TestMovePickerListsInboxAndOpenVersionsWithAreas(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n1"))

	m.startMoveToVersion()

	require.True(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"v0.10.0", "v0.9.1", "  ui"}, targetLabels(m), "no inbox (the source), no shipped version, no plain category")
	assert.Equal(t, 1, m.ui.MovePicker.selected, "the lowest open version is the next to ship")
	assert.Equal(t, "Move 2 task(s) to", m.ui.MovePicker.title, "a parent brings its subtask")
}

func TestMovePickerFromAVersionOffersTheInboxAndNamesOrphanedAreasByPath(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "a1"))

	m.startMoveToVersion()

	assert.Equal(t, []string{"next", "v0.10.0", "v0.9.1 / ui"}, targetLabels(m))
	assert.Equal(t, 1, m.ui.MovePicker.selected)
}

func TestMoveKeepsIDsLeavesTheCursorAndUndoesInOneStep(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n1"))
	m.startMoveToVersion()

	next := m.handleMovePickerKey(keyPress("enter"))
	m = &next

	assert.False(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"n2", "n3"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, []string{"a1", "n1", "n1a"}, taskIDs(m.project.Categories[2]))
	assert.Equal(t, "n1", m.project.Categories[2].Tasks[2].ParentID, "the subtask keeps its parent")
	pos, _ := m.selectedPosition()
	assert.Equal(t, "n2", pos.ID, "the cursor stays in the inbox for the next triage")
	assert.Equal(t, "Moved 2 task(s) to v0.9.1", m.ui.Screen.StatusMsg)

	m.undo()
	assert.Equal(t, []string{"n1", "n1a", "n2", "n3"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, []string{"a1"}, taskIDs(m.project.Categories[2]))
	assert.False(t, m.ui.History.CanUndo())
}

func TestMovePickerRemembersTheLastTarget(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.startMoveToVersion()
	next := m.handleMovePickerKey(keyPress("k"))
	next = next.handleMovePickerKey(keyPress("enter"))
	m = &next
	assert.Equal(t, "v0.10.0", m.project.Categories[1].Name)
	assert.Equal(t, []string{"n2"}, taskIDs(m.project.Categories[1]))

	m.startMoveToVersion()
	assert.Equal(t, "v10", m.ui.MovePicker.targets[m.ui.MovePicker.selected].categoryID)
}

func TestMoveFromTheLastRowLandsOnTheRowAbove(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n3"))
	m.startMoveToVersion()
	next := m.handleMovePickerKey(keyPress("enter"))
	pos, _ := next.selectedPosition()
	assert.Equal(t, "n2", pos.ID)
}

func TestVisualMoveSendsTheRange(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusTask, "n2"))
	m.enterVisualMode()
	m.visualMoveCursor(1)

	m.visualMoveToVersion()
	require.True(t, m.ui.Modes.IsMovePicker())
	assert.Equal(t, []string{"n2", "n3"}, m.ui.MovePicker.taskIDs)
	next := m.handleMovePickerKey(keyPress("enter"))

	assert.Equal(t, []string{"a1", "n2", "n3"}, taskIDs(next.project.Categories[2]))
	pos, _ := next.selectedPosition()
	assert.Equal(t, "n1a", pos.ID)
}

func TestMoveRefusals(t *testing.T) {
	m := newMoveModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "inbox"))
	m.startMoveToVersion()
	assert.False(t, m.ui.Modes.IsMovePicker())
	assert.Contains(t, m.ui.Screen.StatusMsg, "on a task")

	plain := newTestModel(t, sampleProject())
	plain.rebuildPositions()
	require.True(t, plain.selectItem(selection.FocusTask, "t1"))
	plain.startMoveToVersion()
	assert.Equal(t, modes.ModeNormal, plain.ui.Modes.Current())
	assert.Contains(t, plain.ui.Screen.StatusMsg, "No open version")
}

func TestMovePickerScrollsALongList(t *testing.T) {
	p := domain.Project{ID: "p1", Categories: []domain.Category{{ID: "src", Name: "next", Tasks: []domain.Task{{ID: "x", Title: "x", Status: domain.StatusTodo}}}}}
	for i := range 30 {
		p.Categories = append(p.Categories, domain.Category{ID: string(rune('A' + i)), Name: domain.Semver{Minor: 30 - i}.String()})
	}
	m := newTestModel(t, p)
	m.ui.Screen.Height = 20
	m.rebuildPositions()
	require.True(t, m.selectItem(selection.FocusTask, "x"))
	m.startMoveToVersion()
	assert.Equal(t, 29, m.ui.MovePicker.selected, "v0.1.0 is the lowest")
	view := ansi.Strip(m.movePickerView())
	assert.Contains(t, view, "> v0.1.0")
	assert.Contains(t, view, "↑ 22 more")
	assert.NotContains(t, view, "v0.30.0")
}
