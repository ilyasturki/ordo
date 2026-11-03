package app

import (
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
	"github.com/ilyasturki/ordo/internal/ui"
)

func threeCategories() domain.Project {
	return domain.Project{ID: "p1", Name: "P", Categories: []domain.Category{
		{ID: "a", Name: "A"}, {ID: "b", Name: "B"}, {ID: "c", Name: "C"},
	}}
}

func TestGPPinsAndUnpinsTheFocusedCategory(t *testing.T) {
	m := newTestModel(t, threeCategories())
	withStubStateManager(m)
	require.True(t, m.selectItem(selection.FocusCategory, "c"))

	m.togglePin()
	assert.Equal(t, []string{"c", "a", "b"}, categoryParents(m.project))
	assert.True(t, m.project.Categories[0].Pinned)
	assert.Equal(t, "c", focusedCategory(t, m), "the cursor follows the category")
	assert.Equal(t, "Pinned “C”", m.ui.Screen.StatusMsg)

	require.True(t, m.selectItem(selection.FocusCategory, "b"))
	m.togglePin()
	assert.Equal(t, []string{"c", "b", "a"}, categoryParents(m.project))

	require.True(t, m.selectItem(selection.FocusCategory, "c"))
	m.togglePin()
	assert.Equal(t, []string{"b", "c", "a"}, categoryParents(m.project), "an unpinned one starts the unpinned run")
	assert.Equal(t, "Unpinned “C”", m.ui.Screen.StatusMsg)

	m.undo()
	assert.Equal(t, []string{"c", "b", "a"}, categoryParents(m.project))
	assert.True(t, m.project.Categories[0].Pinned)
}

func TestGPOnATaskSaysWhere(t *testing.T) {
	m := newTestModel(t, sampleProject())
	require.True(t, m.selectTaskByID("t1"))
	m.togglePin()
	assert.Equal(t, "Put the cursor on a category to pin it", m.ui.Screen.StatusMsg)
}

func TestAOnAPinnedCategoryAddsAfterThePinnedRun(t *testing.T) {
	p := threeCategories()
	p.Categories[0].Pinned, p.Categories[1].Pinned = true, true
	m := newTestModel(t, p)
	withStubStateManager(m)
	require.True(t, m.selectItem(selection.FocusCategory, "a"))

	m.startAddingCategory()
	typeText(m, "New")
	pressKey(m, tea.KeyEnter)

	assert.Equal(t, []string{"A", "B", "New", "C"}, categoryNamesOf(m))
	assert.False(t, m.project.Categories[2].Pinned)
}

func TestPasteKeepsPinsFirst(t *testing.T) {
	pinnedA := func() domain.Project {
		p := threeCategories()
		p.Categories[0].Pinned = true
		return p
	}
	t.Run("a copy pasted above a pinned one lands after the pinned run", func(t *testing.T) {
		m := newTestModel(t, pinnedA())
		withStubStateManager(m)
		selectRow(t, m, selection.FocusCategory, "c")
		m.enterVisualMode()
		_ = m.visualCopyBullets()
		selectRow(t, m, selection.FocusCategory, "a")
		m.pasteInternal()
		assert.Equal(t, []string{"A", "C (2)", "B", "C"}, categoryNamesOf(m))
	})
	t.Run("a cut one pasted above a pinned one lands after the pinned run", func(t *testing.T) {
		m := newTestModel(t, pinnedA())
		withStubStateManager(m)
		selectRow(t, m, selection.FocusCategory, "c")
		m.enterVisualMode()
		m.visualCut()
		selectRow(t, m, selection.FocusCategory, "a")
		m.pasteInternal()
		assert.Equal(t, []string{"a", "c", "b"}, categoryParents(m.project))
		assert.Equal(t, "c", focusedCategory(t, m))
	})
	t.Run("a pinned copy pasted among unpinned ones ends the pinned run", func(t *testing.T) {
		m := newTestModel(t, pinnedA())
		withStubStateManager(m)
		selectRow(t, m, selection.FocusCategory, "a")
		m.enterVisualMode()
		_ = m.visualCopyBullets()
		selectRow(t, m, selection.FocusCategory, "c")
		m.pasteInternal()
		assert.Equal(t, []string{"A", "A (2)", "B", "C"}, categoryNamesOf(m))
		assert.True(t, m.project.Categories[1].Pinned)
	})
}

func TestThePinMarkFollowsTheName(t *testing.T) {
	h := categoryHeader{name: "Release", pinned: true}
	line := ansi.Strip(renderCategoryLine(h, false, 40, true, false, false, "", ui.MutedStyle))
	assert.True(t, strings.HasPrefix(strings.TrimSpace(line), "▼ Release"+ui.PinMark), line)
	assert.Equal(t, categoryNameWidth(40, categoryHeader{name: "Release"})-ansi.StringWidth(ui.PinMark), categoryNameWidth(40, h))
}
