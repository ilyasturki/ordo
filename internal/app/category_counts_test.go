package app

import (
	"slices"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ilyasturki/ordo/internal/domain"
)

func countsProject() domain.Project {
	p := nestedProject()
	p.Categories = append(p.Categories,
		domain.Category{ID: "inbox", Name: "inbox", Tasks: []domain.Task{
			{ID: "i1", Title: "Idea", Status: domain.StatusInProgress},
			{ID: "i2", Title: "Dropped", Status: domain.StatusCancelled},
		}},
		domain.Category{ID: "empty", Name: "empty"},
	)
	return p
}

func setCategoryCounts(t *testing.T, m *model, on string) {
	t.Helper()
	i := slices.IndexFunc(optionSpecs, func(s optionSpec) bool { return s.name == "Category Counts" })
	require.GreaterOrEqual(t, i, 0)
	optionSpecs[i].apply(m, on)
}

func TestCategoryCountsAreOffByDefault(t *testing.T) {
	m := newTestModel(t, countsProject())
	withStubStateManager(m)
	lines := screen(m)
	for _, name := range []string{"v0.2.0", "▼ ui", "▼ api", "v0.1.0", "inbox"} {
		assert.NotRegexp(t, `\d+/\d+`, lineWith(t, lines, name), "no count on %s", name)
	}
	assert.Contains(t, lineWith(t, lines, "Rest timer"), "1/2", "subtask badges are not category counts")
}

func TestCategoryCountsOnEveryTopLevelCategoryWithTasks(t *testing.T) {
	m := newTestModel(t, countsProject())
	withStubStateManager(m)
	setCategoryCounts(t, m, "on")
	lines := screen(m)

	assert.Contains(t, lineWith(t, lines, "v0.2.0"), "[/] 2/6", "a parent counts its whole tree")
	assert.Contains(t, lineWith(t, lines, "v0.1.0"), "1/1")
	assert.Contains(t, lineWith(t, lines, "inbox"), "[/] 1/2", "a plain category counts too, cancelled as done")
	assert.NotRegexp(t, `\d+/\d+`, lineWith(t, lines, "empty"), "a category without tasks has nothing to count")
	assert.NotRegexp(t, `\d+/\d+`, lineWith(t, lines, "▼ ui"), "sub-categories stay without counts")
	assert.NotContains(t, strings.Join(lines, "\n"), "in progress", "the [/] badge already says it")

	setCategoryCounts(t, m, "off")
	assert.NotContains(t, lineWith(t, screen(m), "inbox"), "1/2")
}

func TestCategoryCountsOptionReflowsHeaders(t *testing.T) {
	// 30 columns leave 22 for the name beside " [ ]", and 18 once " 0/1" joins them.
	name := strings.Repeat("x", 22)
	m := newTestModel(t, domain.Project{ID: "p1", Name: "P", Categories: []domain.Category{
		{ID: "c", Name: name, Tasks: []domain.Task{{ID: "t", Title: "t", Status: domain.StatusTodo}}},
	}})
	withStubStateManager(m)
	m.ui.Screen.Width = 30
	height := func() int {
		for _, item := range m.buildLayout().Items {
			if item.Kind == LayoutCategory {
				return item.Height
			}
		}
		t.Fatal("no category row")
		return 0
	}

	require.Equal(t, 1, height())
	setCategoryCounts(t, m, "on")
	assert.Equal(t, 2, height(), "the layout cache drops the old row height")
}
