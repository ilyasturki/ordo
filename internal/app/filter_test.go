package app

import (
	"fmt"
	"strings"
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/modes"
	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/data"
	"github.com/ilyasturki/ordo/internal/domain"
)

// newTodoFilteredModel shows Alpha, Delta and Epsilon; Beta and Gamma hide in Cat A.
func newTodoFilteredModel(t *testing.T) *model {
	t.Helper()
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	return m
}

func onHiddenRow(t *testing.T) *model {
	t.Helper()
	m := newTodoFilteredModel(t)
	selectRow(t, m, selection.FocusHidden, "c1")
	return m
}

func dispatchKeys(m *model, keys ...string) tea.Cmd {
	var cmd tea.Cmd
	for _, k := range keys {
		cmd = m.dispatchNormalKey(k)
	}
	return cmd
}

func TestHiddenRowFollowsTheCategorysLastShownRow(t *testing.T) {
	m := newTodoFilteredModel(t)

	assert.Equal(t, []string{"project:", "category:c1", "task:t1", "hidden:c1", "category:c2", "task:t4", "task:t5"}, rowIDs(m))
	lineWith(t, screen(m), "(2 hidden by the filter)")
}

func TestHiddenRowOnlyWhileAFilterHidesTasks(t *testing.T) {
	m := newTestModel(t, sampleProject())
	assert.NotContains(t, rowIDs(m), "hidden:c1")

	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.ui.Filter.statuses[domain.StatusInProgress] = true
	m.ui.Filter.statuses[domain.StatusCompleted] = true
	m.rebuildPositions()
	assert.NotContains(t, rowIDs(m), "hidden:c1", "a filter that hides nothing adds no row")
}

func TestHiddenRowReplacesTheFalseNoTasksPlaceholder(t *testing.T) {
	m := newTestModel(t, sampleProject())
	m.ui.Filter.categories["c1"] = true
	m.rebuildPositions()

	lines := screen(m)
	assert.NotContains(t, strings.Join(lines, "\n"), "(no tasks)")
	header := lineWith(t, lines, "Cat B")
	for i, l := range lines {
		if l == header {
			assert.Contains(t, lines[i+1], "(2 hidden by the filter)", "no blank between a header and its only row")
		}
	}
}

func TestHiddenRowCountsNoSeparatorAndNoContextParent(t *testing.T) {
	m := newTestModel(t, domain.Project{ID: "p1", Categories: []domain.Category{{ID: "c1", Name: "Cat", Tasks: []domain.Task{
		{ID: "parent", Title: "Parent", Status: domain.StatusCompleted},
		{ID: "child", Title: "Child", Status: domain.StatusTodo, ParentID: "parent"},
		{ID: "sep", Kind: domain.KindSeparator},
		{ID: "done", Title: "Done", Status: domain.StatusCompleted},
	}}}})
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()

	lineWith(t, screen(m), "(1 hidden by the filter)")
}

func TestEnterOnHiddenRowClearsTheFilterOntoItsCategory(t *testing.T) {
	for _, key := range []string{"enter", "space"} {
		t.Run(key, func(t *testing.T) {
			m := onHiddenRow(t)

			dispatchKeys(m, key)

			assert.False(t, m.ui.Filter.HasActiveFilter())
			assert.Equal(t, "category:c1", cursorItem(t, m))
			assert.Equal(t, domain.StatusInProgress, m.project.Categories[0].Tasks[1].Status, "no task changed")
			assert.NotContains(t, rowIDs(m), "hidden:c1")
		})
	}
}

func TestHiddenRowRefusesRowActions(t *testing.T) {
	for _, keys := range [][]string{
		{"d"}, {"y"}, {"Y"}, {"p"}, {"x"}, {"e"}, {"ctrl+t"}, {"v"}, {"-"},
		{"J"}, {"K"}, {">"}, {"<"}, {"m"}, {"h"}, {"l"}, {"t"}, {"T"},
		{"shift+space"}, {"shift+enter"}, {"g", "t"}, {"g", "x"}, {"g", "y"},
	} {
		t.Run(strings.Join(keys, ""), func(t *testing.T) {
			m := onHiddenRow(t)
			before := cloneProject(m.project)

			cmd := dispatchKeys(m, keys...)

			assert.Nil(t, cmd)
			assert.Equal(t, before, m.project)
			assert.True(t, m.ui.Modes.IsNormal(), "no dialog opens")
			assert.False(t, m.ui.Visual.Active)
			assert.Equal(t, "hidden:c1", cursorItem(t, m))
			assert.True(t, m.ui.Filter.HasActiveFilter())
		})
	}
}

