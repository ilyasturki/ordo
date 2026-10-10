package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"phasionary/internal/app/selection"
	"phasionary/internal/domain"
)

// assertNestingValid checks the nesting rules hold across the project.
func assertNestingValid(t *testing.T, p domain.Project) {
	t.Helper()
	want := domain.Project{Categories: make([]domain.Category, len(p.Categories))}
	for i, c := range p.Categories {
		c.Tasks = append([]domain.Task(nil), c.Tasks...)
		want.Categories[i] = c
	}
	domain.NormalizeNesting(&want)
	assert.Equal(t, categoryOrder(want), categoryOrder(p), "categories break the nesting rules")
	for i := range p.Categories {
		assert.Equal(t, shapeOf(want.Categories[i]), shapeOf(p.Categories[i]), "tasks of %s break the nesting rules", p.Categories[i].ID)
	}
}

func TestCutPasteMovesAParentWithItsSubtasks(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusTask, "timer")
	m.cutSelectedTask()
	assert.Contains(t, m.ui.Screen.StatusMsg, "2 subtasks")
	selectRow(t, m, selection.FocusTask, "auth")
	m.pasteInternal()

	assert.Equal(t, []string{"sets"}, shapeOf(catByID(t, m, "ui")))
	assert.Equal(t, []string{"auth", "timer", "pick<timer", "sound<timer"}, shapeOf(catByID(t, m, "api")))
	assert.Equal(t, "task:timer", cursorItem(t, m))
	assertNestingValid(t, m.project)
}

func TestCutPasteTakesSubtasksAddedSinceTheCut(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusTask, "timer")
	m.cutSelectedTask()
	ui, _ := m.project.FindCategoryByID("ui")
	m.project.Categories[ui].Tasks = append(m.project.Categories[ui].Tasks[:3:3],
		domain.Task{ID: "late", ParentID: "timer", Title: "Late", Status: domain.StatusTodo},
		m.project.Categories[ui].Tasks[3])
	m.rebuildPositions()
	selectRow(t, m, selection.FocusTask, "auth")
	m.pasteInternal()
	assert.Equal(t, []string{"auth", "timer", "pick<timer", "sound<timer", "late<timer"}, shapeOf(catByID(t, m, "api")))
	assertNestingValid(t, m.project)
}

func TestPasteLandsAtTheCursorsDepth(t *testing.T) {
	t.Run("a leaf pasted among subtasks becomes one", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sets")
		m.cutSelectedTask()
		selectRow(t, m, selection.FocusTask, "pick")
		m.pasteInternal()
		assert.Equal(t, []string{"timer", "pick<timer", "sets<timer", "sound<timer"}, shapeOf(catByID(t, m, "ui")))
	})
	t.Run("a subtask pasted at the top level becomes top-level", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "pick")
		m.cutSelectedTask()
		selectRow(t, m, selection.FocusTask, "sets")
		m.pasteInternal()
		assert.Equal(t, []string{"timer", "sound<timer", "sets", "pick"}, shapeOf(catByID(t, m, "ui")))
	})
	t.Run("a parent pasted among subtasks lands after their parent", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "timer")
		m.copySelected()
		selectRow(t, m, selection.FocusTask, "pick")
		m.pasteInternal()
		ui := catByID(t, m, "ui")
		require.Len(t, ui.Tasks, 7)
		assert.Equal(t, []string{"timer", "pick<timer", "sound<timer"}, shapeOf(ui)[:3])
		copied := ui.Tasks[3:6]
		assert.Equal(t, "Rest timer", copied[0].Title)
		assert.NotEqual(t, "timer", copied[0].ID, "a copy gets fresh IDs")
		assert.Empty(t, copied[0].ParentID)
		assert.Equal(t, copied[0].ID, copied[1].ParentID, "the copied subtasks follow the copied parent")
		assert.Equal(t, copied[0].ID, copied[2].ParentID)
		assert.Equal(t, "sets", ui.Tasks[6].ID)
		assertNestingValid(t, m.project)
	})
	t.Run("pasting a cut onto itself keeps it in place", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "timer")
		m.cutSelectedTask()
		selectRow(t, m, selection.FocusTask, "sound")
		m.pasteInternal()
		assert.Equal(t, []string{"timer", "pick<timer", "sound<timer", "sets"}, shapeOf(catByID(t, m, "ui")))
	})
}

