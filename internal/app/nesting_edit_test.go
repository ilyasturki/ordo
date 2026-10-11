package app

import (
	"testing"

	tea "charm.land/bubbletea/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

// shapeOf renders a category's tasks as IDs, a subtask as "id<parent".
func shapeOf(cat domain.Category) []string {
	out := make([]string, len(cat.Tasks))
	for i, t := range cat.Tasks {
		out[i] = t.ID
		if t.ParentID != "" {
			out[i] += "<" + t.ParentID
		}
	}
	return out
}

func categoryOrder(p domain.Project) []string {
	out := make([]string, len(p.Categories))
	for i, c := range p.Categories {
		out[i] = c.ID
		if c.ParentID != "" {
			out[i] += "<" + c.ParentID
		}
	}
	return out
}

func catByID(t *testing.T, m *model, id string) domain.Category {
	t.Helper()
	i, err := m.project.FindCategoryByID(id)
	require.NoError(t, err)
	return m.project.Categories[i]
}

func typeTitle(m *model, title string) {
	m.ui.Edit.input.SetValue(title)
	m.finishEditing()
}

func TestAddAtTheCursorsDepth(t *testing.T) {
	t.Run("a on a subtask adds a sibling subtask", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "pick")
		m.startAddingTask()
		typeTitle(m, "Vibrate")
		ui := catByID(t, m, "ui")
		require.Len(t, ui.Tasks, 5)
		assert.Equal(t, "timer", ui.Tasks[2].ParentID)
		assert.Equal(t, "Vibrate", ui.Tasks[2].Title)
	})
	t.Run("a on a parent adds after its subtasks", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "timer")
		m.startAddingTask()
		typeTitle(m, "Next")
		ui := catByID(t, m, "ui")
		assert.Equal(t, "Next", ui.Tasks[3].Title)
		assert.Empty(t, ui.Tasks[3].ParentID)
	})
	t.Run("- below a subtask lands after the parent's block", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "pick")
		m.startAddingSeparator()
		ui := catByID(t, m, "ui")
		assert.True(t, ui.Tasks[3].IsSeparator())
		assert.Empty(t, ui.Tasks[3].ParentID)
		assert.Equal(t, "sets", ui.Tasks[4].ID)
	})
	t.Run("A on a sub-category adds a sibling sub-category", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sets")
		m.startAddingCategory()
		typeTitle(m, "data")
		assert.Equal(t, []string{"v2", "ui<v2", m.project.Categories[2].ID + "<v2", "api<v2", "v1"}, categoryOrder(m.project))
	})
	t.Run("A on a parent category adds after its sub-categories", func(t *testing.T) {
		m := newNestedModel(t)
		m.project.Categories[0].Name = "areas" // a version would propose the next one at the top instead
		selectRow(t, m, selection.FocusCategory, "v2")
		m.startAddingCategory()
		typeTitle(m, "later")
		assert.Equal(t, []string{"v2", "ui<v2", "api<v2", m.project.Categories[3].ID, "v1"}, categoryOrder(m.project))
	})
	t.Run("a sub-category may reuse a name another parent's area carries", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "auth")
		m.startAddingCategory()
		typeTitle(m, "v0.1.0")
		assert.Empty(t, m.ui.Edit.err, "only siblings must differ")
	})
}