func TestEveryBindingSurvivesTheHiddenRow(t *testing.T) {
	for _, b := range normalBindings {
		name := fmt.Sprintf("%c%s", max(b.prefix, ' '), b.keys[0])
		t.Run(name, func(t *testing.T) {
			m := onHiddenRow(t)
			m.deps.Store = data.NewStore(t.TempDir())
			assert.NotPanics(t, func() { b.action(m) })
		})
	}
}

func TestAddOnHiddenRowAppendsToItsCategory(t *testing.T) {
	m := onHiddenRow(t)

	m.startAddingTask()

	require.True(t, m.ui.Modes.IsEdit())
	tasks := m.project.Categories[0].Tasks
	assert.Equal(t, m.ui.Edit.newItemID, tasks[len(tasks)-1].ID)
	assert.Equal(t, []string{"project:", "category:c1", "task:t1", "task:" + m.ui.Edit.newItemID, "hidden:c1"}, rowIDs(m)[:5])
}

func TestVisualModeStepsOverTheHiddenRow(t *testing.T) {
	m := newTodoFilteredModel(t)
	selectRow(t, m, selection.FocusTask, "t1")
	m.enterVisualMode()

	m.visualMoveCursor(1)

	assert.Equal(t, "task:t4", cursorItem(t, m))
	for _, p := range m.visualSelectedPositions() {
		assert.NotEqual(t, selection.FocusHidden, p.Kind)
	}
}

func TestSearchSkipsTheHiddenRow(t *testing.T) {
	m := newTodoFilteredModel(t)
	for _, r := range m.searchRows() {
		if r.Kind == selection.FocusHidden {
			assert.False(t, m.rowMatches(r, "hidden"))
		}
	}
}

func TestCursorOnHiddenRowIsRemembered(t *testing.T) {
	m := onHiddenRow(t)
	cursor, ok := m.currentCursor()
	require.True(t, ok)
	assert.Equal(t, data.Cursor{Kind: "hidden", CategoryID: "c1"}, cursor)

	m.ui.Selection.SetSelected(0)
	require.True(t, m.restoreCursor(cursor))
	assert.Equal(t, "hidden:c1", cursorItem(t, m))

	m.ui.Filter.ClearAll()
	m.rebuildPositions()
	require.True(t, m.restoreCursor(cursor))
	assert.Equal(t, "category:c1", cursorItem(t, m), "without the filter it lands on the category")
}

func TestShortcutBarShowsTheFilterIsOn(t *testing.T) {
	m := newTestModel(t, sampleProject())
	assert.Contains(t, ansi.Strip(m.renderShortcutBar()), "f filter ")

	m = newTodoFilteredModel(t)
	assert.Contains(t, ansi.Strip(m.renderShortcutBar()), "f filtered")
	assert.NotContains(t, ansi.Strip(m.renderShortcutBar()), "clear filter")

	selectRow(t, m, selection.FocusHidden, "c1")
	assert.True(t, strings.HasPrefix(ansi.Strip(m.renderShortcutBar()), "enter clear filter"))
}

func TestFilterKeyOpensTheDialogFromTheHiddenRow(t *testing.T) {
	m := onHiddenRow(t)

	dispatchKeys(m, "f")

	assert.Equal(t, modes.ModeFilter, m.ui.Modes.Current())
}

func taskStatus(m *model, id string) string {
	return m.taskByID(id).Status
}

