package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

const viewAppliedMessage = "Changed on disk: folds and filter applied"

// viewModel opens the sample project on real files, with a second store as the other writer.
func viewModel(t *testing.T) (*model, *data.Store) {
	t.Helper()
	dir := t.TempDir()
	projects := filepath.Join(dir, "projects")
	store := data.NewStore(projects)
	require.NoError(t, store.Ensure())
	p, err := store.ImportProject(sampleProject())
	require.NoError(t, err)
	p, err = store.LoadProjectByID(p.ID)
	require.NoError(t, err)
	m := newTestModel(t, p)
	m.deps.Store = store
	state := data.NewStateManager(dir, "")
	require.NoError(t, state.Load())
	m.deps.StateManager = state
	m.rebuildPositions()
	tick(t, m)
	return m, data.NewStore(projects)
}

func setFolds(t *testing.T, other *data.Store, id string, folds ...string) {
	t.Helper()
	require.NoError(t, other.MergeView(id, data.ViewChange{Folds: &folds}))
}

func setFilter(t *testing.T, other *data.Store, id string, filter data.Filter) {
	t.Helper()
	require.NoError(t, other.MergeView(id, data.ViewChange{Filter: &filter}))
}

func TestAFoldFromElsewhereAppliesWithoutAKey(t *testing.T) {
	m, other := viewModel(t)
	selectRow(t, m, selection.FocusTask, "t1")

	setFolds(t, other, m.project.ID, "c2")
	tick(t, m)

	assert.True(t, m.ui.Fold.IsFolded("c2"))
	assert.NotContains(t, rowIDs(m), "task:t4")
	assert.Equal(t, "task:t1", cursorItem(t, m), "the cursor stays on its row")
	assert.Equal(t, viewAppliedMessage, m.ui.Screen.StatusMsg)

	hit(m, "j")
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg, "told once")
}

func TestAFoldOverTheCursorLandsItOnTheHeader(t *testing.T) {
	m, other := viewModel(t)
	selectRow(t, m, selection.FocusTask, "t1")

	setFolds(t, other, m.project.ID, "c1")
	tick(t, m)

	assert.Equal(t, "category:c1", cursorItem(t, m))
}

func TestAFilterFromElsewhereAppliesAndEndsAPeek(t *testing.T) {
	m, other := viewModel(t)
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.saveFilter()
	m.rebuildPositions()
	tick(t, m)
	selectRow(t, m, selection.FocusHidden, "c1")
	m, _ = pressKeys(m, "enter")
	require.True(t, m.ui.Filter.peeks("c1"))

	setFolds(t, other, m.project.ID, "c2")
	tick(t, m)
	assert.True(t, m.ui.Filter.peeks("c1"), "a fold leaves the peek")

	setFilter(t, other, m.project.ID, data.Filter{Statuses: []string{domain.StatusCompleted}})
	tick(t, m)
	assert.False(t, m.ui.Filter.peeks("c1"), "another filter ends it")
	assert.True(t, m.ui.Filter.IsStatusEnabled(domain.StatusCompleted))
	assert.False(t, m.ui.Filter.IsStatusEnabled(domain.StatusTodo))
}

func TestAViewChangeWaitsForADialogToClose(t *testing.T) {
	m, other := viewModel(t)
	hit(m, "?")
	require.False(t, m.ui.Modes.IsNormal())

	setFolds(t, other, m.project.ID, "c2")
	tick(t, m)
	assert.False(t, m.ui.Fold.IsFolded("c2"))

	hit(m, "esc")
	require.True(t, m.ui.Modes.IsNormal())
	assert.True(t, m.ui.Fold.IsFolded("c2"))
}

func TestOwnFoldAndFilterAreNoOutsideChange(t *testing.T) {
	m, _ := viewModel(t)
	selectRow(t, m, selection.FocusCategory, "c1")
	m.toggleFold()
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg)

	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.saveFilter()
	tick(t, m)
	assert.Empty(t, m.ui.Screen.StatusMsg)
}

func TestAFoldToggleKeepsAFilterSetElsewhere(t *testing.T) {
	m, other := viewModel(t)
	setFilter(t, other, m.project.ID, data.Filter{Statuses: []string{domain.StatusTodo}})

	selectRow(t, m, selection.FocusCategory, "c1")
	m.toggleFold()

	view, ok := other.View(m.project.ID)
	require.True(t, ok)
	assert.Equal(t, []string{"c1"}, view.Folds)
	assert.Equal(t, []string{domain.StatusTodo}, view.Filter.Statuses)

	tick(t, m)
	assert.True(t, m.ui.Filter.IsStatusEnabled(domain.StatusTodo), "the filter it wrote past reaches the screen")
	assert.True(t, m.ui.Fold.IsFolded("c1"))
}

func TestAFilterArrivingBeforeItsCategoryKeepsIt(t *testing.T) {
	m, other := viewModel(t)
	id := m.project.ID
	setFilter(t, other, id, data.Filter{Categories: []string{"new-cat"}})
	path := filepath.Join(other.Dir, id+".view.json")
	before, err := os.ReadFile(path)
	require.NoError(t, err)

	tick(t, m)
	assert.Equal(t, 0, m.ui.Filter.CategoryCount(), "a category the copy doesn't hold can't hide every row")
	after, err := os.ReadFile(path)
	require.NoError(t, err)
	assert.Equal(t, string(before), string(after), "applying a view never writes it")

	onWeb(t, other, id, func(p *domain.Project) {
		p.Categories = append(p.Categories, domain.Category{ID: "new-cat", Name: "Arrived"})
	})
	tick(t, m)

	assert.True(t, m.ui.Filter.IsCategoryEnabled("new-cat"), "back with the project that holds it")
}

func TestReloadTakesTheViewBesideIt(t *testing.T) {
	m, other := viewModel(t)
	setFolds(t, other, m.project.ID, "c2")
	m.reloadProject()
	assert.True(t, m.ui.Fold.IsFolded("c2"))
	tick(t, m)
	assert.NotEqual(t, viewAppliedMessage, m.ui.Screen.StatusMsg, "nothing left to apply")
}

func TestDeletingAProjectFromThePickerRemovesItsView(t *testing.T) {
	m, other := viewModel(t)
	second, err := other.CreateProject("Second")
	require.NoError(t, err)
	setFolds(t, other, second.ID, "c1")

	m.ui.Picker = ProjectPickerState{projects: []domain.Project{m.project, second}, selected: 1}
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteProject, ProjectID: second.ID}
	m.confirmDeleteProject()

	_, ok := other.View(second.ID)
	assert.False(t, ok)
}