func TestNestKeys(t *testing.T) {
	t.Run("> nests a task under the one above and < brings it back", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sets")
		m.nestSelected(true)
		assert.Equal(t, []string{"timer", "pick<timer", "sound<timer", "sets<timer"}, shapeOf(catByID(t, m, "ui")))
		assert.Equal(t, "task:sets", cursorItem(t, m))

		m.nestSelected(false)
		assert.Equal(t, []string{"timer", "pick<timer", "sound<timer", "sets"}, shapeOf(catByID(t, m, "ui")))

		m.undo()
		assert.Equal(t, []string{"timer", "pick<timer", "sound<timer", "sets<timer"}, shapeOf(catByID(t, m, "ui")))
	})
	t.Run("nesting under a folded parent unfolds it", func(t *testing.T) {
		m := newNestedModel(t)
		m.ui.Fold.Toggle("timer")
		m.rebuildPositions()
		selectRow(t, m, selection.FocusTask, "sets")
		m.nestSelected(true)
		assert.False(t, m.ui.Fold.IsFolded("timer"))
		assert.Equal(t, "task:sets", cursorItem(t, m))
	})
	t.Run("refusals say why", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "pick")
		m.nestSelected(true)
		assert.Contains(t, m.ui.Screen.StatusMsg, "one level deep")

		selectRow(t, m, selection.FocusTask, "timer")
		m.nestSelected(true)
		assert.Contains(t, m.ui.Screen.StatusMsg, "Nothing above")

		m.nestSelected(false)
		assert.Contains(t, m.ui.Screen.StatusMsg, "top level")
		assert.Equal(t, 0, m.ui.History.UndoDepth(), "a refused nest leaves no undo step")
	})
	t.Run("> moves a category into the one above, < out of it", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusCategory, "v1")
		m.nestSelected(true)
		assert.Equal(t, []string{"v2", "ui<v2", "api<v2", "v1<v2"}, categoryOrder(m.project))

		selectRow(t, m, selection.FocusCategory, "ui")
		m.nestSelected(false)
		assert.Equal(t, []string{"v2", "api<v2", "v1<v2", "ui"}, categoryOrder(m.project))
		assert.Equal(t, "category:ui", cursorItem(t, m))
	})
	t.Run("a category with sub-categories doesn't nest", func(t *testing.T) {
		p := nestedProject()
		p.Categories = append([]domain.Category{{ID: "v3", Name: "v0.3.0"}}, p.Categories...)
		m := newTestModel(t, p)
		withStubStateManager(m)
		selectRow(t, m, selection.FocusCategory, "v2")
		m.nestSelected(true)
		assert.Contains(t, m.ui.Screen.StatusMsg, "sub-categories can't nest")
	})
	t.Run("a sibling name blocks the move", func(t *testing.T) {
		p := nestedProject()
		p.Categories[3].Name = "UI"
		p.Categories = append(p.Categories, domain.Category{ID: "ui2", Name: "ui"})
		m := newTestModel(t, p)
		withStubStateManager(m)
		selectRow(t, m, selection.FocusCategory, "ui")
		m.nestSelected(false)
		assert.Contains(t, m.ui.Screen.StatusMsg, `top-level category is already named "ui"`)
	})
	t.Run("visual > nests every selected task", func(t *testing.T) {
		p := sampleProject()
		m := newTestModel(t, p)
		selectRow(t, m, selection.FocusTask, "t2")
		m.enterVisualMode()
		m.visualMoveCursor(1)
		m.visualNest(true)
		assert.Equal(t, []string{"t1", "t2<t1", "t3<t1"}, shapeOf(m.project.Categories[0]))
		m.visualNest(false)
		assert.Equal(t, []string{"t1", "t2", "t3"}, shapeOf(m.project.Categories[0]))
	})
}

