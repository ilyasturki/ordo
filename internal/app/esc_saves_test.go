package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
)

func TestEscInTitle_SavesAndUndoTakesItBack(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m = pressAll(m, "enter")
	typeText(m, " and more")

	m = pressAll(m, "esc")

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Alpha and more", m.project.Categories[0].Tasks[0].Title)
	m = pressAll(m, "u")
	assert.Equal(t, "Alpha", m.project.Categories[0].Tasks[0].Title)
}

func TestEscInTitle_AnEmptiedTitleKeepsTheOldOne(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m = pressAll(m, "enter")
	m.ui.Edit.input.SetValue("  ")

	m = pressAll(m, "esc")

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Alpha", m.project.Categories[0].Tasks[0].Title)
	assert.False(t, m.ui.History.CanUndo())
}

func TestEscWhileAdding_KeepsTheTaskAndOneUndoRemovesIt(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m = pressAll(m, "a")
	typeText(m, "A long title I typed")

	m = pressAll(m, "esc")

	require.True(t, m.ui.Modes.IsNormal())
	require.Len(t, m.project.Categories[0].Tasks, 4)
	assert.Equal(t, "A long title I typed", m.project.Categories[0].Tasks[1].Title)
	m = pressAll(m, "u")
	assert.Len(t, m.project.Categories[0].Tasks, 3)
}

func TestEscWhileAdding_AnEmptyRowGoes(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m = pressAll(m, "a", "esc")

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Len(t, m.project.Categories[0].Tasks, 3)
	assert.False(t, m.ui.History.CanUndo(), "nothing was added, nothing to undo")
}

func TestEscWhileAddingANote_KeepsIt(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m = pressAll(m, "_")
	require.True(t, m.ui.Modes.IsEdit())
	typeText(m, "remember the ferry")

	m = pressAll(m, "esc")

	assert.True(t, m.ui.Modes.IsNormal())
	var notes []string
	for _, task := range m.project.Categories[0].Tasks {
		if task.IsNote() {
			notes = append(notes, task.Title)
		}
	}
	assert.Equal(t, []string{"remember the ferry"}, notes)
}

func TestEscInSeparatorLabel_Saves(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m = pressAll(m, "-")
	require.True(t, m.ui.Modes.IsNormal())
	m = pressAll(m, "enter")
	require.True(t, m.ui.Modes.IsEdit())
	typeText(m, "Later")

	m = pressAll(m, "esc")

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	require.Equal(t, selection.FocusSeparator, pos.Kind)
	assert.Equal(t, "Later", m.project.Categories[pos.CategoryIndex].Tasks[pos.TaskIndex].Title)
}

func TestEscInCategoryName_SavesANewOne(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1)
	m.startAddingCategory()
	typeText(m, "Someday")

	pressKey(m, tea.KeyEscape)

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, []string{"Cat A", "Someday", "Cat B"}, categoryNamesOf(m))
}

func TestEscInCategoryName_ATakenNameNeedsASecondEscToDrop(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1)
	m.startEditing()
	m.ui.Edit.input.SetValue("Cat B")

	pressKey(m, tea.KeyEscape)
	require.True(t, m.ui.Modes.IsEdit(), "a refused save keeps the editor")
	require.NotEmpty(t, m.ui.Edit.clash, "and offers what to do about the name")
	pressKey(m, tea.KeyEscape)
	require.True(t, m.ui.Modes.IsEdit(), "the offer's esc goes back to the name")
	assert.Equal(t, "Cat B", m.ui.Edit.input.Value())

	pressKey(m, tea.KeyEscape)
	assert.True(t, m.ui.Modes.IsNormal(), "esc after a refusal drops the edit")
	assert.Equal(t, []string{"Cat A", "Cat B"}, categoryNamesOf(m))
}

func TestEscInProjectName_ATakenNameSaysEscAgainDrops(t *testing.T) {
	other := sampleProject()
	other.ID, other.Name = "p2", "Roadmap"
	m := newStoredModel(t, sampleProject(), other)
	m.ui.Selection.MoveTo(0)
	m.startEditing()
	m.ui.Edit.input.SetValue("Roadmap")

	pressKey(m, tea.KeyEscape)
	require.True(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "Not saved: esc again drops it", m.ui.Screen.StatusMsg)

	pressKey(m, tea.KeyEscape)
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "P", m.project.Name)
}

func TestEscInProjectName_Saves(t *testing.T) {
	m := newStoredModel(t, sampleProject())
	m.ui.Selection.MoveTo(0)
	m.startEditing()
	m.ui.Edit.input.SetValue("Renamed")

	pressKey(m, tea.KeyEscape)

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Renamed", m.project.Name)
}

func TestEscInSearch_StillCancels(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m = pressAll(m, "/")
	require.True(t, m.ui.Modes.IsSearch())
	typeText(m, "Gam")

	m = pressAll(m, "esc")

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Empty(t, m.ui.Search.input.Value())
}
