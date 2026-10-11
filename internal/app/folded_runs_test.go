package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/app/selection"
	"github.com/ilyasturki/ordo/internal/domain"
)

func foldedRunsProject() domain.Project {
	done := func(id string) []domain.Task {
		return []domain.Task{{ID: id, Title: "task " + id, Status: domain.StatusCompleted}}
	}
	return domain.Project{
		ID:   "p1",
		Name: "P",
		Categories: []domain.Category{
			{ID: "a", Name: "v0.0.1", Tasks: done("a1")},
			{ID: "b", Name: "v0.0.2", Tasks: done("b1")},
			{ID: "c", Name: "v0.0.3", Tasks: done("c1")},
			{ID: "now", Name: "now", Tasks: []domain.Task{{ID: "n1", Title: "open work", Status: domain.StatusTodo}}},
			{ID: "area", Name: "area"},
			{ID: "ui", Name: "ui", ParentID: "area", Tasks: done("u1")},
			{ID: "ux", Name: "ux", ParentID: "area", Tasks: done("x1")},
			{ID: "lone", Name: "lone", Tasks: done("l1")},
			{ID: "tail", Name: "tail", Tasks: []domain.Task{{ID: "t1", Title: "last task", Status: domain.StatusTodo}}},
		},
	}
}

func newFoldedRunsModel(t *testing.T) *model {
	t.Helper()
	m := newTestModel(t, foldedRunsProject())
	withStubStateManager(m)
	for _, id := range []string{"a", "b", "c", "ui", "ux", "lone"} {
		m.ui.Fold.Toggle(id)
	}
	m.rebuildPositions()
	return m
}

func lineOf(t *testing.T, lines []string, name string) int {
	t.Helper()
	i := slices.IndexFunc(lines, func(l string) bool {
		return strings.Contains(l, "▶ "+name+" ") || strings.Contains(l, "▼ "+name+" ")
	})
	require.GreaterOrEqual(t, i, 0, "no header for %q in\n%s", name, strings.Join(lines, "\n"))
	return i
}

func TestFoldedRunsStackOneHeaderPerRow(t *testing.T) {
	lines := screen(newFoldedRunsModel(t))
	a := lineOf(t, lines, "v0.0.1")
	assert.Equal(t, a+1, lineOf(t, lines, "v0.0.2"))
	assert.Equal(t, a+2, lineOf(t, lines, "v0.0.3"))
	assert.Equal(t, "", strings.TrimSpace(lines[a-1]), "a blank line still parts the run from the project line")
	assert.Equal(t, "", strings.TrimSpace(lines[a+3]), "a blank line still parts the run from the open category")
	assert.Equal(t, a+4, lineOf(t, lines, "now"))
	assert.NotContains(t, strings.Join(lines, "\n"), "(folded)")
}

func TestFoldedSubCategoriesStackAtTheirIndent(t *testing.T) {
	lines := screen(newFoldedRunsModel(t))
	ui := lineOf(t, lines, "ui")
	assert.Equal(t, ui+1, lineOf(t, lines, "ux"))
	assert.True(t, strings.HasPrefix(lines[ui], "    ▶ ui"), "sub-category keeps its indent: %q", lines[ui])
	assert.True(t, strings.HasPrefix(lines[ui+1], "    ▶ ux"), "sub-category keeps its indent: %q", lines[ui+1])
	assert.Equal(t, "", strings.TrimSpace(lines[ui-1]), "the run sits one blank line under its parent")
}

func TestLoneFoldedCategoryKeepsItsGapsButNoPlaceholder(t *testing.T) {
	lines := screen(newFoldedRunsModel(t))
	lone := lineOf(t, lines, "lone")
	assert.Equal(t, "", strings.TrimSpace(lines[lone-1]), "a top-level category never stacks under another parent's sub-categories")
	assert.Equal(t, "", strings.TrimSpace(lines[lone+1]))
	assert.Equal(t, lone+2, lineOf(t, lines, "tail"))
}

func TestUnfoldingSplitsTheRun(t *testing.T) {
	m := newFoldedRunsModel(t)
	m.ui.Selection.MoveTo(slices.IndexFunc(m.positions(), func(p selection.Position) bool { return p.ID == "b" }))
	m.toggleFold()
	lines := screen(m)
	a, b, c := lineOf(t, lines, "v0.0.1"), lineOf(t, lines, "v0.0.2"), lineOf(t, lines, "v0.0.3")
	assert.Equal(t, a+2, b, "the unfolded category gets its gap back")
	assert.Contains(t, lines[b+2], "task b1")
	assert.Equal(t, b+4, c, "the category after it starts a new run")
}

func TestVisualOverAFoldedRunIsOneBand(t *testing.T) {
	m := newFoldedRunsModel(t)
	m.ui.Selection.MoveTo(slices.IndexFunc(m.positions(), func(p selection.Position) bool { return p.ID == "a" }))
	m.enterVisualMode()
	m.visualMoveCursor(2)
	lines := screen(m)
	a := lineOf(t, lines, "v0.0.1")
	for i := a; i <= a+2; i++ {
		assert.Equal(t, m.ui.Screen.Width, ansi.StringWidth(lines[i]), "row %d is padded to the full width: %q", i, lines[i])
	}
	assert.Equal(t, a+2, lineOf(t, lines, "v0.0.3"))
}
