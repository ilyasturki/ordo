package app

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func nestedProject() domain.Project {
	todo, done, doing := domain.StatusTodo, domain.StatusCompleted, domain.StatusInProgress
	return domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{ID: "v2", Name: "v0.2.0", Tasks: []domain.Task{
				{ID: "loose", Title: "Loose", Status: todo},
			}},
			{ID: "ui", Name: "ui", ParentID: "v2", Tasks: []domain.Task{
				{ID: "timer", Title: "Rest timer", Status: todo},
				{ID: "pick", ParentID: "timer", Title: "Pick a default", Status: done},
				{ID: "sound", ParentID: "timer", Title: "Add a sound", Status: todo},
				{ID: "sets", Title: "Fix sets", Status: done},
			}},
			{ID: "api", Name: "api", ParentID: "v2", Tasks: []domain.Task{
				{ID: "auth", Title: "Auth", Status: doing},
			}},
			{ID: "v1", Name: "v0.1.0", Tasks: []domain.Task{
				{ID: "old", Title: "Old", Status: done},
			}},
		},
	}
}

// rowIDs lists the shown rows as "kind:id", the project row as "project:".
func rowIDs(m *model) []string {
	kinds := map[selection.FocusKind]string{
		selection.FocusProject:     "project",
		selection.FocusCategory:    "category",
		selection.FocusTask:        "task",
		selection.FocusSeparator:   "separator",
		selection.FocusDescription: "description",
	}
	var out []string
	for _, p := range m.positions() {
		out = append(out, kinds[p.Kind]+":"+p.ID)
	}
	return out
}

func newNestedModel(t *testing.T) *model {
	t.Helper()
	m := newTestModel(t, nestedProject())
	withStubStateManager(m)
	return m
}

func TestNestedRowsShowInOrder(t *testing.T) {
	m := newNestedModel(t)
	assert.Equal(t, []string{
		"project:", "category:v2", "task:loose",
		"category:ui", "task:timer", "task:pick", "task:sound", "task:sets",
		"category:api", "task:auth",
		"category:v1", "task:old",
	}, rowIDs(m))
}

func TestFoldingNestedItems(t *testing.T) {
	t.Run("a folded parent category hides its sub-categories", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sound")
		m.ui.Fold.Toggle("v2")
		m.rebuildPositions()
		assert.Equal(t, []string{"project:", "category:v2", "category:v1", "task:old"}, rowIDs(m))
		assert.Equal(t, "category:v2", cursorItem(t, m), "the cursor climbs to the nearest shown ancestor")
	})
	t.Run("tab on a subtask folds its parent", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sound")
		m.toggleFold()
		assert.True(t, m.ui.Fold.IsFolded("timer"))
		assert.Equal(t, "task:timer", cursorItem(t, m))
		assert.NotContains(t, rowIDs(m), "task:pick")
		assert.Contains(t, rowIDs(m), "task:sets")

		m.toggleFold()
		assert.False(t, m.ui.Fold.IsFolded("timer"))
		assert.Contains(t, rowIDs(m), "task:pick")
	})
	t.Run("tab on a leaf task folds its sub-category", func(t *testing.T) {
		m := newNestedModel(t)
		selectRow(t, m, selection.FocusTask, "sets")
		m.toggleFold()
		assert.True(t, m.ui.Fold.IsFolded("ui"))
		assert.Equal(t, "category:ui", cursorItem(t, m))
		assert.Contains(t, rowIDs(m), "category:api")
	})
	t.Run("fold state is saved by ID", func(t *testing.T) {
		m := newTestModel(t, nestedProject())
		stub := withStubStateManager(m)
		selectRow(t, m, selection.FocusTask, "timer")
		m.toggleFold()
		assert.Equal(t, []string{"timer"}, stub.folded["p1"])
	})
	t.Run("zc folds sub-categories too and zo unfolds tasks", func(t *testing.T) {
		m := newNestedModel(t)
		m.ui.Fold.Toggle("timer")
		m.foldAll()
		assert.True(t, m.ui.Fold.IsFolded("ui"))
		assert.True(t, m.ui.Fold.IsFolded("v2"))
		m.unfoldAll()
		assert.False(t, m.ui.Fold.HasFolded())
	})
}

func TestFilterShowsAParentForContext(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Filter.statuses[domain.StatusCompleted] = true
	m.rebuildPositions()
	rows := rowIDs(m)
	assert.Contains(t, rows, "task:timer", "the open parent of a matching subtask stays")
	assert.Contains(t, rows, "task:pick")
	assert.NotContains(t, rows, "task:sound")
	assert.NotContains(t, rows, "task:loose")
}

func TestCategoryFilterTakesSubCategories(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Filter.categories["v2"] = true
	m.rebuildPositions()
	rows := rowIDs(m)
	assert.Contains(t, rows, "task:loose")
	assert.Contains(t, rows, "task:auth", "picking a parent picks its areas")
	assert.NotContains(t, rows, "task:old")
}

