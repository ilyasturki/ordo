package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func TestCTurnsATaskIntoANoteAndBack(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectRow(t, m, selection.FocusTask, "t2")
	depth := m.ui.History.UndoDepth()

	m, _ = pressKeys(m, "c")
	note := m.project.Categories[0].Tasks[1]
	assert.True(t, note.IsNote())
	assert.Empty(t, note.Status)
	assert.Equal(t, "Turned into a note", m.ui.Screen.StatusMsg)
	assert.Equal(t, "separator:t2", cursorItem(t, m))
	assert.Equal(t, depth+1, m.ui.History.UndoDepth())

	m, _ = pressKeys(m, "c")
	task := m.project.Categories[0].Tasks[1]
	assert.True(t, task.IsTask())
	assert.Equal(t, domain.StatusTodo, task.Status)

	m, _ = pressKeys(m, "u", "u")
	assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[1].Status, "undo brings the status back")
}

func TestCRefusesATaskWithADescription(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "why"
	m := newTestModel(t, p)
	selectRow(t, m, selection.FocusTask, "t1")
	depth := m.ui.History.UndoDepth()

	m, _ = pressKeys(m, "c")
	assert.True(t, m.project.Categories[0].Tasks[0].IsTask())
	assert.Equal(t, "“Alpha” has a description, and a note holds one line of text", m.ui.Screen.StatusMsg)
	assert.Equal(t, depth, m.ui.History.UndoDepth())
}

func TestVisualCTurnsTheSelectionFromItsFirstRow(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks = append(p.Categories[0].Tasks, domain.Task{ID: "n1", Title: "a note", Kind: domain.KindNote})
	m := newTestModel(t, p)
	selectRow(t, m, selection.FocusTask, "t2")
	depth := m.ui.History.UndoDepth()

	m, _ = pressKeys(m, "v", "j", "j", "c")
	for _, row := range m.project.Categories[0].Tasks[1:] {
		assert.True(t, row.IsNote(), row.ID)
	}
	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, "3 notes", visualLabel(m))
	assert.Equal(t, depth+1, m.ui.History.UndoDepth(), "one undo step")

	m, _ = pressKeys(m, "c")
	for _, row := range m.project.Categories[0].Tasks[1:] {
		assert.True(t, row.IsTask(), row.ID)
	}
}

func TestCOnACategorySaysWhatItDoes(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectRow(t, m, selection.FocusCategory, "c1")
	m, _ = pressKeys(m, "c")
	require.Equal(t, "c turns a task into a note, or a note into a task", m.ui.Screen.StatusMsg)
}
