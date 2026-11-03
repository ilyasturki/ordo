package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
)

func addCategoryNamed(t *testing.T, m *model, name string) {
	t.Helper()
	m.ui.Selection.MoveTo(1) // Cat A
	m.startAddingCategory()
	typeText(m, name)
	pressKey(m, tea.KeyEnter)
	require.Equal(t, "c2", m.ui.Edit.clash, "the taken name opens the chooser")
}

func TestClashChooserNamesTheNewOneWithAFreeNumber(t *testing.T) {
	m := newTestModel(t, sampleProject())
	addCategoryNamed(t, m, "cat b")

	pressKey(m, 't')

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, []string{"Cat A", "cat b (2)", "Cat B"}, categoryNamesOf(m))
	assert.Equal(t, "cat b (2)", m.project.Categories[focusIndex(t, m)].Name)
	m.undo()
	assert.Equal(t, []string{"Cat A", "Cat B"}, categoryNamesOf(m))
}

func TestClashChooserEnterTakesTheHighlightedChoice(t *testing.T) {
	m := newTestModel(t, sampleProject())
	addCategoryNamed(t, m, "Cat B")
	pressKey(m, tea.KeyEnter)
	assert.Equal(t, []string{"Cat A", "Cat B (2)", "Cat B"}, categoryNamesOf(m))
}

func TestClashChooserRenamesTheOtherOne(t *testing.T) {
	m := newTestModel(t, sampleProject())
	addCategoryNamed(t, m, "Cat B")

	pressKey(m, 'o')

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, []string{"Cat A", "Cat B", "Cat B (2)"}, categoryNamesOf(m))
	assert.Equal(t, "c2", m.project.Categories[2].ID)
	assert.Equal(t, "Cat B", m.project.Categories[focusIndex(t, m)].Name)
	assert.Equal(t, `Renamed "Cat B" to "Cat B (2)"`, m.ui.Screen.StatusMsg)
	m.undo()
	assert.Equal(t, []string{"Cat A", "Cat B"}, categoryNamesOf(m), "one undo takes back both names")
}

func TestClashChooserOnARename(t *testing.T) {
	t.Run("this one", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		m.ui.Selection.MoveTo(1)
		m.startEditing()
		m.ui.Edit.input.SetValue("Cat B")
		pressKey(m, tea.KeyEnter)
		pressKey(m, 't')
		assert.Equal(t, []string{"Cat B (2)", "Cat B"}, categoryNamesOf(m))
		m.undo()
		assert.Equal(t, []string{"Cat A", "Cat B"}, categoryNamesOf(m))
	})
	t.Run("the other, from the name it would free", func(t *testing.T) {
		p := sampleProject()
		p.Categories[0].Name = "Cat B (2)"
		m := newTestModel(t, p)
		m.ui.Selection.MoveTo(1)
		m.startEditing()
		m.ui.Edit.input.SetValue("Cat B")
		pressKey(m, tea.KeyEnter)
		pressKey(m, 'o')
		assert.Equal(t, []string{"Cat B", "Cat B (2)"}, categoryNamesOf(m))
		assert.Equal(t, "c2", m.project.Categories[1].ID)
		m.undo()
		assert.Equal(t, []string{"Cat B (2)", "Cat B"}, categoryNamesOf(m))
	})
}

func TestClashChooserEscKeepsEditing(t *testing.T) {
	m := newTestModel(t, sampleProject())
	addCategoryNamed(t, m, "Cat B")
	pressKey(m, tea.KeyEscape)
	require.True(t, m.ui.Modes.IsEdit())
	assert.Empty(t, m.ui.Edit.clash)
	assert.Equal(t, "Cat B", m.ui.Edit.input.Value())
}

func TestClashChooserCountsOnlySiblings(t *testing.T) {
	m := newNestedModel(t)
	require.True(t, m.selectItem(selection.FocusCategory, "v1"))
	m.startAddingSubCategory()
	typeText(m, "ui")
	pressKey(m, tea.KeyEnter)
	assert.True(t, m.ui.Modes.IsNormal(), "ui under v0.2.0 is no sibling of one under v0.1.0")

	require.True(t, m.selectItem(selection.FocusCategory, "v2"))
	m.startAddingSubCategory()
	typeText(m, "UI")
	pressKey(m, tea.KeyEnter)
	require.Equal(t, "ui", m.ui.Edit.clash)
	pressKey(m, 't')
	assert.Equal(t, "UI (2)", m.project.Categories[focusIndex(t, m)].Name)
	assert.Equal(t, "v2", m.project.Categories[focusIndex(t, m)].ParentID)
}