func TestSearchRevealsAFoldedSubtask(t *testing.T) {
	m := newNestedModel(t)
	m.ui.Fold.Toggle("timer")
	m.ui.Fold.Toggle("v2")
	m.rebuildPositions()
	selectRow(t, m, selection.FocusProject, "")

	m.ui.Search.query = "sound"
	m.searchNext()
	assert.Equal(t, "task:sound", cursorItem(t, m))
	assert.False(t, m.ui.Fold.IsFolded("timer"))
	assert.False(t, m.ui.Fold.IsFolded("v2"))
}

func screen(m *model) []string {
	return strings.Split(ansi.Strip(m.renderView()), "\n")
}

func lineWith(t *testing.T, lines []string, text string) string {
	t.Helper()
	for _, l := range lines {
		if strings.Contains(l, text) {
			return l
		}
	}
	require.Failf(t, "no line", "no line holds %q in:\n%s", text, strings.Join(lines, "\n"))
	return ""
}

func column(line, text string) int {
	return ansi.StringWidth(line[:strings.Index(line, text)])
}

func TestNestedRowsIndent(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusProject, "")
	lines := screen(m)

	top := column(lineWith(t, lines, "v0.2.0"), "▼")
	sub := column(lineWith(t, lines, "▼ ui"), "▼")
	assert.Equal(t, top+nestIndent, sub, "a sub-category sits a level in")

	loose := column(lineWith(t, lines, "Loose"), "[")
	timer := column(lineWith(t, lines, "Rest timer"), "[")
	pick := column(lineWith(t, lines, "Pick a default"), "[")
	assert.Equal(t, loose+nestIndent, timer, "a sub-category's tasks sit a level in")
	assert.Equal(t, timer+nestIndent, pick, "a subtask sits a level in")
}

func TestParentsShowProgress(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusProject, "")
	lines := screen(m)
	assert.Contains(t, lineWith(t, lines, "Rest timer"), "1/2")
	assert.Contains(t, lineWith(t, lines, "v0.2.0"), "2/6", "a parent category counts its areas' tasks")
	assert.NotContains(t, lineWith(t, lines, "Fix sets"), "/")

	m.ui.Fold.Toggle("timer")
	m.rebuildPositions()
	assert.Contains(t, lineWith(t, screen(m), "Rest timer"), foldedMark+" 1/2")
}

func TestParentCategoryWithOnlyAreasShowsNoPlaceholder(t *testing.T) {
	p := nestedProject()
	p.Categories[0].Tasks = nil
	m := newTestModel(t, p)
	selectRow(t, m, selection.FocusProject, "")
	assert.NotContains(t, strings.Join(screen(m), "\n"), "(no tasks)")
}

func TestCategoryJumpsStopAtEveryHeader(t *testing.T) {
	m := newNestedModel(t)
	selectRow(t, m, selection.FocusTask, "loose")
	var visited []string
	for range 4 {
		m.jumpToNextCategory()
		visited = append(visited, cursorItem(t, m))
	}
	assert.Equal(t, []string{"category:ui", "category:api", "category:v1", "category:v1"}, visited)
	m.jumpToPrevCategory()
	m.jumpToPrevCategory()
	assert.Equal(t, "category:ui", cursorItem(t, m))
	m.jumpToLast()
	assert.Equal(t, "task:old", cursorItem(t, m))
}

func TestSubtaskDescriptionKeepsTheCapAndExpands(t *testing.T) {
	p := nestedProject()
	body := strings.Repeat("line\n", 30) + "last"
	p.Categories[1].Tasks[2].Description = body
	m := newTestModel(t, p)
	m.ui.Screen.Height = 40
	m.ui.Screen.ExpandDescriptions = true
	m.rebuildPositions()

	selectRow(t, m, selection.FocusDescription, "sound")
	desc := lineWith(t, screen(m), "line")
	assert.Equal(t, column(lineWith(t, screen(m), "Add a sound"), "Add"), column(desc, "line")-2, "the bar sits flush with the subtask's title")
	start, end, ok := m.buildLayout().rowRange(m.selected())
	require.True(t, ok)
	assert.Equal(t, maxFocusedDescriptionRows+1, end-start, "a focused description is capped, plus its tail row")

	selectRow(t, m, selection.FocusTask, "sound")
	m.toggleExpandRow()
	selectRow(t, m, selection.FocusDescription, "sound")
	start, end, _ = m.buildLayout().rowRange(m.selected())
	assert.Equal(t, 31, end-start, "ze shows the subtask's description whole")
}

func TestPickerCountsNestedTasks(t *testing.T) {
	open, inProgress := projectStats(nestedProject())
	assert.Equal(t, 4, open, "loose, timer, sound and auth are open")
	assert.Equal(t, 1, inProgress)
}

func TestExpandDescriptionsKeepsTheCursorOnASubtask(t *testing.T) {
	p := nestedProject()
	p.Categories[1].Tasks[0].Description = "why"
	m := newTestModel(t, p)
	selectRow(t, m, selection.FocusTask, "sound")
	m.toggleExpandDescriptions()
	assert.Equal(t, "task:sound", cursorItem(t, m))
}
