package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func visualLabel(m *model) string {
	return m.visualRowsLabel(m.visualSelectedPositions())
}

func TestVisualShiftOfPeekedRowsKeepsThemSelected(t *testing.T) {
	m, _ := pressKeys(peekingModel(t), "v", "j")
	require.Equal(t, "-- VISUAL -- 2 tasks selected", m.statusText())

	m, _ = pressKeys(m, "J")
	assert.Equal(t, []string{"t1"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, []string{"t2", "t3", "t4", "t5"}, taskIDs(m.project.Categories[1]))
	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, "2 tasks", visualLabel(m))
	assert.Equal(t, "Hidden by the filter once you move off it", m.statusText())
	assert.Contains(t, rowIDs(m), "task:t2")
	assert.Contains(t, rowIDs(m), "task:t3")

	m, _ = pressKeys(m, "esc", "k", "k")
	assert.Equal(t, "category:c2", cursorItem(t, m))
	assert.NotContains(t, rowIDs(m), "task:t2", "hidden once the cursor leaves them in normal mode")
}

func TestVisualShiftUpOfPeekedRowsKeepsThemSelected(t *testing.T) {
	m := newTodoFilteredModel(t)
	m.project.Categories[1].Tasks[0].Status = domain.StatusCompleted
	m.rebuildPositions()
	selectRow(t, m, selection.FocusHidden, "c2")
	m, _ = pressKeys(m, "enter", "v", "K")

	assert.Equal(t, []string{"t1", "t2", "t3", "t4"}, taskIDs(m.project.Categories[0]))
	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, "1 task", visualLabel(m))
}

func TestShiftOfAPeekedRowKeepsItUnderTheCursor(t *testing.T) {
	m, _ := pressKeys(peekingModel(t), "j", "J")
	assert.Equal(t, []string{"t3", "t4", "t5"}, taskIDs(m.project.Categories[1]))
	assert.Equal(t, "task:t3", cursorItem(t, m))
}

func TestShiftOfAPeekedParentKeepsItsSubtasks(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks = append(p.Categories[0].Tasks,
		domain.Task{ID: "c3", Title: "Child", Status: domain.StatusCompleted, ParentID: "t3"})
	m := newTestModel(t, p)
	withStubStateManager(m)
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	selectRow(t, m, selection.FocusHidden, "c1")
	m, _ = pressKeys(m, "enter", "j", "J")

	assert.Equal(t, []string{"t3", "c3", "t4", "t5"}, taskIDs(m.project.Categories[1]))
	assert.Contains(t, rowIDs(m), "task:c3")
	assert.Equal(t, "task:t3", cursorItem(t, m))
}

func TestShiftOfALoneNoteIntoAHiddenSectionKeepsIt(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks = append(p.Categories[0].Tasks, domain.Task{ID: "n1", Title: "a note", Kind: domain.KindNote})
	p.Categories[1].Tasks[0].Status = domain.StatusCompleted
	p.Categories[1].Tasks[1].Status = domain.StatusCompleted
	m := newTestModel(t, p)
	withStubStateManager(m)
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	selectRow(t, m, selection.FocusSeparator, "n1")
	m, _ = pressKeys(m, "v", "J")

	assert.Equal(t, "n1", m.project.Categories[1].Tasks[0].ID)
	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, "1 note", visualLabel(m))
}

func TestVisualModeEndsWhenItsRowsHide(t *testing.T) {
	m, _ := pressKeys(peekingModel(t), "v", "j")
	m.ui.Visual.AnchorTaskID = "gone"
	m, _ = pressKeys(m, "j")

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Visual mode off: its rows are hidden", m.ui.Screen.StatusMsg)
}

func TestVisualLabelNamesNotes(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks = append(p.Categories[0].Tasks,
		domain.Task{ID: "n1", Title: "one", Kind: domain.KindNote},
		domain.Task{ID: "n2", Title: "two", Kind: domain.KindNote})
	m := newTestModel(t, p)
	selectRow(t, m, selection.FocusSeparator, "n1")
	m, _ = pressKeys(m, "v")
	assert.Equal(t, "-- VISUAL -- 1 note selected", m.statusText())

	m, _ = pressKeys(m, "j")
	assert.Equal(t, "-- VISUAL -- 2 notes selected", m.statusText())

	m, _ = pressKeys(m, "k", "k")
	assert.Equal(t, "-- VISUAL -- 1 task, 1 note selected", m.statusText())
}

