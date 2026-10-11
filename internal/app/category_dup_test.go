package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

func TestYankPasteCategory_SuffixesTheName(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A header
	m.enterVisualMode()
	_ = m.visualCopyBullets()
	require.Len(t, m.ui.Clipboard.Categories, 1)

	m.pasteFromClipboard()

	assert.ElementsMatch(t, []string{"Cat A", "Cat A (2)", "Cat B"}, categoryNamesOf(m))
}

func categoryNamesOf(m *model) []string {
	var names []string
	for _, c := range m.project.Categories {
		names = append(names, c.Name)
	}
	return names
}

// newStoredModel opens the first project from a real store holding all of them,
// for paths that go through the store's cross-project checks.
func newStoredModel(t *testing.T, projects ...domain.Project) *model {
	t.Helper()
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	var first domain.Project
	for i, p := range projects {
		saved, err := store.ImportProject(p)
		require.NoError(t, err)
		if i == 0 {
			first = saved
		}
	}
	m := newTestModel(t, first)
	m.deps.Store = store
	return m
}

func typeText(m *model, s string) {
	for _, r := range s {
		m.handleEditKey(tea.KeyPressMsg{Code: r, Text: string(r)})
	}
}

func pressKey(m *model, code rune) {
	m.handleEditKey(tea.KeyPressMsg{Code: code})
}

func TestAddCategory_PasteIgnoredUnderMergeOfferAndClearsError(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.startAddingCategory()
	typeText(m, "cat b")
	pressKey(m, tea.KeyEnter)
	require.NotEmpty(t, m.ui.Edit.mergeInto)

	m.handlePaste("x")
	assert.Equal(t, "cat b", m.ui.Edit.input.Value(), "a paste cannot edit the name under the offer")

	pressKey(m, tea.KeyEscape)
	m.handlePaste("x")
	assert.Equal(t, "cat bx", m.ui.Edit.input.Value())
	assert.Empty(t, m.ui.Edit.err, "changing the text by paste clears the error")
}

func TestAddCategory_DuplicateKeepsTheEditorAndOffersMerge(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A
	m.startAddingCategory()
	typeText(m, " cat b")
	pressKey(m, tea.KeyEnter)

	require.True(t, m.ui.Modes.IsEdit(), "the editor stays open")
	assert.Equal(t, " cat b", m.ui.Edit.input.Value(), "the typed name survives")
	assert.Contains(t, m.ui.Edit.err, `"Cat B" already exists`)
	assert.Equal(t, "c2", m.ui.Edit.mergeInto)
	assert.Contains(t, m.View().Content, "Merge into")

	// Declining the merge returns to the editor with the error still shown.
	pressKey(m, tea.KeyEscape)
	require.True(t, m.ui.Modes.IsEdit())
	assert.Empty(t, m.ui.Edit.mergeInto)
	assert.NotEmpty(t, m.ui.Edit.err)

	// Editing the text clears the error, and a free name goes through.
	typeText(m, "2")
	assert.Empty(t, m.ui.Edit.err)
	pressKey(m, tea.KeyEnter)
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, []string{"Cat A", "cat b2", "Cat B"}, categoryNamesOf(m))
}

func TestAddCategory_MergeDropsTheNewRowAndJumpsToTheExisting(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1)
	m.startAddingCategory()
	typeText(m, "CAT B")
	pressKey(m, tea.KeyEnter)
	pressKey(m, 'y')

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, []string{"Cat A", "Cat B"}, categoryNamesOf(m))
	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusCategory, pos.Kind)
	assert.Equal(t, "c2", m.project.Categories[pos.CategoryIndex].ID)
	assert.False(t, m.ui.History.CanUndo(), "dropping the new row leaves nothing to undo")
}

func TestRenameCategory_MergeMovesTasksUnderTheOldNameInOneUndo(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1) // Cat A, three tasks
	m.startEditing()
	m.ui.Edit.input.SetValue("cat b")
	pressKey(m, tea.KeyEnter)
	require.Equal(t, "c2", m.ui.Edit.mergeInto)
	pressKey(m, tea.KeyEnter)

	require.True(t, m.ui.Modes.IsNormal())
	require.Len(t, m.project.Categories, 1)
	tasks := m.project.Categories[0].Tasks
	require.Len(t, tasks, 6)
	assert.Equal(t, []string{"t4", "t5"}, []string{tasks[0].ID, tasks[1].ID})
	assert.True(t, tasks[2].IsSeparator())
	assert.Equal(t, "Cat A", tasks[2].Title)
	assert.Equal(t, []string{"t1", "t2", "t3"}, []string{tasks[3].ID, tasks[4].ID, tasks[5].ID})

	pos, ok := m.selectedPosition()
	require.True(t, ok)
	assert.Equal(t, selection.FocusSeparator, pos.Kind, "the cursor lands on the new separator")

	m.undo()
	assert.Equal(t, []string{"Cat A", "Cat B"}, categoryNamesOf(m))
	assert.Len(t, m.project.Categories[0].Tasks, 3)
	assert.Len(t, m.project.Categories[1].Tasks, 2)
}

func TestRenameCategory_EscapeAfterTheOfferStillCancels(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Selection.MoveTo(1)
	m.startEditing()
	m.ui.Edit.input.SetValue("Cat B")
	pressKey(m, tea.KeyEnter)
	pressKey(m, 'n')
	pressKey(m, tea.KeyEscape)

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, []string{"Cat A", "Cat B"}, categoryNamesOf(m))
	assert.False(t, m.ui.History.CanUndo())
}

func TestRenameProject_InlineRefusesAnotherProjectsName(t *testing.T) {
	other := sampleProject()
	other.ID, other.Name = "p2", "Roadmap"
	m := newStoredModel(t, sampleProject(), other)
	m.ui.Selection.MoveTo(0)
	m.startEditing()
	m.ui.Edit.input.SetValue(" roadmap ")
	pressKey(m, tea.KeyEnter)

	require.True(t, m.ui.Modes.IsEdit(), "the editor stays open")
	assert.Contains(t, m.ui.Edit.err, "already exists")
	assert.Equal(t, "P", m.project.Name)

	m.ui.Edit.input.SetValue("Road")
	pressKey(m, tea.KeyEnter)
	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "Road", m.project.Name)
}
