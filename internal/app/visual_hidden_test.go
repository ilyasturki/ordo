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