func TestBlockMoves(t *testing.T) {
	t.Run("J moves a parent with its subtasks", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "timer")
		m.moveSelectedRow(1)
		assert.Equal(t, []string{"sets", "timer", "pick<timer", "sound<timer"}, shapeOf(catByID(t, m, "ui")))
		assert.Equal(t, "task:timer", cursorItem(t, m))
	})
	t.Run("a subtask moves among its siblings only", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "pick")
		m.moveSelectedRow(1)
		assert.Equal(t, []string{"timer", "sound<timer", "pick<timer", "sets"}, shapeOf(catByID(t, m, "ui")))
		m.moveSelectedRow(1)
		assert.Equal(t, "Already at the bottom of its parent", m.ui.Screen.StatusMsg)
	})
	t.Run("K hops a parent's whole block", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sets")
		m.moveSelectedRow(-1)
		assert.Equal(t, []string{"sets", "timer", "pick<timer", "sound<timer"}, shapeOf(catByID(t, m, "ui")))
	})
	t.Run("a block crosses into the next category with its subtasks", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "timer")
		m.moveSelectedRow(1)
		m.moveSelectedRow(1)
		assert.Equal(t, []string{"sets"}, shapeOf(catByID(t, m, "ui")))
		assert.Equal(t, []string{"timer", "pick<timer", "sound<timer", "auth"}, shapeOf(catByID(t, m, "api")))
	})
	t.Run("J on a parent category moves its areas along", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusCategory, "v2")
		m.moveSelectedRow(1)
		assert.Equal(t, []string{"v1", "v2", "ui<v2", "api<v2"}, categoryOrder(m.project))
	})
	t.Run("a sub-category moves among its siblings only", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusCategory, "api")
		m.moveSelectedRow(-1)
		assert.Equal(t, []string{"v2", "api<v2", "ui<v2", "v1"}, categoryOrder(m.project))
		m.moveSelectedRow(-1)
		assert.Equal(t, "Already at the top of its parent", m.ui.Screen.StatusMsg)
	})
	t.Run("S reverses whole blocks", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "auth")
		m.reverseCategories()
		assert.Equal(t, []string{"v1", "v2", "ui<v2", "api<v2"}, categoryOrder(m.project))
		assert.Equal(t, "task:auth", cursorItem(t, m))
	})
}

func TestDeleteHandlesChildren(t *testing.T) {
	press := func(m *model, key string) *model {
		next, _ := m.handleKeyMsg(tea.KeyPressMsg{Text: key, Code: rune(key[0])})
		mm := next.(model)
		return &mm
	}
	t.Run("deleting a parent asks, then takes its subtasks", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "timer")
		m.deleteSelected()
		require.True(t, m.ui.Modes.IsConfirmDelete())
		assert.Contains(t, m.confirmDeleteView(), "its 2 subtasks")
		m = press(m, "y")
		assert.Equal(t, []string{"sets"}, shapeOf(catByID(t, m, "ui")))
		assert.Len(t, m.ui.Clipboard.Tasks, 3, "p puts the whole block back")
	})
	t.Run("k keeps the subtasks, top-level", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "timer")
		m.deleteSelected()
		m = press(m, "k")
		assert.Equal(t, []string{"pick", "sound", "sets"}, shapeOf(catByID(t, m, "ui")))
	})
	t.Run("k keeps a parent category's areas", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusCategory, "v2")
		m.deleteSelected()
		assert.Contains(t, m.confirmDeleteView(), "2 sub-categories")
		m = press(m, "k")
		assert.Equal(t, []string{"ui", "api", "v1"}, categoryOrder(m.project))
	})
	t.Run("y takes the areas along", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusCategory, "v2")
		m.deleteSelected()
		m = press(m, "y")
		assert.Equal(t, []string{"v1"}, categoryOrder(m.project))
		m.undo()
		assert.Equal(t, []string{"v2", "ui<v2", "api<v2", "v1"}, categoryOrder(m.project))
	})
	t.Run("k does nothing for a leaf", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sets")
		m.deleteSelected()
		m = press(m, "k")
		assert.True(t, m.ui.Modes.IsConfirmDelete())
	})
}