func TestVisualCutTakesFoldedSubtasks(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Fold.Toggle("timer")
	m.rebuildPositions()
	selectRow(t, m, selection.FocusTask, "timer")
	m.enterVisualMode()
	m.visualMoveCursor(1)
	m.visualCut()
	assert.Equal(t, []string{"timer", "pick", "sound", "sets"}, m.ui.Clipboard.TaskIDs)

	selectRow(t, m, selection.FocusTask, "loose")
	m.pasteInternal()
	assert.Equal(t, []string{"loose", "timer", "pick<timer", "sound<timer", "sets"}, shapeOf(catByID(t, m, "v2")))
	assert.Empty(t, catByID(t, m, "ui").Tasks)
	assertNestingValid(t, m.project)
}

func TestVisualCopyTextIndentsSubtasks(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusTask, "timer")
	m.enterVisualMode()
	assert.Equal(t, "- Rest timer\n  - Pick a default\n  - Add a sound", m.visualMarkdownText(m.visualSelectedPositions(), false))
	assert.Equal(t, "- [ ] Rest timer\n  - [x] Pick a default\n  - [ ] Add a sound", m.visualMarkdownText(m.visualSelectedPositions(), true))
}

func TestVisualDeleteTakesSubtasks(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusTask, "timer")
	m.enterVisualMode()
	m.visualDelete()
	assert.Contains(t, m.confirmDeleteView(), "1 task(s) and 2 subtask(s)")
	m.confirmDeleteVisualRange()
	assert.Equal(t, []string{"sets"}, shapeOf(catByID(t, m, "ui")))
}

func TestCategoryClipboardCarriesAreas(t *testing.T) {
	t.Run("a copied parent pastes with fresh, linked areas", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusCategory, "v2")
		m.enterVisualMode()
		m.visualCopyBullets()
		assert.Equal(t, []string{"v2", "ui", "api"}, m.ui.Clipboard.CategoryIDs)

		selectRow(t, m, selection.FocusCategory, "v1")
		m.pasteInternal()
		require.Len(t, m.project.Categories, 7)
		copied := m.project.Categories[3]
		assert.Equal(t, "v0.2.0 (2)", copied.Name)
		assert.Equal(t, copied.ID, m.project.Categories[4].ParentID)
		assert.Equal(t, "ui", m.project.Categories[4].Name, "an area keeps its name under a new parent")
		sub := m.project.Categories[4]
		require.Len(t, sub.Tasks, 4)
		assert.Equal(t, sub.Tasks[0].ID, sub.Tasks[1].ParentID, "subtasks follow their copied parent")
		assertNestingValid(t, m.project)
	})
	t.Run("a cut area pasted at the top level becomes top-level", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusCategory, "ui")
		m.enterVisualMode()
		m.visualCut()
		selectRow(t, m, selection.FocusCategory, "v1")
		m.pasteInternal()
		assert.Equal(t, []string{"v2", "api<v2", "ui", "v1"}, categoryOrder(m.project))
	})
	t.Run("a cut area pasted among another parent's areas joins them", func(t *testing.T) {
		p := nestedProject()
		p.Categories = append(p.Categories, domain.Category{ID: "web", Name: "web", ParentID: "v1"})
		m := newTestModel(t, p)
		withStubStateManager(m)
		selectRow(t, m, selection.FocusCategory, "api")
		m.enterVisualMode()
		m.visualCut()
		selectRow(t, m, selection.FocusCategory, "web")
		m.pasteInternal()
		assert.Equal(t, []string{"v2", "ui<v2", "v1", "api<v1", "web<v1"}, categoryOrder(m.project))
	})
}