func TestStatusCycleKeepsTheRowUntilTheCursorLeaves(t *testing.T) {
	m := newTodoFilteredModel(t)
	selectRow(t, m, selection.FocusTask, "t1")

	m, _ = pressKeys(m, "space")
	assert.Equal(t, domain.StatusInProgress, taskStatus(m, "t1"))
	assert.Equal(t, "task:t1", cursorItem(t, m))
	assert.Contains(t, lineWith(t, screen(m), "Alpha"), "Alpha (filtered out)")
	assert.Equal(t, "Hidden by the filter once you move off it", m.ui.Screen.StatusMsg)

	m, _ = pressKeys(m, "space", "space")
	assert.Equal(t, domain.StatusCancelled, taskStatus(m, "t1"), "space walks the whole cycle")
	assert.Equal(t, "task:t1", cursorItem(t, m))

	m, _ = pressKeys(m, "space")
	assert.Equal(t, domain.StatusTodo, taskStatus(m, "t1"))
	assert.NotContains(t, lineWith(t, screen(m), "Alpha"), "filtered out", "back in the filter, no mark")

	m, _ = pressKeys(m, "space", "j")
	assert.NotContains(t, rowIDs(m), "task:t1")
	assert.Equal(t, "hidden:c1", cursorItem(t, m))
	lineWith(t, screen(m), "(3 hidden by the filter)")
}

func TestEveryFilterEditKeepsTheRow(t *testing.T) {
	green := func(m *model) {
		m.ui.TagClip = TagClipboard{Set: true, Color: domain.TagGreen}
		m.ui.TagCopiedLast = true
	}
	pickGreen := func(m *model) {
		m.startTagEdit()
		m.ui.TagEdit.colorIdx = tagColorIndex(domain.TagGreen)
		m.finishTagEdit()
	}
	untagged := func(f *FilterState) { f.tags[""] = true }
	cases := map[string]struct {
		filter func(f *FilterState)
		setup  func(m *model)
		keys   []string
		act    func(m *model)
	}{
		"priority up":    {filter: func(f *FilterState) { f.priorities[""] = true }, keys: []string{"l", "l"}},
		"priority down":  {filter: func(f *FilterState) { f.priorities[""] = true }, keys: []string{"h"}},
		"reverse status": {filter: func(f *FilterState) { f.statuses[domain.StatusInProgress] = true }, keys: []string{"shift+space"}},
		"tag cycle":      {filter: untagged, keys: []string{"t", "t"}},
		"tag picker":     {filter: untagged, act: pickGreen},
		"tag paste":      {filter: untagged, setup: green, keys: []string{"p"}},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			m := newTestModel(t, sampleProject())
			withStubStateManager(m)
			tc.filter(&m.ui.Filter)
			m.rebuildPositions()
			if tc.setup != nil {
				tc.setup(m)
			}
			selectRow(t, m, selection.FocusTask, "t2")
			before := cloneProject(m.project)

			if tc.act != nil {
				tc.act(m)
			}
			m, _ = pressKeys(m, tc.keys...)

			require.NotEqual(t, before, m.project, "the edit landed")
			assert.Equal(t, "task:t2", cursorItem(t, m))
			assert.Contains(t, lineWith(t, screen(m), "Beta"), "(filtered out)")

			m, _ = pressKeys(m, "k")
			assert.NotEqual(t, "task:t2", cursorItem(t, m))
			assert.NotContains(t, rowIDs(m), "task:t2")
		})
	}
}

func TestVisualEditsKeepTheRangeUntilVisualModeEnds(t *testing.T) {
	m := newTodoFilteredModel(t)
	selectRow(t, m, selection.FocusTask, "t4")

	m, _ = pressKeys(m, "v", "j", "space")
	assert.True(t, m.ui.Modes.IsVisual())
	assert.Equal(t, domain.StatusInProgress, taskStatus(m, "t5"))
	m, _ = pressKeys(m, "k", "j")
	assert.Len(t, m.visualSelectedPositions(), 2, "moving inside the range drops nothing")

	m, _ = pressKeys(m, "esc")
	assert.Contains(t, rowIDs(m), "task:t4", "the cursor still sits on a held row")
	assert.Contains(t, rowIDs(m), "task:t5")

	m, _ = pressKeys(m, "g", "g")
	assert.NotContains(t, rowIDs(m), "task:t4")
	assert.NotContains(t, rowIDs(m), "task:t5")
	lineWith(t, screen(m), "(2 hidden by the filter)")
}

func TestVisualTagPasteKeepsTheRows(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	m.ui.Filter.tags[""] = true
	m.rebuildPositions()
	m.ui.TagClip = TagClipboard{Set: true, Color: domain.TagGreen}
	selectRow(t, m, selection.FocusTask, "t4")

	m, _ = pressKeys(m, "v", "j", "p")

	assert.True(t, m.ui.Modes.IsNormal())
	assert.Equal(t, "task:t5", cursorItem(t, m))
	assert.Contains(t, rowIDs(m), "task:t4")
	assert.Contains(t, rowIDs(m), "task:t5")
}

