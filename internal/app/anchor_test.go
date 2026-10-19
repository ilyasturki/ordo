package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

// cursorItem names the row under the cursor as "kind:id".
func cursorItem(t *testing.T, m *model) string {
	t.Helper()
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	kinds := map[selection.FocusKind]string{
		selection.FocusProject:     "project",
		selection.FocusCategory:    "category",
		selection.FocusTask:        "task",
		selection.FocusSeparator:   "separator",
		selection.FocusDescription: "description",
		selection.FocusHidden:      "hidden",
	}
	return kinds[pos.Kind] + ":" + pos.ID
}

func selectRow(t *testing.T, m *model, kind selection.FocusKind, id string) {
	t.Helper()
	require.True(t, m.ui.Selection.SelectByPredicate(func(p selection.Position) bool {
		return p.Kind == kind && p.ID == id
	}), "no %v row for %s", kind, id)
}

func describedProject() domain.Project {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "first body"
	p.Categories[0].Tasks[1].Description = "second body\nmore"
	return p
}

func TestExpandDescriptionsKeepsTheCursorOnItsTask(t *testing.T) {
	m := newTestModel(t, describedProject())
	selectRow(t, m, selection.FocusTask, "t3")

	m.toggleExpandDescriptions()
	assert.Equal(t, "task:t3", cursorItem(t, m), "rows inserted above must not move the cursor")

	selectRow(t, m, selection.FocusDescription, "t2")
	m.toggleExpandDescriptions()
	assert.Equal(t, "task:t2", cursorItem(t, m), "a hidden description hands the cursor to its task")
}

func TestFilterKeepsTheCursorOnItsTask(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectRow(t, m, selection.FocusTask, "t5")

	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	assert.Equal(t, "task:t5", cursorItem(t, m))

	selectRow(t, m, selection.FocusTask, "t4")
	m.ui.Filter.statuses = map[string]bool{domain.StatusCompleted: true}
	m.rebuildPositions()
	assert.Equal(t, "category:c2", cursorItem(t, m), "a filtered-out task hands the cursor to its category")
}

func TestFilterKeepsTheCursorInView(t *testing.T) {
	p := longProject()
	for i := range p.Categories[0].Tasks[:30] {
		p.Categories[0].Tasks[i].Status = domain.StatusCompleted
	}
	m := newTestModel(t, p)
	m.ui.Screen.Height = 12
	selectRow(t, m, selection.FocusTask, "t35")
	m.ensureVisible()
	require.Positive(t, m.ui.Screen.TopRow)

	m.ui.Modes.ToFilter()
	m.ui.Filter.SetView(FilterViewStatus)
	m.ui.Filter.statusSelected = 0
	next := m.handleFilterKey(tea.KeyPressMsg{Code: tea.KeySpace, Text: " "})
	m = &next
	assert.Equal(t, "task:t35", cursorItem(t, m))
	start, end, ok := m.buildLayout().rowRange(m.selected())
	require.True(t, ok)
	assert.GreaterOrEqual(t, start, m.ui.Screen.TopRow, "the cursor row must stay on screen")
	assert.LessOrEqual(t, end, m.ui.Screen.TopRow+m.availableHeight())
}

func TestFoldAllFromASeparatorLandsOnItsCategory(t *testing.T) {
	p := sampleProject()
	p.Categories[1].Tasks = append(p.Categories[1].Tasks, domain.Task{ID: "s1", Kind: domain.KindSeparator})
	m := newTestModel(t, p)
	withStubStateManager(m)
	selectRow(t, m, selection.FocusSeparator, "s1")

	m.foldAll()
	assert.Equal(t, "category:c2", cursorItem(t, m))
}

func TestRebuildAfterAnEditElsewhereKeepsTheCursor(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectRow(t, m, selection.FocusTask, "t4")

	m.project.Categories[0].Tasks = append([]domain.Task{{ID: "new", Title: "New", Status: domain.StatusTodo}}, m.project.Categories[0].Tasks...)
	m.rebuildPositions()
	assert.Equal(t, "task:t4", cursorItem(t, m))
}

func TestDeletingTheCursorTaskKeepsItsRow(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectRow(t, m, selection.FocusTask, "t2")
	m.deleteTask(selection.Position{Kind: selection.FocusTask, CategoryIndex: 0, TaskIndex: 1}, false)
	assert.Equal(t, "task:t3", cursorItem(t, m), "the row below takes the deleted task's place")
}

func TestUndoReturnsTheCursorToASeparator(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks = append(p.Categories[0].Tasks, domain.Task{ID: "s1", Kind: domain.KindSeparator})
	m := newTestModel(t, p)
	selectRow(t, m, selection.FocusSeparator, "s1")

	m.recordHistory()
	m.project.Categories[0].Tasks[0].Title = "renamed"
	selectRow(t, m, selection.FocusTask, "t4")
	m.undo()
	assert.Equal(t, "separator:s1", cursorItem(t, m))
}
