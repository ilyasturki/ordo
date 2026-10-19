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