func TestHeldRowMovesWithJAndKInsteadOfItsCategory(t *testing.T) {
	m := newTodoFilteredModel(t)
	selectRow(t, m, selection.FocusTask, "t4")

	m, _ = pressKeys(m, "space", "J")
	assert.Equal(t, []string{"t5", "t4"}, taskIDs(m.project.Categories[1]))
	assert.Equal(t, "task:t4", cursorItem(t, m))

	m, _ = pressKeys(m, "K")
	assert.Equal(t, []string{"t4", "t5"}, taskIDs(m.project.Categories[1]))
	assert.Equal(t, []string{"c1", "c2"}, []string{m.project.Categories[0].ID, m.project.Categories[1].ID})
	assert.Equal(t, "task:t4", cursorItem(t, m))
}

func TestHeldRowsGoOnFilterChangeUndoAndRedo(t *testing.T) {
	held := func(t *testing.T) *model {
		m := newTodoFilteredModel(t)
		selectRow(t, m, selection.FocusTask, "t1")
		m, _ = pressKeys(m, "space")
		require.True(t, m.ui.Filter.holding())
		return m
	}

	t.Run("filter change", func(t *testing.T) {
		m := held(t)
		m.ui.Filter.SetView(FilterViewPriority)
		m.ui.Filter.ToggleSelected(m.project.Categories)
		assert.False(t, m.ui.Filter.holding())
	})
	t.Run("undo and redo", func(t *testing.T) {
		m := held(t)
		m, _ = pressKeys(m, "u")
		assert.False(t, m.ui.Filter.holding())
		assert.Equal(t, domain.StatusTodo, taskStatus(m, "t1"))
		m, _ = pressKeys(m, "ctrl+r")
		assert.False(t, m.ui.Filter.holding())
		assert.NotContains(t, rowIDs(m), "task:t1")
	})
	t.Run("reload", func(t *testing.T) {
		m := newStoredModel(t, sampleProject())
		m.ui.Filter.statuses[domain.StatusTodo] = true
		m.rebuildPositions()
		first := m.project.Categories[0].Tasks[0].ID
		selectRow(t, m, selection.FocusTask, first)
		m, _ = pressKeys(m, "space")
		require.True(t, m.ui.Filter.holding())

		m.reloadProject()

		assert.False(t, m.ui.Filter.holding())
		assert.NotContains(t, rowIDs(m), "task:"+first)
	})
}

func TestSavedFilterNamesTheEmptyBucketsNone(t *testing.T) {
	f := NewFilterState()
	f.statuses[domain.StatusTodo] = true
	f.priorities[""] = true
	f.priorities[domain.PriorityHigh] = true
	f.tags[""] = true
	f.categories["c1"] = true

	saved := f.saved()

	assert.Equal(t, data.Filter{Statuses: []string{"todo"}, Priorities: []string{"high", "none"}, Categories: []string{"c1"}, Tags: []string{"none"}}, saved)
	back := filterStateFrom(saved)
	assert.Equal(t, f.saved(), back.saved())
	assert.True(t, back.IsPriorityEnabled(""))
	assert.True(t, back.IsTagEnabled(""))
}

func TestRestoredFilterDropsValuesNoRowOffers(t *testing.T) {
	f := filterStateFrom(data.Filter{Statuses: []string{"todo", "someday"}, Tags: []string{"purple"}})

	assert.Equal(t, 1, f.StatusCount())
	assert.Zero(t, f.TagCount(), "a value no row could untick is dropped")
}

func TestFilterSavesOnEveryChange(t *testing.T) {
	m := newTestModel(t, sampleProject())
	stub := withStubStateManager(m)
	m.ui.Modes.ToFilter()

	m.handleFilterKey(keyPress("enter")) // Status
	m.handleFilterKey(keyPress("space")) // Todo
	assert.Equal(t, data.Filter{Statuses: []string{"todo"}}, stub.filters["p1"])

	m.ui.Filter.SetView(FilterViewHub)
	m.ui.Filter.hubSelected = FilterHubClearAll
	m.handleFilterKey(keyPress("enter"))
	assert.NotContains(t, stub.filters, "p1", "clear all leaves nothing behind")

	m = newTodoFilteredModel(t)
	stub = m.deps.StateManager.(*stubStateManager)
	m.saveFilter()
	require.Contains(t, stub.filters, "p1")
	selectRow(t, m, selection.FocusHidden, "c1")
	dispatchKeys(m, "enter")
	assert.NotContains(t, stub.filters, "p1", "enter on a hidden row saves the cleared filter")
}