func TestAddNeverEditsAnotherRow(t *testing.T) {
	t.Run("a filter that would hide the new task refuses it", func(t *testing.T) {
		for _, anchor := range []string{"sets", "pick"} {
			m := newNestedModel(t)
			m.ui.Filter.statuses[domain.StatusCompleted] = true
			m.rebuildPositions()
			selectRow(t, m, selection.FocusTask, anchor)
			before := shapeOf(catByID(t, m, "ui"))
			m.startAddingTask()
			assert.False(t, m.ui.Modes.IsEdit(), anchor)
			assert.Equal(t, "Clear the filter to add a task here", m.ui.Screen.StatusMsg, anchor)
			assert.Equal(t, before, shapeOf(catByID(t, m, "ui")), anchor)
			assert.Equal(t, "ui", catByID(t, m, "ui").Name, anchor)
		}
	})
	t.Run("a folded category unfolds for the new task", func(t *testing.T) {
		m := newNestedModel(t)
		m.ui.Fold.Toggle("api")
		m.rebuildPositions()
		selectRow(t, m, selection.FocusCategory, "api")
		m.startAddingTask()
		assert.False(t, m.ui.Fold.IsFolded("api"))
		typeTitle(m, "Login")
		api := catByID(t, m, "api")
		assert.Equal(t, "api", api.Name)
		assert.Equal(t, "Login", api.Tasks[0].Title)
	})
	t.Run("a folded category unfolds for a new separator", func(t *testing.T) {
		m := newNestedModel(t)
		m.ui.Fold.Toggle("api")
		m.rebuildPositions()
		selectRow(t, m, selection.FocusCategory, "api")
		m.startAddingSeparator()
		assert.False(t, m.ui.Fold.IsFolded("api"))
		assert.Equal(t, "separator:"+catByID(t, m, "api").Tasks[0].ID, cursorItem(t, m))
	})
}

func TestAddedSubtaskTurnedSeparatorLeavesTheBlock(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusTask, "pick")
	m.startAddingTask()
	require.True(t, m.pasteLinesInEdit("# Heading\n- a"))
	ui := catByID(t, m, "ui")
	assert.Equal(t, []string{"timer", "pick<timer", "sound<timer"}, shapeOf(ui)[:3])
	assert.True(t, ui.Tasks[3].IsSeparator())
	assert.Equal(t, "a", ui.Tasks[4].Title)
	assertNestingValid(t, m.project)
}

func TestEditorPasteShowsSubtasksOfAFoldedParent(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Fold.Toggle("timer")
	m.rebuildPositions()
	selectRow(t, m, selection.FocusTask, "timer")
	m.startEditing()
	require.True(t, m.pasteLinesInEdit("- Rest timer\n  - vibrate"))
	assert.False(t, m.ui.Fold.IsFolded("timer"))
	ui := catByID(t, m, "ui")
	assert.Equal(t, "task:"+ui.Tasks[3].ID, cursorItem(t, m))
	assert.NotEmpty(t, ui.UpdatedAt)
}

func TestNestRefusesARowTheFilterHides(t *testing.T) {
	m := newTestModel(t, sampleProject())
	withStubStateManager(m)
	require.NoError(t, m.project.Categories[0].Tasks[2].SetStatus(domain.StatusTodo))
	m.ui.Filter.statuses[domain.StatusTodo] = true
	m.rebuildPositions()
	selectRow(t, m, selection.FocusTask, "t3")
	m.nestSelected(true)
	assert.Equal(t, "Can't nest: the filter hides the row above", m.ui.Screen.StatusMsg)
	assert.Equal(t, []string{"t1", "t2", "t3"}, shapeOf(m.project.Categories[0]))
	assert.Equal(t, 0, m.ui.History.UndoDepth())
}

func TestMergeOfferNamesTheSubCategories(t *testing.T) {
	p := nestedProject()
	p.Categories[3].Tasks = nil
	p.Categories = append(p.Categories, domain.Category{ID: "web", Name: "web", ParentID: "v1",
		Tasks: []domain.Task{{ID: "w1", Title: "W", Status: domain.StatusTodo}}})
	m := newTestModel(t, p)
	withStubStateManager(m)
	selectRow(t, m, selection.FocusCategory, "v1")
	m.startEditing()
	typeTitle(m, "v0.2.0")
	require.Equal(t, "v2", m.ui.Edit.mergeInto)
	view := m.confirmMergeView()
	assert.NotContains(t, view, "is empty")
	assert.Contains(t, view, "1 sub-category joins")
}

func TestReverseOfOneTopLevelBlockIsANoop(t *testing.T) {
	p := nestedProject()
	p.Categories = p.Categories[:3]
	m := newTestModel(t, p)
	withStubStateManager(m)
	m.reverseCategories()
	assert.Equal(t, 0, m.ui.History.UndoDepth())
	assert.Equal(t, []string{"v2", "ui<v2", "api<v2"}, categoryOrder(m.project))
}
