package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
)

func pressAll(m *model, keys ...string) *model {
	for _, k := range keys {
		next, _ := m.Update(keyPress(k))
		updated := asModel(next)
		m = &updated
	}
	return m
}

func shiftEnter(m *model) *model {
	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModShift})
	updated := asModel(next)
	return &updated
}

func editTitleWithCursor(t *testing.T, m *model, text string, at int) *model {
	t.Helper()
	selectFirstTask(m)
	m = pressAll(m, "enter")
	require.True(t, m.ui.Modes.IsEdit())
	m.ui.Edit.input.SetValue(text)
	m.ui.Edit.input.SetCursor(at)
	return m
}

func TestShiftEnterInTitle_MovesTheRestToTheDescription(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m = editTitleWithCursor(t, m, "Alpha it fails on DST", len("Alpha "))

	m = shiftEnter(m)

	task := m.project.Categories[0].Tasks[0]
	assert.Equal(t, "Alpha", task.Title)
	assert.Equal(t, "it fails on DST", task.Description)
	require.Equal(t, modes.ModeDescriptionEdit, m.ui.Modes.Current())
	ta := m.ui.DescriptionEdit.textarea
	assert.Equal(t, 0, ta.Line())
	assert.Equal(t, len("it fails on DST"), ta.Column(), "the cursor ends the moved text")
}

func TestShiftEnterInTitle_PutsTheMovedTextAboveTheDescription(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "older notes"
	m := newTestModel(t, p)
	m = editTitleWithCursor(t, m, "Alpha it fails on DST", len("Alpha "))

	m = shiftEnter(m)

	assert.Equal(t, "it fails on DST\n\nolder notes", m.project.Categories[0].Tasks[0].Description)
	ta := m.ui.DescriptionEdit.textarea
	assert.Equal(t, 0, ta.Line())
	assert.Equal(t, len("it fails on DST"), ta.Column())
}

func TestShiftEnterInTitle_ShowsTheMovedTextAboveALongDescription(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = numberedLines(100)
	m := newTestModel(t, p)
	m = editTitleWithCursor(t, m, "Alpha it fails on DST", len("Alpha "))

	m = shiftEnter(m)

	assert.Equal(t, 0, descriptionMarkerRow(t, m), "the editor shows the moved text, its cursor on the top row")
	assert.Contains(t, ansi.Strip(m.ui.DescriptionEdit.textarea.View()), "it fails on DST")
}

func TestShiftEnterInTitle_IsOneUndoStep(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m = editTitleWithCursor(t, m, "Alpha it fails on DST", len("Alpha "))
	m = shiftEnter(m)
	m = pressAll(m, "esc")
	require.True(t, m.ui.Modes.IsNormal())

	m = pressAll(m, "u")

	task := m.project.Categories[0].Tasks[0]
	assert.Equal(t, "Alpha", task.Title, "one u takes the split back")
	assert.Empty(t, task.Description)
}

func TestShiftEnterInTitle_WhileAddingIsTheAddsUndoStep(t *testing.T) {
	m := newTestModel(t, sampleProject())
	selectFirstTask(m)
	m = pressAll(m, "a")
	require.True(t, m.ui.Modes.IsEdit())
	m.ui.Edit.input.SetValue("Ferry a much longer story")
	m.ui.Edit.input.SetCursor(len("Ferry"))

	m = shiftEnter(m)
	require.Equal(t, modes.ModeDescriptionEdit, m.ui.Modes.Current())
	m = pressAll(m, "enter")

	added := m.project.Categories[0].Tasks[1]
	assert.Equal(t, "Ferry", added.Title)
	assert.Equal(t, "a much longer story", added.Description)
	require.Len(t, m.ui.History.undo, 1)

	m = pressAll(m, "u")
	assert.Len(t, m.project.Categories[0].Tasks, 3, "one u removes the added task")
}

func TestShiftEnterInTitle_AtTheEndSavesAndOpensTheDescription(t *testing.T) {
	p := sampleProject()
	p.Categories[0].Tasks[0].Description = "kept"
	m := newTestModel(t, p)
	m = editTitleWithCursor(t, m, "Alpha renamed", len("Alpha renamed"))

	m = shiftEnter(m)

	task := m.project.Categories[0].Tasks[0]
	assert.Equal(t, "Alpha renamed", task.Title)
	assert.Equal(t, "kept", task.Description)
	require.Equal(t, modes.ModeDescriptionEdit, m.ui.Modes.Current())
	assert.Equal(t, "kept", m.ui.DescriptionEdit.textarea.Value())
}

func TestShiftEnterInTitle_RefusesAnEmptyTitle(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m = editTitleWithCursor(t, m, "  everything after", 2)

	m = shiftEnter(m)

	assert.True(t, m.ui.Modes.IsEdit(), "the editor stays open")
	assert.Equal(t, "  everything after", m.ui.Edit.input.Value())
	assert.Equal(t, "Put the cursor where the title ends", m.ui.Screen.StatusMsg)
	assert.Equal(t, "Alpha", m.project.Categories[0].Tasks[0].Title)
}

func TestShiftEnterInTitle_AltEnterToo(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m = editTitleWithCursor(t, m, "Alpha rest", len("Alpha"))

	next, _ := m.Update(tea.KeyPressMsg{Code: tea.KeyEnter, Mod: tea.ModAlt})
	updated := asModel(next)

	assert.Equal(t, "rest", updated.project.Categories[0].Tasks[0].Description)
}

func TestShiftEnterInCategoryName_SaysOnlyATaskHasOne(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.SelectByPredicate(func(p selection.Position) bool { return p.Kind == selection.FocusCategory })
	m = pressAll(m, "enter")
	require.True(t, m.ui.Modes.IsEdit())

	m = shiftEnter(m)

	assert.True(t, m.ui.Modes.IsEdit())
	assert.Equal(t, "Only a task has a description", m.ui.Screen.StatusMsg)
}