func TestFilterAndItsCursorSurviveARestart(t *testing.T) {
	dir := t.TempDir()
	before := data.NewStateManager(dir, "")
	require.NoError(t, before.Load())
	m := newTestModel(t, sampleProject())
	m.deps.StateManager = before
	m.ui.Modes.ToFilter()
	m.handleFilterKey(keyPress("enter"))
	m.handleFilterKey(keyPress("space"))
	m.handleFilterKey(keyPress("q"))
	selectRow(t, m, selection.FocusHidden, "c1")
	m.saveCursorState()

	after := data.NewStateManager(dir, "")
	require.NoError(t, after.Load())
	restarted := newTestModel(t, sampleProject())
	restarted.deps.StateManager = after
	restarted.ui.Filter = loadFilterState(after, restarted.project)
	restarted.rebuildPositions()
	restarted.applyStoredCursor()

	assert.True(t, restarted.ui.Filter.IsStatusEnabled(domain.StatusTodo))
	assert.Equal(t, "hidden:c1", cursorItem(t, restarted), "the cursor lands on a row only the filter makes")
}

func TestEachProjectOpensWithItsOwnFilter(t *testing.T) {
	store := data.NewStore(t.TempDir())
	require.NoError(t, store.Ensure())
	alpha, err := store.ImportProject(domain.Project{Name: "Alpha", Categories: []domain.Category{{ID: "ca", Name: "Cat"}}})
	require.NoError(t, err)
	beta, err := store.ImportProject(domain.Project{Name: "Beta", Categories: []domain.Category{{ID: "cb", Name: "Cat"}}})
	require.NoError(t, err)

	m := newTestModel(t, alpha)
	stub := withStubStateManager(m)
	m.deps.Store = store
	stub.filters[beta.ID] = data.Filter{Statuses: []string{"todo"}}
	open := func(target int) {
		m.ui.Picker = ProjectPickerState{projects: []domain.Project{alpha, beta}, selected: target}
		m.selectProject()
	}

	open(1)
	assert.True(t, m.ui.Filter.IsStatusEnabled(domain.StatusTodo))
	open(0)
	assert.False(t, m.ui.Filter.HasActiveFilter())

	open(1)
	m.ui.Picker.startAdding()
	m.ui.Picker.input.SetValue("Gamma")
	m.createProjectFromPicker()
	assert.False(t, m.ui.Filter.HasActiveFilter(), "a new project starts clean")

	m.ui.Picker = ProjectPickerState{projects: []domain.Project{alpha, beta}, selected: 1}
	m.ui.ConfirmDelete = ConfirmDeleteState{Kind: ConfirmDeleteProject, ProjectID: beta.ID}
	m.confirmDeleteProject()
	assert.NotContains(t, stub.filters, beta.ID)
}

func TestDeletedCategoryLeavesTheFilter(t *testing.T) {
	t.Run("on load", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		stub := withStubStateManager(m)
		stub.filters["p1"] = data.Filter{Categories: []string{"gone", "c2"}}

		f := loadFilterState(stub, m.project)

		assert.Equal(t, 1, f.CategoryCount())
		assert.Equal(t, []string{"c2"}, stub.filters["p1"].Categories)
	})
	t.Run("on delete", func(t *testing.T) {
		m := newTestModel(t, sampleProject())
		stub := withStubStateManager(m)
		m.ui.Filter.categories["c1"] = true
		m.saveFilter()
		m.rebuildPositions()
		selectRow(t, m, selection.FocusCategory, "c1")

		m.confirmDeleteAction()

		assert.False(t, m.ui.Filter.HasActiveFilter())
		assert.NotContains(t, stub.filters, "p1")
		assert.Contains(t, rowIDs(m), "task:t4", "the rest of the project shows again")
	})
}