func TestMoveOfPeekedRowsSaysTheFilterHidesThem(t *testing.T) {
	m, _ := pressKeys(peekingModel(t), "v", "j", "m", "enter")
	assert.Equal(t, []string{"t4", "t5", "t2", "t3"}, taskIDs(m.project.Categories[1]))
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Contains(t, m.ui.Screen.StatusMsg, "(hidden by the filter)")
}

func hiddenSectionModel(t *testing.T) *model {
	t.Helper()
	p := domain.Project{ID: "p1", Name: "P", Categories: []domain.Category{{ID: "c1", Name: "Inbox", Tasks: []domain.Task{
		{ID: "p", Title: "p", Status: domain.StatusTodo},
		{ID: "done", Title: "done", Kind: domain.KindSeparator},
		{ID: "b", Title: "b", Status: domain.StatusCompleted},
		{ID: "k", Title: "k", Status: domain.StatusCompleted},
		{ID: "next", Title: "next", Kind: domain.KindSeparator},
		{ID: "c", Title: "c", Status: domain.StatusTodo},
		{ID: "r", Title: "r", Status: domain.StatusTodo},
	}}}}
	m := newTestModel(t, p)
	withStubStateManager(m)
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	require.Equal(t, []string{"project:", "category:c1", "task:p", "section:done", "separator:next", "task:c", "task:r"}, rowIDs(m))
	return m
}

func TestShiftStepsOverAHiddenSectionsLine(t *testing.T) {
	m := hiddenSectionModel(t)
	selectRow(t, m, selection.FocusTask, "c")
	m, _ = pressKeys(m, "K")
	assert.Equal(t, []string{"p", "done", "b", "k", "c", "next", "r"}, taskIDs(m.project.Categories[0]), "between the line and next: the section's end")
	assert.Equal(t, []string{"project:", "category:c1", "task:p", "separator:done", "task:c", "separator:next", "task:r", "hidden:c1"}, rowIDs(m))
	assert.Equal(t, "task:c", cursorItem(t, m))

	m, _ = pressKeys(m, "K")
	assert.Equal(t, []string{"p", "c", "done", "b", "k", "next", "r"}, taskIDs(m.project.Categories[0]), "above the line")
	assert.Equal(t, []string{"project:", "category:c1", "task:p", "task:c", "section:done", "separator:next", "task:r"}, rowIDs(m))

	m, _ = pressKeys(m, "J")
	assert.Equal(t, []string{"p", "done", "b", "k", "c", "next", "r"}, taskIDs(m.project.Categories[0]), "J steps over it the same way")
}

func TestVisualShiftStepsOverAHiddenSectionsLine(t *testing.T) {
	m := hiddenSectionModel(t)
	selectRow(t, m, selection.FocusTask, "c")
	m, _ = pressKeys(m, "v", "j", "K", "K")
	assert.Equal(t, []string{"p", "c", "r", "done", "b", "k", "next"}, taskIDs(m.project.Categories[0]))
	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, []string{"project:", "category:c1", "task:p", "task:c", "task:r", "section:done", "section:next"}, rowIDs(m))
}

func TestAHiddenSectionsLineMovesItsSeparatorAlone(t *testing.T) {
	m := hiddenSectionModel(t)
	selectRow(t, m, selection.FocusSection, "done")
	m, _ = pressKeys(m, "K")
	assert.Equal(t, []string{"done", "p", "b", "k", "next", "c", "r"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, "separator:done", cursorItem(t, m), "the section shows p now")

	m = hiddenSectionModel(t)
	selectRow(t, m, selection.FocusSection, "done")
	m, _ = pressKeys(m, "J")
	assert.Equal(t, []string{"p", "b", "k", "next", "done", "c", "r"}, taskIDs(m.project.Categories[0]))
	assert.Equal(t, "separator:done", cursorItem(t, m))
}
